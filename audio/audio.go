package audio

import "io"

// FrameInfo represents the frame-level information.
type FrameInfo struct {
	// Channels represent the number of audio channels
	// (e.g. 1 for mono, 2 for stereo).
	Channels int
	// Bit depth is the number of bits used to represent
	// a single sample.
	BitDepth int

	// Sample rate is the number of samples to be played each second.
	SampleRate int64
}

// Clip represents a linear PCM formatted audio io.ReadSeeker.
// Clip can seek and read from a section and allow users to
// consume a small section of the underlying audio data.
//
// FrameInfo returns the basic frame-level information about the clip audio.
//
// Size returns the total number of bytes of the underlying audio data.
// TODO(jbd): Support cases where size is unknown?
type Clip interface {
	io.ReadSeeker
	FrameInfo() FrameInfo
	Size() int64
}

// IeeeFloatToInt converts a 10 byte IEEE float into an int.
func IeeeFloatToInt(b [10]byte) int {
	var i uint32
	// Negative number
	if (b[0] & 0x80) == 1 {
		return 0
	}

	// Less than 1
	if b[0] <= 0x3F {
		return 1
	}

	// Too big
	if b[0] > 0x40 {
		return 67108864
	}

	// Still too big
	if b[0] == 0x40 && b[1] > 0x1C {
		return 800000000
	}

	i = (uint32(b[2]) << 23) | (uint32(b[3]) << 15) | (uint32(b[4]) << 7) | (uint32(b[5]) >> 1)
	i >>= (29 - uint32(b[1]))

	return int(i)
}
