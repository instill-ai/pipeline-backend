package transformer

import (
	_ "embed"
)

const (
	pythonInterpreter string = "/opt/venv/bin/python"
)

var (

	//go:embed execution/task_convert_to_markdown.py
	taskConvertToMarkdownExecution string
	//go:embed pdf_to_markdown/pdf_transformer.py
	pdfTransformer string
	//go:embed pdf_to_markdown/page_image_processor.py
	imageProcessor string

	//go:embed execution/task_convert_to_images.py
	taskConvertToImagesExecution string

	//go:embed execution/pdf_checker.py
	pdfChecker string
)
