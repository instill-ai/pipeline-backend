import base64
from io import BytesIO
from PIL import Image

from pdfplumber.page import Page
from pdfplumber.display import PageImage

class PageImageProcessor:
    page: Page
    errors: list[str]
    images: list[dict]

    def __init__(self, page: Page, image_index: int):
        self.page = page
        self.lines = page.extract_text_lines(layout=True, strip=True, return_chars=False)
        self.images = []
        self.errors = []
        page.flush_cache()
        page.get_textmap.cache_clear()
        self.image_index = image_index

    def produce_images_by_blocks(self) -> None:
        saved_blocks = []
        page = self.page
        images = page.images

        # Process images detect by pdfplumber
        for i, image in enumerate(images):
            bbox = (image["x0"], image["top"], image["x1"], image["bottom"])
		    # There is a bug in pdfplumber that it can't target the image position correctly.
            try:
                img_page = page.crop(bbox=bbox)
            except Exception as e:
                self.errors.append(f"image {i} got error: {str(e)}, so it convert all pages into image.")
                bbox = (0, 0, page.width, page.height)
                img_page = page

            img_obj = img_page.to_image(resolution=500)
            img_base64 = self.__class__.encode_image(image=img_obj)

            image["page_number"] = page.page_number
            image["img_number"] = self.image_index
            self.image_index += 1
            image["img_base64"] = img_base64
            saved_blocks.append(bbox)
            self.images.append(image)

        # (x0, top, x1, bottom)
        blocks = self.calculate_blank_blocks(page=page)

        for i, block in enumerate(blocks):
            block_dict = self.get_block_dict(block)
            block_image = {
                "page_number": int,
                "img_number":  int,
                "img_base64":  str,
                "top":         block_dict["top"],
                "bottom":      block_dict["bottom"],
            }
            overlap = False
            for saved_block in saved_blocks:
                if self.is_overlap(block1=saved_block, block2=block):
                    overlap = True
                    break
            if overlap:
                continue

            if self.image_too_small(block=block):
                continue

            if self.low_possibility_to_be_image(block=block):
                continue

            try:
                cropped_page = page.crop(block)
            except Exception as e:
                self.errors.append(f"image {i} got error: {str(e)}, do not convert the images.")
                continue

            im = cropped_page.to_image(resolution=200)

            if self.is_blank_pil_image(im=im):
                continue

            img_base64 = self.__class__.encode_image(image=im)
            block_image["page_number"] = page.page_number
            block_image["img_number"] = self.image_index
            self.image_index += 1
            block_image["img_base64"] = img_base64
            self.images.append(block_image)

    def encode_image(image: PageImage) -> str:
        buffer = BytesIO()
        image.save(buffer, format="PNG")
        buffer.seek(0)
        img_data = buffer.getvalue()
        return "data:image/png;base64," + base64.b64encode(img_data).decode("utf-8")

    def get_block_dict(self, block: tuple) -> dict:
        return {
            "x0":     block[0],
            "top":    block[1],
            "x1":     block[2],
            "bottom": block[3],
        }

    def calculate_blank_blocks(self, page: Page) -> list[tuple]:
        page_width = page.width
        page_height = page.height
        lines = self.lines

        page.flush_cache()
        page.get_textmap.cache_clear()

        blank_blocks = []

        # Track the bottom of the last line processed
        last_bottom = 0  # Start from the top of the page

        # Check for empty spaces before the first line
        if lines:
            first_line = lines[0]
            if first_line["top"] > 0:
                blank_blocks.append((0, 0, page_width, first_line["top"]))

        # Process each line to find blank areas between them
        for i, line in enumerate(lines):
            # Calculate the blank space above the current line
            if i > 0:
                previous_line = lines[i - 1]
                if line["top"] > previous_line["bottom"]:
                    # (x0, top, x1, bottom)
                    blank_blocks.append((0, previous_line["bottom"], page_width, line["top"]))

            # Update last_bottom to the current line's bottom
            last_bottom = line["bottom"]

        # Check for empty spaces after the last line
        if last_bottom < page_height:
            blank_blocks.append((0, last_bottom, page_width, page_height))


        return blank_blocks + self.calculate_horizontal_blocks(lines=lines, page_width=page_width, tolerance=30)

    def calculate_horizontal_blocks(self, lines: list[dict[str, any]], page_width: float, tolerance: float) -> list[tuple]:
        """
        Calculates horizontal blocks (blank spaces) on the left or right side of text lines.

        Parameters:
        - lines: A list of dictionaries, each representing a line of text with 'x0', 'x1', 'top', and 'bottom' attributes.
        - page_width: The width of the page being processed.
        - tolerance: to tolerate the block judgement

        Returns:
        - A list of tuples representing the horizontal blocks (x0, top, x1, bottom) where no text exists.
        """
        if not lines:
            return []

        left_blocks = []
        right_blocks = []

        # Sort the lines by their vertical position (top)
        sorted_lines = sorted(lines, key=lambda l: l["top"])

        found_block = False
        block_start_line = None

        for i in range(1, len(sorted_lines)):

            # Check if the block starts with first line
            # 4 is number to be tuned
            if not block_start_line and page_width / 4 < sorted_lines[i]["x0"]:
                block_start_line = sorted_lines[i]
                line_count = 1

            current_line = sorted_lines[i]
            previous_line = sorted_lines[i - 1]
            if not found_block and block_start_line and abs(current_line["x0"] - block_start_line["x0"]) < tolerance:
                line_count += 1
                if line_count > 5:
                    found_block = True
                    block_start_top = block_start_line["top"]

            elif not block_start_line and abs(current_line["x0"] - previous_line["x0"]) > tolerance:
                block_start_line = current_line
                line_count = 1
            elif not found_block and block_start_line:
                block_start_line = None
                line_count = 0

            if found_block and abs(current_line["x0"] - block_start_line["x0"]) > tolerance:
                # Finalize the left block up to the previous line
                left_blocks.append((0, block_start_top, previous_line["x0"], previous_line["bottom"]))
                found_block= False
                block_start_line = None
                line_count = 0


        found_block = False
        block_start_line = None

        for i in range(1, len(sorted_lines)):
            if not block_start_line and page_width / 4 < page_width - sorted_lines[i]["x1"]:
                block_start_line = sorted_lines[i]
                line_count = 1

            current_line = sorted_lines[i]
            previous_line = sorted_lines[i - 1]

            if not found_block and block_start_line and abs(current_line["x1"] - block_start_line["x1"]) < tolerance:
                line_count += 1
                if line_count > 5:
                    found_block = True
                    block_start_top = block_start_line["top"]

            elif not block_start_line and (current_line["x1"] - previous_line["x1"]) > tolerance:
                block_start_line = current_line
                line_count = 1

            elif not found_block and block_start_line:
                block_start_line = None
                line_count = 0

            if found_block and abs(current_line["x1"] - block_start_line["x1"]) > tolerance:
                # Finalize the right block up to the previous line
                right_blocks.append((previous_line["x1"], block_start_top, page_width, previous_line["bottom"]))
                found_block= False
                block_start_line = None
                line_count = 0

        return left_blocks + right_blocks

    # (x0, top, x1, bottom)
    def image_too_small(self, block: tuple) -> bool:
        image_width = block[2] - block[0]
        image_height = block[3] - block[1]
        size = image_width * image_height
        # This is a number to be tuned
        return size < 15000

    def low_possibility_to_be_image(self, block: tuple, min_size: int = 20, max_aspect_ratio: float = 10.0) -> bool:
        """
        Determine if a block has a low likelihood of being an image based on its dimensions.

        Parameters:
        - block: A 4-tuple (x0, top, x1, bottom) representing the coordinates of the block.
        - min_size: The minimum width/height required for a block to be considered a potential image.
        - max_aspect_ratio: The maximum allowed width-to-height (or height-to-width) ratio for a block to be considered an image.

        Returns:
        - True if the block is unlikely to be an image, False otherwise.
        """
        # Calculate the width and height of the block
        image_width = block[2] - block[0]  # x1 - x0
        image_height = block[3] - block[1]  # bottom - top

        # Check if the block is too small to be an image
        if image_width < min_size or image_height < min_size:
            return True

        # Calculate the aspect ratio
        aspect_ratio = image_width / image_height if image_height != 0 else float('inf')

        # Check if the aspect ratio is too extreme to be an image
        if aspect_ratio > max_aspect_ratio or aspect_ratio < 1 / max_aspect_ratio:
            return True

        # Otherwise, it's not considered "low possibility" of being an image
        return False

    def is_blank_pil_image(self, im: Image.Image) -> bool:
        """
        Check if an in-memory image (from pdfplumber) is blank using Pillow, without NumPy.
        """
        pil_image = im.original  # im.original is a PIL Image object

        # Get extrema (min, max) for each channel (for grayscale, there will be one tuple)
        extrema = pil_image.getextrema()

        # If the extrema (min, max) values are the same, the image is uniform (blank)
        if isinstance(extrema, tuple) and all(min_val == max_val for min_val, max_val in extrema):
            return True
        return False

    def is_overlap(self, block1: tuple, block2: tuple) -> bool:
        """
        Determines if two blocks (x0, top, x1, bottom) overlap.

        Parameters:
        - block1: A tuple representing the first block (x0, top, x1, bottom).
        - block2: A tuple representing the second block (x0, top, x1, bottom).

        Returns:
        - True if the blocks overlap, False otherwise.
        """
        x0_1, top_1, x1_1, bottom_1 = block1
        x0_2, top_2, x1_2, bottom_2 = block2

        # Check for horizontal overlap
        horizontal_overlap = (x0_1 < x1_2 and x1_1 > x0_2)

        # Check for vertical overlap
        vertical_overlap = (top_1 < bottom_2 and bottom_1 > top_2)

        # If both horizontal and vertical overlaps are true, the blocks overlap
        return horizontal_overlap and vertical_overlap
