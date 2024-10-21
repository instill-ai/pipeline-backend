package data

import "github.com/instill-ai/pipeline-backend/pkg/data/value"

type ReferenceString interface {
	Resolvable()
	String() string
}
type OmittableField interface {
	Omittable()
}

type Null interface {
	value.Value
}

type Number interface {
	value.Value
	Integer() int
	Float64() float64
	String() string
}

type String interface {
	value.Value
	String() string
}

type Boolean interface {
	value.Value
	Boolean() bool
	String() string
}

type ByteArray interface {
	value.Value
	ByteArray() []byte
	String() string
}

type File interface {
	value.Value
	String() string

	Binary(contentType string) (ba *byteArrayData, err error)
	DataURI(contentType string) (url *stringData, err error)
	FileSize() (size *numberData)
	ContentType() (t *stringData)
	FileName() (t *stringData)
	SourceURL() (t *stringData)
}

type Document interface {
	File
	Text() (val *stringData, err error)
	PDF() (val *documentData, err error)
}

type Image interface {
	File
	Width() *numberData
	Height() *numberData
}

type Video interface {
	File
}

type Audio interface {
	File
}
