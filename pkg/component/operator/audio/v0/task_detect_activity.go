//go:build onnx
// +build onnx

// This task requires ONNX Runtime to be installed. Follow these steps to set it up:
//
// 1. Download ONNX Runtime:
//   - Visit the official repository: https://github.com/microsoft/onnxruntime/releases
//   - Choose the latest version compatible with your OS architecture
//
// 2. Install ONNX Runtime:
//   - Extract the downloaded tar file to a directory (referred to as ONNXRUNTIME_ROOT_PATH)
//   - Set up the environment:
//     export C_INCLUDE_PATH=$ONNXRUNTIME_ROOT_PATH/include
//     export LD_RUN_PATH=$ONNXRUNTIME_ROOT_PATH/lib
//     export LIBRARY_PATH=$ONNXRUNTIME_ROOT_PATH/lib

// This task requires the following libraries to be installed:
//   - libsoxr-dev (required for github.com/zaf/resample)

package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/streamer45/silero-vad-go/speech"
	"github.com/zaf/resample"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func detectActivity(ctx context.Context, job *base.Job) error {
	var input detectActivityInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return err
	}

	audioBuf, dec, err := decodeAudioWAV(input.Audio)
	if err != nil {
		return err
	}

	if audioBuf.Format.NumChannels > numChannel {
		audioBuf = toMono(audioBuf)
	}

	if audioBuf.Format.SampleRate != sampleRate {
		resampledData, err := resampleAudio(audioBuf.AsFloatBuffer().Data, float64(dec.SampleRate), float64(sampleRate), audioBuf.Format.NumChannels)
		if err != nil {
			return fmt.Errorf("resampling audio: %w", err)
		}
		audioBuf.Format.SampleRate = sampleRate
		audioBuf.Data = resampledData
	}

	sd, err := speech.NewDetector(speech.DetectorConfig{
		ModelPath:            filepath.Join(os.Getenv("ONNX_MODEL_FOLDER_PATH"), "silero_vad.onnx"),
		SampleRate:           sampleRate,
		Threshold:            0.5,
		MinSilenceDurationMs: input.MinSilenceDuration,
		SpeechPadMs:          input.SpeechPad,
	})
	if err != nil {
		return fmt.Errorf("creating voice activity detector: %w", err)
	}

	defer func() {
		if removeErr := sd.Destroy(); removeErr != nil {
			if err == nil {
				err = fmt.Errorf("destroy speech detector: %w", removeErr)
			}
		}
	}()

	segments, err := sd.Detect(audioBuf.AsFloat32Buffer().Data)
	if err != nil {
		return fmt.Errorf("detect voice activity: %w", err)
	}

	dao := detectActivityOutput{
		Segments: make([]segmentData, len(segments)),
	}
	for i, s := range segments {
		dao.Segments[i] = segmentData{StartTime: s.SpeechStartAt, EndTime: s.SpeechEndAt}
	}

	if err := job.Output.WriteData(ctx, dao); err != nil {
		return err
	}

	return nil
}

func toMono(buffer *audio.IntBuffer) *audio.IntBuffer {
	for i := 0; i < len(buffer.Data)/2; i++ {
		buffer.Data[i] = (buffer.Data[2*i] + buffer.Data[2*i+1]) / 2
	}
	buffer.Data = buffer.Data[:len(buffer.Data)/2]
	buffer.Format.NumChannels = 1
	return buffer
}

func resampleAudio(input []float64, inputRate, outputRate float64, channels int) ([]int, error) {
	var buf bytes.Buffer
	resampler, err := resample.New(&buf, inputRate, outputRate, channels, resample.F64, resample.HighQ)
	if err != nil {
		return nil, fmt.Errorf("creating resampler: %w", err)
	}
	defer resampler.Close()

	// Convert []float64 to []byte
	inputBytes := make([]byte, len(input)*8)
	for i, v := range input {
		binary.LittleEndian.PutUint64(inputBytes[i*8:], math.Float64bits(v))
	}

	_, err = resampler.Write(inputBytes)
	if err != nil {
		return nil, fmt.Errorf("writing to resampler: %w", err)
	}

	// Convert resampled []byte back to []int
	resampledBytes := buf.Bytes()
	resampledData := make([]int, len(resampledBytes)/8)
	for i := 0; i < len(resampledData); i++ {
		resampledFloat := math.Float64frombits(binary.LittleEndian.Uint64(resampledBytes[i*8:]))
		resampledData[i] = int(resampledFloat)
	}

	return resampledData, nil
}
