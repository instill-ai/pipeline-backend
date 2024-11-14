from io import BytesIO
import json
import sys
import base64
import pdfplumber

if __name__ == "__main__":
    try:
        json_str   = sys.stdin.buffer.read().decode('utf-8')
        params     = json.loads(json_str)
        pdf_string = params["PDF"]
        decoded_bytes = base64.b64decode(pdf_string)
        pdf_file_obj = BytesIO(decoded_bytes)

        pdf = pdfplumber.open(pdf_file_obj)

        output = {
            "page_numbers": len(pdf.pages)
        }

        print(json.dumps(output))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
