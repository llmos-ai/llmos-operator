package chat

import (
	"time"

	typev1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

type Chat struct {
	// ID of the database.
	ID string `json:"id,omitempty"`
	// Title holds the value of the "title" field.
	Title string `json:"title,omitempty"`
	// UserId holds the value of the "userId" field.
	UserId string `json:"userId,omitempty"`
	// Models holds the value of the "models" field.
	Models []string `json:"models,omitempty"`
	// Tags holds the value of the "tags" field.
	Tags []string `json:"tags,omitempty"`
	// History holds the value of the "history" field.
	History typev1.History `json:"history,omitempty"`
	// Messages holds the value of the "messages" field.
	Messages []typev1.Message `json:"messages,omitempty"`
	// CreatedAt holds the value of the "createdAt" field.
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

type NewChatRequest struct {
	History  typev1.History   `json:"history"`
	Messages []typev1.Message `json:"messages"`
	Models   []string         `json:"models,omitempty"`
	Title    string           `json:"title,omitempty"`
	Tags     []string         `json:"tags,omitempty"`
}

type UpdateChatRequest struct {
	Title    string           `json:"title"`
	History  typev1.History   `json:"history"`
	Messages []typev1.Message `json:"messages,omitempty"`
	UserId   string           `json:"userId"`
}
