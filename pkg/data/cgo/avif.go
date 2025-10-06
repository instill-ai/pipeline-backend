package cgo

/*
#cgo pkg-config: libavif
#include <avif/avif.h>
#include <stdlib.h>
#include <string.h>

// Wrapper function to handle type conversion issues
static avifImage* createAvifImage(int width, int height, int depth, avifPixelFormat format) {
    return avifImageCreate((uint32_t)width, (uint32_t)height, (uint32_t)depth, format);
}

*/
import "C"
import (
	"fmt"
	"unsafe"
)

// GetAVIFImageProperties extracts width and height from AVIF files using libavif
func GetAVIFImageProperties(data []byte) (width, height int) {
	if len(data) == 0 {
		return 0, 0
	}

	// Create decoder
	decoder := C.avifDecoderCreate()
	if decoder == nil {
		return 0, 0
	}
	defer C.avifDecoderDestroy(decoder)

	// Parse the AVIF data
	cData := C.CBytes(data)
	defer C.free(cData)

	result := C.avifDecoderSetIOMemory(decoder, (*C.uint8_t)(cData), C.size_t(len(data)))
	if result != C.AVIF_RESULT_OK {
		return 0, 0
	}

	// Parse the image
	result = C.avifDecoderParse(decoder)
	if result != C.AVIF_RESULT_OK {
		return 0, 0
	}

	// Get image dimensions
	width = int(decoder.image.width)
	height = int(decoder.image.height)

	return width, height
}

// DecodeAVIF decodes AVIF data to RGB data using libavif
func DecodeAVIF(data []byte) ([]byte, int, int, error) {
	if len(data) == 0 {
		return nil, 0, 0, fmt.Errorf("empty data")
	}

	// Create decoder
	decoder := C.avifDecoderCreate()
	if decoder == nil {
		return nil, 0, 0, fmt.Errorf("failed to create avif decoder")
	}
	defer C.avifDecoderDestroy(decoder)

	// Parse the AVIF data
	cData := C.CBytes(data)
	defer C.free(cData)

	result := C.avifDecoderSetIOMemory(decoder, (*C.uint8_t)(cData), C.size_t(len(data)))
	if result != C.AVIF_RESULT_OK {
		return nil, 0, 0, fmt.Errorf("failed to set AVIF data: %d", result)
	}

	// Parse the image
	result = C.avifDecoderParse(decoder)
	if result != C.AVIF_RESULT_OK {
		return nil, 0, 0, fmt.Errorf("failed to parse AVIF: %d", result)
	}

	// Decode next image
	result = C.avifDecoderNextImage(decoder)
	if result != C.AVIF_RESULT_OK {
		return nil, 0, 0, fmt.Errorf("failed to decode AVIF image: %d", result)
	}

	// Get image dimensions
	width := int(decoder.image.width)
	height := int(decoder.image.height)

	// Create RGB image
	var rgb C.avifRGBImage
	C.avifRGBImageSetDefaults(&rgb, decoder.image)
	rgb.format = C.AVIF_RGB_FORMAT_RGB

	// Allocate RGB pixels
	C.avifRGBImageAllocatePixels(&rgb)
	defer C.avifRGBImageFreePixels(&rgb)

	// Convert to RGB
	result = C.avifImageYUVToRGB(decoder.image, &rgb)
	if result != C.AVIF_RESULT_OK {
		return nil, 0, 0, fmt.Errorf("failed to convert AVIF to RGB: %d", result)
	}

	// Copy RGB data
	dataSize := width * height * 3 // 3 bytes per pixel (RGB)
	rgbData := C.GoBytes(unsafe.Pointer(rgb.pixels), C.int(dataSize))

	return rgbData, width, height, nil
}

// EncodeAVIF encodes RGB data to AVIF format using libavif
func EncodeAVIF(rgbData []byte, width, height int, quality int) ([]byte, error) {
	if len(rgbData) == 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid input parameters")
	}

	// Create image using wrapper function
	image := C.createAvifImage(C.int(width), C.int(height), C.int(8), C.AVIF_PIXEL_FORMAT_YUV420)
	if image == nil {
		return nil, fmt.Errorf("failed to create AVIF image")
	}
	defer C.avifImageDestroy(image)

	// Create RGB image
	var rgb C.avifRGBImage
	C.avifRGBImageSetDefaults(&rgb, image)
	rgb.format = C.AVIF_RGB_FORMAT_RGB

	// Allocate RGB pixels
	C.avifRGBImageAllocatePixels(&rgb)
	defer C.avifRGBImageFreePixels(&rgb)

	// Copy RGB data
	cRgbData := C.CBytes(rgbData)
	defer C.free(cRgbData)
	C.memcpy(unsafe.Pointer(rgb.pixels), cRgbData, C.size_t(len(rgbData)))

	// Convert RGB to YUV
	result := C.avifImageRGBToYUV(image, &rgb)
	if result != C.AVIF_RESULT_OK {
		return nil, fmt.Errorf("failed to convert RGB to YUV: %d", result)
	}

	// Create encoder
	encoder := C.avifEncoderCreate()
	if encoder == nil {
		return nil, fmt.Errorf("failed to create AVIF encoder")
	}
	defer C.avifEncoderDestroy(encoder)

	// Set quality using encoder settings
	if quality >= 0 && quality <= 100 {
		qualityStr := C.CString(fmt.Sprintf("%d", 63-quality*63/100))
		defer C.free(unsafe.Pointer(qualityStr))
		cqLevelStr := C.CString("cq-level")
		defer C.free(unsafe.Pointer(cqLevelStr))
		C.avifEncoderSetCodecSpecificOption(encoder, cqLevelStr, qualityStr)
	}

	// Encode image
	var output C.avifRWData
	C.avifRWDataRealloc(&output, 0)
	defer C.avifRWDataFree(&output)

	result = C.avifEncoderWrite(encoder, image, &output)
	if result != C.AVIF_RESULT_OK {
		return nil, fmt.Errorf("failed to encode AVIF: %d", result)
	}

	// Copy output data
	avifData := C.GoBytes(unsafe.Pointer(output.data), C.int(output.size))

	return avifData, nil
}
