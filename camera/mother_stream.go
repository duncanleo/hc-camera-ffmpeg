package camera

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
	"github.com/duncanleo/hc-camera-ffmpeg/mp4"
)

var (
	initChunks    = make([]mp4.Chunk, 0)
	prebufferData = make([]mp4.Chunk, 0)

	prebufferDataMaxLength = int(hsv.PrebufferLengthStandard/FragmentDuration)*2 + int((2*time.Second)/FragmentDuration)*2 // padding

	liveStreamConsumers = make(map[string]io.Writer)
)

func motherStream(inputCfg InputConfiguration, encoderProfile EncoderProfile) error {
	var args = generateMotherStreamArguments(inputCfg, encoderProfile)

	var ffmpegProcess = exec.Command(
		"ffmpeg",
		args...,
	)

	log.Println(ffmpegProcess.String())

	ffmpegOut, err := ffmpegProcess.StdoutPipe()
	if err != nil {
		return err
	}

	if isDebugEnabled {
		ffmpegProcess.Stderr = os.Stdout
	}

	ffmpegOutBuffer := bufio.NewReaderSize(ffmpegOut, 1000000)

	go func() {
		defer ffmpegOut.Close()

		if ffmpegProcess.ProcessState != nil && ffmpegProcess.ProcessState.Exited() {
			return
		}

		var collectedChunks = make([]mp4.Chunk, 0)

		for {
			if isDebugEnabled {
				log.Println("[MOTHER STREAM] Waiting for data. Buffer size=", ffmpegOutBuffer.Buffered())
			}

			var chunkHeader = make([]byte, 8)
			_, err = io.ReadFull(ffmpegOutBuffer, chunkHeader)
			if err != nil {
				log.Println(err)
				return
			}

			if isDebugEnabled {
				log.Println("[MOTHER STREAM] Read chunk header", chunkHeader)
			}

			var chunkTypeBytes = chunkHeader[4:]
			var chunkType = string(chunkTypeBytes)

			var prependChunkData = make([]byte, 0)

			if chunkType == "ftyp" {
				// Must read the sub-type as well
				prependChunkData = make([]byte, 4)

				_, err = io.ReadFull(ffmpegOutBuffer, prependChunkData)
				if err != nil {
					log.Println(err)
					return
				}

				log.Println("[MOTHER STREAM] Read chunk sub-type", string(prependChunkData))
			}

			var chunkSizeBytes = chunkHeader[0:4]
			var chunkSizeComplete = binary.BigEndian.Uint32(chunkSizeBytes)
			var chunkSize = chunkSizeComplete - uint32(len(chunkHeader)) - uint32(len(prependChunkData))

			if isDebugEnabled {
				log.Printf("[MOTHER STREAM] Chunk type=%s size=%d", chunkType, chunkSize)
			}

			// Sanity check
			switch chunkType {
			case "ftyp", "mdat", "moov", "pnot", "udta", "uuid", "moof", "free", "skip", "jP2 ", "wide", "load", "ctab", "imap", "matt", "kmat", "clip", "crgn", "sync", "chap", "tmcd", "scpt", "ssrc", "PICT":
			default:
				log.Println("[MOTHER STREAM] Unknown chunk type", chunkType, "discarding")
				discardCount, err := ffmpegOutBuffer.Discard(int(chunkSize))
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("[MOTHER STREAM] Discarded", discardCount, "bytes")
				continue
			}

			var chunkData = make([]byte, chunkSize)
			_, err = io.ReadFull(ffmpegOutBuffer, chunkData)
			if err != nil {
				log.Println(err)
				return
			}

			var chunk = mp4.Chunk{
				Size:     chunkSizeComplete,
				MainType: chunkType,
				SubType:  string(prependChunkData),
				Data:     chunkData,
			}

			collectedChunks = append(collectedChunks, chunk)

			if len(collectedChunks) == 2 {
				if isDebugEnabled {
					log.Println("[MOTHER STREAM] Flushing")
				}

				switch collectedChunks[0].MainType {
				case "ftyp":
					initChunks = collectedChunks
				default:
					if len(prebufferData)+2 > prebufferDataMaxLength {
						prebufferData = prebufferData[2:]
					}

					prebufferData = append(prebufferData, collectedChunks...)
				}

				collectedChunks = make([]mp4.Chunk, 0)

				if isDebugEnabled {
					log.Printf("[MOTHER STREAM] Prebuffer data %d/%d\n", len(prebufferData), prebufferDataMaxLength)
				}
			}

			if isDebugEnabled {
				log.Printf("[MOTHER STREAM] Writing to %d consumers\n", len(liveStreamConsumers))
			}

			dat, _ := chunk.Assemble()
			for _, consumer := range liveStreamConsumers {
				_, err := consumer.Write(dat)
				if err != nil {
					log.Println(err)
				}
			}
		}

	}()

	err = ffmpegProcess.Start()
	if err != nil {
		return err
	}

	log.Println("[MOTHER STREAM] Spawn PID", ffmpegProcess.Process.Pid)

	defer func() {
		if ffmpegProcess.ProcessState != nil && !ffmpegProcess.ProcessState.Exited() {
			log.Println("[MOTHER STREAM] Terminating PID", ffmpegProcess.Process.Pid)
			ffmpegProcess.Process.Kill()
		}
	}()

	return nil
}
