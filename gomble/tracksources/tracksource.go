package tracksources

type TrackSource interface {
	// returns PCM Frame:
	// samplerate: 48000
	// channel: 1 (Mono)
	GetPCMFrame(duration int) ([]int16, error)
	GetTitle() string
}
