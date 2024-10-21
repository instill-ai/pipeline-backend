package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/streamer45/silero-vad-go/speech"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type detectVoiceActivityInput struct {
	Audio                Audio `json:"audio"`
	MinSilenceDurationMs int   `json:"min-silence-duration-ms"`
	SpeechPadMs          int   `json:"speech-pad-ms"`
}

type detectVoiceActivityOutput struct {
	Segments []segment `json:"segments"`
}

func detectVoiceActivity(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	inputStruct := &detectVoiceActivityInput{}
	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, fmt.Errorf("converting input to struct: %v", err)
	}

	buf, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(string(inputStruct.Audio)))
	if err != nil {
		return nil, err
	}

	dec := wav.NewDecoder(bytes.NewReader(buf))
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// Create an IntBuffer
	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: int(dec.NumChans),
			SampleRate:  int(dec.SampleRate),
		},
		Data:           make([]int, len(buf)),
		SourceBitDepth: int(dec.BitDepth),
	}

	// Read the full audio buffer
	_, err = dec.PCMBuffer(audioBuf)
	if err != nil {
		return nil, fmt.Errorf("error reading audio data: %w", err)
	}

	if audioBuf.Format.NumChannels > numChannel {
		audioBuf = toMono(audioBuf)
	}

	resampledData := resample(audioBuf.AsFloatBuffer().Data, dec.SampleRate, sampleRate)
	audioBuf.Format.SampleRate = sampleRate
	audioBuf.Data = resampledData

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}

	sd, err := speech.NewDetector(speech.DetectorConfig{
		ModelPath:            filepath.Join(cwd, "model/silero_vad.onnx"),
		SampleRate:           16000,
		Threshold:            0.5,
		MinSilenceDurationMs: inputStruct.MinSilenceDurationMs,
		SpeechPadMs:          inputStruct.SpeechPadMs,
	})
	if err != nil {
		return nil, fmt.Errorf("detect voice activity: %w", err)
	}

	segments, err := sd.Detect(audioBuf.AsFloat32Buffer().Data)
	if err != nil {
		return nil, fmt.Errorf("detect voice activity: %w", err)
	}

	if err := sd.Destroy(); err != nil {
		return nil, fmt.Errorf("error destroying speech detector: %w", err)
	}

	output := detectVoiceActivityOutput{
		Segments: make([]segment, len(segments)),
	}
	for i, s := range segments {
		output.Segments[i] = segment{StartTime: s.SpeechStartAt, EndTime: s.SpeechEndAt}
	}

	return base.ConvertToStructpb(output)
}

func toMono(buffer *audio.IntBuffer) *audio.IntBuffer {
	monoData := make([]int, len(buffer.Data)/2)
	for i := range monoData {
		monoData[i] = (buffer.Data[2*i] + buffer.Data[2*i+1]) / 2
	}
	buffer.Data = monoData
	buffer.Format.NumChannels = 1
	return buffer
}

func resample(input []float64, inputRate, outputRate uint32) []int {
	ratio := float64(outputRate) / float64(inputRate)
	n := len(input)
	out := make([]float64, int(float64(n)*ratio))

	fftData := fft.FFTReal(input)
	fftData = fftData[:len(out)]

	ifftData := fft.IFFT(fftData)

	resampledData := make([]int, len(ifftData))
	for i, v := range ifftData {
		resampledData[i] = int(real(v))
	}
	return resampledData
}
