package youtube

import (
    "github.com/CodingVoid/gomble/gomble/audioformats"
    "github.com/CodingVoid/gomble/gomble/container/matroska"
    "github.com/CodingVoid/gomble/logger"
    "encoding/json"
    "os/exec"
    "runtime"
    "fmt"
    "io"
)

type YoutubedlVideo struct {
    // used for json unmarshal of youtube-dl output
    Id string
    Title string
    Formats []YoutubedlFormat

    // used internally to get audio data
    matroskacont *matroska.Matroska
    blockOffset  int
    pcmbuff      []int16
    pcmbuffoff   int
    timeoffset   int
    // opus decoder
    dec         *audioformats.OpusDecoder
    doneReading bool
}

type YoutubedlFormat struct {
    Asr float64
    Filesize float64
    Url string
    Acodec string
    Ext string
}

func NewYoutubedlVideo(path string) (*YoutubedlVideo, error) {// {{{
    cmd := exec.Command("youtube-dl", "-j", path)
    buf, err := cmd.Output()
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    var video YoutubedlVideo
    err = json.Unmarshal(buf, &video)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    format := getBestFormat(video.Formats[:])
    if format == nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }

    logger.Debugf("Youtube Url: %s", format.Url)
    // Now create a new youtube stream with our final url
    ystream, err := NewPersistentYoutubeStream(format.Url, int(format.Filesize))
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }

    odec, err := audioformats.NewOpusDecoder(48000, 2)

    matroskcont := matroska.NewMatroska(ystream)
    matroskcont.ReadHeader()

    video.matroskacont = matroskcont
    video.dec = odec
    video.pcmbuff = make([]int16, 0, 960)
    video.doneReading = false

    return &video, nil
}// }}}

func getBestFormat(formats []YoutubedlFormat) *YoutubedlFormat {// {{{
    for _,v := range formats {
        if (v.Acodec == "opus" && v.Ext == "webm" && v.Asr == 48000) {
            return &v
        }
    }
    return nil
}// }}}

// Returns a Mono PCMFrame with the given duration in milliseconds
func (y *YoutubedlVideo) GetPCMFrame(duration int) ([]int16, error) { // {{{
    logger.Debugf("Called GetPCMFrame\n")
    neededSamples := 48 * duration * audioformats.OPUS_CHANNELS // 48kHz * duration in ms
    // wait till we have the necessary pcm samples and buffer if possible
    for len(y.pcmbuff) < neededSamples && !y.doneReading {
        nextFrame, err := y.matroskacont.GetNextFrames(1)
        if err != nil {
            if err != io.EOF {
                _, file, line, _ := runtime.Caller(0)
                return nil, fmt.Errorf("GetPCMFrame(%s:%d): %w", file, line, err)
            }
            y.doneReading = true
        }
        logger.Debugf("len(nextFrame): %d\n", len(nextFrame))
        //samples, err := audioformats.GetPacketFrameSize(48000, nextFrame[0].Audiodata)
        //if err != nil {
        //	return nil, err
        //}
        //frameduration := 48000 / samples
        for i := 0; i < len(nextFrame); i++ {
            pcm, err := y.dec.Decode(nextFrame[i].Audiodata)
            if err != nil {
                _, file, line, _ := runtime.Caller(0)
                return nil, fmt.Errorf("GetPCMFrame(%s:%d): %w", file, line, err)
            }
            //mono := make([]int16, len(pcm)/2)
            //// Convert Stereo to Mono
            //for j := 0; j < len(pcm)/2; j++ {
            //    mono[j] = (pcm[j*2]) //+ pcm[i*2+1]) / 2 // to take the average of both channels did really not work as expected... sounded awfull...
            //}
            //logger.Debugf("append mono length: %d to pcmbuff\n", len(mono))
            y.pcmbuff = append(y.pcmbuff, pcm...)
        }
    }
    // if we still don't have enough samples, we probably have reached EOF and return the remaining samples
    if len(y.pcmbuff) < neededSamples {
        ret := make([]int16, len(y.pcmbuff))
        copy(ret, y.pcmbuff[:])
        logger.Debugf("GetPCMFrame EOF\n")
        return ret, io.EOF // return rest of pcmbuffer
    }
    logger.Debugf("Returned PCM length: %d pcmbuffer length: %d\n", neededSamples, len(y.pcmbuff))
    ret := make([]int16, neededSamples)
    copy(ret, y.pcmbuff[:neededSamples])
    y.pcmbuff = y.pcmbuff[neededSamples:]
    return ret, nil
} // }}}

func (y *YoutubedlVideo) GetTitle() string {// {{{
    return y.Title
}// }}}

// youtube-dl -j
