package main

import (
	"strings"
	"regexp"

	"github.com/CodingVoid/gomble/gomble"
	"github.com/CodingVoid/gomble/logger"
	"github.com/CodingVoid/gomble/gomble/audiosources"
	"github.com/CodingVoid/gomble/gomble/audiosources/youtube"
	"github.com/CodingVoid/gomble/gomble/audiosources/oggopusfile"
)

// queue of tracks
var queue []audiosources.Audiosource
// current Channel the bot is in
var currentChannel gomble.Channel

func main() {
	gomble.Init(logger.DEBUG, "127.0.0.1:64738")
	gomble.Listener.OnPrivateMessageReceived = OnPrivateMessageReceived
	gomble.Listener.OnChannelMessageReceived = OnChannelMessageReceived
	gomble.Listener.OnTrackEnded = OnTrackEnded

	gomble.Begin()
}

func OnPrivateMessageReceived(e gomble.PrivateMessageReceivedEvent) {
	gomble.GetUser(e.Actor).SendMessage("Send Back Private")
}

func OnChannelMessageReceived(e gomble.ChannelMessageReceivedEvent) {
	// set current channel
	currentChannel = gomble.GetChannel(e.Channel)
	if strings.HasPrefix(e.Message, "#play ") {
		var url string
		fields := strings.Fields(e.Message)
		if fields[1] == "<a" {
			// there is an html encoded link
			// <a href=link>link</a>
			re := regexp.MustCompile(`https://www.youtube.com/watch\?v=[a-zA-Z0-9\-\_]+`) // need to use ` character otherwise \character are recognized as escape characters
			matches := re.FindStringSubmatch(e.Message)
			url = matches[0]
		} else {
			// probably not html encoded (maybe allowHTML is off?)
			url = strings.Fields(e.Message)[1]
		}
		logger.Debugf(url + "\n")
		yt, err := youtube.NewYoutubeVideo(url)
		if err != nil {
			logger.Fatalf("%v", err)
		}

		queue = append(queue, yt)
		startNextTrack()
	} else if strings.HasPrefix(e.Message, "#youtube") {
		yt, err := youtube.NewYoutubeVideo("https://www.youtube.com/watch?v=YO1GBsuzTWU")
		if err != nil {
			logger.Fatalf("%v", err)
		}
		queue = append(queue, yt)
		startNextTrack()
	} else if strings.HasPrefix(e.Message, "#file") {
		of, err := oggopusfile.NewOggOpusfile("/home/max/Programming/gomble/gomble/audiosources/oggopusfile/example.opus")
		if err != nil {
			logger.Fatalf("%v", err)
		}
		queue = append(queue, of)
		startNextTrack()
	}
}

func OnTrackEnded(e gomble.TrackEndedEvent) {
	startNextTrack()
}


func startNextTrack() {
	if len(queue) > 0 {
		t := queue[0]
		// returns false if a track is already playing (or t == nil). returns true if starting was successfull
		if gomble.Play(t, false) {
			currentChannel.SendMessage("Start playing Track " + t.GetTitle())
			// If successfull remove the track from the queue
			queue = queue[1:]
		}
	}
}
