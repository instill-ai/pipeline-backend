package constant

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32
const MaxPayloadSize = 1024 * 1024 * 256

// Constants for resource owner
const DefaultUserID string = "admin"

const (
	HeaderInstillCodeKey  = "Instill-Share-Code"
	HeaderReturnTracesKey = "Instill-Return-Traces"

	// HeaderServiceKey is the context key for service-to-service communication.
	// When present, it indicates the request originates from an internal
	// service rather than an end user. Valid values are restricted to "instill"
	// and can only be set by internal services. Requests with this header
	// bypass standard authorization checks since they represent trusted
	// internal traffic.
	HeaderServiceKey = "Instill-Service"

	HeaderAccept           = "Accept"
	HeaderValueEventStream = "text/event-stream"

	SegMemory     = "memory"
	SegVariable   = "variable"
	SegSecret     = "secret"
	SegConnection = "connection"
	SegComponent  = "component"
	SegIteration  = "iterator"
	SegInput      = "input"
	SegOutput     = "output"
)

// GlobalSecretKey can be used to reference a global secret in the
// configuration of a component (i.e, ${secrets.INSTILL_SECRET}).
//
// Components can be configured to hold global secrets for certain parameters.
// Given this configuration, a component will override the value of a parameter
// with its global secret when it finds this keyword.
const GlobalSecretKey = "INSTILL_SECRET"

// The ID within the preset namespace must consistently match between
// mgmt and pipeline-backend.
const (
	PresetNamespaceID = "preset"
	// PresetNamespaceUID is the UUID for the preset namespace organization.
	// This must match the UID in the mgmt-backend database.
	PresetNamespaceUID = "63616e63-6f6d-7065-7465-796f75726461"
)
