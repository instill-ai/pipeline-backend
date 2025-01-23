from io import BytesIO
import json
import base64
import sys
import re
from docling.document_converter import DocumentConverter, PdfFormatOption
from docling.datamodel.base_models import DocumentStream, InputFormat
from docling.datamodel.pipeline_options import PdfPipelineOptions
from docling_core.types.doc import ImageRefMode, PictureItem


if __name__ == "__main__":
    json_str = sys.stdin.buffer.read().decode('utf-8')
    params = json.loads(json_str)
    display_image_tag = params["display-image-tag"]
    display_all_page_image = params["display-all-page-image"]
    pdf_string = params["PDF"]
    if ("resolution" in params and
            params["resolution"] != 0 and
            params["resolution"] is not None):
        resolution = params["resolution"]
    else:
        resolution = 300
    decoded_bytes = base64.b64decode(pdf_string)
    pdf_file_obj = BytesIO(decoded_bytes)

    # Convert resolution DPI to image resolution scale
    image_resolution_scale = resolution / 72.0

    # Initialize variables
    images = []
    all_page_images = []
    page_numbers_with_images = []
    elements = []
    errors = []

    try:
        # Configure the pipeline options
        pipeline_options = PdfPipelineOptions()
        pipeline_options.images_scale = image_resolution_scale
        pipeline_options.generate_page_images = display_all_page_image
        pipeline_options.generate_picture_images = True

        # Initialize the document converter
        source = DocumentStream(name="document.pdf", stream=pdf_file_obj)
        converter = DocumentConverter(
            format_options={
                    InputFormat.PDF: PdfFormatOption(
                        pipeline_options=pipeline_options
                    )
            }
        )

        # Process the PDF document
        doc = converter.convert(source)

        # Extract the markdown text per page
        markdown_pages = [
            doc.document.export_to_markdown(
                page_no=i + 1,
                image_mode=ImageRefMode.PLACEHOLDER
            )
            for i in range(doc.document.num_pages())
        ]

        # Format the image placeholder according to current convention
        image_counter = [0]

        def replace_image(match):
            if display_image_tag:
                replacement = f"![image {image_counter[0]}]({image_counter[0]})"
                image_counter[0] += 1
                return replacement
            else:
                return ""  # Remove the image tag if display-image-tag is False

        for page in range(len(markdown_pages)):
            updated_page = re.sub(
                r"<!-- image -->", replace_image, markdown_pages[page]
            )
            markdown_pages[page] = updated_page

        # Join the markdown pages for the body output
        result = "\n\n".join(markdown_pages)

        # Extract the images/figures from the document
        for element, _level in doc.document.iterate_items():
            if isinstance(element, PictureItem):
                image = element.get_image(doc.document)
                images.append(str(element.image.uri))
                page_numbers_with_images.append(element.prov[0].page_no)

        # Extract images of the full pages for pages that contain images/figures
        if display_all_page_image:
            for page_no, page in doc.document.pages.items():
                if page_no in page_numbers_with_images:
                    all_page_images.append(str(page.image.uri))

        # Collate the output
        output = {
            "body": result,
            "images": images,
            "parsing_error": errors,
            "all_page_images": all_page_images,
            "display_all_page_image": display_all_page_image,
            "markdowns": markdown_pages,
        }
        print(json.dumps(output))
    except Exception as e:
        print(json.dumps({"system_error": str(e)}))
