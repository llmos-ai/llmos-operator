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
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/chat"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/predicate"
	v1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

const (
	// Operation types.
	OpCreate    = ent.OpCreate
	OpDelete    = ent.OpDelete
	OpDeleteOne = ent.OpDeleteOne
	OpUpdate    = ent.OpUpdate
	OpUpdateOne = ent.OpUpdateOne

	// Node types.
	TypeChat = "Chat"
)

// ChatMutation represents an operation that mutates the Chat nodes in the graph.
type ChatMutation struct {
	config
	op             Op
	typ            string
	id             *uuid.UUID
	title          *string
	userId         *uuid.UUID
	models         *[]string
	appendmodels   []string
	tags           *[]string
	appendtags     []string
	history        *v1.History
	messages       *[]v1.Message
	appendmessages []v1.Message
	createdAt      *time.Time
	clearedFields  map[string]struct{}
	done           bool
	oldValue       func(context.Context) (*Chat, error)
	predicates     []predicate.Chat
}

var _ ent.Mutation = (*ChatMutation)(nil)

// chatOption allows management of the mutation configuration using functional options.
type chatOption func(*ChatMutation)

// newChatMutation creates new mutation for the Chat entity.
func newChatMutation(c config, op Op, opts ...chatOption) *ChatMutation {
	m := &ChatMutation{
		config:        c,
		op:            op,
		typ:           TypeChat,
		clearedFields: make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// withChatID sets the ID field of the mutation.
func withChatID(id uuid.UUID) chatOption {
	return func(m *ChatMutation) {
		var (
			err   error
			once  sync.Once
			value *Chat
		)
		m.oldValue = func(ctx context.Context) (*Chat, error) {
			once.Do(func() {
				if m.done {
					err = errors.New("querying old values post mutation is not allowed")
				} else {
					value, err = m.Client().Chat.Get(ctx, id)
				}
			})
			return value, err
		}
		m.id = &id
	}
}

// withChat sets the old Chat of the mutation.
func withChat(node *Chat) chatOption {
	return func(m *ChatMutation) {
		m.oldValue = func(context.Context) (*Chat, error) {
			return node, nil
		}
		m.id = &node.ID
	}
}

// Client returns a new `ent.Client` from the mutation. If the mutation was
// executed in a transaction (ent.Tx), a transactional client is returned.
func (m ChatMutation) Client() *Client {
	client := &Client{config: m.config}
	client.init()
	return client
}

// Tx returns an `ent.Tx` for mutations that were executed in transactions;
// it returns an error otherwise.
func (m ChatMutation) Tx() (*Tx, error) {
	if _, ok := m.driver.(*txDriver); !ok {
		return nil, errors.New("ent: mutation is not running in a transaction")
	}
	tx := &Tx{config: m.config}
	tx.init()
	return tx, nil
}

// SetID sets the value of the id field. Note that this
// operation is only accepted on creation of Chat entities.
func (m *ChatMutation) SetID(id uuid.UUID) {
	m.id = &id
}

// ID returns the ID value in the mutation. Note that the ID is only available
// if it was provided to the builder or after it was returned from the database.
func (m *ChatMutation) ID() (id uuid.UUID, exists bool) {
	if m.id == nil {
		return
	}
	return *m.id, true
}

// IDs queries the database and returns the entity ids that match the mutation's predicate.
// That means, if the mutation is applied within a transaction with an isolation level such
// as sql.LevelSerializable, the returned ids match the ids of the rows that will be updated
// or updated by the mutation.
func (m *ChatMutation) IDs(ctx context.Context) ([]uuid.UUID, error) {
	switch {
	case m.op.Is(OpUpdateOne | OpDeleteOne):
		id, exists := m.ID()
		if exists {
			return []uuid.UUID{id}, nil
		}
		fallthrough
	case m.op.Is(OpUpdate | OpDelete):
		return m.Client().Chat.Query().Where(m.predicates...).IDs(ctx)
	default:
		return nil, fmt.Errorf("IDs is not allowed on %s operations", m.op)
	}
}

// SetTitle sets the "title" field.
func (m *ChatMutation) SetTitle(s string) {
	m.title = &s
}

// Title returns the value of the "title" field in the mutation.
func (m *ChatMutation) Title() (r string, exists bool) {
	v := m.title
	if v == nil {
		return
	}
	return *v, true
}

// OldTitle returns the old "title" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldTitle(ctx context.Context) (v string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTitle is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTitle requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTitle: %w", err)
	}
	return oldValue.Title, nil
}

// ResetTitle resets all changes to the "title" field.
func (m *ChatMutation) ResetTitle() {
	m.title = nil
}

// SetUserId sets the "userId" field.
func (m *ChatMutation) SetUserId(u uuid.UUID) {
	m.userId = &u
}

// UserId returns the value of the "userId" field in the mutation.
func (m *ChatMutation) UserId() (r uuid.UUID, exists bool) {
	v := m.userId
	if v == nil {
		return
	}
	return *v, true
}

// OldUserId returns the old "userId" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldUserId(ctx context.Context) (v uuid.UUID, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldUserId is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldUserId requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldUserId: %w", err)
	}
	return oldValue.UserId, nil
}

// ResetUserId resets all changes to the "userId" field.
func (m *ChatMutation) ResetUserId() {
	m.userId = nil
}

// SetModels sets the "models" field.
func (m *ChatMutation) SetModels(s []string) {
	m.models = &s
	m.appendmodels = nil
}

// Models returns the value of the "models" field in the mutation.
func (m *ChatMutation) Models() (r []string, exists bool) {
	v := m.models
	if v == nil {
		return
	}
	return *v, true
}

// OldModels returns the old "models" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldModels(ctx context.Context) (v []string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldModels is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldModels requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldModels: %w", err)
	}
	return oldValue.Models, nil
}

// AppendModels adds s to the "models" field.
func (m *ChatMutation) AppendModels(s []string) {
	m.appendmodels = append(m.appendmodels, s...)
}

// AppendedModels returns the list of values that were appended to the "models" field in this mutation.
func (m *ChatMutation) AppendedModels() ([]string, bool) {
	if len(m.appendmodels) == 0 {
		return nil, false
	}
	return m.appendmodels, true
}

// ResetModels resets all changes to the "models" field.
func (m *ChatMutation) ResetModels() {
	m.models = nil
	m.appendmodels = nil
}

// SetTags sets the "tags" field.
func (m *ChatMutation) SetTags(s []string) {
	m.tags = &s
	m.appendtags = nil
}

// Tags returns the value of the "tags" field in the mutation.
func (m *ChatMutation) Tags() (r []string, exists bool) {
	v := m.tags
	if v == nil {
		return
	}
	return *v, true
}

// OldTags returns the old "tags" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldTags(ctx context.Context) (v []string, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldTags is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldTags requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldTags: %w", err)
	}
	return oldValue.Tags, nil
}

// AppendTags adds s to the "tags" field.
func (m *ChatMutation) AppendTags(s []string) {
	m.appendtags = append(m.appendtags, s...)
}

// AppendedTags returns the list of values that were appended to the "tags" field in this mutation.
func (m *ChatMutation) AppendedTags() ([]string, bool) {
	if len(m.appendtags) == 0 {
		return nil, false
	}
	return m.appendtags, true
}

// ResetTags resets all changes to the "tags" field.
func (m *ChatMutation) ResetTags() {
	m.tags = nil
	m.appendtags = nil
}

// SetHistory sets the "history" field.
func (m *ChatMutation) SetHistory(v v1.History) {
	m.history = &v
}

// History returns the value of the "history" field in the mutation.
func (m *ChatMutation) History() (r v1.History, exists bool) {
	v := m.history
	if v == nil {
		return
	}
	return *v, true
}

// OldHistory returns the old "history" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldHistory(ctx context.Context) (v v1.History, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldHistory is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldHistory requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldHistory: %w", err)
	}
	return oldValue.History, nil
}

// ResetHistory resets all changes to the "history" field.
func (m *ChatMutation) ResetHistory() {
	m.history = nil
}

// SetMessages sets the "messages" field.
func (m *ChatMutation) SetMessages(v []v1.Message) {
	m.messages = &v
	m.appendmessages = nil
}

// Messages returns the value of the "messages" field in the mutation.
func (m *ChatMutation) Messages() (r []v1.Message, exists bool) {
	v := m.messages
	if v == nil {
		return
	}
	return *v, true
}

// OldMessages returns the old "messages" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldMessages(ctx context.Context) (v []v1.Message, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldMessages is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldMessages requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldMessages: %w", err)
	}
	return oldValue.Messages, nil
}

// AppendMessages adds v to the "messages" field.
func (m *ChatMutation) AppendMessages(v []v1.Message) {
	m.appendmessages = append(m.appendmessages, v...)
}

// AppendedMessages returns the list of values that were appended to the "messages" field in this mutation.
func (m *ChatMutation) AppendedMessages() ([]v1.Message, bool) {
	if len(m.appendmessages) == 0 {
		return nil, false
	}
	return m.appendmessages, true
}

// ResetMessages resets all changes to the "messages" field.
func (m *ChatMutation) ResetMessages() {
	m.messages = nil
	m.appendmessages = nil
}

// SetCreatedAt sets the "createdAt" field.
func (m *ChatMutation) SetCreatedAt(t time.Time) {
	m.createdAt = &t
}

// CreatedAt returns the value of the "createdAt" field in the mutation.
func (m *ChatMutation) CreatedAt() (r time.Time, exists bool) {
	v := m.createdAt
	if v == nil {
		return
	}
	return *v, true
}

// OldCreatedAt returns the old "createdAt" field's value of the Chat entity.
// If the Chat object wasn't provided to the builder, the object is fetched from the database.
// An error is returned if the mutation operation is not UpdateOne, or the database query fails.
func (m *ChatMutation) OldCreatedAt(ctx context.Context) (v time.Time, err error) {
	if !m.op.Is(OpUpdateOne) {
		return v, errors.New("OldCreatedAt is only allowed on UpdateOne operations")
	}
	if m.id == nil || m.oldValue == nil {
		return v, errors.New("OldCreatedAt requires an ID field in the mutation")
	}
	oldValue, err := m.oldValue(ctx)
	if err != nil {
		return v, fmt.Errorf("querying old value for OldCreatedAt: %w", err)
	}
	return oldValue.CreatedAt, nil
}

// ResetCreatedAt resets all changes to the "createdAt" field.
func (m *ChatMutation) ResetCreatedAt() {
	m.createdAt = nil
}

// Where appends a list predicates to the ChatMutation builder.
func (m *ChatMutation) Where(ps ...predicate.Chat) {
	m.predicates = append(m.predicates, ps...)
}

// WhereP appends storage-level predicates to the ChatMutation builder. Using this method,
// users can use type-assertion to append predicates that do not depend on any generated package.
func (m *ChatMutation) WhereP(ps ...func(*sql.Selector)) {
	p := make([]predicate.Chat, len(ps))
	for i := range ps {
		p[i] = ps[i]
	}
	m.Where(p...)
}

// Op returns the operation name.
func (m *ChatMutation) Op() Op {
	return m.op
}

// SetOp allows setting the mutation operation.
func (m *ChatMutation) SetOp(op Op) {
	m.op = op
}

// Type returns the node type of this mutation (Chat).
func (m *ChatMutation) Type() string {
	return m.typ
}

// Fields returns all fields that were changed during this mutation. Note that in
// order to get all numeric fields that were incremented/decremented, call
// AddedFields().
func (m *ChatMutation) Fields() []string {
	fields := make([]string, 0, 7)
	if m.title != nil {
		fields = append(fields, chat.FieldTitle)
	}
	if m.userId != nil {
		fields = append(fields, chat.FieldUserId)
	}
	if m.models != nil {
		fields = append(fields, chat.FieldModels)
	}
	if m.tags != nil {
		fields = append(fields, chat.FieldTags)
	}
	if m.history != nil {
		fields = append(fields, chat.FieldHistory)
	}
	if m.messages != nil {
		fields = append(fields, chat.FieldMessages)
	}
	if m.createdAt != nil {
		fields = append(fields, chat.FieldCreatedAt)
	}
	return fields
}

// Field returns the value of a field with the given name. The second boolean
// return value indicates that this field was not set, or was not defined in the
// schema.
func (m *ChatMutation) Field(name string) (ent.Value, bool) {
	switch name {
	case chat.FieldTitle:
		return m.Title()
	case chat.FieldUserId:
		return m.UserId()
	case chat.FieldModels:
		return m.Models()
	case chat.FieldTags:
		return m.Tags()
	case chat.FieldHistory:
		return m.History()
	case chat.FieldMessages:
		return m.Messages()
	case chat.FieldCreatedAt:
		return m.CreatedAt()
	}
	return nil, false
}

// OldField returns the old value of the field from the database. An error is
// returned if the mutation operation is not UpdateOne, or the query to the
// database failed.
func (m *ChatMutation) OldField(ctx context.Context, name string) (ent.Value, error) {
	switch name {
	case chat.FieldTitle:
		return m.OldTitle(ctx)
	case chat.FieldUserId:
		return m.OldUserId(ctx)
	case chat.FieldModels:
		return m.OldModels(ctx)
	case chat.FieldTags:
		return m.OldTags(ctx)
	case chat.FieldHistory:
		return m.OldHistory(ctx)
	case chat.FieldMessages:
		return m.OldMessages(ctx)
	case chat.FieldCreatedAt:
		return m.OldCreatedAt(ctx)
	}
	return nil, fmt.Errorf("unknown Chat field %s", name)
}

// SetField sets the value of a field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *ChatMutation) SetField(name string, value ent.Value) error {
	switch name {
	case chat.FieldTitle:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTitle(v)
		return nil
	case chat.FieldUserId:
		v, ok := value.(uuid.UUID)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetUserId(v)
		return nil
	case chat.FieldModels:
		v, ok := value.([]string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetModels(v)
		return nil
	case chat.FieldTags:
		v, ok := value.([]string)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetTags(v)
		return nil
	case chat.FieldHistory:
		v, ok := value.(v1.History)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetHistory(v)
		return nil
	case chat.FieldMessages:
		v, ok := value.([]v1.Message)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetMessages(v)
		return nil
	case chat.FieldCreatedAt:
		v, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf("unexpected type %T for field %s", value, name)
		}
		m.SetCreatedAt(v)
		return nil
	}
	return fmt.Errorf("unknown Chat field %s", name)
}

// AddedFields returns all numeric fields that were incremented/decremented during
// this mutation.
func (m *ChatMutation) AddedFields() []string {
	return nil
}

// AddedField returns the numeric value that was incremented/decremented on a field
// with the given name. The second boolean return value indicates that this field
// was not set, or was not defined in the schema.
func (m *ChatMutation) AddedField(name string) (ent.Value, bool) {
	return nil, false
}

// AddField adds the value to the field with the given name. It returns an error if
// the field is not defined in the schema, or if the type mismatched the field
// type.
func (m *ChatMutation) AddField(name string, value ent.Value) error {
	switch name {
	}
	return fmt.Errorf("unknown Chat numeric field %s", name)
}

// ClearedFields returns all nullable fields that were cleared during this
// mutation.
func (m *ChatMutation) ClearedFields() []string {
	return nil
}

// FieldCleared returns a boolean indicating if a field with the given name was
// cleared in this mutation.
func (m *ChatMutation) FieldCleared(name string) bool {
	_, ok := m.clearedFields[name]
	return ok
}

// ClearField clears the value of the field with the given name. It returns an
// error if the field is not defined in the schema.
func (m *ChatMutation) ClearField(name string) error {
	return fmt.Errorf("unknown Chat nullable field %s", name)
}

// ResetField resets all changes in the mutation for the field with the given name.
// It returns an error if the field is not defined in the schema.
func (m *ChatMutation) ResetField(name string) error {
	switch name {
	case chat.FieldTitle:
		m.ResetTitle()
		return nil
	case chat.FieldUserId:
		m.ResetUserId()
		return nil
	case chat.FieldModels:
		m.ResetModels()
		return nil
	case chat.FieldTags:
		m.ResetTags()
		return nil
	case chat.FieldHistory:
		m.ResetHistory()
		return nil
	case chat.FieldMessages:
		m.ResetMessages()
		return nil
	case chat.FieldCreatedAt:
		m.ResetCreatedAt()
		return nil
	}
	return fmt.Errorf("unknown Chat field %s", name)
}

// AddedEdges returns all edge names that were set/added in this mutation.
func (m *ChatMutation) AddedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// AddedIDs returns all IDs (to other nodes) that were added for the given edge
// name in this mutation.
func (m *ChatMutation) AddedIDs(name string) []ent.Value {
	return nil
}

// RemovedEdges returns all edge names that were removed in this mutation.
func (m *ChatMutation) RemovedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// RemovedIDs returns all IDs (to other nodes) that were removed for the edge with
// the given name in this mutation.
func (m *ChatMutation) RemovedIDs(name string) []ent.Value {
	return nil
}

// ClearedEdges returns all edge names that were cleared in this mutation.
func (m *ChatMutation) ClearedEdges() []string {
	edges := make([]string, 0, 0)
	return edges
}

// EdgeCleared returns a boolean which indicates if the edge with the given name
// was cleared in this mutation.
func (m *ChatMutation) EdgeCleared(name string) bool {
	return false
}

// ClearEdge clears the value of the edge with the given name. It returns an error
// if that edge is not defined in the schema.
func (m *ChatMutation) ClearEdge(name string) error {
	return fmt.Errorf("unknown Chat unique edge %s", name)
}

// ResetEdge resets all changes to the edge with the given name in this mutation.
// It returns an error if the edge is not defined in the schema.
func (m *ChatMutation) ResetEdge(name string) error {
	return fmt.Errorf("unknown Chat edge %s", name)
}
