package transformer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

func RequiredToRepair(pdfBase64 string) bool {

	paramsJSON := map[string]interface{}{
		"PDF": base.TrimBase64Mime(pdfBase64),
	}

	pythonCode := pdfTransformer + pdfChecker

	outputBytes, err := util.ExecutePythonCode(pythonCode, paramsJSON)

	if err != nil {
		// It shouldn't block the original process.
		log.Println("failed to run python script: %w", err)
		return false
	}

	var output struct {
		Repair bool `json:"required"`
	}

	err = json.Unmarshal(outputBytes, &output)

	if err != nil {
		// It shouldn't block the original process.
		log.Println("failed to unmarshal output: %w", err)
	}

	return output.Repair
}

// RepairPDF repairs the PDF file if it is required. It will respond the base64 encoded PDF file.
func RepairPDF(pdfBase64 string) (string, error) {

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

	// qpdf can still repair the PDF when there are warnings.
	// So, we do not raise error here.
	if err := cmd.Run(); err != nil {
		log.Printf("qpdf failed to repair the PDF: %s", stderr.String())
	}

	repairedPDF, err := os.ReadFile(tempOutputFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read repaired PDF: %w", err)
	}

	repairedPDFBase64 := base64.StdEncoding.EncodeToString(repairedPDF)

	return repairedPDFBase64, nil
}
