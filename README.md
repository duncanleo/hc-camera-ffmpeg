# hc-camera-ffmpeg
This is a DIY HomeKit IP Camera accessory, built with [hc](https://github.com/brutella/hc)

### Usage
```shell
Usage of hc-camera-ffmpeg:
  -cameraAudio
    	whether the camera has audio
  -cameraFormat string
    	input for the camera (default "v4l2")
  -cameraInput string
    	input for the camera (default "/dev/video0")
  -encoderProfile string
    	encoder profile for FFMPEG. Accepts: CPU, OMX, VAAPI (default "CPU")
  -manufacturer string
    	manufacturer for the HomeKit Camera (default "Raspberry Pi Foundation")
  -model string
    	model for the HomeKit Camera (default "Camera Module")
  -name string
    	name for the HomeKit Camera (default "hc-doorbell")
  -pin string
    	pairing PIN for the accessory) (default "00102003")
  -port string
    	port for the HC accessory, leave empty to randomise
  -storagePath string
    	storage path (default "hc-camera-ffmpeg-storage")
```

### Encoder Profiles
Three encoder profiles are supported:

#### CPU
This uses the x264 encoder with no acceleration. This works on all computers and usually produces the best result, but it uses up the CPU.

#### OMX
This utilises the Raspberry Pi's GPU for accelerated encoding.

#### VAAPI
This utilises the VAAPI standard for accelerated decoding and encoding.

