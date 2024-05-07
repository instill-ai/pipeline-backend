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
const HeaderReturnTracesKey = "Instill-Return-Traces"

// GlobalConnectionSecretKey can be used to reference a global secret
// secret in a connection configuration (i.e, ${secrets.INSTILL_CONNECTION}).
//
// This key will be interpreted during execution by each connector and
// transformed into the value of the secret, in case a secret for that
// connection parameter was injected.
const GlobalConnectionSecretKey = "INSTILL_CONNECTION"
