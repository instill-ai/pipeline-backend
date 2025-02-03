package transformer

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"

	md "github.com/JohannesKaufmann/html-to-markdown"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

// MarkdownTransformerGetterFunc is a function that returns a MarkdownTransformer.
type MarkdownTransformerGetterFunc func(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (MarkdownTransformer, error)

// MarkdownTransformer is an interface for all transformers that convert a document to markdown.
type MarkdownTransformer interface {
	// Transform converts a document to markdown.
	Transform() (converterOutput, error)
}

// PDFToMarkdownTransformer is a transformer for PDF files. It converts the file to markdown.
type PDFToMarkdownTransformer struct {
	// FileExtension is the file extension of the document.
	FileExtension string
	// PDFToMarkdownStruct is the input struct for the PDF to markdown converter.
	PDFToMarkdownStruct pdfToMarkdownInputStruct
	// PDFConvertFunc is the function that converts the PDF to markdown.
	PDFConvertFunc func(pdfToMarkdownInputStruct) (converterOutput, error)
}

// Transform converts a PDF file to markdown.
func (t PDFToMarkdownTransformer) Transform() (converterOutput, error) {
	return t.PDFConvertFunc(t.PDFToMarkdownStruct)
}

// DocxDocToMarkdownTransformer is a transformer for DOC and DOCX files. It converts the file to PDF and then to markdown.
type DocxDocToMarkdownTransformer struct {
	// FileExtension is the file extension of the document.
	FileExtension string
	// Base64EncodedText is the base64 encoded DOC or DOCX file.
	Base64EncodedText string
	// PDFToMarkdownStruct is the input struct for the PDF to markdown converter.
	PDFToMarkdownStruct pdfToMarkdownInputStruct
	// PDFConvertFunc is the function that converts the PDF to markdown.
	PDFConvertFunc func(pdfToMarkdownInputStruct) (converterOutput, error)
}

// Transform converts a DOC or DOCX file to markdown.
func (t DocxDocToMarkdownTransformer) Transform() (converterOutput, error) {

	if err := t.validate(); err != nil {
		return converterOutput{}, fmt.Errorf("validate input: %w", err)
	}

	base64PDF, err := ConvertToPDF(t.Base64EncodedText, t.FileExtension)

	if err != nil {
		return converterOutput{}, fmt.Errorf("convert file to PDF: %w", err)
	}

	t.PDFToMarkdownStruct.Base64Text = base64PDF

	return t.PDFConvertFunc(t.PDFToMarkdownStruct)
}

func (t DocxDocToMarkdownTransformer) validate() error {
	if t.PDFToMarkdownStruct.Base64Text != "" {
		return fmt.Errorf("PDF struct base64 text should be empty before transformation")
	}
	return nil
}

// PptPptxToMarkdownTransformer is a transformer for PPT and PPTX files. It converts the file to PDF and then to markdown.
type PptPptxToMarkdownTransformer struct {
	// FileExtension is the file extension of the document.
	FileExtension string
	// Base64EncodedText is the base64 encoded PPT or PPTX file.
	Base64EncodedText string
	// PDFToMarkdownStruct is the input struct for the PDF to markdown converter.
	PDFToMarkdownStruct pdfToMarkdownInputStruct
	// PDFConvertFunc is the function that converts the PDF to markdown.
	PDFConvertFunc func(pdfToMarkdownInputStruct) (converterOutput, error)
}

// Transform converts a PPT or PPTX file to markdown.
func (t PptPptxToMarkdownTransformer) Transform() (converterOutput, error) {

	if err := t.validate(); err != nil {
		return converterOutput{}, fmt.Errorf("validate input: %w", err)
	}

	base64PDF, err := ConvertToPDF(t.Base64EncodedText, t.FileExtension)

	if err != nil {
		return converterOutput{}, fmt.Errorf("convert file to PDF: %w", err)
	}

	t.PDFToMarkdownStruct.Base64Text = base64PDF

	return t.PDFConvertFunc(t.PDFToMarkdownStruct)
}

func (t PptPptxToMarkdownTransformer) validate() error {
	if t.PDFToMarkdownStruct.Base64Text != "" {
		return fmt.Errorf("PDF struct base64 text should be empty before transformation")
	}
	return nil
}

// HTMLToMarkdownTransformer is a transformer for HTML files. It converts the file to markdown.
type HTMLToMarkdownTransformer struct {
	// Base64EncodedText is the base64 encoded HTML file.
	Base64EncodedText string
}

// Transform converts an HTML file to markdown.
func (t HTMLToMarkdownTransformer) Transform() (converterOutput, error) {

	data, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(t.Base64EncodedText))
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

// XlsxToMarkdownTransformer is a transformer for XLSX files. It converts the file to markdown.
type XlsxToMarkdownTransformer struct {
	// Base64EncodedText is the base64 encoded XLSX file.
	Base64EncodedText string
}

// Transform converts an XLSX file to markdown.
func (t XlsxToMarkdownTransformer) Transform() (converterOutput, error) {

	base64String := strings.Split(t.Base64EncodedText, ",")[1]
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

// XlsToMarkdownTransformer is a transformer for XLS files. It converts the file to markdown.
type XlsToMarkdownTransformer struct {
	Base64EncodedText string
}

// Transform converts an XLS file to markdown.
func (t XlsToMarkdownTransformer) Transform() (converterOutput, error) {

	base64String := strings.Split(t.Base64EncodedText, ",")[1]
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

// CSVToMarkdownTransformer is a transformer for CSV files. It converts the file to markdown.
type CSVToMarkdownTransformer struct {
	Base64EncodedText string
}

// Transform converts a CSV file to markdown.
func (t CSVToMarkdownTransformer) Transform() (converterOutput, error) {

	base64String := strings.Split(t.Base64EncodedText, ",")[1]
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
