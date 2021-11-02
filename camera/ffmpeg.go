package camera

import (
	"fmt"

	"github.com/brutella/hc/rtp"
	"github.com/duncanleo/hc-camera-ffmpeg/hsv"
)

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
	case OMX:
		encoder = "h264_omx"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("scale=%d:-1", videoAttributes.ImageWidth),
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
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
		hksvVideoProfile(videoCodecParams, encoderProfile),
		"-level:v",
		hksvVideoCodecLevel(videoCodecParams),
		"-r",
		fmt.Sprintf("%d", videoAttributes.FrameRate),
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
			// "-map",
			// "0:1",
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
		"100000",
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
		}
		break
	case OMX:
		encoder = "h264_omx"
		encoderOpts = []string{
			"-vf",
			fmt.Sprintf("scale=%d:-1", streamCfg.Video.Attributes.Width),
		}
		break
	case VAAPI:
		inputOpts = []string{
			"-vaapi_device",
			"/dev/dri/renderD128",
			"-hwaccel",
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
		streamVideoProfile(streamCfg),
		"-level:v",
		streamVideoCodecLevel(streamCfg),
		"-r",
		fmt.Sprintf("%d", streamCfg.Video.Attributes.Framerate),
	)

	args = append(args, encoderOpts...)
	args = append(args,
		"-preset",
		"veryfast",
	)

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
			"-b:a",
			fmt.Sprintf("%dk", streamCfg.Audio.RTP.Bitrate),
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
