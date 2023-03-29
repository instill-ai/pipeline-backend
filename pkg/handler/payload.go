package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

func parseImageFormDataInputsToBytes(req *http.Request) (ImageInput *service.ImageInput, err error) {

	inputs := req.MultipartForm.File["file"]
	content := make([]byte, 0, len(inputs))
	fileNames := make([]string, 0, len(inputs))
	fileLengths := make([]uint64, 0, len(inputs))
	for _, input := range inputs {
		file, err := input.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			return nil, fmt.Errorf("Unable to open file for image")
		}

		buff := new(bytes.Buffer)
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			return nil, fmt.Errorf("Unable to read content body from image")
		}
		if numBytes > int64(config.Config.Server.MaxDataSize*constant.MB) {
			return nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(constant.MB),
			)
		}

		content = append(content, buff.Bytes()...)
		fileNames = append(fileNames, input.Filename)
		fileLengths = append(fileLengths, uint64(buff.Len()))
	}
	return &service.ImageInput{
		Content:     content,
		FileNames:   fileNames,
		FileLengths: fileLengths,
	}, nil
}

func parseImageFormDataTextToImageInputs(req *http.Request) (textToImageInput *service.TextToImageInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) == 0 {
		return nil, fmt.Errorf("missing prompt input")
	}
	if len(prompts) > 1 {
		return nil, fmt.Errorf("invalid prompt input, only support a single prompt")
	}
	stepStr := req.MultipartForm.Value["steps"]
	cfgScaleStr := req.MultipartForm.Value["cfg_scale"]
	seedStr := req.MultipartForm.Value["seed"]
	samplesStr := req.MultipartForm.Value["samples"]

	if len(stepStr) > 1 {
		return nil, fmt.Errorf("invalid steps input, only support a single steps")
	}
	if len(cfgScaleStr) > 1 {
		return nil, fmt.Errorf("invalid cfg_scale input, only support a single cfg_scale")
	}
	if len(seedStr) > 1 {
		return nil, fmt.Errorf("invalid seed input, only support a single seed")
	}
	if len(samplesStr) > 1 {
		return nil, fmt.Errorf("invalid samples input, only support a single samples")
	}

	step := constant.DefaultStep
	if len(stepStr) > 0 {
		step, err = strconv.Atoi(stepStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid step input %w", err)
		}
	}

	cfgScale := constant.DefaultCfgScale
	if len(cfgScaleStr) > 0 {
		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid cfgScale input %w", err)
		}
	}

	seed := constant.DefaultSeed
	if len(seedStr) > 0 {
		seed, err = strconv.Atoi(seedStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid seed input %w", err)
		}
	}

	samples := constant.DefaultSamples
	if len(samplesStr) > 0 {
		samples, err = strconv.Atoi(samplesStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid samples input %w", err)
		}
	}

	if samples > 1 {
		return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
	}

	return &service.TextToImageInput{
		Prompt:   prompts[0],
		Steps:    int64(step),
		CfgScale: float32(cfgScale),
		Seed:     int64(seed),
		Samples:  int64(samples),
	}, nil
}

func parseTextFormDataTextGenerationInputs(req *http.Request) (textGeneration *service.TextGenerationInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("only support batchsize 1")
	}
	badWordsListInput := req.MultipartForm.Value["stop_words_list"]
	stopWordsListInput := req.MultipartForm.Value["stop_words_list"]
	outputLenInput := req.MultipartForm.Value["output_len"]
	topKInput := req.MultipartForm.Value["topk"]
	seedInput := req.MultipartForm.Value["seed"]

	badWordsList := string("")
	if len(badWordsListInput) > 0 {
		badWordsList = badWordsListInput[0]
	}

	stopWordsList := string("")
	if len(stopWordsListInput) > 0 {
		stopWordsList = stopWordsListInput[0]
	}

	outputLen := constant.DefaultOutputLen
	if len(outputLenInput) > 0 {
		outputLen, err = strconv.Atoi(outputLenInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	topK := constant.DefaultTopK
	if len(topKInput) > 0 {
		topK, err = strconv.Atoi(topKInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	seed := constant.DefaultSeed
	if len(seedInput) > 0 {
		seed, err = strconv.Atoi(seedInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	// TODO: add support for bad/stop words
	return &service.TextGenerationInput{
		Prompt:        prompts[0],
		OutputLen:     int64(outputLen),
		BadWordsList:  badWordsList,
		StopWordsList: stopWordsList,
		TopK:          int64(topK),
		Seed:          int64(seed),
	}, nil
}
