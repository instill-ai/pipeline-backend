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
}

func convertPDFToMarkdownWithPDFPlumber(base64Text string, displayImageTag bool, displayAllPage bool) (converterOutput, error) {

	var pdfBase64 string
	var err error
	pdfBase64WithoutMime := util.TrimBase64Mime(base64Text)
	if RequiredToRepair(base64Text) {
		pdfBase64, err = RepairPDF(pdfBase64WithoutMime)
		if err != nil {
			return converterOutput{}, fmt.Errorf("failed to repair PDF: %w", err)
		}
	} else {
		pdfBase64 = pdfBase64WithoutMime
	}

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"PDF":                    pdfBase64,
		"display-image-tag":      displayImageTag,
		"display-all-page-image": displayAllPage,
	})
	var output converterOutput

	if err != nil {
		return output, fmt.Errorf("failed to marshal params: %w", err)
	}

	pythonCode := imageProcessor + pdfTransformer + taskConvertToMarkdownExecution

	cmdRunner := exec.Command(pythonInterpreter, "-c", pythonCode)
	stdin, err := cmdRunner.StdinPipe()

	if err != nil {
		return output, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	errChan := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, err := stdin.Write(paramsJSON)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	outputBytes, err := cmdRunner.CombinedOutput()
	if err != nil {
		errorStr := string(outputBytes)
		return output, fmt.Errorf("failed to run python script: %w, %s", err, errorStr)
	}

	writeErr := <-errChan
	if writeErr != nil {
		return output, fmt.Errorf("failed to write to stdin: %w", writeErr)
	}

	err = json.Unmarshal(outputBytes, &output)
	if err != nil {
		return output, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	if output.SystemError != "" {
		return output, fmt.Errorf("failed to convert pdf to markdown: %s", output.SystemError)
	}

	return output, nil
}
