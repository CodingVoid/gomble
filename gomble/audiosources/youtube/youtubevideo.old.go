package youtube

//import (
//	"bytes"
//	"io"
//
//	"os/exec"
//	"errors"
//
//	"gomble/logger"
//)
//// after much time spent in trial and error I found out that youtube-dl does seem to ignore any kind of parameter if output is set to stdout (-o -), although I can't find any of this in the documentation. Furthermore most problems I got had their origin in using youtube-dl for downloading the video/audio. It's safer to use youtube-dl only as plain downloading and outputting to stdout of anything it can find under this url:
//// youtube-dl -o - --rm-cache-dir --quiet {url}
//
//// -i -			takes the input of stdin
//// -ac 1		downmixes to mono
//// -ar 48000	tells ffmpeg to resample the audio to 48000Hz
//// -f s16le		sets the output format to signed 16 bit little endian pcm audio
//// ffmpeg -i - -ac 1 -ar 48000 -f s16le -
//type youtubevideo struct {
//	ycmd *exec.Cmd
//	ycmdError bytes.Buffer
//	fcmd *exec.Cmd
//	fcmdError bytes.Buffer
//
//	pcmbuffer bytes.Buffer
//	pcmReaded int
//	pcmWritten int
//
//	url string
//
//	done chan bool
//}
//
//func NewYoutubeVideo(url string) (*youtubevideo, error) {
//	var yv youtubevideo
//	yv.ycmd = exec.Command("youtube-dl", "--quiet", "--rm-cache-dir", "-o", "-",  url)
//	// "--audio-quality", "4", url) // got some weird problems with that. it caused youtube-dl to print some 403 forbidden errors SOMETIMES, actually like every 4th call or so it worked totally random...
//	//ycmd := exec.Command("youtube-dl", "--quiet", "-o", "-", "--extract-audio", url) // "--audio-quality", "4", url) // got some weird problems with that. it caused youtube-dl to print some 403 forbidden errors SOMETIMES, actually like every 4th call or so it worked totally random...
//
//	yv.fcmd = exec.Command("ffmpeg", "-i", "-", "-ac", "1", "-ar", "48000", "-f", "s16le", "-")
//
//	r, w := io.Pipe()
//	yv.ycmd.Stdout = w
//	yv.fcmd.Stdin = r
//	//yv.pcmbuffer = //&bytes.Buffer{}
//	yv.fcmd.Stdout = &yv.pcmbuffer
//
//	yv.ycmd.Stderr = &yv.ycmdError
//	yv.fcmd.Stderr = &yv.fcmdError
//
//	logger.Debug("Starting youtube-dl\n")
//	err := yv.ycmd.Start()
//	if err != nil {
//		return nil, errors.New("Could not start youtube-dl command with url: " + url + " error: " + err.Error())
//	}
//
//	logger.Debug("Starting ffmpeg\n")
//	err = yv.fcmd.Start()
//	if err != nil {
//		return nil, errors.New("Could not start ffmpeg command with url: " + url + " error: " + err.Error())
//	}
//	yv.done = make(chan bool)
//	go func() {
//		yv.ycmd.Wait()
//		w.Close()
//		yv.fcmd.Wait()
//		yv.done <- true
//	}()
////	err = ycmd.Wait()
////	if err != nil {
////		return nil, errors.New("Could not complete youtube-dl command with url: " + url + " stderr: " + ycmdError.String())
////	}
////	w.Close()
////	logger.Debugf("Done youtube-dl with url: %s\n", url)
////	fcmd.ProcessState.
////	err = fcmd.Wait()
////	if err != nil {
////		return nil, errors.New("Could not complete ffmpeg command with url: " + url + " stderr: " + fcmdError.String())
////	}
////	logger.Debug("Done ffmpeg\n")
////
//	//pcmbuff := outpcm.Bytes()
//	//for i := 0; i < len(yv.pcmbuffer); i++ {
//	//	yv.pcmbuffer[i] = int16(pcmbuff[i*2+1]) << 8
//	//	yv.pcmbuffer[i] += int16(pcmbuff[i*2]) << 0
//	//}
//	yv.url = url
//	return &yv, nil
//}
//// Current ffmpeg conversion from stereo to mono:
////  Warning: Any out of phase stereo will cancel out.
//// The following filtergraph can be used to bring out of phase stereo in phase prior to downmixing:
//// -af "asplit[a],aphasemeter=video=0,ametadata=select:key=lavfi.aphasemeter.phase:value=-0.005:function=less,pan=1c|c0=c0,aresample=async=1:first_pts=0,[a]amix"
//
//// return a buffered pcm frame with framesize number of samples
//func (y *youtubevideo) GetPCMFrame(framesize int) ([]int16, error) {
//	buf := y.pcmbuffer.Next(framesize*2) // *2, da int16
//	logger.Debugf("pcmbuffer len: %d\n", y.pcmbuffer.Len())
//	logger.Debugf("ret buf len: %d\n", len(buf))
//	if len(buf) < framesize*2 {
//		// we didn't got enough data, is ffmpeg and youtube-dl done?
//		select {
//		case _ = <- y.done:
//			// Check if any problems occurred with youtube-dl
//			if !y.ycmd.ProcessState.Success() {
//				return nil, errors.New("Could not complete youtube-dl command with url: " + y.url + " stderr: " + y.ycmdError.String())
//			}
//
//			// Did an error with ffmpeg occur?
//			if y.fcmd.ProcessState.Success() {
//				//TODO there could still be pcm data left that needs to be padded and returned
//				// No errors by ffmpeg or youtube-dl. return to tell that the video is done
//				return nil, nil
//			} else {
//				return nil, errors.New("Could not complete ffmpeg command with url: " + y.url + " stderr: " + y.fcmdError.String())
//			}
//		default:
//			// got not enough pcm data in time. Not acceptable return error
//			return nil, errors.New("Could not get enough PCM data in time\n")
//		}
//	}
//
//	pcmslice := make([]int16, framesize)
//	for i := 0; i < len(pcmslice); i++ {
//		pcmslice[i] = int16(buf[i*2+1]) << 8
//		pcmslice[i] += int16(buf[i*2]) << 0
//	}
//	return pcmslice, nil
//}
//
////func (y youtubevideo) GetTotalSamples() (int64, error) {
////	return int64(len(y.pcmbuffer)), nil
////}
////
////func (y youtubevideo) GetChannels() (int, error) {
////	return 1, nil
////}
