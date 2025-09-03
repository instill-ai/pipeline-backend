package data

import (
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestEncodeDecode(t *testing.T) {
	c := qt.New(t)

	// Fill data structure
	data := Array(make([]format.Value, 1))

	variable := Map{}
	variable["true"] = NewBoolean(true)
	variable["false"] = NewBoolean(true)
	variable["number"] = NewNumberFromFloat(12.34)
	variable["integer"] = NewNumberFromInteger(1234)
	variable["null"] = NewNull()
	variable["string"] = NewString("asdf")
	variable["nested"] = Map{"string": NewString("ghjk")}
	variable["bytes"] = NewByteArray([]byte(`1234`))

	filename := "sample_640_426.jpeg"
	img, err := os.ReadFile("testdata/" + filename)
	c.Assert(err, qt.IsNil)

	variable["file"], err = NewFileFromBytes([]byte(img), "", "")
	c.Assert(err, qt.IsNil)

	variable["img"], err = NewImageFromBytes([]byte(img), JPEG, filename, true)
	c.Assert(err, qt.IsNil)

	filename = "sample1.mp3"
	audio, err := os.ReadFile("testdata/" + filename)
	c.Assert(err, qt.IsNil)

	variable["audio"], err = NewAudioFromBytes([]byte(audio), MP3, filename, true)
	c.Assert(err, qt.IsNil)

	filename = "sample_640_360.mp4"
	vid, err := os.ReadFile("testdata/" + filename)
	c.Assert(err, qt.IsNil)

	variable["video"], err = NewVideoFromBytes([]byte(vid), MP4, filename, true)
	c.Assert(err, qt.IsNil)

	data[0] = variable
	want, err := data.ToJSONValue()
	c.Assert(err, qt.IsNil)

	// Encode, decode and compare
	b, err := Encode(data)
	c.Assert(err, qt.IsNil)

	v, err := Decode(b)
	c.Assert(err, qt.IsNil)

	got, err := v.ToJSONValue()
	c.Assert(err, qt.IsNil)

	c.Check(got, qt.ContentEquals, want)
}
