from io import BytesIO
import json
import base64
import sys

# TODO chuang8511:
# Deal with the import error when running the code in the docker container.
# Now, we combine all python code into one file to avoid the import error.
# from pdf_to_markdown import PDFTransformer
# from pdf_to_markdown import PageImageProcessor

if __name__ == "__main__":
    json_str   = sys.stdin.buffer.read().decode('utf-8')
    params     = json.loads(json_str)
    filename   = params["filename"]
    pdf_string = params["PDF"]

    decoded_bytes = base64.b64decode(pdf_string)
    pdf_file_obj = BytesIO(decoded_bytes)
    pdf = PDFTransformer(x=pdf_file_obj)
    pages = pdf.raw_pages
    exclude_file_extension = filename.split(".")[0]
    filenames = []
    images = []

    for i, page in enumerate(pages):
        page_image = page.to_image(resolution=500)
        encoded_image = PageImageProcessor.encode_image(page_image)
        images.append(encoded_image)
        filenames.append(f"{exclude_file_extension}_{i}.png")


    output = {
        "images": images,
        "filename": filenames,
    }

    print(json.dumps(output))
