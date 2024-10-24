package audio

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/iFaceless/godub"
	"github.com/iFaceless/godub/wav"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ChunkAudiosInput struct {
	Audio      Audio `json:"audio"`
	ChunkCount int   `json:"chunk-count"`
}

type ChunkAudiosOutput struct {
	Audios []Audio `json:"audios"`
}

type SliceAudioInput struct {
	Audio     Audio `json:"audio"`
	StartTime int   `json:"start-time"`
	EndTime   int   `json:"end-time"`
}

type SliceAudioOutput struct {
	Audio Audio `json:"audio"`
}

type ConcatenateInput struct {
	Audios []Audio `json:"audios"`
}

type ConcatenateOutput struct {
	Audio Audio `json:"audio"`
}

// Base64 encoded audio
type Audio string

func chunkAudios(input *structpb.Struct) (*structpb.Struct, error) {

	var inputStruct ChunkAudiosInput

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}

	buf, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(string(inputStruct.Audio)))
	if err != nil {
		return nil, err
	}

	segment, err := godub.NewLoader().Load(bytes.NewReader(buf))

	if err != nil {
		return nil, fmt.Errorf("failed to load audio: %w", err)
	}

	duration := segment.Duration()

	chunkSeconds := float64(duration) / float64(inputStruct.ChunkCount)

	var audioSegments []*godub.AudioSegment

	var startTime time.Duration
	for i := 0; i < inputStruct.ChunkCount; i++ {
		startTime = getStartTime(chunkSeconds, i)
		endTime := getEndTime(chunkSeconds, i, inputStruct.ChunkCount, duration)

		slicedSegment, err := segment.Slice(startTime, endTime)
		if err != nil {
			return nil, fmt.Errorf("failed to slice audio: %w in chunk %v", err, i)
		}
		audioSegments = append(audioSegments, slicedSegment)
	}

	var audios []Audio
	prefix := "data:audio/wav;base64,"
	for _, segment := range audioSegments {
		var wavBuf bytes.Buffer
		err = wav.Encode(&wavBuf, segment.AsWaveAudio())

		if err != nil {
			return nil, fmt.Errorf("failed to encode audio to wav: %w", err)
		}

		audios = append(audios, Audio(prefix+base64.StdEncoding.EncodeToString(wavBuf.Bytes())))
	}

	output := ChunkAudiosOutput{
		Audios: audios,
	}

	return base.ConvertToStructpb(output)
}

func sliceAudio(input *structpb.Struct) (*structpb.Struct, error) {

	var inputStruct SliceAudioInput

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}

	buf, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(string(inputStruct.Audio)))
	if err != nil {
		return nil, err
	}

	segment, err := godub.NewLoader().Load(bytes.NewReader(buf))

	if err != nil {
		return nil, fmt.Errorf("failed to load audio: %w", err)
	}

	startTime := time.Duration(inputStruct.StartTime) * time.Second
	endTime := time.Duration(inputStruct.EndTime) * time.Second

	slicedSegment, err := segment.Slice(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to slice audio: %w", err)
	}

	var wavBuf bytes.Buffer
	err = wav.Encode(&wavBuf, slicedSegment.AsWaveAudio())
	if err != nil {
		return nil, fmt.Errorf("failed to encode audio to wav: %w", err)
	}

	output := SliceAudioOutput{
		Audio: Audio("data:audio/wav;base64," + base64.StdEncoding.EncodeToString(wavBuf.Bytes())),
	}

	return base.ConvertToStructpb(output)
}

func getStartTime(chunkSeconds float64, i int) time.Duration {
	return time.Duration(chunkSeconds * float64(i))
}

func getEndTime(chunkSeconds float64, i, totalCount int, duration time.Duration) time.Duration {
	if i == totalCount-1 {
		return duration
	}
	return time.Duration(chunkSeconds * float64(i+1))
}
