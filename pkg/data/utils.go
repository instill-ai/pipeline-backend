package data

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func decodeDataURI(s string) (b []byte, contentType string, fileName string, err error) {
	slices := strings.Split(s, ",")
	if len(slices) == 1 {
		b, err = base64.StdEncoding.DecodeString(s)
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		mime := strings.Split(slices[0], ":")
		tags := ""
		contentType, tags, _ = strings.Cut(mime[1], ";")
		b, err = base64.StdEncoding.DecodeString(slices[1])
		for _, tag := range strings.Split(tags, ";") {

			key, value, _ := strings.Cut(tag, "=")
			if key == "filename" || key == "fileName" || key == "file-name" {
				fileName = value
			}
		}
	}

	return
}

func encodeDataURI(b []byte, contentType string) (s string, err error) {
	s = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(b))
	return
}

func StandardizePath(path string) (newPath string, err error) {
	splits := strings.FieldsFunc(path, func(s rune) bool {
		return s == '.' || s == '['
	})
	for _, split := range splits {
		if strings.HasSuffix(split, "]") {
			// Array Index
			newPath += fmt.Sprintf("[%s", split)
		} else {
			// Map Key
			newPath += fmt.Sprintf("[\"%s\"]", split)
		}
	}
	return newPath, err
}
