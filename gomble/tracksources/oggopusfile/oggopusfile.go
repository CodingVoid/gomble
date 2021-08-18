package oggopusfile

/*
#cgo LDFLAGS: -lopusfile
#include <opus/opus_defines.h>
#include <opus/opusfile.h>
*/
import "C"
import "unsafe"
import "errors"
import "strconv"
import "io"
import "runtime"
import "fmt"
import "github.com/CodingVoid/gomble/logger"

type OggOpusfile struct {
    path        string
    pcmbuffer   []int16
    doneReading bool
    opFile      *C.struct_OggOpusFile
}

// Opens the Opus File for later Decoding (see DecodeOpusFile)
// path The path to the file to open.
func NewOggOpusfile(path string) (*OggOpusfile, error) { // {{{
    var f OggOpusfile
    f.path = path
    f.pcmbuffer = make([]int16, 0)

    var err int32
    errPointer := (*C.int)(unsafe.Pointer(&err))

    f.opFile = C.op_open_file(C.CString(path), errPointer)
    if err != 0 {
        //C.free(unsafe.Pointer(oggFile))
        return nil, getOpusFileError("", err)
    }
    //C.free(unsafe.Pointer(errPointer))
    return &f, nil
} // }}}

//Get the total PCM length (number of samples at 48 kHz) of the stream, or of an individual link in a (possibly-chained) Ogg Opus stream.
//	link	The index of the link whose PCM length should be computed. Use a negative number to get the PCM length of the entire stream.
//	Returns The PCM length of the entire stream if _li is negative, the PCM length of link _li if it is non-negative
func (o *OggOpusfile) GetTotalPCMSamples(link int) (int64, error) { // {{{
    var csamples C.ogg_int64_t
    csamples = C.op_pcm_total(o.opFile, C.int(link))
    samples := int64(csamples)

    if samples < 0 {
        _, file, line, _ := runtime.Caller(0)
        return samples, fmt.Errorf("GetPCMFrame(%s:%d): %s", file, line, getOpusFileError("", int32(samples)))
    }
    return samples, nil
} // }}}

// Returns channel count of the audiosource
func (o *OggOpusfile) GetChannels() (int, error) {
    return 2, nil // I currently read each link of opusfile with stereo (op_read_stereo). Even if the original file is mono, it is returned as stereo
}

// Get a slice of PCM Data of Opus File
func (o *OggOpusfile) GetPCMFrame(duration int) ([]int16, error) { // {{{
    neededSamples := duration * 48 // duration * 48kHz sampleRate
    for len(o.pcmbuffer) < neededSamples*2 && !o.doneReading {
        var pcm [11520]int16
        cpcm := (*C.opus_int16)(unsafe.Pointer(&pcm[0]))

        cntSamples := C.op_read_stereo(o.opFile, cpcm, C.int(len(pcm))) // returns the readed samples per channel
        if cntSamples < 0 {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("GetPCMFrame(%s:%d): %s", file, line, getOpusFileError("", int32(cntSamples)))
        }
        if cntSamples == 0 {
            //logger.Debugf("End-Of-File reached or Buffer to small\n");
            o.doneReading = true
            break
        }
        mono := make([]int16, cntSamples)
        // Convert Stereo to Mono
        for j := 0; j < int(cntSamples); j++ {
            mono[j] = (pcm[j*2])
        }
        o.pcmbuffer = append(o.pcmbuffer, mono...)
    }
    if len(o.pcmbuffer) < neededSamples*2 {
        ret := make([]int16, len(o.pcmbuffer))
        copy(ret, o.pcmbuffer[:])
        logger.Debugf("GetPCMFrame EOF\n")
        return ret, io.EOF // return rest of pcmbuffer
    }
    logger.Debugf("Returned PCM length: %d pcmbuffer length: %d\n", neededSamples, len(o.pcmbuffer))
    ret := make([]int16, neededSamples)
    copy(ret, o.pcmbuffer[:neededSamples])
    o.pcmbuffer = o.pcmbuffer[neededSamples:]
    return ret, nil
} // }}}

//func OpRead(oggFile OggOpusFile, pcm []int16, pcmlen uint32) (int32, error) {
//	var ccntSamples C.int
//	cpcm := (*C.opus_int16)(unsafe.Pointer(&pcm[0]))
//	ccntSamples = C.op_read(oggFile, cpcm, C.int(pcmlen), nil) // returns the readed samples per channel
//	cntSamples := int32(ccntSamples)
//	if (cntSamples < 0) {
//		return cntSamples, getOpusFileError("Failure op_read: ", cntSamples)
//	}
//	if (cntSamples == 0) {
//		//logger.Debugf("End-Of-File reached or Buffer to small\n");
//		return 0, nil
//	}
//	return cntSamples, nil
//}

func (o *OggOpusfile) GetTitle() string {
    return "example title"
}

func getOpusFileError(prefix string, errorCode int32) error { // {{{
    str := prefix
    switch errorCode {
    case 0:
        str += "SUCCESS"
    case C.OP_FALSE:
        str += "OP_FALSE: A request did not succeed"
    case C.OP_EOF:
        str += "OP_EOF: Currently not used externally"
    case C.OP_HOLE:
        str += "OP_HOLE:There was a hole in the page sequence numbers (e.g., a page was corrupt or missing"
    case C.OP_EREAD:
        str += "OP_EREAD: An underlying read, seek, or tell operation failed when it should have succeeded"
    case C.OP_EFAULT:
        str += "OP_EFAULT: A <code>NULL</code> pointer was passed where one was unexpected, or an internal memory allocation failed, or an internal library error was encountered"
    case C.OP_EIMPL:
        str += "OP_EIMPL: The stream used a feature that is not implemented, such as an unsupported channel family"
    case C.OP_EINVAL:
        str += "One or more parameters to a function were invalid"
    case C.OP_ENOTFORMAT:
        str += "A purported Ogg Opus stream did not begin with an Ogg page, a purported header packet did not start with one of the required strings, 'OpusHead' or 'OpusTags', or a link in a chained file was encountered that did not contain any logical Opus streams."
    case C.OP_EBADHEADER:
        str += "A required header packet was not properly formatted, contained illegal values, or was missing altogether."
    case C.OP_EVERSION:
        str += "The ID header contained an unrecognized version number."
    case C.OP_ENOTAUDIO:
        str += "Currently not used at all."
    case C.OP_EBADPACKET:
        str += "An audio packet failed to decode properly. This is usually caused by a multistream Ogg packet where the durations of the individual Opus packets contained in it are not all the same."
    case C.OP_EBADLINK:
        str += "We failed to find data we had seen before, or the bitstream structure was sufficiently malformed that seeking to the target destination was impossible."
    case C.OP_ENOSEEK:
        str += "An operation that requires seeking was requested on an unseekable stream."
    case C.OP_EBADTIMESTAMP:
        str += "The first or last granule position of a link failed basic validity checks."
    default:
        str += "unknown errorCode: " + strconv.Itoa(int(errorCode))
    }
    return errors.New(str)
} // }}}
