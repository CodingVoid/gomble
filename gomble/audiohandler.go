package gomble

import "time"
import "io"

//import "os"
import "fmt"

import "github.com/CodingVoid/gomble/gomble/audiosources"
import "github.com/CodingVoid/gomble/gomble/audioformats"
import "github.com/CodingVoid/gomble/logger"

type Audiohandler struct {
	playing bool
}

var audiohandler Audiohandler

//param track The track to start playing, passing nil and interrupt true will stop the current track and return false
//param interrupt Whether to only start if nothing else is playing, passing an audiosource and interrupt false (and a audiotrack is currently playing) will return false and not start the track
//return True if the track was started
func Play(track audiosources.Audiosource, interrupt bool) bool { // {{{
	if interrupt {
		Stop()
	}
	if !interrupt && audiohandler.playing || track == nil {
		return false
	}
	audiohandler.playing = true
	go audioroutine(track)

	return true
} // }}}

// Stops the current Track if one is playing
func Stop() { // {{{
	audiohandler.playing = false
} // }}}

// This audioroutine gets called whenever a new audio stream should be played
func audioroutine(track audiosources.Audiosource) { // {{{
	enc, err := audioformats.NewOpusEncoder(audioformats.OPUS_SAMPLE_RATE, audioformats.OPUS_CHANNELS, audioformats.OPUS_APPLICATION) // initializes a new encoder
	if err != nil {
		logger.Errorf("Could not create Opus Encoder. End Track\n") //TODO break and raise done/exception event with error parameter or whatsoever
		eventpuffer <- TrackEndedEvent{
			Track:  track,
			Reason: TRACK_OTHER,
		}
		return
	}
	timer := time.NewTicker((audioformats.OPUS_FRAME_DURATION) * time.Millisecond)
	var last bool
	//file, err := os.OpenFile("send.opus", os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0666)
	//if err != nil {
	//	logger.Fatal("...")
	//}
	for range timer.C {

		if audiohandler.playing == false {
			logger.Infof("track/audiosource wurde unterbrochen\n")
			eventpuffer <- TrackEndedEvent{
				Track:  track,
				Reason: TRACK_INTERRUPTED,
			}
			break
		}
		logger.Debugf("Get next opus frame\n")
		lastTime := time.Now()
		opusPayload, err := getNextOpusFrame(track, enc)
		elapsed := time.Since(lastTime)
		if elapsed.Microseconds() > 4000 {
			logger.Debugf("elapsed time: %s\n", elapsed)
		}
		if err != nil {
			logger.Errorf("Could not get next opus frame: %v\n", err)
			break
		}
		if opusPayload == nil {
			// track is done
			break
		}
		sendAudioPacket(opusPayload, uint16(len(opusPayload)), last)

		//file.Write(opusPayload)
	}
	//WriteInt16InFile("pcm.final", allpcm)
	audiohandler.playing = false
	logger.Infof("Done playing Track\n")
	eventpuffer <- TrackEndedEvent{
		Track:  track,
		Reason: TRACK_ENDED,
	}
} // }}}

//var allpcm []int16
func getNextOpusFrame(track audiosources.Audiosource, encoder *audioformats.OpusEncoder) ([]byte, error) { // {{{
	//pcm, err := track.GetPCMFrame(audioformats.OPUS_PCM_FRAME_SIZE * audioformats.OPUS_CHANNELS)
	pcm, err := track.GetPCMFrame(audioformats.OPUS_FRAME_DURATION)
	//allpcm = append(allpcm, pcm...)
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
