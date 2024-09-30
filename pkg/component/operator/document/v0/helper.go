package document

import (
	"encoding/json"
	"log"

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

func RepairPDF(pdfBase64 string) (string, error) {
	return ConvertToPDF(pdfBase64, "pdf")
}
