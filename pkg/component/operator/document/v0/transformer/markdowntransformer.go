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
	"sync"
	"time"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"

	md "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

var (
	// libreOfficeMutex ensures only one LibreOffice process runs at a time
	// This prevents race conditions and permission issues when multiple processes
	// try to initialize LibreOffice user profiles simultaneously
	libreOfficeMutex sync.Mutex
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

	params := map[string]interface{}{
		"PDF":                    pdfBase64,
		"display-image-tag":      t.pdfToMarkdownStruct.displayImageTag,
		"display-all-page-image": t.pdfToMarkdownStruct.displayAllPageImage,
		"resolution":             t.pdfToMarkdownStruct.resolution,
	}

	var pythonCode string
	switch t.engine {
	case "docling":
		pythonCode = doclingPDFToMDConverter
	default:
		pythonCode = pageImageProcessor + pdfTransformer + pdfPlumberPDFToMDConverter
	}

	outputBytes, err := util.ExecutePythonCode(pythonCode, params)

	benchmarkLog = benchmarkLog.With(zap.Time("convert", time.Now()))
	if err != nil {
		return output, fmt.Errorf("running Python script: %w", err)
	}

	err = json.Unmarshal(outputBytes, &output)
	if err != nil {
		return output, fmt.Errorf("unmarshalling output: %w", err)
	}

	if output.SystemError != "" {
		// There are documents that will fail to be converted to MD. Usually,
		// the document-to-markdwon task is a step in a pipeline to obtain an
		// approximate result or to feed it into other components that will
		// refine it. In most cases, we don't want this failure to stop the
		// execution of the pipeline, so we continue with a blank result.
		//
		// TODO jvallesm: INS-8156 implements a failover mechanism so pipeline
		// recipes can determine if a component failure is fatal or not. When
		// that's implemented, we should return an error here instead of
		// continuing.
		t.logger.Error("Failed to convert PDF to Markdown. Continuing with empty result.", zap.String("systemError", output.SystemError))
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

// ConvertToFormat converts a base64 encoded document to the specified target
// format using LibreOffice (or wkhtmltopdf for HTML-to-PDF). It serializes
// operations with a mutex to prevent race conditions.
func ConvertToFormat(base64Encoded, fileExtension, targetFormat string) (string, error) {
	libreOfficeMutex.Lock()
	defer libreOfficeMutex.Unlock()

	tempFile, err := os.CreateTemp("", "temp_document.*."+fileExtension)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary document: %w", err)
	}
	inputFileName := tempFile.Name()
	defer os.Remove(inputFileName)

	err = writeDecodeToFile(base64Encoded, tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 to file: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "conversion")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Chmod(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to set permissions on temporary directory: %w", err)
	}

	var cmd *exec.Cmd
	var outputFileName string

	// Use wkhtmltopdf for HTML-to-PDF if available, otherwise use LibreOffice
	if fileExtension == "html" && targetFormat == "pdf" {
		outputFileName = filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(inputFileName), ".html")+".pdf")

		if _, err := exec.LookPath("wkhtmltopdf"); err == nil {
			cmd = exec.Command("wkhtmltopdf", inputFileName, outputFileName)
		} else {
			cmd = exec.Command("libreoffice", "--headless", "--infilter=HTML", "--convert-to", "pdf", "--outdir", tempDir, inputFileName)
			cmd.Env = append(os.Environ(), "HOME="+tempDir)
			outputFileName = filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(inputFileName), ".html")+".pdf")
		}
	} else {
		cmd = exec.Command("libreoffice", "--headless", "--convert-to", targetFormat, "--outdir", tempDir, inputFileName)
		cmd.Env = append(os.Environ(), "HOME="+tempDir)
		outputFileName = filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(inputFileName), "."+fileExtension)+"."+targetFormat)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute conversion command: %s (output: %s)", err.Error(), string(output))
	}

	if _, err := os.Stat(outputFileName); os.IsNotExist(err) {
		noPathFileName := filepath.Base(inputFileName)
		standardName := filepath.Join(tempDir, strings.TrimSuffix(noPathFileName, filepath.Ext(inputFileName))+"."+targetFormat)
		if _, err := os.Stat(standardName); err == nil {
			outputFileName = standardName
		} else {
			return "", fmt.Errorf("output %s file not found at expected location: %s", targetFormat, outputFileName)
		}
	}

	defer os.Remove(outputFileName)

	base64Output, err := encodeFileToBase64(outputFileName)
	if err != nil {
		if fileExtension == targetFormat {
			base64Output, err := encodeFileToBase64(inputFileName)
			if err != nil {
				return "", fmt.Errorf("failed to encode file to base64: %w", err)
			}
			return base64Output, nil
		}
		return "", fmt.Errorf("failed to encode %s file to base64: %w", targetFormat, err)
	}

	return base64Output, nil
}

// ConvertToPDF converts a base64 encoded document to PDF.
func ConvertToPDF(base64Encoded, fileExtension string) (string, error) {
	return ConvertToFormat(base64Encoded, fileExtension, "pdf")
}
