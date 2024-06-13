package v1

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Chat holds the schema definition for the Chat entity.
type Chat struct {
	ent.Schema
}

// note:run go generate if the filed changes
type History struct {
	CurrentID string             `json:"currentId"`
	Messages  map[string]Message `json:"messages"`
}

type Message struct {
	ChildrenIds []string    `json:"childrenIds,omitempty"`
	Content     string      `json:"content,omitempty"`
	Context     string      `json:"context,omitempty"`
	ID          string      `json:"id,omitempty"`
	ParentId    string      `json:"parentId,omitempty"`
	Role        string      `json:"role,omitempty"`
	Timestamp   int64       `json:"timestamp,omitempty"`
	Done        bool        `json:"done,omitempty"`
	Info        interface{} `json:"info,omitempty"`
}

// Fields of the User.
func (Chat) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).Unique(),
		field.String("title").NotEmpty(),
		field.UUID("userId", uuid.UUID{}).StorageKey("user_id"),
		field.JSON("models", []string{}),
		field.JSON("tags", []string{}),
		field.JSON("history", History{}),
		field.JSON("messages", []Message{}),
		field.Time("createdAt").StorageKey("created_at").Default(time.Now()).Immutable(),
	}
}

func (Chat) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userId"),
	}
}
