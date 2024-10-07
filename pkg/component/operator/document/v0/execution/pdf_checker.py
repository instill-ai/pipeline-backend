from io import BytesIO
import json
import base64
import sys

# TODO chuang8511:
# Deal with the import error when running the code in the docker container.
# Now, we combine all python code into one file to avoid the import error.
# from pdf_to_markdown import PDFTransformer

if __name__ == "__main__":
    json_str   = sys.stdin.buffer.read().decode('utf-8')
    params     = json.loads(json_str)
    pdf_string = params["PDF"]

    decoded_bytes = base64.b64decode(pdf_string)
    pdf_file_obj = BytesIO(decoded_bytes)
    pdf = PDFTransformer(x=pdf_file_obj)
    pages = pdf.raw_pages
    output = {
        "required": len(pages) == 0,
    }
    print(json.dumps(output))
