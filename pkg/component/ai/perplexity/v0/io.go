package perplexity

// We expose them because we will use them to calculate the Instill Credit
// after the IO struct is finished.

// TextChatInput is the input for the TASK_CHAT task.
type TextChatInput struct {
	// Data contains the input data.
	Data InputData `instill:"data"`
	// Parameter contains the input parameter.
	Parameter Parameter `instill:"parameter,omitempty"`
}

// InputData contains the input data.
type InputData struct {
	// Model is the model to use.
	Model string `instill:"model"`
	// Messages contains the input messages.
	Messages []InputMessage `instill:"messages"`
}

// InputMessage is the structure of a message.
type InputMessage struct {
	// Content is the content of the message.
	Content []Content `instill:"content"`
	// Role is the role of the message.
	Role string `instill:"role"`
	// Name is the name of the message.
	Name string `instill:"name"`
}

// Content is the content of a message.
// We remain the structure of standardised AI design.
// So, in Content, even though we only support Text, we still keep the Type field.
type Content struct {
	// Text is the text of the message.
	Text string `instill:"text"`
	// Type is the type of the message.
	Type string `instill:"type"`
}

type WebSearchOptions struct {
	// SearchContextSize is the search context size.
	SearchContextSize *string `instill:"search-context-size,default=low" json:"search_context_size,omitempty"`
	// UserLocation is the user location.
	UserLocation *UserLocation `instill:"user-location" json:"user_location,omitempty"`
}

type UserLocation struct {
	// Latitude is the latitude of the user's location.
	Latitude *float64 `instill:"latitude" json:"latitude,omitempty"`
	// Longitude is the longitude of the user's location.
	Longitude *float64 `instill:"longitude" json:"longitude,omitempty"`
	// Country is the country of the user's location.
	Country *string `instill:"country" json:"country,omitempty"`
}

// Parameter contains the input parameter.
type Parameter struct {
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens *int `instill:"max-tokens,default=50"`
	// Temperature is the temperature of the model.
	Temperature *float64 `instill:"temperature,default=0.2"`
	// TopP is the top-p value of the model.
	TopP *float64 `instill:"top-p,default=0.9"`
	// Stream is whether to stream the output.
	Stream *bool `instill:"stream,default=false"`
	// SearchMode is the search mode to be used.
	SearchMode *string `instill:"search-mode,default=web"`
	// SearchDomainFilter gives the list of domains,
	// limit the citations used by the online model to URLs from the specified
	// domains. Currently limited to only 3 domains for whitelisting and
	// blacklisting. For blacklisting add a `-` to the beginning of the domain
	// string.
	SearchDomainFilter []string `instill:"search-domain-filter"`
	// SearchRecencyFilter returns search results within the specified time interval
	// - does not apply to images. Values include `month`, `week`, `day`, `year`."
	// ReturnRelatedQuestions determines whether related questions should be returned.
	ReturnRelatedQuestions *bool   `instill:"return-related-questions,default=false"`
	SearchRecencyFilter    *string `instill:"search-recency-filter"`
	// SearchAfterDateFilter filters search results to only include content published after this date.
	SearchAfterDateFilter *string `instill:"search-after-date-filter"`
	// SearchBeforeDateFilter filters search results to only include content published before this date.
	SearchBeforeDateFilter *string `instill:"search-before-date-filter"`
	// LastUpdatedAfterFilter filters search results to only include content last updated after this date.
	LastUpdatedAfterFilter *string `instill:"last-updated-after-filter"`
	// LastUpdatedBeforeFilter filters search results to only include content last updated before this date.
	LastUpdatedBeforeFilter *string `instill:"last-updated-before-filter"`
	// WebSearchOptions is the web search options.
	WebSearchOptions *WebSearchOptions `instill:"web-search-options"`
	// TopK is the top-k value of the model.
	TopK *int `instill:"top-k,default=0"`
	// PresencePenalty is a value between -2.0 and 2.0. Positive values penalize new
	// tokens based on whether they appear in the text so far, increasing the
	// model's likelihood to talk about new topics. Incompatible with
	// `frequency_penalty`.
	PresencePenalty *float64 `instill:"presence-penalty,default=0"`
	// FrequencyPenalty is a multiplicative penalty greater than 0. Values greater
	// than 1.0 penalize new tokens based on their existing frequency in the text so
	// far, decreasing the model's likelihood to repeat the same line verbatim. A
	// value of 1.0 means no penalty. Incompatible with `presence_penalty`.
	FrequencyPenalty *float64 `instill:"frequency-penalty,default=1"`

	// EnableSearchClassifier is whether to enable search classifier.
	EnableSearchClassifier *bool `instill:"enable-search-classifier,default=false"`
}

// TextChatOutput is the output for the TASK_CHAT task.
type TextChatOutput struct {
	// Data contains the output data.
	Data OutputData `instill:"data"`
	// Metadata contains the output metadata.
	Metadata Metadata `instill:"metadata"`
}

// OutputData contains the output data.
type OutputData struct {
	// Choice is list of chat completion choices
	Choices []Choice `instill:"choices"`
	// Citations is the citation of the output.
	Citations []string `instill:"citations"`
	// SearchResults is the search results of the output.
	SearchResults []SearchResult `instill:"search-results"`
}

// SearchResult is the structure of a search result.
type SearchResult struct {
	// Title is the title of the search result.
	Title string `instill:"title" json:"title"`
	// URL is the URL of the search result.
	URL string `instill:"url" json:"url"`
	// Date is the date of the search result.
	Date string `instill:"date" json:"date"`
}

// Choice is the structure of a chat completion choice.
type Choice struct {
	// FinishReason is the reason the chat was finished.
	FinishReason string `instill:"finish-reason"`
	// Index is the index of the choice.
	Index int `instill:"index"`
	// Message is the message of the choice.
	Message OutputMessage `instill:"message"`
	// Created is the timestamp of when the chat completion was created.
	// Format is in ISO 8601. Example: 2024-07-01T11:47:40.388Z.
	Created string `instill:"created"`
}

// OutputMessage is the structure of a chat completion message.
type OutputMessage struct {
	// Content is the content of the message.
	Content string `instill:"content"`
	// Role is the role of the message.
	Role string `instill:"role"`
}

// Metadata contains the output metadata.
type Metadata struct {
	// Usage contains the token usages.
	Usage Usage `instill:"usage"`
}

// Usage contains the token usages.
type Usage struct {
	// CompletionTokens is the number of completion tokens.
	CompletionTokens int `instill:"completion-tokens"`
	// PromptTokens is the number of prompt tokens.
	PromptTokens int `instill:"prompt-tokens"`
	// TotalTokens is the total number of tokens.
	TotalTokens int `instill:"total-tokens"`
}
