package instillapp

// ReadChatHistoryInput is the input struct for the ReadChatHistory operation
type ReadChatHistoryInput struct {
	// Namespace is the namespace of the app
	Namespace       string `instill:"namespace"`
	// AppID is the ID of the app
	AppID           string `instill:"app-id"`
	// ConversationID is the ID of the conversation
	ConversationID  string `instill:"conversation-id"`
	// Role can be either 'user' or 'assistant'
	Role            string `instill:"role"`
	// MessageType is the type of the message, now only support `MESSAGE_TYPE_TEXT`
	MessageType     string `instill:"message-type"`
	// Duration is the duration between now and how long ago to retrieve the chat history from. i.e. 2h45m5s. Valid time units are \"ns\", \"us\" (or \"\u00b5s\"), \"ms\", \"s\", \"m\", \"h\".
	Duration        string `instill:"duration"`
	// MaxMessageCount is the maximum number of messages to return
	MaxMessageCount int    `instill:"max-message-count"`
}

// ReadChatHistoryOutput is the output struct for the ReadChatHistory operation
type ReadChatHistoryOutput struct {
	// Messages is the list of messages
	Messages []Message `instill:"messages"`
}

// Message is the struct for a message
type Message struct {
	// Content is the content of the message
	Content []Content `instill:"content"`
	// Role can be either 'user' or 'assistant'
	Role string `instill:"role"`
	// Name is the name of the user who sent the message
	Name string `instill:"name,omitempty"`
}

// Content is the struct for the content of a message.
// It can be either text, image_url, or image_base64.
// Only one of the fields should be set, and Type should be set to the type of the content.
type Content struct {
	// Type is the type of the content. It can be either 'text', 'image_url', or 'image_base64'
	Type        string `instill:"type"`
	// Text is the text content of the message
	Text        string `instill:"text,omitempty"`
	// ImageURL is the URL of the image
	ImageURL    string `instill:"image-url,omitempty"`
	// ImageBase64 is the base64 encoded image
	ImageBase64 string `instill:"image-base64,omitempty"`
}

// WriteChatMessageInput is the input struct for the WriteChatMessage operation
type WriteChatMessageInput struct {
	// Namespace is the namespace of the app
	Namespace      string       `instill:"namespace"`
	// AppID is the ID of the app
	AppID          string       `instill:"app-id"`
	// ConversationID is the ID of the conversation
	ConversationID string       `instill:"conversation-id"`
	// Message is the message to be written to the chat
	Message        WriteMessage `instill:"message"`
}

// WriteMessage is the struct for a message to be written to the chat
type WriteMessage struct {
	// Content is the content of the message
	Content string `instill:"content"`
	// Role can be either 'user' or 'assistant'
	Role    string `instill:"role,omitempty"`
}

// WriteChatMessageOutput is the output struct for the WriteChatMessage operation
type WriteChatMessageOutput struct {
	// MessageUID is the UID of the message
	MessageUID string `instill:"message-uid"`
	// CreateTime is the time the message was created
	CreateTime string `instill:"create-time"`
	// UpdateTime is the time the message was updated
	UpdateTime string `instill:"update-time"`
}
