package gomble

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

// Track got interrupted/stopped by user
type TrackStoppedEvent struct {
	Track  *Track
}

// Track got paused
type TrackPausedEvent struct {
	Track  *Track
}

// Track Finished playing
type TrackFinishedEvent struct {
	Track  *Track
}

// Some other failure occurred. The Logs should be checked for further Information
type TrackExceptionEvent struct {
	Track  *Track
	err	   error
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
