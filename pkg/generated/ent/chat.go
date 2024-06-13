/*
Copyright YEAR llmos.ai.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/chat"
	v1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

// Chat is the model entity for the Chat schema.
type Chat struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Title holds the value of the "title" field.
	Title string `json:"title,omitempty"`
	// UserId holds the value of the "userId" field.
	UserId uuid.UUID `json:"userId,omitempty"`
	// Models holds the value of the "models" field.
	Models []string `json:"models,omitempty"`
	// Tags holds the value of the "tags" field.
	Tags []string `json:"tags,omitempty"`
	// History holds the value of the "history" field.
	History v1.History `json:"history,omitempty"`
	// Messages holds the value of the "messages" field.
	Messages []v1.Message `json:"messages,omitempty"`
	// CreatedAt holds the value of the "createdAt" field.
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Chat) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case chat.FieldModels, chat.FieldTags, chat.FieldHistory, chat.FieldMessages:
			values[i] = new([]byte)
		case chat.FieldTitle:
			values[i] = new(sql.NullString)
		case chat.FieldCreatedAt:
			values[i] = new(sql.NullTime)
		case chat.FieldID, chat.FieldUserId:
			values[i] = new(uuid.UUID)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Chat fields.
func (c *Chat) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case chat.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				c.ID = *value
			}
		case chat.FieldTitle:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field title", values[i])
			} else if value.Valid {
				c.Title = value.String
			}
		case chat.FieldUserId:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field userId", values[i])
			} else if value != nil {
				c.UserId = *value
			}
		case chat.FieldModels:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field models", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &c.Models); err != nil {
					return fmt.Errorf("unmarshal field models: %w", err)
				}
			}
		case chat.FieldTags:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field tags", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &c.Tags); err != nil {
					return fmt.Errorf("unmarshal field tags: %w", err)
				}
			}
		case chat.FieldHistory:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field history", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &c.History); err != nil {
					return fmt.Errorf("unmarshal field history: %w", err)
				}
			}
		case chat.FieldMessages:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field messages", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &c.Messages); err != nil {
					return fmt.Errorf("unmarshal field messages: %w", err)
				}
			}
		case chat.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field createdAt", values[i])
			} else if value.Valid {
				c.CreatedAt = value.Time
			}
		default:
			c.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Chat.
// This includes values selected through modifiers, order, etc.
func (c *Chat) Value(name string) (ent.Value, error) {
	return c.selectValues.Get(name)
}

// Update returns a builder for updating this Chat.
// Note that you need to call Chat.Unwrap() before calling this method if this Chat
// was returned from a transaction, and the transaction was committed or rolled back.
func (c *Chat) Update() *ChatUpdateOne {
	return NewChatClient(c.config).UpdateOne(c)
}

// Unwrap unwraps the Chat entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (c *Chat) Unwrap() *Chat {
	_tx, ok := c.config.driver.(*txDriver)
	if !ok {
		panic("ent: Chat is not a transactional entity")
	}
	c.config.driver = _tx.drv
	return c
}

// String implements the fmt.Stringer.
func (c *Chat) String() string {
	var builder strings.Builder
	builder.WriteString("Chat(")
	builder.WriteString(fmt.Sprintf("id=%v, ", c.ID))
	builder.WriteString("title=")
	builder.WriteString(c.Title)
	builder.WriteString(", ")
	builder.WriteString("userId=")
	builder.WriteString(fmt.Sprintf("%v", c.UserId))
	builder.WriteString(", ")
	builder.WriteString("models=")
	builder.WriteString(fmt.Sprintf("%v", c.Models))
	builder.WriteString(", ")
	builder.WriteString("tags=")
	builder.WriteString(fmt.Sprintf("%v", c.Tags))
	builder.WriteString(", ")
	builder.WriteString("history=")
	builder.WriteString(fmt.Sprintf("%v", c.History))
	builder.WriteString(", ")
	builder.WriteString("messages=")
	builder.WriteString(fmt.Sprintf("%v", c.Messages))
	builder.WriteString(", ")
	builder.WriteString("createdAt=")
	builder.WriteString(c.CreatedAt.Format(time.ANSIC))
	builder.WriteByte(')')
	return builder.String()
}

// Chats is a parsable slice of Chat.
type Chats []*Chat
