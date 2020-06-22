package youtube

import "io"
import "net/http"
import "net/url"
import "fmt"
import "runtime"
import "github.com/CodingVoid/gomble/logger"

type persistentYoutubeStream struct {
	contentUrl string
	// Position for the current http GET response
	cposition int
	// Position of the entire youtube content
	aposition int
	// The overall length of the youtube content
	contentLength int
	// The current http response that is in use
	httpResp *http.Response
	// byte array to save all the incoming data (usefull for seeking)
	//save []byte
}

func NewPersistentYoutubeStream(contentUrl string, contentLength int) (*persistentYoutubeStream, error) {
	stream := persistentYoutubeStream {
		contentUrl: contentUrl,
		contentLength: contentLength,
	}
	err := stream.RequestNext()
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return nil, fmt.Errorf("NewPersistentYoutubeStream(%s:%d): %w", file, line, err)
	}
	return &stream, nil
}

// Read len(p) bytes (it does not guarantee that len(p) bytes are read). returns io.EOF on end of file
func (s *persistentYoutubeStream) Read(p []byte) (int, error) {// {{{
	n, err := s.httpResp.Body.Read(p)
	if err != nil {
		// Check if error was end of file
		if err != io.EOF {
			logger.Debugf("Hier IST DAS MUHAHAHAH, body length: %d, readed: %d, to read: %d, already readed: %d, %v\n", s.httpResp.ContentLength, n, len(p), s.cposition, err)
			// try to "reconnect" (just request the remaining audiodata again)
			err = s.RequestNext()
			// here we will return 0 data even though there is still data left (so keep in mind that this function does not always return len(p) audiodata)
			return n, err
		}
	}
	s.cposition += n
	s.aposition += n
	//s.save = append(s.save, p[:n]...) // save everything that has been readed so far (//TODO implement seeking)
	// Check if that is the end of the http response
	if s.httpResp.ContentLength == int64(s.cposition) {
		// Check if the complete stream is at the end
		if s.aposition == s.contentLength {
			return n, io.EOF
		}
		err = s.RequestNext()
		return n, err
	}
	return n, nil
}// }}}

func (s *persistentYoutubeStream) RequestNext() error {// {{{
	logger.Debug("Requesting next Youtube media/audio data\n")
	burl, err := url.Parse(s.contentUrl)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("RequestNext(%s:%d): %w", file, line, err)
	}
	val := burl.Query() // get values of url
	val.Set("range", fmt.Sprintf("%d-%d", s.aposition, s.contentLength)) // set range paramter for value
	burl.RawQuery = val.Encode() // set values to url
	resp, err := http.Get(burl.String())
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("RequestNext(%s:%d): %w", file, line, err)
	}
	if resp.StatusCode != 200 {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("RequestNext(%s:%d): youtube returned %s", file, line, resp.Status)
	}
	logger.Debug("Successfully got next Youtube media/audio data\n")
	s.httpResp = resp
	s.cposition = 0
	return nil
}// }}}
