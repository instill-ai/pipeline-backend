package data

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func decodeDataURL(s string) (b []byte, contentType string, fileName string, err error) {
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

func encodeDataURL(b []byte, contentType, fileName string) (s string, err error) {
	s = fmt.Sprintf("data:%s;filename=%s;base64,%s", contentType, fileName, base64.StdEncoding.EncodeToString(b))
	return
}

func standardizePath(path string) (newPath string, err error) {
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

func trimFirstKeyFromPath(path string) (key, remainingPath string, err error) {
	key, remainingPath, _ = strings.Cut(path, "]")
	if strings.HasPrefix(key, "[\"") && strings.HasSuffix(key, "\"") {
		return key[2 : len(key)-1], remainingPath, nil
	}
	return "", "", fmt.Errorf("can not parse key from path: %s", path)
}

func trimFirstIndexFromPath(path string) (index int, remainingPath string, err error) {
	key, remainingPath, _ := strings.Cut(path, "]")
	if strings.HasPrefix(key, "[") {
		index, err := strconv.Atoi(key[1:])
		if err == nil {
			return index, remainingPath, nil
		}

	}
	return 0, "", fmt.Errorf("can not parse index from path: %s", path)
}

func comparePath(path1, path2 string) bool {
	var err error
	path1, err = standardizePath(path1)
	if err != nil {
		return false
	}
	path2, err = standardizePath(path2)
	if err != nil {
		return false
	}
	return path1 == path2
}

func matchPathPrefix(path, prefix string) bool {
	var err error
	path, err = standardizePath(path)
	if err != nil {
		return false
	}
	prefix, err = standardizePath(prefix)
	if err != nil {
		return false
	}
	return strings.HasPrefix(path, prefix)
}
