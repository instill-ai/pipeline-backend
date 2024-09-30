from io import BytesIO
import json
import base64
import sys

# TODO chuang8511:
# Deal with the import error when running the code in the docker container.
# Now, we combine all python code into one file to avoid the import error.
# from pdf_to_markdown import PDFTransformer


if __name__ == "__main__":
	json_str = sys.stdin.buffer.read().decode('utf-8')
	params = json.loads(json_str)
	display_image_tag = params["display-image-tag"]
	display_all_page_image = params["display-all-page-image"]
	pdf_string = params["PDF"]
	decoded_bytes = base64.b64decode(pdf_string)
	pdf_file_obj = BytesIO(decoded_bytes)
	pdf = PDFTransformer(pdf_file_obj, display_image_tag)

	result = ""
	images = []
	separator_number = 30
	image_idx = 0
	errors = []
	all_page_images = []

	try:
		times = len(pdf.raw_pages) // separator_number + 1
		for i in range(times):
			pdf = PDFTransformer(pdf_file_obj, display_image_tag, image_idx)
			if i == times - 1:
				pdf.pages = pdf.raw_pages[i*separator_number:]
			else:
				pdf.pages = pdf.raw_pages[i*separator_number:(i+1)*separator_number]

			pdf.preprocess()
			image_idx = pdf.image_index
			result += pdf.execute()
			for image in pdf.base64_images:
				images.append(image)

			if display_all_page_image:
				raw_pages = pdf.raw_pages

				for page_number in pdf.page_numbers_with_images:
					page = raw_pages[page_number - 1]
					page_image = page.to_image(resolution=500)
					encoded_image = PageImageProcessor.encode_image(page_image)
					all_page_images.append(encoded_image)

			errors += pdf.errors

		output = {
			"body": result,
			"images": images,
			"parsing_error": errors,
			"all_page_images": all_page_images,
			"display_all_page_image": display_all_page_image,
		}
		print(json.dumps(output))
	except Exception as e:
		print(json.dumps({"system_error": str(e)}))
