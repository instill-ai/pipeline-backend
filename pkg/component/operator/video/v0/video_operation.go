package video

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type SubsampleVideoInput struct {
	Video     Video   `json:"video"`
	Fps       float32 `json:"fps"`
	StartTime string  `json:"start-time"`
	Duration  string  `json:"duration"`
}

type SubsampleVideoOutput struct {
	Video Video `json:"video"`
}

type SubsampleVideoFramesInput struct {
	Video     Video   `json:"video"`
	Fps       float32 `json:"fps"`
	StartTime string  `json:"start-time"`
	Duration  string  `json:"duration"`
}

type SubsampleVideoFramesOutput struct {
	Frames []Frame `json:"frames"`
}

// Base64 encoded video
type Video string

// Base64 encoded frame
type Frame string

func subsampleVideo(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := SubsampleVideoInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	base64Video := string(inputStruct.Video)

	videoBytes, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(base64Video))

	if err != nil {
		return nil, fmt.Errorf("error in decoding for inner: %s", err)
	}

	// TODO: chuang8511 map the file extension to the correct format
	tempInputFile, err := os.CreateTemp("", "temp.*.mp4")

	if err != nil {
		return nil, fmt.Errorf("error in creating temp input file: %s", err)
	}

	tempInputFileName := tempInputFile.Name()
	defer os.Remove(tempInputFileName)

	if _, err := tempInputFile.Write(videoBytes); err != nil {
		return nil, fmt.Errorf("error in writing file: %s", err)
	}

	split := ffmpeg.Input(tempInputFileName)

	tempOutputFile, err := os.CreateTemp("", "temp_out.*.mp4")
	if err != nil {
		return nil, fmt.Errorf("error in creating temp output file: %s", err)
	}
	tempOutputFileName := tempOutputFile.Name()
	defer os.Remove(tempOutputFileName)

	split = split.OverWriteOutput()
	err = split.Output(tempOutputFileName, getKwArgs(inputStruct)).Run()

	if err != nil {
		return nil, fmt.Errorf("error in running ffmpeg: %s", err)
	}

	byOut, _ := os.ReadFile(tempOutputFileName)
	base64Subsample := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString(byOut)

	output := SubsampleVideoOutput{
		Video: Video(base64Subsample),
	}

	return base.ConvertToStructpb(output)
}

func getKwArgs(inputStruct SubsampleVideoInput) ffmpeg.KwArgs {
	kwArgs := ffmpeg.KwArgs{"r": inputStruct.Fps}
	if inputStruct.StartTime != "" {
		kwArgs["ss"] = inputStruct.StartTime
	}
	if inputStruct.Duration != "" {
		kwArgs["t"] = inputStruct.Duration
	}
	// Set yuv420p to ensure video compatibility across various operating
	// systems video viewer.
	kwArgs["pix_fmt"] = "yuv420p"

	return kwArgs
}

func subsampleVideoFrames(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := SubsampleVideoFramesInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	base64Video := string(inputStruct.Video)

	videoBytes, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(base64Video))

	if err != nil {
		return nil, fmt.Errorf("error in decoding for inner: %s", err)
	}

	tempInputFile, err := os.CreateTemp("", "temp.*.mp4")
	if err != nil {
		return nil, fmt.Errorf("error in creating temp input file: %s", err)
	}
	tempInputFileName := tempInputFile.Name()
	defer os.Remove(tempInputFileName)

	if _, err := tempInputFile.Write(videoBytes); err != nil {
		return nil, fmt.Errorf("error in writing file: %s", err)
	}

	random := uuid.New().String()
	// TODO: chuang8511 confirm the reasonable numbers for outputPattern.
	// In the future, we will support bigger size of video, so we set the frame number to 8 digits.
	// Because the sequence is important, we need to use pattern
	// with frame number rather than uuid as suffix.
	outputPattern := random + "_frame_%08d.jpeg"

	kwArgs, errArgs := getFramesKwArgs(inputStruct)
	if errArgs != nil {
		return nil, fmt.Errorf("unable to convert fps to float number: %s", errArgs)
	}

	err = ffmpeg.Input(tempInputFileName).
		Output(outputPattern,
			kwArgs,
		).
		//.GlobalArgs("-report").OverWriteOutput().
		Run()

	if err != nil {
		return nil, fmt.Errorf("error in running ffmpeg: %s", err)
	}

	files, err := filepath.Glob(random + "_frame_*.jpeg")
	if err != nil {
		return nil, fmt.Errorf("error listing frames: %s", err)
	}
	defer removeFiles(files)

	sort.Strings(files)
	jpegPrefix := "data:image/jpeg;base64,"
	var frames []Frame
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %v", file, err)
		}

		encoded := base64.StdEncoding.EncodeToString(data)

		frames = append(frames, Frame(jpegPrefix+encoded))
	}

	output := SubsampleVideoFramesOutput{
		Frames: frames,
	}

	return base.ConvertToStructpb(output)
}

func getFramesKwArgs(inputStruct SubsampleVideoFramesInput) (ffmpeg.KwArgs, error) {
	// formattedFps, err := getFormattedFPS(string(inputStruct.Fps))
	// if err != nil {
	// 	return nil, err
	// }

	kwArgs := ffmpeg.KwArgs{"vf": "fps=" + fmt.Sprintf("%f", inputStruct.Fps)}
	if inputStruct.StartTime != "" {
		kwArgs["ss"] = inputStruct.StartTime
	}
	if inputStruct.Duration != "" {
		kwArgs["t"] = inputStruct.Duration
	}
	// kwArgs["loglevel"] = "warning"
	return kwArgs, nil
}

func removeFiles(files []string) {
	for _, file := range files {
		os.Remove(file)
	}
}

// func getFormattedFPS(fpsLiteral string) (float32, error) {
// 	fps, err := strconv.ParseFloat(fpsLiteral, 32) // handles int and float inputs (30, 10, 0.5 etc)
// 	if err != nil {
// 		// to handle fraction inputs like 1/2, 1/4, 1/30
// 		split := strings.Split(fpsLiteral, "/")
// 		if len(split) != 2 {
// 			return 0, errors.New("invalid fraction input")
// 		}

// 		numerator, errN := strconv.Atoi(split[0])
// 		denominator, errD := strconv.Atoi(split[1])
// 		if errN != nil || errD != nil {
// 			return 0, errors.New("fraction numerator and denominator should be ints")
// 		}

// 		fps := float32(numerator) / float32(denominator)

// 		return fps, nil
// 	}
// 	return float32(fps), nil
// }
