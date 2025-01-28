package transformer

import (
	_ "embed"
)

const (
	pythonInterpreter string = "/opt/venv/bin/python"
)

var (
	//go:embed pdf_to_markdown/pdf_transformer.py
	pdfTransformer string
	//go:embed pdf_to_markdown/page_image_processor.py
	pageImageProcessor string

	//go:embed execution/docling_pdf_to_md_converter.py
	doclingPDFToMDConverter string

	//go:embed execution/pdfplumber_pdf_to_md_converter.py
	pdfPlumberPDFToMDConverter string

	//go:embed execution/image_converter.py
	imageConverter string

	//go:embed execution/pdf_checker.py
	pdfChecker string

	//go:embed execution/get_page_numbers.py
	getPageNumbersExecution string
)
