// go wrapper for c opus library. All c functions have a 'w' prefix (wrapper function)
package audioformats

/*
#cgo LDFLAGS: -lopus
#include <opus/opus.h>
#include <opus/opus_defines.h>

static int wopusEncoderCtlSetBitrate(OpusEncoder *st, int bitrate) {
    return opus_encoder_ctl(st, OPUS_SET_BITRATE(bitrate));
}

static int wopusEncoderCtlSetVbr(OpusEncoder *st, int vbr) {
    return opus_encoder_ctl(st, OPUS_SET_VBR(vbr));
}
*/
import "C"
import "unsafe"
import "errors"
import "strconv"

// Possible constants to use
const (
	// Best for most VoIP/videoconference applications where listening quality and intelligibility matter most
	OPUS_APPICATION_VOIP = 2048

	// Best for broadcast/high-fidelity application where the decoded audio should be as close as possible to the input
	OPUS_APPLICATION_AUDIO = 2049

	// Only use when lowest-achievable latency is what matters most. Voice-optimized modes cannot be used.
	OPUS_APPLICATION_RESTRICTEDLOWDELAY = 2051

	// Signal being encoded is voice
	OPUS_SIGNAL_VOICE = 3001
	// Signal being encoded is music
	OPUS_SIGNAL_MUSIC = 3002
)

// Settings for encoding to opus
const (
	// samplerate of 48kHz is opus default setting
	OPUS_SAMPLE_RATE = 48000

	// number of channels to use (2 for Stereo, 1 for Mono).
	OPUS_CHANNELS = 2

	//OPUS_BITRATE=128000 // Opus at 128 KB/s (VBR) is pretty much transparentk
	//-1 means MAX_BITRATE
	// MAX_BITRATE means libopus will use as much space it can put in opus packets. So it's controlled by the MAX_PACKET_SIZE in some way.
	OPUS_BITRATE = 40000

	// size in pcm samples (1 sample is int16) (2 bytes = 1 sample) (480 frame_size = 960 bytes
	// samplerate * frame_duration = frame_size = number of pcm sampels in one frame per channel
	OPUS_PCM_FRAME_SIZE = 480 // 48000*10ms = 480
	OPUS_FRAME_DURATION = 10  // 10 ms

	//	OPUS_MAX_FRAME_SIZE=6*960 only for decoding
	//OPUS_MAX_PACKET_SIZE = 1275

	// The Application to use for opus. libopus will automatically
	OPUS_APPLICATION = OPUS_APPLICATION_AUDIO

	// Variable Bit Rate, if set to one libopus will automatically change the bitrate as it sees fit.
	OPUS_VBR = 0
)

type OpusEncoder struct {
	// the underlying c encoder of libopus
	cencoder *C.struct_OpusEncoder
	// memory where the c encoder is allocated. managed by GO Garbace Collector
	mem []byte
	// number of channels in which to encode the pcm data to opus encoding
	channels int
}

type OpusDecoder struct {
	// the underlying c decoder of libopus
	cdecoder *C.struct_OpusDecoder
	// memory where the c decoder is allocated (managed by GO Garbage Collector)
	mem []byte

	sample_rate int32
	channels    int
}

//[in]	Fs	opus_int32: Sampling rate of input signal (Hz) This must be one of 8000, 12000, 16000, 24000, or 48000.
//[in]	channels	int: Number of channels (1 or 2) in input signal
//[in]	application	int: Coding mode (OPUS_APPLICATION_VOIP/OPUS_APPLICATION_AUDIO/OPUS_APPLICATION_RESTRICTED_LOWDELAY). This parameter is currently ignored and instead OPUS_APPLICATION_AUDIO will be used
//[out]	error	int*: Error codes
func NewOpusEncoder() (*OpusEncoder, error) {
	var enc OpusEncoder
	//var err int
	//enc = C.opus_encoder_create(C.int(Fs), C.int(channels), C.int(C.gopus_application_audio), errPtr)
	size := C.opus_encoder_get_size(C.int(OPUS_CHANNELS))
	enc.mem = make([]byte, size)
	enc.cencoder = (*C.OpusEncoder)(unsafe.Pointer(&enc.mem[0]))
	err := C.opus_encoder_init(enc.cencoder, C.opus_int32(OPUS_SAMPLE_RATE), C.int(OPUS_CHANNELS), C.int(OPUS_APPLICATION))
	if err != C.OPUS_OK {
		return nil, getOpusError("", int32(err))
	}
	enc.channels = OPUS_CHANNELS
	if err := enc.CtlSetBitrate(OPUS_BITRATE); err != nil {
		return nil, err
	}
	if err := enc.CtlSetVbr(OPUS_VBR); err != nil {
		return nil, err
	}
	return &enc, nil
}

func NewOpusDecoder(samplerate int32, channels int) (*OpusDecoder, error) {
	var dec OpusDecoder
	if channels != 1 && channels != 2 {
		return nil, errors.New("channels must be mono or stereo")
	}
	size := C.opus_decoder_get_size(C.int(channels))
	dec.sample_rate = samplerate
	dec.channels = channels
	dec.mem = make([]byte, size)
	dec.cdecoder = (*C.OpusDecoder)(unsafe.Pointer(&dec.mem[0]))
	err := C.opus_decoder_init(dec.cdecoder, C.opus_int32(samplerate), C.int(channels))
	if err != C.OPUS_OK {
		return nil, getOpusError("", int32(err))
	}
	return &dec, nil
}

func (enc *OpusEncoder) CtlSetBitrate(bitrate int) error {
	err := C.wopusEncoderCtlSetBitrate(enc.cencoder, C.int(bitrate))
	if err < 0 {
		return getOpusError("Gopus_encoder_ctl_set_bitrate error: ", int32(err))
	}
	return nil
}

func (enc *OpusEncoder) CtlSetVbr(vbr int) error {
	err := C.wopusEncoderCtlSetVbr(enc.cencoder, C.int(vbr))
	if err < 0 {
		return getOpusError("Gopus_encoder_ctl_set_vbr error: ", int32(err))
	}
	return nil
}

//[in]	st	OpusEncoder*: Encoder state
//[in]	pcm	opus_int16*: Input signal (interleaved if 2 channels). length is frame_size*channels*sizeof(opus_int16)
//[in]	frame_size	int: Number of samples per channel in the input signal. This must be an Opus frame size for the encoder's sampling rate. For example, at 48 kHz the permitted values are 120, 240, 480, 960, 1920, and 2880. Passing in a duration of less than 10 ms (480 samples at 48 kHz) will prevent the encoder from using the LPC or hybrid modes.
//[out]	data	unsigned char*: Output payload. This must contain storage for at least max_data_bytes.
//[in]	max_data_bytes	opus_int32: Size of the allocated memory for the output payload. This may be used to impose an upper limit on the instant bitrate, but should not be used as the only bitrate control. Use OPUS_SET_BITRATE to control the bitrate.
// returns The length of the encoded packet (in bytes) on success or a negative error code (see Error codes) on failure.
func (enc *OpusEncoder) Encode(pcm []int16) ([]byte, error) {
	var data [512]byte
	samples := len(pcm) / enc.channels

	n := C.opus_encode(enc.cencoder, (*C.opus_int16)(&pcm[0]), C.int(samples), (*C.uchar)(&data[0]), C.opus_int32(len(data)))
	ngo := int32(n)

	if ngo < 0 {
		return nil, getOpusError("", int32(ngo))
	}

	return data[:ngo], nil
}

func (dec *OpusDecoder) Decode(opus []byte) ([]int16, error) {
	var pcm [2 * 2880]int16 //MAX_CHANNELS * MAX_FRAME_SIZE
	n := C.opus_decode(
		dec.cdecoder,
		(*C.uchar)(&opus[0]),
		C.opus_int32(len(opus)),
		(*C.opus_int16)(&pcm[0]),
		C.int(len(pcm)/dec.channels),
		0)
	ngo := int32(n)
	if ngo < 0 {
		return nil, getOpusError("", ngo)
	}
	return pcm[:ngo*int32(dec.channels)], nil
}

// get the frame size from an opus packet
func GetPacketFrameSize(samplerate int, packet []byte) (int, error) { // {{{
	framecount := 2
	switch packet[0] & 0x03 {
	case 0:
		framecount = 1
	case 3:
		if len(packet) < 2 {
			return -1, errors.New("Invalid opus Frame")
		} else {
			framecount = int(packet[1] & 0x3F)
		}
	}

	samplesPerFrame := 0
	shiftBits := (packet[0] >> 3) & 0x03
	if (packet[0] & 0x80) != 0 {
		samplesPerFrame = (samplerate << shiftBits) / 400
	} else if (packet[0] & 0x60) == 0x60 {
		if (packet[0] & 0x08) != 0 {
			samplesPerFrame = samplerate / 50
		} else {
			samplesPerFrame = samplerate / 100
		}
	} else if shiftBits == 3 {
		samplesPerFrame = samplerate * 60 / 1000
	} else {
		samplesPerFrame = int(packet[0]<<shiftBits) / 100
	}
	samples := framecount * samplesPerFrame
	return samples, nil
} // }}}

func getOpusError(prefix string, errorCode int32) error {
	str := prefix
	switch errorCode {
	case C.OPUS_OK:
		str += "No error"
	case C.OPUS_BAD_ARG:
		str += "One or more invalid/out of range arguments"
	case C.OPUS_BUFFER_TOO_SMALL:
		str += "Not enough bytes allocated in the buffer"
	case C.OPUS_INTERNAL_ERROR:
		str += "An internal error was detected"
	case C.OPUS_INVALID_PACKET:
		str += "The compressed data passed is corrupted"
	case C.OPUS_UNIMPLEMENTED:
		str += "Invalid/unsupported request number"
	case C.OPUS_INVALID_STATE:
		str += "An encoder or decoder structure is invalid or already freed"
	case C.OPUS_ALLOC_FAIL:
		str += "Memory allocation has failed"
	default:
		str += "unknown errorCode: " + strconv.Itoa(int(errorCode))
	}
	return errors.New(str)
}
