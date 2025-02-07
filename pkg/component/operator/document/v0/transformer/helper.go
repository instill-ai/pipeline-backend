package transformer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"go.uber.org/zap"
)

func requiresRepair(pdfBase64 string) (bool, error) {
	paramsJSON := map[string]interface{}{
		"PDF": util.TrimBase64Mime(pdfBase64),
	}

	pythonCode := pdfTransformer + pdfChecker
	outputBytes, err := util.ExecutePythonCode(pythonCode, paramsJSON)
	if err != nil {
		return false, fmt.Errorf("executing Python script: %w", err)
	}

	var output struct {
		Repair bool `json:"required"`
	}
	err = json.Unmarshal(outputBytes, &output)
	if err != nil {
		return false, fmt.Errorf("unmarshalling output: %w", err)
	}

	return output.Repair, nil
}

// repairPDF repairs the PDF file if it is required. It will respond the base64
// encoded PDF file.
func repairPDF(pdfBase64 string, logger *zap.Logger) (string, error) {
	pdfData, err := base64.StdEncoding.DecodeString(pdfBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 PDF: %w", err)
	}

	tempInputFile, err := os.CreateTemp("", "input_*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary input file: %w", err)
	}
	defer os.Remove(tempInputFile.Name())

	_, err = tempInputFile.Write(pdfData)
	if err != nil {
		return "", fmt.Errorf("failed to write to temporary input file: %w", err)
	}
	tempInputFile.Close()

	tempOutputFile, err := os.CreateTemp("", "output_*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary output file: %w", err)
	}
	defer os.Remove(tempOutputFile.Name())
	tempOutputFile.Close()

	cmd := exec.Command("qpdf", "--linearize", tempInputFile.Name(), tempOutputFile.Name())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// qpdf might raise warnings during the repair and still succeed.
	// Therefore, we log and continue.
	if err := cmd.Run(); err != nil {
		logger.Error("PDF repair caused standard error output", zap.String("error", stderr.String()))
	}

	repairedPDF, err := os.ReadFile(tempOutputFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read repaired PDF: %w", err)
	}

	repairedPDFBase64 := base64.StdEncoding.EncodeToString(repairedPDF)

	return repairedPDFBase64, nil
}
