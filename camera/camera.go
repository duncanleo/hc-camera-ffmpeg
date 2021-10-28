package camera

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/rtp"
	"github.com/brutella/hc/service"
)

type EncoderProfile int

const (
	_                  = iota
	CPU EncoderProfile = 1 << (10 * iota)
	OMX
	VAAPI
)

const (
	protocolWhitelist = "file,udp,tcp,rtp,http"
)

var (
	debug          = os.Getenv("DEBUG")
	isDebugEnabled = debug == "*" || debug == "ffmpeg"
)

type InputConfiguration struct {
	Source           string
	Format           string
	Audio            bool
	AudioAAC         bool
	TimestampOverlay bool
}





	})

	})




			},
	})



	var snapshot = func(width, height uint) (*image.Image, error) {
		var args = generateSnapshotArguments(inputCfg, width)

		var ffmpegProcess = exec.Command(
			"ffmpeg",
			args...,
		)
		var stdoutPipe bytes.Buffer

		log.Println(ffmpegProcess.String())

		ffmpegProcess.Stdout = &stdoutPipe
		if isDebugEnabled {
			ffmpegProcess.Stderr = os.Stderr
		}
		err := ffmpegProcess.Run()
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(&stdoutPipe)
		return &img, err
	}

	return camera, snapshot
}
