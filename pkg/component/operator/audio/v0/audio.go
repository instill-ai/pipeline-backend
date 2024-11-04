package audio

import (
	"bytes"
	"fmt"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

const (
	sampleRate = 16000
	numChannel = 1
)

func decodeAudioWAV(audioData format.Audio) (*audio.IntBuffer, *wav.Decoder, error) {

	wavAudioData := audioData
	var err error
	if audioData.ContentType().String() != data.WAV {
		wavAudioData, err = audioData.Convert(data.WAV)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting audio data to WAV: %v", err)
		}
	}

	binary, err := wavAudioData.Binary()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting binary data for image: %v", err)
	}

	dec := wav.NewDecoder(bytes.NewReader(binary.ByteArray()))
	if !dec.IsValidFile() {
		return nil, nil, fmt.Errorf("invalid WAV file")
	}

	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: int(dec.NumChans),
			SampleRate:  int(dec.SampleRate),
		},
		Data:           make([]int, len(binary.ByteArray())),
		SourceBitDepth: int(dec.BitDepth),
	}

	if _, err := dec.PCMBuffer(audioBuf); err != nil {
		return nil, nil, fmt.Errorf("reading audio data: %w", err)
	}

	return audioBuf, dec, nil
}
