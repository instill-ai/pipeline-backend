package transformer

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type converterOutput struct {
	Body          string   `json:"body"`
	Images        []string `json:"images"`
	ParsingError  []string `json:"parsing_error"`
	SystemError   string   `json:"system_error"`
	AllPageImages []string `json:"all_page_images"`
	AllPage       bool     `json:"display_all_page_image"`
	Markdowns     []string `json:"markdowns"`
}

func convertPDFToMarkdown(pythonCode string) func(input pdfToMarkdownInputStruct) (converterOutput, error) {
	return func(input pdfToMarkdownInputStruct) (converterOutput, error) {
		var output converterOutput

		pdfBase64 := util.TrimBase64Mime(input.base64Text)
		if requiresRepair(input.base64Text) {
			repairedPDF, err := repairPDF(pdfBase64)
			if err != nil {
				return output, fmt.Errorf("repairing PDF: %w", err)
			}

			pdfBase64 = repairedPDF
		}

		paramsJSON, err := json.Marshal(map[string]interface{}{
			"PDF":                    pdfBase64,
			"display-image-tag":      input.displayImageTag,
			"display-all-page-image": input.displayAllPageImage,
			"resolution":             input.resolution,
		})
		if err != nil {
			return output, fmt.Errorf("marshalling conversion params: %w", err)
		}

		cmdRunner := exec.Command(pythonInterpreter, "-c", pythonCode)
		stdin, err := cmdRunner.StdinPipe()
		if err != nil {
			return output, fmt.Errorf("creating stdin pipe: %w", err)
		}

		errChan := make(chan error, 1)
		go func() {
			defer stdin.Close()
			_, err := stdin.Write(paramsJSON)
			if err != nil {
				errChan <- fmt.Errorf("writing to stdin: %w", err)
				return
			}
			errChan <- nil
		}()

		outputBytes, err := cmdRunner.CombinedOutput()
		if err != nil {
			errorStr := string(outputBytes)
			return output, fmt.Errorf("running Python script: %w, %s", err, errorStr)
		}

		err = <-errChan
		if err != nil {
			return output, err
		}

		err = json.Unmarshal(outputBytes, &output)
		if err != nil {
			return output, fmt.Errorf("unmarshalling output: %w", err)
		}

		if output.SystemError != "" {
			return output, fmt.Errorf("converting PDF to Markdown: %s", output.SystemError)
		}

		return output, nil
	}
}
