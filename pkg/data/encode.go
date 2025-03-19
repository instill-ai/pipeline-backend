package data

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// init registers the concrete format.Value types for the gob encoder and
// decoder. On each end, this will tell the engine which concrete type is being
// sent that implements the interface.
func init() {
	gob.Register(Array{})
	gob.Register(Map{})

	gob.Register(&audioData{})
	gob.Register(&booleanData{})
	gob.Register(&byteArrayData{})
	gob.Register(&documentData{})
	gob.Register(&fileData{})
	gob.Register(&imageData{})
	gob.Register(&nullData{})
	gob.Register(&numberData{})
	gob.Register(&stringData{})
	gob.Register(&videoData{})
}

// Encode transforms any implementation of format.Value into its byte
// representation, which can be later recovered respecting the concrete type
// implementation.
func Encode(v format.Value) ([]byte, error) {
	var bb bytes.Buffer
	enc := gob.NewEncoder(&bb)

	// Encoding will fail unless the concrete type has been registered. All the
	// format.Value implementations in this package should be registered in the
	// init() function.

	// A pointer to interface is passed so Encode sees (and hence sends) a
	// value of interface type. If we passed v directly it would see the
	// concrete type instead.
	if err := enc.Encode(&v); err != nil {
		return nil, fmt.Errorf("encoding gob: %w", err)
	}

	return bb.Bytes(), nil
}

// Decode transforms an encoded implementation of the format.Value interface
// into an object that implements format.Value (and respects the original
// type).
func Decode(b []byte) (format.Value, error) {
	var v format.Value
	dec := gob.NewDecoder(bytes.NewBuffer(b))

	// The decode function will fail unless the concrete type on the wire has
	// been registered, which was done in the init() function.
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("decoding gob: %w", err)
	}

	return v, nil
}
