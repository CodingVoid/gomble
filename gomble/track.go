package gomble

import (
    "regexp"
    "time"
    "errors"
    "io"
    "fmt"
    "os"
    "github.com/CodingVoid/gomble/logger"
    "github.com/CodingVoid/gomble/gomble/audioformats"
    "github.com/CodingVoid/gomble/gomble/tracksources"
    "github.com/CodingVoid/gomble/gomble/tracksources/youtube"
)

type Track struct {
    trackSrc tracksources.TrackSource
    // should be a multiple of audioformats.OPUS_FRAME_DURATION
    buffer_ms int
    // Never ever change that outside of this source file
    Done bool
}

var stop chan bool = make(chan bool)
var pause chan bool = make(chan bool)
var currTrack *Track

const (
    TRACK_TYPE_YOUTUBE = iota
    TRACK_TYPE_OGGFILE
)

var ytregex *regexp.Regexp = regexp.MustCompile(`https://www.youtube.com/watch\?v=([a-zA-Z0-9\-\_]+)`) // need to use ` character otherwise \character are recognized as escape characters

func LoadTrack(url string) (*Track, error) {
    ytmatches := ytregex.FindStringSubmatch(url)
    if len(ytmatches) > 0 {
        surl := ytmatches[1]
        var err error
        var src tracksources.TrackSource
        if _, err := os.Stat("/bin/youtube-dl"); err == nil {
            // use youtube-dl if it exists
            src, err = youtube.NewYoutubedlVideo(surl)
        } else {
            // otherwise use native youtube stream implementation (probably doesn't work, but worth a try)
            // src, err = youtube.NewYoutubeVideo(surl)
            logger.Fatalf("No Youtube-dl installed. exiting\n")
        }
        if err != nil {
            return nil, err
        }
        return &Track{
            trackSrc: src,
        }, nil
    }
    return nil, errors.New("LoadTrack (track.go): No Youtube Video Found under URL");
}

func (t *Track) GetTitle() string {
    return t.trackSrc.GetTitle()
}

func GetCurrentTrack() *Track {
    return currTrack
}

func Play(t *Track) bool { // {{{
    if t.Done || currTrack != nil {
        return false
    }
    currTrack = t
    t.buffer_ms = 500 // buffering of 500 ms should be enough, I think...
    go audioroutine(t)
    return true
} // }}}

// Stops the current Track if one is playing
func Stop() {
    stop <- true
}

// Pauses the current Track if one is playing
func Pause() {
    pause <- true
}

func Resume() {
    pause <- false
}

// This audioroutine gets called whenever a new audio stream should be played
func audioroutine(t *Track) { // {{{
    enc, err := audioformats.NewOpusEncoder() // initializes a new encoder
    if err != nil {
        eventpuffer <- TrackExceptionEvent{
            Track:  t,
            err: fmt.Errorf("Could not create Opus Encoder: %v\n", err),
        }
        return
    }

    opusbuf := make(chan []byte, t.buffer_ms / audioformats.OPUS_FRAME_DURATION) // make channel with buffer_ms/frame_duration number of frames as buffer
    // our producer
    go func () {
        for true {
            opusPayload, err := getNextOpusFrame(t, enc)
            if err != nil {
                logger.Errorf("Could not get next opus frame: %v\n", err)
                opusbuf <- nil
                return
            }
            if opusPayload == nil {
                // close channel instead of writing nil to channel (cleaner i think...)
                close(opusbuf)
                return
            }
            select {
            case opusbuf <- opusPayload:
                break
            case <- stop:
                return
            }
        }
    }()

    timer := time.NewTicker((audioformats.OPUS_FRAME_DURATION) * time.Millisecond)
    var last bool
    //file, err := os.OpenFile("send.opus", os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
    //if err != nil {
    //	logger.Fatal("...")
    //}
    loop:for range timer.C {
        select {
        case <- stop:
            t.Done = true
            currTrack = nil
            eventpuffer <- TrackStoppedEvent{
                Track:  t,
            }
            break loop
        case <- pause:
            eventpuffer <- TrackPausedEvent {
                Track: t,
            }
            select {
            case <- stop:
                t.Done = true
                currTrack = nil
                eventpuffer <- TrackStoppedEvent{
                    Track:  t,
                }
                break loop
            case <- pause:
                // Track resumed
                //TODO Check if pause is true or false (pause or resume)
            }
        default:
        }
        logger.Debugf("Get next opus frame, number of frames in buffer: %d\n", len(opusbuf))
        lastTime := time.Now()
        opusPayload, ok := <-opusbuf
        elapsed := time.Since(lastTime)
        if elapsed.Microseconds() > 4000 {
            logger.Warnf("elapsed time from getting next opus frame: %s\n", elapsed)
        }
        if !ok {
            // Channel got closed -> track is done
            t.Done = true
            currTrack = nil
            eventpuffer <- TrackFinishedEvent {
                Track: t,
            }
            break
        }
        if opusPayload == nil {
            // happens if there was an error getting the opus payload from our producer
            t.Done = true
            currTrack = nil
            eventpuffer <- TrackExceptionEvent {
                Track: t,
            }
            break
        }
        sendAudioPacket(opusPayload, uint16(len(opusPayload)), last)

        //file.Write(opusPayload)
    }
    //WriteInt16InFile("pcm.final", allpcm)
} // }}}

//var allpcm []int16
func getNextOpusFrame(t *Track, encoder *audioformats.OpusEncoder) ([]byte, error) { // {{{
    pcm, err := t.trackSrc.GetPCMFrame(audioformats.OPUS_FRAME_DURATION)
    if err != nil {
        if err == io.EOF {
            return nil, nil
        }
        return nil, fmt.Errorf("Could not get PCM-Frame: %v", err)
    }
    opus, err := encoder.Encode(pcm)
    if err != nil {
        return nil, fmt.Errorf("Could not encode PCM-Frame: %v", err)
    }
    return opus, nil
} // }}}

