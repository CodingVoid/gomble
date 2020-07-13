package gomble

import "github.com/CodingVoid/gomble/gomble/audiosources"

// I need a base type.
type event interface {
}

type PrivateMessageReceivedEvent struct {
	Actor   uint32
	Message string
}

type ChannelMessageReceivedEvent struct {
	Actor   uint32
	Channel uint32
	Message string
}

// Track related

// Reasons why track ended playing
type TrackEndedReason int

const (
	// Track got interrupted by user
	TRACK_INTERRUPTED = iota
	// Track ended playing because Track was over
	TRACK_ENDED
	// Some other failure occured. The Logs should be checked for further Information
	TRACK_OTHER
)

type TrackEndedEvent struct {
	Track  audiosources.Audiosource
	Reason TrackEndedReason
}

/* Audiosource/Track related stuff
type TrackStartedEvent struct {
	listener []func(author string, msg string)
	callListener()
}

type TrackEndedEvent struct {
	listener []func(author string, msg string)
	callListener()
}*/
