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
	// HeaderUserUIDKey is the context key for the authenticated user.
	HeaderUserUIDKey = "Instill-User-Uid"
	// HeaderRequesterUIDKey is the context key for the requester. An
	// authenticated user can use different namespaces (e.g. an organization
	// they belong to) to make requests, as long as they have permissions.
	HeaderRequesterUIDKey = "Instill-Requester-Uid"
	// HeaderVisitorUIDKey is the context key for the visitor UID when requests
	// are made without authentication.
	HeaderVisitorUIDKey = "Instill-Visitor-Uid"
	// HeaderAuthTypeKey is the context key the authentication type (user or
	// visitor).
	HeaderAuthTypeKey = "Instill-Auth-Type"

	HeaderInstillCodeKey  = "Instill-Share-Code"
	HeaderReturnTracesKey = "Instill-Return-Traces"

	HeaderUserAgentKey = "Instill-User-Agent"

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

const ContentTypeJSON = "application/json"

// GlobalSecretKey can be used to reference a global secret in the
// configuration of a component (i.e, ${secrets.INSTILL_SECRET}).
//
// Components can be configured to hold global secrets for certain parameters.
// Given this configuration, a component will override the value of a parameter
// with its global secret when it finds this keyword.
const GlobalSecretKey = "INSTILL_SECRET"
