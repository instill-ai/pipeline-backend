package base

import (
	"bufio"
	"encoding/base64"
	"io"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestUtil_GetFileExtension(t *testing.T) {
	c := qt.New(t)

	file, err := os.Open("./testdata/test_image.png")
	c.Assert(err, qt.IsNil)
	defer file.Close()
	wantFileExtension := "png"

	reader := bufio.NewReader(file)
	content, err := io.ReadAll(reader)
	c.Assert(err, qt.IsNil)

	fileBase64 := base64.StdEncoding.EncodeToString(content)
	fileBase64 = "data:image/png;base64," + fileBase64
	gotFileExtension := GetBase64FileExtension(fileBase64)
	c.Check(gotFileExtension, qt.Equals, wantFileExtension)
}
