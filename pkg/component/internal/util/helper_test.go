package util

import (
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	timestampPB "google.golang.org/protobuf/types/known/timestamppb"
)

func TestDecodeBase46(t *testing.T) {
	c := qt.New(t)

	c.Run("ok - with MIME prepended", func(c *qt.C) {
		in := "data:text/plain;base64,aG9sYQ=="
		got, err := DecodeBase64(in)
		c.Check(err, qt.IsNil)
		c.Check(got, qt.ContentEquals, []byte("hola"))
	})

	c.Run("ok - with MIME prepended", func(c *qt.C) {
		in := "aG9sYQ=="
		got, err := DecodeBase64(in)
		c.Check(err, qt.IsNil)
		c.Check(got, qt.ContentEquals, []byte("hola"))
	})

	c.Run("nok - invalid", func(c *qt.C) {
		in := "hola=="
		_, err := DecodeBase64(in)
		c.Check(err, qt.IsNotNil)
	})
}

func TestFormatToISO8601(t *testing.T) {
	c := qt.New(t)

	c.Run("FormatToISO8601", func(c *qt.C) {
		in := timestampPB.New(time.Date(2020, 8, 1, 12, 0, 0, 0, time.UTC))
		got := FormatToISO8601(in)
		c.Check(got, qt.Equals, "2020-08-01T12:00:00Z")
	})
}
