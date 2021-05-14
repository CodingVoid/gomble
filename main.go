package main

import (
	"strings"

	"github.com/CodingVoid/gomble/gomble"
	"github.com/CodingVoid/gomble/logger"
)

// queue of tracks
var queue []*gomble.Track

// current Channel the bot is in
var currentChannel gomble.Channel

func main() {
	gomble.Init(logger.TRACE, "127.0.0.1:64738")
	gomble.Listener.OnPrivateMessageReceived = OnPrivateMessageReceived
	gomble.Listener.OnChannelMessageReceived = OnChannelMessageReceived
	gomble.Listener.OnTrackFinished = OnTrackFinished
	gomble.Listener.OnTrackPaused = OnTrackPaused
	gomble.Listener.OnTrackStopped = OnTrackStopped
	gomble.Listener.OnTrackException = OnTrackException

	gomble.Begin()
}

func OnPrivateMessageReceived(e gomble.PrivateMessageReceivedEvent) {
	gomble.GetUser(e.Actor).SendMessage("Send Back Private")
}

func OnChannelMessageReceived(e gomble.ChannelMessageReceivedEvent) {
	// set current channel
	currentChannel = gomble.GetChannel(e.Channel)
	if strings.HasPrefix(e.Message, "#play ") {
		logger.Debugf(e.Message + "\n")
		yt, err := gomble.LoadTrack(e.Message)
		if err != nil {
			logger.Errorf("%v", err)
            return
		}
		queue = append(queue, yt)
		startNextTrack()
	} else if strings.HasPrefix(e.Message, "#stop") {
		gomble.Stop()
	} else if strings.HasPrefix(e.Message, "#pause") {
		gomble.Pause()
	} else if strings.HasPrefix(e.Message, "#resume") {
		gomble.Resume()
	}
}

func OnTrackFinished(e gomble.TrackFinishedEvent) {
	startNextTrack()
}

func OnTrackPaused(e gomble.TrackPausedEvent) {
	logger.Infof("Paused Track: %s", e.Track.GetTitle())
}

func OnTrackStopped(e gomble.TrackStoppedEvent) {
	logger.Infof("Stopped Track: %s", e.Track.GetTitle())
}

func OnTrackException(e gomble.TrackExceptionEvent) {
	logger.Warnf("Got an Exception while playing Track: %s", e.Track.GetTitle())
}

func startNextTrack() {
	if len(queue) > 0 {
		t := queue[0]
		// returns false if a track is already playing (or t == nil). returns true if starting was successful
		if gomble.Play(t) {
			currentChannel.SendMessage("Start playing Track " + t.GetTitle())
			// If successful remove the track from the queue
			queue = queue[1:]
		}
	}
}

