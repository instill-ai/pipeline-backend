package transformer

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"

	md "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type converterOutput struct {
	Body          string   `json:"body"`
	Images        []string `json:"images"`
	AllPageImages []string `json:"all_page_images"`
	AllPage       bool     `json:"display_all_page_image"`
	Markdowns     []string `json:"markdowns"`

	Logs         []string `json:"logs"`
	ParsingError []string `json:"parsing_error"`
	SystemError  string   `json:"system_error"`
}

type markdownTransformer interface {
	transform() (converterOutput, error)
}

type pdfToMarkdownInputStruct struct {
	base64Text          string
	displayImageTag     bool
	displayAllPageImage bool
	resolution          int
}

type pdfToMarkdownTransformer struct {
	fileExtension       string
	engine              string
	pdfToMarkdownStruct pdfToMarkdownInputStruct
	logger              *zap.Logger
}

func (t *pdfToMarkdownTransformer) transform() (converterOutput, error) {
	var output converterOutput

	t0 := time.Now()
	ok := false
	benchmarkLog := t.logger.With(zap.Time("start", t0))
	defer func() {
		benchmarkLog.Info("PDF to Markdown conversion",
			zap.Float64("durationInSecs", time.Since(t0).Seconds()),
			zap.Bool("ok", ok),
		)
	}()

	pdfBase64 := util.TrimBase64Mime(t.pdfToMarkdownStruct.base64Text)
	shouldRepair, err := requiresRepair(t.pdfToMarkdownStruct.base64Text)
	if err != nil { // Non-blocking error
		t.logger.Error("Failed to check PDF state", zap.Error(err))
	}

	if shouldRepair {
		repairedPDF, err := repairPDF(pdfBase64, t.logger)
		if err != nil {
			return output, fmt.Errorf("repairing PDF: %w", err)
		}

		pdfBase64 = repairedPDF
	}
	benchmarkLog = benchmarkLog.With(zap.Time("repair", time.Now()))

	paramsJSON, err := json.Marshal(map[string]interface{}{
		"PDF":                    pdfBase64,
		"display-image-tag":      t.pdfToMarkdownStruct.displayImageTag,
		"display-all-page-image": t.pdfToMarkdownStruct.displayAllPageImage,
		"resolution":             t.pdfToMarkdownStruct.resolution,
	})
	if err != nil {
		return output, fmt.Errorf("marshalling conversion params: %w", err)
	}

	var pythonCode string
	switch t.engine {
	case "docling":
		pythonCode = doclingPDFToMDConverter
	default:
		pythonCode = pageImageProcessor + pdfTransformer + pdfPlumberPDFToMDConverter
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

	outputBytes, err := cmdRunner.Output()
	benchmarkLog = benchmarkLog.With(zap.Time("convert", time.Now()))
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

	if len(output.Logs) > 0 {
		t.logger.Info("PDF to Markdown Python script produced conversion logs",
			zap.Strings("conversionLogs", output.Logs),
		)
	}

	benchmarkLog = benchmarkLog.With(zap.Time("handleOutput", time.Now()))
	ok = true

	return output, nil
}

// docToMarkdownTransformer is a transformer for DOC and DOCX files. It converts
// the file to PDF and then to Markdown.
type docToMarkdownTransformer struct {
	*pdfToMarkdownTransformer
	base64EncodedText string
}

func (t *docToMarkdownTransformer) transform() (converterOutput, error) {
	if err := t.validate(); err != nil {
		return converterOutput{}, fmt.Errorf("validate input: %w", err)
	}

	base64PDF, err := ConvertToPDF(t.base64EncodedText, t.fileExtension)

	if err != nil {
		return converterOutput{}, fmt.Errorf("convert file to PDF: %w", err)
	}

	t.pdfToMarkdownStruct.base64Text = base64PDF

	return t.pdfToMarkdownTransformer.transform()
}

func (t docToMarkdownTransformer) validate() error {
	if t.pdfToMarkdownStruct.base64Text != "" {
		return fmt.Errorf("PDF struct base64 text should be empty before transformation")
	}
	return nil
}

// pptToMarkdownTransformer is a transformer for PPT and PPTX files. It converts
// the file to PDF and then to markdown.
type pptToMarkdownTransformer struct {
	*pdfToMarkdownTransformer
	base64EncodedText string
}

func (t *pptToMarkdownTransformer) transform() (converterOutput, error) {

	if err := t.validate(); err != nil {
		return converterOutput{}, fmt.Errorf("validate input: %w", err)
	}

	base64PDF, err := ConvertToPDF(t.base64EncodedText, t.fileExtension)

	if err != nil {
		return converterOutput{}, fmt.Errorf("convert file to PDF: %w", err)
	}

	t.pdfToMarkdownStruct.base64Text = base64PDF

	return t.pdfToMarkdownTransformer.transform()
}

func (t pptToMarkdownTransformer) validate() error {
	if t.pdfToMarkdownStruct.base64Text != "" {
		return fmt.Errorf("PDF struct base64 text should be empty before transformation")
	}
	return nil
}

type htmlToMarkdownTransformer struct {
	base64EncodedText string
}

func (t *htmlToMarkdownTransformer) transform() (converterOutput, error) {

	data, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(t.base64EncodedText))
	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	converter := md.NewConverter("", true, nil)

	html := string(data)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to convert HTML to markdown: %w", err)
	}

	return converterOutput{Body: markdown}, nil
}

type xlsxToMarkdownTransformer struct {
	base64EncodedText string
}

func (t *xlsxToMarkdownTransformer) transform() (converterOutput, error) {
	base64String := strings.Split(t.base64EncodedText, ",")[1]
	fileContent, err := base64.StdEncoding.DecodeString(base64String)

	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	reader := bytes.NewReader(fileContent)

	f, err := excelize.OpenReader(reader)
	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to open reader: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()

	var result string
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)

		if err != nil {
			return converterOutput{}, fmt.Errorf("failed to get rows: %w", err)
		}

		if len(rows) == 0 {
			result += fmt.Sprintf("# %s\n", sheet)
			result += "No data found\n\n"
			continue
		}

		result += fmt.Sprintf("# %s\n", sheet)
		result += util.ConvertDataFrameToMarkdownTable(rows)
		result += "\n\n"
	}

	return converterOutput{Body: result}, nil
}

type xlsToMarkdownTransformer struct {
	base64EncodedText string
}

func (t *xlsToMarkdownTransformer) transform() (converterOutput, error) {
	base64String := strings.Split(t.base64EncodedText, ",")[1]
	fileContent, err := base64.StdEncoding.DecodeString(base64String)

	output := converterOutput{}

	if err != nil {
		return output, fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	reader := bytes.NewReader(fileContent)

	xlsFile, err := xls.OpenReader(reader, "utf-8")
	if err != nil {
		return output, fmt.Errorf("failed to open XLS reader: %w", err)
	}

	result := ""
	for i := 0; i < xlsFile.NumSheets(); i++ {
		sheet := xlsFile.GetSheet(i)
		if sheet == nil {
			continue
		}

		result += fmt.Sprintf("# %s\n", sheet.Name)
		dataFrame := make([][]string, 0)

		for rowIndex := 0; rowIndex <= int(sheet.MaxRow); rowIndex++ {
			row := sheet.Row(rowIndex)
			if row == nil {
				continue
			}
			dataRow := make([]string, 0)
			for colIndex := 0; colIndex <= int(row.LastCol()); colIndex++ {
				cell := row.Col(colIndex)
				dataRow = append(dataRow, cell)
			}
			dataFrame = append(dataFrame, dataRow)
		}

		result += util.ConvertDataFrameToMarkdownTable(dataFrame)
		result += "\n\n"
	}

	output.Body = result
	return output, nil

}

type csvToMarkdownTransformer struct {
	base64EncodedText string
}

func (t *csvToMarkdownTransformer) transform() (converterOutput, error) {
	base64String := strings.Split(t.base64EncodedText, ",")[1]
	fileContent, err := base64.StdEncoding.DecodeString(base64String)

	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(fileContent))

	records, err := reader.ReadAll()

	if err != nil {
		return converterOutput{}, fmt.Errorf("failed to read csv: %w", err)
	}

	result := util.ConvertDataFrameToMarkdownTable(records)

	return converterOutput{Body: result}, nil
}

func writeDecodeToFile(base64Str string, file *os.File) error {
	data, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(base64Str))
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

func encodeFileToBase64(inputPath string) (string, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ConvertToPDF converts a base64 encoded document to a PDF.
func ConvertToPDF(base64Encoded, fileExtension string) (string, error) {
	tempPpt, err := os.CreateTemp("", "temp_document.*."+fileExtension)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary document: %w", err)
	}
	inputFileName := tempPpt.Name()
	defer os.Remove(inputFileName)

	err = writeDecodeToFile(base64Encoded, tempPpt)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "libreoffice")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %s", err.Error())
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", inputFileName)
	cmd.Env = append(os.Environ(), "HOME="+tempDir)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute LibreOffice command: %s", err.Error())
	}

	// LibreOffice is not executed in temp directory like inputFileName.
	// The generated PDF is not in temp directory.
	// So, we need to remove the path and keep only the file name.
	noPathFileName := filepath.Base(inputFileName)
	tempPDFName := strings.TrimSuffix(noPathFileName, filepath.Ext(inputFileName)) + ".pdf"
	defer os.Remove(tempPDFName)

	base64PDF, err := encodeFileToBase64(tempPDFName)

	if err != nil {
		// In the different containers, we have the different versions of LibreOffice, which means the behavior of LibreOffice may be different.
		// So, we need to handle the case when the generated PDF is not in the temp directory.
		if fileExtension == "pdf" {
			base64PDF, err := encodeFileToBase64(inputFileName)
			if err != nil {
				return "", fmt.Errorf("failed to encode file to base64: %w", err)
			}
			return base64PDF, nil
		}
	}
	return base64PDF, nil
}
