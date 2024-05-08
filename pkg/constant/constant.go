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

// GlobalCredentialSecretKey can be used to reference a global secret
// in the configuration of a component (i.e, ${secrets.INSTILL_CREDENTIAL}).
//
// Components can be configured to hold global secrets for certain parameters.
// Given this configuration, a component will override the value of a parameter
// with its global secret when it finds this keyword.
const GlobalCredentialSecretKey = "INSTILL_CREDENTIAL"
