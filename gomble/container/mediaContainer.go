package youtube

import "io"

type MediaContainer interface {
	ReadHeader(reader io.Reader) error
	ReadContent(reader io.Reader) error
}
