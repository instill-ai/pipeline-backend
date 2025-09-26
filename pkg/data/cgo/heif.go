package cgo

/*
#cgo pkg-config: libheif
#include <libheif/heif.h>
#include <stdlib.h>
#include <string.h>

*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"unsafe"
)

// GetHEIFImageProperties extracts width and height from HEIC/HEIF files using libheif
func GetHEIFImageProperties(data []byte) (width, height int) {
	if len(data) == 0 {
		return 0, 0
	}

	// Initialize libheif
	ctx := C.heif_context_alloc()
	if ctx == nil {
		return 0, 0
	}
	defer C.heif_context_free(ctx)

	// Read from memory
	cData := C.CBytes(data)
	defer C.free(cData)

	err := C.heif_context_read_from_memory_without_copy(ctx, cData, C.size_t(len(data)), nil)
	if err.code != C.heif_error_Ok {
		return 0, 0
	}

	// Get primary image handle
	var handle *C.struct_heif_image_handle
	err = C.heif_context_get_primary_image_handle(ctx, &handle)
	if err.code != C.heif_error_Ok {
		return 0, 0
	}
	defer C.heif_image_handle_release(handle)

	// Get image dimensions
	width = int(C.heif_image_handle_get_width(handle))
	height = int(C.heif_image_handle_get_height(handle))

	return width, height
}

// DecodeHEIF decodes HEIC/HEIF data to RGB data using libheif
func DecodeHEIF(data []byte) ([]byte, int, int, error) {
	if len(data) == 0 {
		return nil, 0, 0, fmt.Errorf("empty data")
	}

	// Initialize libheif
	ctx := C.heif_context_alloc()
	if ctx == nil {
		return nil, 0, 0, fmt.Errorf("failed to allocate heif context")
	}
	defer C.heif_context_free(ctx)

	// Read from memory
	cData := C.CBytes(data)
	defer C.free(cData)

	err := C.heif_context_read_from_memory_without_copy(ctx, cData, C.size_t(len(data)), nil)
	if err.code != C.heif_error_Ok {
		return nil, 0, 0, fmt.Errorf("failed to read HEIF data: %s", C.GoString(err.message))
	}

	// Get primary image handle
	var handle *C.struct_heif_image_handle
	err = C.heif_context_get_primary_image_handle(ctx, &handle)
	if err.code != C.heif_error_Ok {
		return nil, 0, 0, fmt.Errorf("failed to get primary image handle: %s", C.GoString(err.message))
	}
	defer C.heif_image_handle_release(handle)

	// Decode the image
	var img *C.struct_heif_image
	err = C.heif_decode_image(handle, &img, C.heif_colorspace_RGB, C.heif_chroma_interleaved_RGB, nil)
	if err.code != C.heif_error_Ok {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %s", C.GoString(err.message))
	}
	defer C.heif_image_release(img)

	// Get image dimensions
	width := int(C.heif_image_get_width(img, C.heif_channel_interleaved))
	height := int(C.heif_image_get_height(img, C.heif_channel_interleaved))

	// Get image data
	var stride C.int
	plane := C.heif_image_get_plane_readonly(img, C.heif_channel_interleaved, &stride)
	if plane == nil {
		return nil, 0, 0, fmt.Errorf("failed to get image plane")
	}

	// Copy RGB data
	dataSize := width * height * 3 // 3 bytes per pixel (RGB)
	rgbData := C.GoBytes(unsafe.Pointer(plane), C.int(dataSize))

	return rgbData, width, height, nil
}

// EncodeHEIF encodes RGB data to HEIC format using libheif
func EncodeHEIF(rgbData []byte, width, height int, quality int) ([]byte, error) {
	if len(rgbData) == 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid input parameters")
	}

	// Initialize libheif
	ctx := C.heif_context_alloc()
	if ctx == nil {
		return nil, fmt.Errorf("failed to allocate heif context")
	}
	defer C.heif_context_free(ctx)

	// Create image
	var img *C.struct_heif_image
	err := C.heif_image_create(C.int(width), C.int(height), C.heif_colorspace_RGB, C.heif_chroma_interleaved_RGB, &img)
	if err.code != C.heif_error_Ok {
		return nil, fmt.Errorf("failed to create image: %s", C.GoString(err.message))
	}
	defer C.heif_image_release(img)

	// Add image plane
	err = C.heif_image_add_plane(img, C.heif_channel_interleaved, C.int(width), C.int(height), 24)
	if err.code != C.heif_error_Ok {
		return nil, fmt.Errorf("failed to add image plane: %s", C.GoString(err.message))
	}

	// Get plane for writing
	var stride C.int
	plane := C.heif_image_get_plane(img, C.heif_channel_interleaved, &stride)
	if plane == nil {
		return nil, fmt.Errorf("failed to get image plane for writing")
	}

	// Copy RGB data to image plane
	cRgbData := C.CBytes(rgbData)
	defer C.free(cRgbData)

	// Copy line by line to handle stride differences
	for y := 0; y < height; y++ {
		srcOffset := y * width * 3
		dstOffset := y * int(stride)
		C.memcpy(unsafe.Pointer(uintptr(unsafe.Pointer(plane))+uintptr(dstOffset)),
			unsafe.Pointer(uintptr(cRgbData)+uintptr(srcOffset)),
			C.size_t(width*3))
	}

	// Get encoder
	var encoder *C.struct_heif_encoder
	err = C.heif_context_get_encoder_for_format(ctx, C.heif_compression_HEVC, &encoder)
	if err.code != C.heif_error_Ok {
		return nil, fmt.Errorf("failed to get encoder: %s", C.GoString(err.message))
	}
	defer C.heif_encoder_release(encoder)

	// Set quality
	if quality >= 0 && quality <= 100 {
		C.heif_encoder_set_lossy_quality(encoder, C.int(quality))
	}

	// Encode image
	var handle *C.struct_heif_image_handle
	err = C.heif_context_encode_image(ctx, img, encoder, nil, &handle)
	if err.code != C.heif_error_Ok {
		return nil, fmt.Errorf("failed to encode image: %s", C.GoString(err.message))
	}
	defer C.heif_image_handle_release(handle)

	// Write to temporary file (simpler approach)
	tempFile := C.CString("/tmp/temp_heif_output.heic")
	defer C.free(unsafe.Pointer(tempFile))

	err = C.heif_context_write_to_file(ctx, tempFile)
	if err.code != C.heif_error_Ok {
		return nil, fmt.Errorf("failed to write HEIF data: %s", C.GoString(err.message))
	}

	// Read the file back
	file, fileErr := os.Open("/tmp/temp_heif_output.heic")
	if fileErr != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", fileErr)
	}
	defer file.Close()
	defer os.Remove("/tmp/temp_heif_output.heic")

	result, fileErr := io.ReadAll(file)
	if fileErr != nil {
		return nil, fmt.Errorf("failed to read temp file: %w", fileErr)
	}

	return result, nil
}
