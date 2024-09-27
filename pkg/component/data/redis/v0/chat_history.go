package redis

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	// DefaultLatestK is the default number of latest conversation turns to retrieve
	DefaultLatestK = 5
)

type Message struct {
	Role     string                  `json:"role"`
	Content  string                  `json:"content"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

type MultiModalMessage struct {
	Role     string                  `json:"role"`
	Content  []MultiModalContent     `json:"content"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

type MultiModalContent struct {
	Type     string  `json:"type"`
	Text     *string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image-url,omitempty"`
}

type MessageWithTime struct {
	Message
	Timestamp int64 `json:"timestamp"`
}

type MultiModalMessageWithTime struct {
	MultiModalMessage
	Timestamp int64 `json:"timestamp"`
}

type ChatMessageWriteInput struct {
	SessionID string `json:"session-id"`
	Message
}

type ChatMultiModalMessageWriteInput struct {
	SessionID string `json:"session-id"`
	MultiModalMessage
}

type ChatMessageWriteOutput struct {
	Status bool `json:"status"`
}

type ChatHistoryRetrieveInput struct {
	SessionID            string `json:"session-id"`
	LatestK              *int   `json:"latest-k,omitempty"`
	IncludeSystemMessage bool   `json:"include-system-message"`
}

// ChatHistoryReadOutput is a wrapper struct for the messages associated with a session ID
type ChatHistoryRetrieveOutput struct {
	Messages []*MultiModalMessage `json:"messages"`
	Status   bool                 `json:"status"`
}

// WriteSystemMessage writes system message for a given session ID
func WriteSystemMessage(client *goredis.Client, sessionID string, message MultiModalMessageWithTime) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Store in a hash with a unique SessionID
	return client.HSet(context.Background(), "chat_history:system_messages", sessionID, messageJSON).Err()
}

func WriteNonSystemMessage(client *goredis.Client, sessionID string, message MultiModalMessageWithTime) error {
	// Marshal the MessageWithTime struct to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Index by Timestamp: Add to the Sorted Set
	return client.ZAdd(context.Background(), "chat_history:"+sessionID+":timestamps", goredis.Z{
		Score:  float64(message.Timestamp),
		Member: string(messageJSON),
	}).Err()
}

// RetrieveSystemMessage gets system message based on a given session ID
func RetrieveSystemMessage(client *goredis.Client, sessionID string) (bool, *MultiModalMessageWithTime, error) {
	serializedMessage, err := client.HGet(context.Background(), "chat_history:system_messages", sessionID).Result()

	// Check if the messageID does not exist
	if err == goredis.Nil {
		// Handle the case where the message does not exist
		return false, nil, nil
	} else if err != nil {
		// Handle other types of errors
		return false, nil, err
	}

	var message MultiModalMessageWithTime
	if err := json.Unmarshal([]byte(serializedMessage), &message); err != nil {
		return false, nil, err
	}

	return true, &message, nil
}

func WriteMessage(client *goredis.Client, input ChatMessageWriteInput) ChatMessageWriteOutput {
	// Current time
	currTime := time.Now().Unix()

	// Create a MessageWithTime struct with the provided input and timestamp
	messageWithTime := MultiModalMessageWithTime{
		MultiModalMessage: MultiModalMessage{
			Role: input.Role,
			Content: []MultiModalContent{
				{
					Type: "text",
					Text: &input.Content,
				},
			},
			Metadata: input.Metadata,
		},
		Timestamp: currTime,
	}

	// Treat system message differently
	if input.Role == "system" {
		err := WriteSystemMessage(client, input.SessionID, messageWithTime)
		if err != nil {
			return ChatMessageWriteOutput{Status: false}
		} else {
			return ChatMessageWriteOutput{Status: true}
		}
	}

	err := WriteNonSystemMessage(client, input.SessionID, messageWithTime)
	if err != nil {
		return ChatMessageWriteOutput{Status: false}
	} else {
		return ChatMessageWriteOutput{Status: true}
	}
}

func WriteMultiModelMessage(client *goredis.Client, input ChatMultiModalMessageWriteInput) ChatMessageWriteOutput {
	// Current time
	currTime := time.Now().Unix()

	// Create a MessageWithTime struct with the provided input and timestamp
	messageWithTime := MultiModalMessageWithTime{
		MultiModalMessage: MultiModalMessage{
			Role:     input.Role,
			Content:  input.Content,
			Metadata: input.Metadata,
		},
		Timestamp: currTime,
	}

	// Treat system message differently
	if input.Role == "system" {
		err := WriteSystemMessage(client, input.SessionID, messageWithTime)
		if err != nil {
			return ChatMessageWriteOutput{Status: false}
		} else {
			return ChatMessageWriteOutput{Status: true}
		}
	}

	err := WriteNonSystemMessage(client, input.SessionID, messageWithTime)
	if err != nil {
		return ChatMessageWriteOutput{Status: false}
	} else {
		return ChatMessageWriteOutput{Status: true}
	}
}

// RetrieveSessionMessages retrieves the latest K conversation turns from the Redis list for the given session ID
func RetrieveSessionMessages(client *goredis.Client, input ChatHistoryRetrieveInput) ChatHistoryRetrieveOutput {
	if input.LatestK == nil || *input.LatestK <= 0 {
		input.LatestK = &DefaultLatestK
	}
	key := input.SessionID

	messagesWithTime := []MultiModalMessageWithTime{}
	messages := []*MultiModalMessage{}
	ctx := context.Background()

	// Retrieve the latest K conversation turns associated with the session ID by descending timestamp order
	messagesNum := *input.LatestK * 2
	timestampMessages, err := client.ZRevRange(ctx, "chat_history:"+key+":timestamps", 0, int64(messagesNum-1)).Result()
	if err != nil {
		return ChatHistoryRetrieveOutput{
			Messages: messages,
			Status:   false,
		}
	}

	// Iterate through the members and deserialize them into MessageWithTime
	for _, member := range timestampMessages {
		var messageWithTime MultiModalMessageWithTime
		if err := json.Unmarshal([]byte(member), &messageWithTime); err != nil {
			return ChatHistoryRetrieveOutput{
				Messages: messages,
				Status:   false,
			}
		}
		messagesWithTime = append(messagesWithTime, messageWithTime)
	}

	// Sort the messages by timestamp in ascending order (earliest first)
	sort.SliceStable(messagesWithTime, func(i, j int) bool {
		return messagesWithTime[i].Timestamp < messagesWithTime[j].Timestamp
	})

	// Add System message if exist
	if input.IncludeSystemMessage {
		exist, sysMessage, err := RetrieveSystemMessage(client, input.SessionID)
		if err != nil {
			return ChatHistoryRetrieveOutput{
				Messages: messages,
				Status:   false,
			}
		}
		if exist {
			messages = append(messages, &MultiModalMessage{
				Role:     sysMessage.Role,
				Content:  sysMessage.Content,
				Metadata: sysMessage.Metadata,
			})
		}
	}

	// Convert the MessageWithTime structs to Message structs
	for _, m := range messagesWithTime {
		messages = append(messages, &MultiModalMessage{
			Role:     m.Role,
			Content:  m.Content,
			Metadata: m.Metadata,
		})
	}
	return ChatHistoryRetrieveOutput{
		Messages: messages,
		Status:   true,
	}
}
