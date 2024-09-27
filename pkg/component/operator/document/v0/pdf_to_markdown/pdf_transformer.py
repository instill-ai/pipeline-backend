
from io import BytesIO
from collections import Counter

import pdfplumber
from pdfplumber.page import Page

# TODO: Deal with the import error when running the code in the docker container
# from page_image_processor import PageImageProcessor


class PDFTransformer:
	pdf: pdfplumber.PDF
	raw_pages: list[Page]
	metadata: dict
	display_image_tag: bool
	image_index: int
	errors: list[str]
	pages: list[Page]
	lines: list[dict]
	images: list[dict]
	tables: list[dict]
	base64_images: list[dict]
	page_numbers_with_images: list[int]

	def __init__(self, x: BytesIO, display_image_tag: bool = False, image_index: int = 0):
		self.pdf = pdfplumber.open(x)
		self.raw_pages = self.pdf.pages
		self.metadata = self.pdf.metadata
		self.display_image_tag = display_image_tag
		self.image_index = image_index
		self.errors = []
		self.page_numbers_with_images = []

	def preprocess(self):
		self.set_heights()
		self.lines = []
		self.tables = []
		self.images = []
		self.base64_images = []
		if self.display_image_tag:
			self.process_image(self.image_index)

		for page in self.pages:
			page_lines = page.extract_text_lines(layout=True, x_tolerance_ratio=0.1, return_chars= False)
			page.flush_cache()
			page.get_textmap.cache_clear()

			self.process_line(page_lines, page.page_number)
			self.process_table(page)

		self.set_paragraph_information(self.lines)

		self.result = ""

	def process_image(self, i: int):
		image_index = i
		for page in self.pages:
			image_processor = PageImageProcessor(page=page, image_index=image_index)
			image_processor.produce_images_by_blocks()
			processed_images = image_processor.images
			self.images += processed_images
			image_index = image_processor.image_index

			if page.page_number not in self.page_numbers_with_images:
				self.page_numbers_with_images.append(page.page_number)

		self.image_index = image_processor.image_index

	def set_heights(self):
		tolerance = 0.95
		heights = []
		largest_text_height, second_largest_text_height = 0, 0
		for page in self.pages:
			lines = page.extract_text_lines(layout=True, x_tolerance_ratio=0.1, return_chars= False)
			page.flush_cache()
			page.get_textmap.cache_clear()
			for line in lines:
				height = int(line["bottom"] - line["top"])
				heights.append(height)
				if height > largest_text_height:
					second_largest_text_height = largest_text_height
					largest_text_height = height
				elif height > second_largest_text_height and height < largest_text_height:
					second_largest_text_height = height

		counter = Counter(heights)

		# if there are too many subtitles, we don't use the title height.
		# 50 is a temp number. It should be tuned.
		if counter[largest_text_height] > 50:
			self.title_height = float("inf")
		else:
			self.title_height = round(largest_text_height * tolerance)

		if counter[second_largest_text_height] > 50 or self.title_height == float("inf"):
			self.subtitle_height = float("inf")
		else:
			self.subtitle_height = round(second_largest_text_height * tolerance)

	def set_paragraph_information(self, lines: list[dict]):
		def round_to_nearest_upper_bound(value, step=3): # for the golden sample case
			"""
			Round the value to the nearest upper bound based on the given step.
			For example, with step=3: 0~3 -> 3, 3~6 -> 6, etc.
			"""
			return ((value // step) + 1) * step

		distances = []
		paragraph_width = 0
		distances_to_left = []

		for _, line in enumerate(lines):
			if line["distance_to_next_line"] and line["distance_to_next_line"] > 0:
				# Round the distance to the nearest integer and add to the list
				rounded_distance = round_to_nearest_upper_bound(line["distance_to_next_line"])
				distances.append(rounded_distance)

			if line["line_width"] > paragraph_width:
				paragraph_width = line["line_width"]

			if line["x0"]:
				distances_to_left.append(line["x0"])

		# Find the most common distance
		if distances:
			common_distance = Counter(distances).most_common(1)[0][0]
		else:
			common_distance = 10 ## default value

		if distances_to_left:
			zero_indent_distance = min(distances_to_left)
		else:
			zero_indent_distance = 0
		paragraph_distance = common_distance * 1.5
		self.paragraph_distance = paragraph_distance
		self.paragraph_width = paragraph_width
		self.zero_indent_distance = zero_indent_distance

	def execute(self):
		self.set_line_type(self.title_height, self.subtitle_height, "indent")
		self.result = self.transform_line_to_markdown(self.lines)
		return self.result

	# It can add more calculation for the future development when we want to extend more use cases.
	def process_line(self, lines: list[dict], page_number: int):
		for idx, line in enumerate(lines):
			line["line_height"] = line["bottom"] - line["top"]
			line["line_width"] = line["x1"] - line["x0"]
			line["middle"] = (line["x1"] + line["x0"]) / 2
			line["distance_to_next_line"] = lines[idx+1]["top"] - line["bottom"] if idx < len(lines) - 1 else None
			line["page_number"] = page_number
			self.lines.append(line)

	def process_table(self, page: Page):
		tables = page.find_tables(
			table_settings={
				"vertical_strategy": "lines",
				"horizontal_strategy": "lines",
				}
		)
		if tables:
			for table in tables:
				table_info = {}
				table_info["bbox"] = table.bbox
				text = table.extract()
				table_info["text"] = text
				table_info["page_number"] = page.page_number
				self.tables.append(table_info)

	# TODO: Implement paragraph strategy
	def paragraph_strategy(self, lines: list[dict], subtitle_height: int = 14):
		# TODO: Implement paragraph strategy
		# judge the non-title line in a page.
		# If there is a line with indent, return "indent"
		# If there is a line with no indent, return "no-indent"
		return "indent"
		paragraph_lines_start_positions = []
		for line in lines:
			if line["line_height"] < subtitle_height:
				paragraph_lines_start_positions.append(line["x0"])

	def set_line_type(self, title_height: int = 16, subtitle_height: int = 14, paragraph_strategy: str = "indent"):
		lines = self.lines
		current_paragraph = []
		paragraph_start_position = 0
		paragraph_idx = 1

		for i, line in enumerate(lines):
			if line['line_height'] >= title_height:
				line["type"] = 'title'
				if current_paragraph:
					for line_in_paragraph in current_paragraph:
						line_in_paragraph["type"] = f'paragraph {paragraph_idx}'
					paragraph_idx += 1
					current_paragraph = []

			elif line['line_height'] >= subtitle_height:
				line["type"] = 'subtitle'
				if current_paragraph:
					for line_in_paragraph in current_paragraph:
						line_in_paragraph["type"] = f'paragraph {paragraph_idx}'
					paragraph_idx += 1
					current_paragraph = []
			else:
				line["type"] = 'paragraph'
				if current_paragraph:
					current_paragraph.append(line)

					if ((paragraph_strategy == "indent" and i < len(lines) - 1 and
							(   # if the next line starts a new paragraph
								abs(lines[i+1]['x0'] - paragraph_start_position) < 10
								# if the next line is not in the same layer
								# or abs(line["middle"] - lines[i+1]["middle"]) > 5
								)
							) or
						(paragraph_strategy == "no-indent"
							and line["distance_to_next_line"]
							and line["distance_to_next_line"] > 10) or
						(i == len(lines) - 1) # final line
						):

						for line_in_paragraph in current_paragraph:
							line_in_paragraph["type"] = f'paragraph {paragraph_idx}'

						paragraph_idx += 1
						current_paragraph = []
				else:
					current_paragraph = [line]
					paragraph_start_position = line["x0"]
		self.lines = lines

	def transform_line_to_markdown(self, lines: list[dict]):
		result = ""
		to_be_processed_table = []
		for i, line in enumerate(lines):
			table = self.meet_table(line, line["page_number"])
			if table and table not in to_be_processed_table:
				to_be_processed_table.append(table)
			elif table and table in to_be_processed_table:
				continue
			elif to_be_processed_table:
				for table in to_be_processed_table:
					result += "\n\n"
					result += self.transform_table_markdown(table)
					result += "\n\n"
					self.tables.remove(table)
				to_be_processed_table = []

				if (i > 0 and
					("title" == lines[i-1]["type"] and "title" == lines[i]["type"] or
	  				"subtitle" == lines[i-1]["type"] and "subtitle" == lines[i]["type"])
					):
					while len(result) > 0 and result[-1] == "\n":
						result = result[:-1]

					line_text = self.line_process(line, i, lines, result)
					## If line_text prefix or suffix is \n, remove them
					while line_text.startswith("\n") or line_text.endswith("\n"):
						line_text = line_text.strip("\n")
				else:
					line_text = self.line_process(line, i, lines, result)
					while (
						(line_text.startswith("\n") or line_text.endswith("\n"))):
						line_text = line_text.strip("\n")

				result += line_text
				result += "\n"
				## TODO: Do not change another line if it is bullet point or numbered list.
				if (
					(line["distance_to_next_line"] and line["distance_to_next_line"] >= self.paragraph_distance) or
					(
						line["page_number"] != lines[i+1]["page_number"] if i < len(lines) - 1 else False
						and line["line_width"] < self.paragraph_width * 0.8
					)
					):
					result += "\n"

			else:
				if (i > 0 and
					("title" == lines[i-1]["type"] and "title" == lines[i]["type"] or
	  				"subtitle" == lines[i-1]["type"] and "subtitle" == lines[i]["type"])
					):
					while len(result) > 0 and result[-1] == "\n":
						result = result[:-1]

					line_text = self.line_process(line, i, lines, result)
					## If line_text prefix or suffix is \n, remove them
					while line_text.startswith("\n") or line_text.endswith("\n"):
						line_text = line_text.strip("\n")
				else:
					line_text = self.line_process(line, i, lines, result)
					while (
						(line_text.startswith("\n") or line_text.endswith("\n"))):
						line_text = line_text.strip("\n")

				result += line_text

				## TODO: Do not change another line if it is bullet point or numbered list.
				if (
					(line["distance_to_next_line"] and line["distance_to_next_line"] >= self.paragraph_distance) or
					(
						line["page_number"] != lines[i+1]["page_number"] if i < len(lines) - 1 else False
						and line["line_width"] < self.paragraph_width * 0.8
					)
					):
					result += "\n"
				result += "\n"


			if i < len(lines) - 1:
				result += self.insert_image(line, lines[i+1])
			else:
				result += self.insert_image(line, None)
		if self.tables:
			processed_table = []
			for table in self.tables:
				result += "\n\n"
				result += self.transform_table_markdown(table)
				result += "\n\n"
				processed_table.append(table)
			for table in processed_table:
				self.tables.remove(table)

		return result

	def line_process(self, line: dict, i: int, lines: list[dict], current_result: str):
		result = ""
		if "type" not in line:
			return line["text"]
		if line["type"] == "title":
			if current_result != "":
				result += "\n\n"
			if i > 0 and lines[i-1]["type"] == "title":
				result += f" {line['text']}\n"
			else:
				result += f"# {line['text']}\n"
		elif line["type"] == "subtitle":
			if current_result != "":
				result += "\n\n"
			if i > 0 and lines[i-1]["type"] == "subtitle":
				result += f" {line['text']}\n"
			else:
				result += f"## {line['text']}\n"
		elif "paragraph" in line["type"]:
			# Deal with indentation
			if self.zero_indent_distance != 0:
				indent = round((line["x0"] - self.zero_indent_distance) // 10)  # to be tuned
				if indent > 0:
					result += " " * indent

			result += line["text"]
			if (
				(i < len(lines) - 1) and
				"type" in lines[i+1] and
				len(lines[i+1]["type"].split(" ")) == 2 and
				(int(line["type"].split(" ")[1]) < int(lines[i+1]["type"].split(" ")[1]))
			):
				result += "\n"
				result += "\n"
		return result

	def meet_table(self, line: dict, page_number: int):
		tables = self.tables
		for table in tables:
			if table["page_number"] == page_number:
				bbox = table["bbox"]
				top, bottom = bbox[1], bbox[3]
				if line["top"] > top and line["bottom"] < bottom:
					return table
				else:
					None

	def transform_table_markdown(self, table: dict):
		result = ""
		texts = table["text"]
		for i, row in enumerate(texts):
			for j, col in enumerate(row):
				if col:
					if "\n" in col:
						col = col.replace("\n", "<br>")
					result += col

					if j < len(row) - 1:
						result += " | "
				else:
					if j == 0:
						result += "||"
					else:
						result += "|"
			if i == 0:
				result += "\n"
				## TODO: Judge table that cross the page,
				result += "|"
				result += " --- |" * len(row)
				result += "\n"
			elif i < len(texts) - 1:
				result += "\n"

		return result

	def insert_image(self, line: dict, next_line: dict):
		result = ""
		images = self.images
		to_be_removed_images = []

		if images:
			if next_line:
				# If there is image between line and next_line, we insert image.
				if next_line["page_number"] == line["page_number"]:
					for image in images:
						if image["page_number"] == line["page_number"] and image["top"] > line["bottom"] and image["bottom"] < next_line["top"]:
							result += "\n\n"
							result += f"![image {image['img_number']}]({image['img_number']})"
							self.base64_images.append(image["img_base64"])
							result += "\n\n"
							to_be_removed_images.append(image)
				elif next_line["page_number"] > line["page_number"]:
					for image in images:
						if image["page_number"] >= line["page_number"] and image["page_number"] < next_line["page_number"]:
							result += "\n\n"
							result += f"![image {image['img_number']}]({image['img_number']})"
							self.base64_images.append(image["img_base64"])
							result += "\n\n"
							to_be_removed_images.append(image)

			else: # if images exists and there is no next_line, we insert image.
				for image in images:
					result += "\n\n"
					result += f"![image {image['img_number']}]({image['img_number']})"
					self.base64_images.append(image["img_base64"])
					result += "\n\n"
					to_be_removed_images.append(image)
		for image in to_be_removed_images:
			self.images.remove(image)

		return result
