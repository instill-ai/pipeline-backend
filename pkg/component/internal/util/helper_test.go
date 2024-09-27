package util

import (
	"testing"

	qt "github.com/frankban/quicktest"
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
