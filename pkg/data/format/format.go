package format

type ReferenceString interface {
	Resolvable()
	String() string
}
type OmittableField interface {
	Omittable()
}

type Null interface {
	Value
}

type Number interface {
	Value
	Integer() int
	Float64() float64
	String() string
}

type String interface {
	Value
	String() string
}

type Boolean interface {
	Value
	Boolean() bool
	String() string
}

type ByteArray interface {
	Value
	ByteArray() []byte
	String() string
}

type File interface {
	Value

	Binary() (ba ByteArray, err error)
	DataURI() (url String, err error)
	Base64() (url String, err error)
	FileSize() (size Number)
	ContentType() (t String)
	FileName() (t String)
	SourceURL() (t String)
	String() string
}

type Document interface {
	File

	String() string
	Text() (val String, err error)
	PDF() (val Document, err error)
}

type Image interface {
	File
	Width() Number
	Height() Number
	Convert(contentType string) (val Image, err error)
}

type Video interface {
	File

	Width() Number
	Height() Number
	Duration() Number
	FrameRate() Number
	Convert(contentType string) (val Video, err error)
}

type Audio interface {
	File

	Duration() Number
	SampleRate() Number
	Convert(contentType string) (val Audio, err error)
}
