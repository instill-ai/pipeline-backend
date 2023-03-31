package constant

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32

// Constants for text to image task.
const DefaultStep int = 10
const DefaultCfgScale float64 = 7.0
const DefaultSeed int = 1024
const DefaultSamples int = 1

// Constants for text generation task.
const DefaultOutputLen int = 100
const DefaultTopK int = 40

// Constants for resource owner
const DefaultOwnerID string = "local-user"
const HeaderOwnerUIDKey = "jwt-sub"
