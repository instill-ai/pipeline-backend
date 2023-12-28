package constant

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32
const MaxPayloadSize = 1024 * 1024 * 32

// Constants for resource owner
const DefaultUserID string = "admin"
const HeaderUserUIDKey = "Instill-User-Uid"
const HeaderVisitorUIDKey = "Instill-Visitor-Uid"
const HeaderAuthTypeKey = "Instill-Auth-Type"
const HeaderInstillCodeKey = "Instill-Share-Code"
const StartConnectorId = "start"
const EndConnectorId = "end"
const ReturnTracesKey = "instill-return-traces"
