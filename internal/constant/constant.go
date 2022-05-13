package constant

// ConnectionTypeDirectness is a slice records connector names having the connection-type directness
var ConnectionTypeDirectness = []string{
	"source-connectors/http", "source-connectors/gRPC",
	"destination-connectors/http", "destination-connectors/gRPC",
}

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32
const MaxImageSizeBytes int = 4 * MB
