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
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/chat"
	v1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

// ChatCreate is the builder for creating a Chat entity.
type ChatCreate struct {
	config
	mutation *ChatMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetTitle sets the "title" field.
func (cc *ChatCreate) SetTitle(s string) *ChatCreate {
	cc.mutation.SetTitle(s)
	return cc
}

// SetUserId sets the "userId" field.
func (cc *ChatCreate) SetUserId(u uuid.UUID) *ChatCreate {
	cc.mutation.SetUserId(u)
	return cc
}

// SetModels sets the "models" field.
func (cc *ChatCreate) SetModels(s []string) *ChatCreate {
	cc.mutation.SetModels(s)
	return cc
}

// SetTags sets the "tags" field.
func (cc *ChatCreate) SetTags(s []string) *ChatCreate {
	cc.mutation.SetTags(s)
	return cc
}

// SetHistory sets the "history" field.
func (cc *ChatCreate) SetHistory(v v1.History) *ChatCreate {
	cc.mutation.SetHistory(v)
	return cc
}

// SetMessages sets the "messages" field.
func (cc *ChatCreate) SetMessages(v []v1.Message) *ChatCreate {
	cc.mutation.SetMessages(v)
	return cc
}

// SetCreatedAt sets the "createdAt" field.
func (cc *ChatCreate) SetCreatedAt(t time.Time) *ChatCreate {
	cc.mutation.SetCreatedAt(t)
	return cc
}

// SetNillableCreatedAt sets the "createdAt" field if the given value is not nil.
func (cc *ChatCreate) SetNillableCreatedAt(t *time.Time) *ChatCreate {
	if t != nil {
		cc.SetCreatedAt(*t)
	}
	return cc
}

// SetID sets the "id" field.
func (cc *ChatCreate) SetID(u uuid.UUID) *ChatCreate {
	cc.mutation.SetID(u)
	return cc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (cc *ChatCreate) SetNillableID(u *uuid.UUID) *ChatCreate {
	if u != nil {
		cc.SetID(*u)
	}
	return cc
}

// Mutation returns the ChatMutation object of the builder.
func (cc *ChatCreate) Mutation() *ChatMutation {
	return cc.mutation
}

// Save creates the Chat in the database.
func (cc *ChatCreate) Save(ctx context.Context) (*Chat, error) {
	cc.defaults()
	return withHooks(ctx, cc.sqlSave, cc.mutation, cc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (cc *ChatCreate) SaveX(ctx context.Context) *Chat {
	v, err := cc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (cc *ChatCreate) Exec(ctx context.Context) error {
	_, err := cc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cc *ChatCreate) ExecX(ctx context.Context) {
	if err := cc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (cc *ChatCreate) defaults() {
	if _, ok := cc.mutation.CreatedAt(); !ok {
		v := chat.DefaultCreatedAt
		cc.mutation.SetCreatedAt(v)
	}
	if _, ok := cc.mutation.ID(); !ok {
		v := chat.DefaultID()
		cc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (cc *ChatCreate) check() error {
	if _, ok := cc.mutation.Title(); !ok {
		return &ValidationError{Name: "title", err: errors.New(`ent: missing required field "Chat.title"`)}
	}
	if v, ok := cc.mutation.Title(); ok {
		if err := chat.TitleValidator(v); err != nil {
			return &ValidationError{Name: "title", err: fmt.Errorf(`ent: validator failed for field "Chat.title": %w`, err)}
		}
	}
	if _, ok := cc.mutation.UserId(); !ok {
		return &ValidationError{Name: "userId", err: errors.New(`ent: missing required field "Chat.userId"`)}
	}
	if _, ok := cc.mutation.Models(); !ok {
		return &ValidationError{Name: "models", err: errors.New(`ent: missing required field "Chat.models"`)}
	}
	if _, ok := cc.mutation.Tags(); !ok {
		return &ValidationError{Name: "tags", err: errors.New(`ent: missing required field "Chat.tags"`)}
	}
	if _, ok := cc.mutation.History(); !ok {
		return &ValidationError{Name: "history", err: errors.New(`ent: missing required field "Chat.history"`)}
	}
	if _, ok := cc.mutation.Messages(); !ok {
		return &ValidationError{Name: "messages", err: errors.New(`ent: missing required field "Chat.messages"`)}
	}
	if _, ok := cc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "createdAt", err: errors.New(`ent: missing required field "Chat.createdAt"`)}
	}
	return nil
}

func (cc *ChatCreate) sqlSave(ctx context.Context) (*Chat, error) {
	if err := cc.check(); err != nil {
		return nil, err
	}
	_node, _spec := cc.createSpec()
	if err := sqlgraph.CreateNode(ctx, cc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	cc.mutation.id = &_node.ID
	cc.mutation.done = true
	return _node, nil
}

func (cc *ChatCreate) createSpec() (*Chat, *sqlgraph.CreateSpec) {
	var (
		_node = &Chat{config: cc.config}
		_spec = sqlgraph.NewCreateSpec(chat.Table, sqlgraph.NewFieldSpec(chat.FieldID, field.TypeUUID))
	)
	_spec.OnConflict = cc.conflict
	if id, ok := cc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := cc.mutation.Title(); ok {
		_spec.SetField(chat.FieldTitle, field.TypeString, value)
		_node.Title = value
	}
	if value, ok := cc.mutation.UserId(); ok {
		_spec.SetField(chat.FieldUserId, field.TypeUUID, value)
		_node.UserId = value
	}
	if value, ok := cc.mutation.Models(); ok {
		_spec.SetField(chat.FieldModels, field.TypeJSON, value)
		_node.Models = value
	}
	if value, ok := cc.mutation.Tags(); ok {
		_spec.SetField(chat.FieldTags, field.TypeJSON, value)
		_node.Tags = value
	}
	if value, ok := cc.mutation.History(); ok {
		_spec.SetField(chat.FieldHistory, field.TypeJSON, value)
		_node.History = value
	}
	if value, ok := cc.mutation.Messages(); ok {
		_spec.SetField(chat.FieldMessages, field.TypeJSON, value)
		_node.Messages = value
	}
	if value, ok := cc.mutation.CreatedAt(); ok {
		_spec.SetField(chat.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Chat.Create().
//		SetTitle(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.ChatUpsert) {
//			SetTitle(v+v).
//		}).
//		Exec(ctx)
func (cc *ChatCreate) OnConflict(opts ...sql.ConflictOption) *ChatUpsertOne {
	cc.conflict = opts
	return &ChatUpsertOne{
		create: cc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Chat.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (cc *ChatCreate) OnConflictColumns(columns ...string) *ChatUpsertOne {
	cc.conflict = append(cc.conflict, sql.ConflictColumns(columns...))
	return &ChatUpsertOne{
		create: cc,
	}
}

type (
	// ChatUpsertOne is the builder for "upsert"-ing
	//  one Chat node.
	ChatUpsertOne struct {
		create *ChatCreate
	}

	// ChatUpsert is the "OnConflict" setter.
	ChatUpsert struct {
		*sql.UpdateSet
	}
)

// SetTitle sets the "title" field.
func (u *ChatUpsert) SetTitle(v string) *ChatUpsert {
	u.Set(chat.FieldTitle, v)
	return u
}

// UpdateTitle sets the "title" field to the value that was provided on create.
func (u *ChatUpsert) UpdateTitle() *ChatUpsert {
	u.SetExcluded(chat.FieldTitle)
	return u
}

// SetUserId sets the "userId" field.
func (u *ChatUpsert) SetUserId(v uuid.UUID) *ChatUpsert {
	u.Set(chat.FieldUserId, v)
	return u
}

// UpdateUserId sets the "userId" field to the value that was provided on create.
func (u *ChatUpsert) UpdateUserId() *ChatUpsert {
	u.SetExcluded(chat.FieldUserId)
	return u
}

// SetModels sets the "models" field.
func (u *ChatUpsert) SetModels(v []string) *ChatUpsert {
	u.Set(chat.FieldModels, v)
	return u
}

// UpdateModels sets the "models" field to the value that was provided on create.
func (u *ChatUpsert) UpdateModels() *ChatUpsert {
	u.SetExcluded(chat.FieldModels)
	return u
}

// SetTags sets the "tags" field.
func (u *ChatUpsert) SetTags(v []string) *ChatUpsert {
	u.Set(chat.FieldTags, v)
	return u
}

// UpdateTags sets the "tags" field to the value that was provided on create.
func (u *ChatUpsert) UpdateTags() *ChatUpsert {
	u.SetExcluded(chat.FieldTags)
	return u
}

// SetHistory sets the "history" field.
func (u *ChatUpsert) SetHistory(v v1.History) *ChatUpsert {
	u.Set(chat.FieldHistory, v)
	return u
}

// UpdateHistory sets the "history" field to the value that was provided on create.
func (u *ChatUpsert) UpdateHistory() *ChatUpsert {
	u.SetExcluded(chat.FieldHistory)
	return u
}

// SetMessages sets the "messages" field.
func (u *ChatUpsert) SetMessages(v []v1.Message) *ChatUpsert {
	u.Set(chat.FieldMessages, v)
	return u
}

// UpdateMessages sets the "messages" field to the value that was provided on create.
func (u *ChatUpsert) UpdateMessages() *ChatUpsert {
	u.SetExcluded(chat.FieldMessages)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.Chat.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(chat.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *ChatUpsertOne) UpdateNewValues() *ChatUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(chat.FieldID)
		}
		if _, exists := u.create.mutation.CreatedAt(); exists {
			s.SetIgnore(chat.FieldCreatedAt)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Chat.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *ChatUpsertOne) Ignore() *ChatUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *ChatUpsertOne) DoNothing() *ChatUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the ChatCreate.OnConflict
// documentation for more info.
func (u *ChatUpsertOne) Update(set func(*ChatUpsert)) *ChatUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&ChatUpsert{UpdateSet: update})
	}))
	return u
}

// SetTitle sets the "title" field.
func (u *ChatUpsertOne) SetTitle(v string) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetTitle(v)
	})
}

// UpdateTitle sets the "title" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateTitle() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateTitle()
	})
}

// SetUserId sets the "userId" field.
func (u *ChatUpsertOne) SetUserId(v uuid.UUID) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetUserId(v)
	})
}

// UpdateUserId sets the "userId" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateUserId() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateUserId()
	})
}

// SetModels sets the "models" field.
func (u *ChatUpsertOne) SetModels(v []string) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetModels(v)
	})
}

// UpdateModels sets the "models" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateModels() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateModels()
	})
}

// SetTags sets the "tags" field.
func (u *ChatUpsertOne) SetTags(v []string) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetTags(v)
	})
}

// UpdateTags sets the "tags" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateTags() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateTags()
	})
}

// SetHistory sets the "history" field.
func (u *ChatUpsertOne) SetHistory(v v1.History) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetHistory(v)
	})
}

// UpdateHistory sets the "history" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateHistory() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateHistory()
	})
}

// SetMessages sets the "messages" field.
func (u *ChatUpsertOne) SetMessages(v []v1.Message) *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.SetMessages(v)
	})
}

// UpdateMessages sets the "messages" field to the value that was provided on create.
func (u *ChatUpsertOne) UpdateMessages() *ChatUpsertOne {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateMessages()
	})
}

// Exec executes the query.
func (u *ChatUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for ChatCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *ChatUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *ChatUpsertOne) ID(ctx context.Context) (id uuid.UUID, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("ent: ChatUpsertOne.ID is not supported by MySQL driver. Use ChatUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *ChatUpsertOne) IDX(ctx context.Context) uuid.UUID {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// ChatCreateBulk is the builder for creating many Chat entities in bulk.
type ChatCreateBulk struct {
	config
	err      error
	builders []*ChatCreate
	conflict []sql.ConflictOption
}

// Save creates the Chat entities in the database.
func (ccb *ChatCreateBulk) Save(ctx context.Context) ([]*Chat, error) {
	if ccb.err != nil {
		return nil, ccb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(ccb.builders))
	nodes := make([]*Chat, len(ccb.builders))
	mutators := make([]Mutator, len(ccb.builders))
	for i := range ccb.builders {
		func(i int, root context.Context) {
			builder := ccb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*ChatMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, ccb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = ccb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, ccb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, ccb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (ccb *ChatCreateBulk) SaveX(ctx context.Context) []*Chat {
	v, err := ccb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (ccb *ChatCreateBulk) Exec(ctx context.Context) error {
	_, err := ccb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (ccb *ChatCreateBulk) ExecX(ctx context.Context) {
	if err := ccb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.Chat.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.ChatUpsert) {
//			SetTitle(v+v).
//		}).
//		Exec(ctx)
func (ccb *ChatCreateBulk) OnConflict(opts ...sql.ConflictOption) *ChatUpsertBulk {
	ccb.conflict = opts
	return &ChatUpsertBulk{
		create: ccb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.Chat.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (ccb *ChatCreateBulk) OnConflictColumns(columns ...string) *ChatUpsertBulk {
	ccb.conflict = append(ccb.conflict, sql.ConflictColumns(columns...))
	return &ChatUpsertBulk{
		create: ccb,
	}
}

// ChatUpsertBulk is the builder for "upsert"-ing
// a bulk of Chat nodes.
type ChatUpsertBulk struct {
	create *ChatCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.Chat.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(chat.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *ChatUpsertBulk) UpdateNewValues() *ChatUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(chat.FieldID)
			}
			if _, exists := b.mutation.CreatedAt(); exists {
				s.SetIgnore(chat.FieldCreatedAt)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.Chat.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *ChatUpsertBulk) Ignore() *ChatUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *ChatUpsertBulk) DoNothing() *ChatUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the ChatCreateBulk.OnConflict
// documentation for more info.
func (u *ChatUpsertBulk) Update(set func(*ChatUpsert)) *ChatUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&ChatUpsert{UpdateSet: update})
	}))
	return u
}

// SetTitle sets the "title" field.
func (u *ChatUpsertBulk) SetTitle(v string) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetTitle(v)
	})
}

// UpdateTitle sets the "title" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateTitle() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateTitle()
	})
}

// SetUserId sets the "userId" field.
func (u *ChatUpsertBulk) SetUserId(v uuid.UUID) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetUserId(v)
	})
}

// UpdateUserId sets the "userId" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateUserId() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateUserId()
	})
}

// SetModels sets the "models" field.
func (u *ChatUpsertBulk) SetModels(v []string) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetModels(v)
	})
}

// UpdateModels sets the "models" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateModels() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateModels()
	})
}

// SetTags sets the "tags" field.
func (u *ChatUpsertBulk) SetTags(v []string) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetTags(v)
	})
}

// UpdateTags sets the "tags" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateTags() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateTags()
	})
}

// SetHistory sets the "history" field.
func (u *ChatUpsertBulk) SetHistory(v v1.History) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetHistory(v)
	})
}

// UpdateHistory sets the "history" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateHistory() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateHistory()
	})
}

// SetMessages sets the "messages" field.
func (u *ChatUpsertBulk) SetMessages(v []v1.Message) *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.SetMessages(v)
	})
}

// UpdateMessages sets the "messages" field to the value that was provided on create.
func (u *ChatUpsertBulk) UpdateMessages() *ChatUpsertBulk {
	return u.Update(func(s *ChatUpsert) {
		s.UpdateMessages()
	})
}

// Exec executes the query.
func (u *ChatUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("ent: OnConflict was set for builder %d. Set it on the ChatCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for ChatCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *ChatUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
