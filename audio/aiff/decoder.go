package aiff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/mattetti/exp/audio"
)

// Decoder is the wrapper structure for the AIFF container
type Decoder struct {
	r io.Reader
	// ID is always 'FORM'. This indicates that this is a FORM chunk
	ID [4]byte
	// Size contains the size of data portion of the 'FORM' chunk.
	// Note that the data portion has been
	// broken into two parts, formType and chunks
	Size uint32
	// Format describes what's in the 'FORM' chunk. For Audio IFF files,
	// formType (aka Format) is always 'AIFF'.
	// This indicates that the chunks within the FORM pertain to sampled sound.
	Format [4]byte

	// Data coming from the COMM chunk
	commSize        uint32
	NumChans        uint16
	NumSampleFrames uint32
	SampleSize      uint16
	SampleRate      int

	// AIFC data
	Encoding     [4]byte
	EncodingName string
}

// Decode reads from a Read Seeker and converts the input to a PCM
// clip output.
func Decode(r io.ReadSeeker) (audio.Clip, error) {
	d := &Decoder{r: r}
	if err := d.readHeaders(); err != nil {
		return nil, err
	}

	// read the file information to setup the audio clip
	// find the beginning of the SSND chunk and set the clip reader to it.
	clip := &Clip{}

	var err error
	var rewindBytes int64
	for err != io.EOF {
		id, size, err := d.iDnSize()
		if err != nil {
			break
		}
		switch id {
		case commID:
			d.parseCommChunk(size)
			clip.channels = int(d.NumChans)
			clip.bitDepth = int(d.SampleSize)
			clip.sampleRate = int64(d.SampleRate)
			// if we found the sound data before the COMM,
			// we need to rewind the reader so we can properly
			// set the clip reader.
			if rewindBytes > 0 {
				r.Seek(-rewindBytes, 1)
				break
			}
		case ssndID:
			clip.size = int64(size)
			// if we didn't read the COMM, we are going to need to come back
			if clip.sampleRate == 0 {
				rewindBytes += int64(size)
				if err := d.jumpTo(int(size)); err != nil {
					return nil, err
				}
			} else {
				break
			}
		default:
			// if we read SSN but didn't read the COMM, we need to track location
			if clip.size != 0 {
				rewindBytes += int64(size)
			}
			if err := d.jumpTo(int(size)); err != nil {
				return nil, err
			}
		}
	}
	clip.r = r
	return clip, nil
}

// Duration returns the time duration for the current AIFF container
func (d *Decoder) Duration() (time.Duration, error) {
	if d == nil {
		return 0, errors.New("can't calculate the duration of a nil pointer")
	}
	duration := time.Duration(float64(d.NumSampleFrames) / float64(d.SampleRate) * float64(time.Second))
	return duration, nil
}

func (d *Decoder) readHeaders() error {
	if err := binary.Read(d.r, binary.BigEndian, &d.ID); err != nil {
		return err
	}
	// Must start by a FORM header/ID
	if d.ID != formID {
		return fmt.Errorf("%s - %s", ErrFmtNotSupported, d.ID)
	}

	if err := binary.Read(d.r, binary.BigEndian, &d.Size); err != nil {
		return err
	}
	if err := binary.Read(d.r, binary.BigEndian, &d.Format); err != nil {
		return err
	}

	// Must be a AIFF or AIFC form type
	if d.Format != aiffID && d.Format != aifcID {
		return fmt.Errorf("%s - %s", ErrFmtNotSupported, d.Format)
	}

	return nil
}

func (d *Decoder) parseCommChunk(size uint32) error {
	d.commSize = size

	if err := binary.Read(d.r, binary.BigEndian, &d.NumChans); err != nil {
		return fmt.Errorf("num of channels failed to parse - %s", err.Error())
	}
	if err := binary.Read(d.r, binary.BigEndian, &d.NumSampleFrames); err != nil {
		return fmt.Errorf("num of sample frames failed to parse - %s", err.Error())
	}
	if err := binary.Read(d.r, binary.BigEndian, &d.SampleSize); err != nil {
		return fmt.Errorf("sample size failed to parse - %s", err.Error())
	}
	var srBytes [10]byte
	if err := binary.Read(d.r, binary.BigEndian, &srBytes); err != nil {
		return fmt.Errorf("sample rate failed to parse - %s", err.Error())
	}
	d.SampleRate = audio.IeeeFloatToInt(srBytes)

	if d.Format == aifcID {
		if err := binary.Read(d.r, binary.BigEndian, &d.Encoding); err != nil {
			return fmt.Errorf("AIFC encoding failed to parse - %s", err)
		}
		// pascal style string with the description of the encoding
		var size uint8
		if err := binary.Read(d.r, binary.BigEndian, &size); err != nil {
			return fmt.Errorf("AIFC encoding failed to parse - %s", err)
		}

		desc := make([]byte, size)
		if err := binary.Read(d.r, binary.BigEndian, &desc); err != nil {
			return fmt.Errorf("AIFC encoding failed to parse - %s", err)
		}
		d.EncodingName = string(desc)
	}

	return nil

}

// iDnSize returns the next ID + block size
func (d *Decoder) iDnSize() ([4]byte, uint32, error) {
	var ID [4]byte
	var blockSize uint32
	if err := binary.Read(d.r, binary.BigEndian, &ID); err != nil {
		return ID, blockSize, err
	}
	if err := binary.Read(d.r, binary.BigEndian, &blockSize); err != err {
		return ID, blockSize, err
	}
	return ID, blockSize, nil
}

// jumpTo advances the reader to the amount of bytes provided
func (d *Decoder) jumpTo(bytesAhead int) error {
	var err error
	if bytesAhead > 0 {
		_, err = io.CopyN(ioutil.Discard, d.r, int64(bytesAhead))
	}
	return err
}
