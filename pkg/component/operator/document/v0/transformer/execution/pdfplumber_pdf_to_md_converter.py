import base64
import json
import logging
import sys
from io import BytesIO, StringIO

# TODO chuang8511:
# Deal with the import error when running the code in the docker container.
# Now, we combine all python code into one file to avoid the import error.
# from pdf_to_markdown import PDFTransformer
# from pdf_to_markdown import PageImageProcessor


if __name__ == "__main__":
    # Capture all stderr output and logging warnings/errors
    stderr_capture = StringIO()
    conversion_logs = StringIO()

    # Redirect stderr to capture all stderr output
    original_stderr = sys.stderr
    sys.stderr = stderr_capture

    # Set up logging to capture warnings and errors
    log_handler = logging.StreamHandler(conversion_logs)
    log_handler.setLevel(logging.WARNING)

    # Remove any existing handlers to avoid duplicate logging
    logging.getLogger().handlers = []

    # Add the handler to capture warnings/errors
    logging.getLogger().addHandler(log_handler)

    try:
        json_str = sys.stdin.buffer.read().decode('utf-8')
        params = json.loads(json_str)
        display_image_tag = params["display-image-tag"]
        display_all_page_image = params["display-all-page-image"]
        pdf_string = params["PDF"]
        if "resolution" in params and params["resolution"] != 0 and params["resolution"] != None:
            resolution = params["resolution"]
        else:
            resolution = 300
        decoded_bytes = base64.b64decode(pdf_string)
        pdf_file_obj = BytesIO(decoded_bytes)
        pdf = PDFTransformer(pdf_file_obj, display_image_tag)

        result = ""
        images = []
        separator_number = 30
        image_index = 0
        errors = []
        all_page_images = []
        markdowns = []

        times = len(pdf.raw_pages) // separator_number + 1
        for i in range(times):
            pdf = PDFTransformer(x=pdf_file_obj, display_image_tag=display_image_tag, image_index=image_index, resolution=resolution)
            if i == times - 1:
                pdf.pages = pdf.raw_pages[i*separator_number:]
            else:
                pdf.pages = pdf.raw_pages[i*separator_number:(i+1)*separator_number]

            pdf.preprocess()
            image_index = pdf.image_index
            result += pdf.execute()

            for image in pdf.base64_images:
                images.append(image)

            if display_all_page_image:
                raw_pages = pdf.raw_pages

                for page_number in pdf.page_numbers_with_images:
                    page = raw_pages[page_number - 1]
                    page_image = page.to_image(resolution=resolution)
                    encoded_image = PageImageProcessor.encode_image(page_image)
                    all_page_images.append(encoded_image)

            errors += pdf.errors
            markdowns += pdf.markdowns

        # Combine all captured output
        all_logs = []
        stderr_content = stderr_capture.getvalue().strip()
        if stderr_content:
            all_logs.extend(stderr_content.splitlines())
        log_content = conversion_logs.getvalue().strip()
        if log_content:
            all_logs.extend(log_content.splitlines())

        output = {
            "body": result,
            "images": images,
            "parsing_error": errors,
            "all_page_images": all_page_images,
            "display_all_page_image": display_all_page_image,
            "markdowns": markdowns,
            "logs": all_logs,
        }

        # Restore original stderr for the final output
        sys.stderr = original_stderr
        print(json.dumps(output))

    except Exception as e:
        # Restore original stderr before printing error
        sys.stderr = original_stderr
        print(json.dumps({"system_error": str(e)}), file=sys.stderr)
