package camera

import (
	"fmt"
	"strings"
	"time"

	"github.com/brutella/hc/rtp"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
)

const (
	FragmentDuration = 100 * time.Millisecond
)

func generateMotherStreamArguments(inputCfg InputConfiguration, encoderProfile EncoderProfile) []string {
	var inputOpts []string
	var encoder string
	var encoderOpts []string

	var width = 1920

	switch encoderProfile {
	case CPU:
		encoder = "h264"
		encoderOpts = []string{
			"-x264-params",
			"intra-refresh=1:bframes=0",
			"-vf",
			fmt.Sprintf("scale=%d:-1", width),
			"-preset",
			"veryfast",
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
			"vaapi",
			"-hwaccel_output_format",
			"vaapi",
		}
		encoder = "h264_vaapi"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("format=nv12|vaapi,hwupload,scale_vaapi=w=%d:h=-1", width),
			"-bf",
			"0",
		}
		break
	}
	var args []string
	args = append(args, inputOpts...)

	if strings.HasPrefix(inputCfg.Source, "rtsp://") {
		args = append(args,
			"-rtsp_transport",
			"tcp",
		)
	}

	args = append(
		args,
		"-f",
		inputCfg.Format,
		"-protocol_whitelist",
		protocolWhitelist,
		"-ss",
		"00:00:01.000",
		"-i",
		inputCfg.Source,
		"-c:v",
		encoder,
		"-profile:v",
		"high",
		"-level:v",
		"4",
		"-r",
		fmt.Sprintf("%d", 30),
		"-b:v",
		fmt.Sprintf("%dk", 5000),
	)

	args = append(args, encoderOpts...)

	if inputCfg.TimestampOverlay {
		args = append(
			args,
			"-filter_complex",
			"drawtext=text='time\\: %{localtime}':fontcolor=white",
		)
	}

	var audioCodecFfmpeg = "aac"

	if inputCfg.Audio {
		args = append(
			args,
			"-c:a",
			audioCodecFfmpeg,
			"-ar",
			fmt.Sprintf("%d", 44100),
		)
	}

	// Output arguments
	args = append(
		args,
		"-f",
		"mp4",
		"-movflags",
		"frag_keyframe+empty_moov+default_base_moof",
		"-frag_duration",
		fmt.Sprintf("%d", int64(FragmentDuration/time.Microsecond)),
		"pipe:1",
	)

	return args
}

func generateHKSVArguments(inputCfg InputConfiguration, encoderProfile EncoderProfile, recordCfg hsv.SelectedCameraRecordingConfiguration) []string {
	var inputOpts []string
	var encoder string
	var encoderOpts []string

	var videoAttributes = recordCfg.SelectedVideoConfiguration.VideoAttributes[0]
	var videoCodecParams = recordCfg.SelectedVideoConfiguration.VideoCodecParameters[0]
	var audioCodecParams = recordCfg.SelectedAudioConfiguration.AudioCodecParameters[0]
	var audioCodec = recordCfg.SelectedAudioConfiguration.Codec

	switch encoderProfile {
	case CPU:
		encoder = "h264"
		encoderOpts = []string{
			"-x264-params",
			"intra-refresh=1:bframes=0",
			"-vf",
			fmt.Sprintf("scale=%d:-1", videoAttributes.ImageWidth),
			"-preset",
			"veryfast",
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
			"vaapi",
			"-hwaccel_output_format",
			"vaapi",
		}
		encoder = "h264_vaapi"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("format=nv12|vaapi,hwupload,scale_vaapi=w=%d:h=-1", videoAttributes.ImageWidth),
			"-bf",
			"0",
		}
		break
	}
	var args []string
	args = append(args, inputOpts...)

	if strings.HasPrefix(inputCfg.Source, "rtsp://") {
		args = append(args,
			"-rtsp_transport",
			"tcp",
		)
	}

	args = append(
		args,
		"-f",
		"mp4",
		"-protocol_whitelist",
		protocolWhitelist,
		"-i",
		"pipe:0",
		"-c:v",
		encoder,
		"-profile:v",
		hksvVideoProfile(videoCodecParams, encoderProfile),
		"-level:v",
		hksvVideoCodecLevel(videoCodecParams),
		"-r",
		fmt.Sprintf("%d", videoAttributes.FrameRate),
		"-b:v",
		fmt.Sprintf("%dk", videoCodecParams.Bitrate),
	)

	args = append(args, encoderOpts...)

	if inputCfg.TimestampOverlay {
		args = append(
			args,
			"-filter_complex",
			"drawtext=text='time\\: %{localtime}':fontcolor=white",
		)
	}

	var audioCodecFfmpeg = hksvAudioCodec(audioCodec)

	if inputCfg.Audio {
		args = append(
			args,
			"-c:a",
			audioCodecFfmpeg,
			"-ar",
			fmt.Sprintf("%d", hksvAudioSampleRate(audioCodecParams)),
		)
		args = append(args, hksvAudioCodecOptions(audioCodec)...)
	}

	// Output arguments
	args = append(
		args,
		"-f",
		"mp4",
		"-movflags",
		"frag_keyframe+empty_moov+default_base_moof",
		"-frag_duration",
		fmt.Sprintf("%d", int64(FragmentDuration/time.Microsecond)),
		"pipe:1",
	)

	return args
}

func generateArguments(inputCfg InputConfiguration, streamCfg rtp.StreamConfiguration, se rtp.SetupEndpoints, encoderProfile EncoderProfile) []string {
	var inputOpts []string
	var encoder string
	var encoderOpts []string
	switch encoderProfile {
	case CPU:
		encoder = "h264"
		encoderOpts = []string{
			"-x264-params",
			"intra-refresh=1:bframes=0",
			"-vf",
			fmt.Sprintf("scale=%d:-1", streamCfg.Video.Attributes.Width),
			"-preset",
			"veryfast",
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
			"vaapi",
			"-hwaccel_output_format",
			"vaapi",
		}
		encoder = "h264_vaapi"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("format=nv12|vaapi,hwupload,scale_vaapi=w=%d:h=-1", streamCfg.Video.Attributes.Width),
			"-bf",
			"0",
		}
		break
	}
	var args []string

	args = append(args, inputOpts...)

	if strings.HasPrefix(inputCfg.Source, "rtsp://") {
		args = append(args,
			"-rtsp_transport",
			"tcp",
		)
	}

	args = append(
		args,
		"-f",
		"mp4",
		"-protocol_whitelist",
		protocolWhitelist,
		"-i",
		"pipe:0",
		"-c:v",
		encoder,
		"-profile:v",
		streamVideoProfile(streamCfg),
		"-level:v",
		streamVideoCodecLevel(streamCfg),
		"-r",
		fmt.Sprintf("%d", streamCfg.Video.Attributes.Framerate),
	)

	args = append(args, encoderOpts...)

	if inputCfg.TimestampOverlay {
		args = append(
			args,
			"-filter_complex",
			"drawtext=text='time\\: %{localtime}':fontcolor=white",
		)
	}

	args = append(
		args,
		"-payload_type",
		fmt.Sprintf("%d", streamCfg.Video.RTP.PayloadType),
		"-ssrc",
		"1",
		"-map",
		"0:0",
		"-f",
		"rtp",
		"-srtp_out_suite",
		"AES_CM_128_HMAC_SHA1_80",
		"-b:v",
		fmt.Sprintf("%dk", streamCfg.Video.RTP.Bitrate),
		"-srtp_out_params",
		se.Video.SrtpKey(),
		fmt.Sprintf("srtp://%s:%d?rtcpport=%d&localrtcpport=%d&pkt_size=%d&timeout=60", se.ControllerAddr.IPAddr, se.ControllerAddr.VideoRtpPort, se.ControllerAddr.VideoRtpPort, se.ControllerAddr.VideoRtpPort, streamVideoMTP(se)),
	)

	var audioCodec = "libopus"
	if inputCfg.AudioAAC {
		audioCodec = streamAudioCodec(streamCfg)
	}

	if inputCfg.Audio {
		args = append(
			args,
			"-payload_type",
			fmt.Sprintf("%d", streamCfg.Audio.RTP.PayloadType),
			"-ssrc",
			"2",
			"-c:a",
			audioCodec,
		)

		args = append(args,
			streamAudioBitrate(streamCfg.Audio.CodecParams)...,
		)

		args = append(args,
			"-map",
			"0:1",
			"-f",
			"rtp",
			"-ar",
			fmt.Sprintf("%d", streamAudioSampleRate(streamCfg)),
		)
		args = append(args, streamAudioCodecOptions(streamCfg)...)
		args = append(
			args,
			"-srtp_out_suite",
			"AES_CM_128_HMAC_SHA1_80",
		)

		args = append(args,
			"-frame_duration",
			"20",
			"-srtp_out_params",
			se.Audio.SrtpKey(),
			fmt.Sprintf("srtp://%s:%d?rtcpport=%d&localrtcpport=%d&pkt_size=%d&timeout=60", se.ControllerAddr.IPAddr, se.ControllerAddr.AudioRtpPort, se.ControllerAddr.AudioRtpPort, se.ControllerAddr.AudioRtpPort, 3768),
		)
	}

	return args
}

func generateSnapshotArguments(inputCfg InputConfiguration, width uint) []string {
	var args = []string{
		"-f",
		inputCfg.Format,
		"-protocol_whitelist",
		protocolWhitelist,
		"-ss",
		"00:00:01.000",
		"-i",
		inputCfg.Source,
		"-c:v",
		"png",
		"-vframes",
		"1",
		"-vsync",
		"vfr",
		"-compression_level",
		"50",
		"-video_size",
		fmt.Sprintf("%d:-2", width),
		"-f",
		"image2pipe",
		"-",
	}
	return args
}
