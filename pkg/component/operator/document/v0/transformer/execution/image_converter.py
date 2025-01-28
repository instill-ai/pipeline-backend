from io import BytesIO
import json
import base64
import sys
import pdfplumber

# TODO chuang8511:
# Deal with the import error when running the code in the docker container.
# Now, we combine all python code into one file to avoid the import error.
# from pdf_to_markdown import PageImageProcessor

if __name__ == "__main__":
    json_str   = sys.stdin.buffer.read().decode('utf-8')
    params     = json.loads(json_str)
    filename   = params["filename"]
    pdf_string = params["PDF"]
    page_idx   = params["page_idx"]
    if "resolution" in params and params["resolution"] != 0 and params["resolution"] != None:
        resolution = params["resolution"]
    else:
        resolution = 300

    decoded_bytes = base64.b64decode(pdf_string)
    pdf_file_obj = BytesIO(decoded_bytes)
    pdf = pdfplumber.open(pdf_file_obj)

    page = pdf.pages[page_idx]

    page_image = page.to_image(resolution=resolution)
    encoded_image = PageImageProcessor.encode_image(page_image)

    exclude_file_extension = filename.split(".")[0]
    filename = f"{exclude_file_extension}_{page_idx}.png"

    output = {
        "image": encoded_image,
        "filename": filename,
    }

    print(json.dumps(output))
