package oggopusfile

import "testing"
import "io"
import "github.com/CodingVoid/gomble/logger"

func TestRead(t *testing.T) {
	f, err := NewOggOpusfile("/home/max/Programming/gomble/gomble/audiosources/example.opus")
	if err != nil {
		t.Fatal(err)
	}

	for {
		pcm, err := f.GetPCMFrame(20)
		if err != nil {
			if err == io.EOF {
				// Track is done
				break
			}
			t.Fatal(err)
		}
		logger.Debugf("len(pcm): %d\n", len(pcm))
	}
}
