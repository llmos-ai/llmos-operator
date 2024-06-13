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

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/dialect/sql/sqljson"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/chat"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/predicate"
	v1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

// ChatUpdate is the builder for updating Chat entities.
type ChatUpdate struct {
	config
	hooks    []Hook
	mutation *ChatMutation
}

// Where appends a list predicates to the ChatUpdate builder.
func (cu *ChatUpdate) Where(ps ...predicate.Chat) *ChatUpdate {
	cu.mutation.Where(ps...)
	return cu
}

// SetTitle sets the "title" field.
func (cu *ChatUpdate) SetTitle(s string) *ChatUpdate {
	cu.mutation.SetTitle(s)
	return cu
}

// SetNillableTitle sets the "title" field if the given value is not nil.
func (cu *ChatUpdate) SetNillableTitle(s *string) *ChatUpdate {
	if s != nil {
		cu.SetTitle(*s)
	}
	return cu
}

// SetUserId sets the "userId" field.
func (cu *ChatUpdate) SetUserId(u uuid.UUID) *ChatUpdate {
	cu.mutation.SetUserId(u)
	return cu
}

// SetNillableUserId sets the "userId" field if the given value is not nil.
func (cu *ChatUpdate) SetNillableUserId(u *uuid.UUID) *ChatUpdate {
	if u != nil {
		cu.SetUserId(*u)
	}
	return cu
}

// SetModels sets the "models" field.
func (cu *ChatUpdate) SetModels(s []string) *ChatUpdate {
	cu.mutation.SetModels(s)
	return cu
}

// AppendModels appends s to the "models" field.
func (cu *ChatUpdate) AppendModels(s []string) *ChatUpdate {
	cu.mutation.AppendModels(s)
	return cu
}

// SetTags sets the "tags" field.
func (cu *ChatUpdate) SetTags(s []string) *ChatUpdate {
	cu.mutation.SetTags(s)
	return cu
}

// AppendTags appends s to the "tags" field.
func (cu *ChatUpdate) AppendTags(s []string) *ChatUpdate {
	cu.mutation.AppendTags(s)
	return cu
}

// SetHistory sets the "history" field.
func (cu *ChatUpdate) SetHistory(v v1.History) *ChatUpdate {
	cu.mutation.SetHistory(v)
	return cu
}

// SetNillableHistory sets the "history" field if the given value is not nil.
func (cu *ChatUpdate) SetNillableHistory(v *v1.History) *ChatUpdate {
	if v != nil {
		cu.SetHistory(*v)
	}
	return cu
}

// SetMessages sets the "messages" field.
func (cu *ChatUpdate) SetMessages(v []v1.Message) *ChatUpdate {
	cu.mutation.SetMessages(v)
	return cu
}

// AppendMessages appends v to the "messages" field.
func (cu *ChatUpdate) AppendMessages(v []v1.Message) *ChatUpdate {
	cu.mutation.AppendMessages(v)
	return cu
}

// Mutation returns the ChatMutation object of the builder.
func (cu *ChatUpdate) Mutation() *ChatMutation {
	return cu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (cu *ChatUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, cu.sqlSave, cu.mutation, cu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (cu *ChatUpdate) SaveX(ctx context.Context) int {
	affected, err := cu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (cu *ChatUpdate) Exec(ctx context.Context) error {
	_, err := cu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cu *ChatUpdate) ExecX(ctx context.Context) {
	if err := cu.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (cu *ChatUpdate) check() error {
	if v, ok := cu.mutation.Title(); ok {
		if err := chat.TitleValidator(v); err != nil {
			return &ValidationError{Name: "title", err: fmt.Errorf(`ent: validator failed for field "Chat.title": %w`, err)}
		}
	}
	return nil
}

func (cu *ChatUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := cu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(chat.Table, chat.Columns, sqlgraph.NewFieldSpec(chat.FieldID, field.TypeUUID))
	if ps := cu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := cu.mutation.Title(); ok {
		_spec.SetField(chat.FieldTitle, field.TypeString, value)
	}
	if value, ok := cu.mutation.UserId(); ok {
		_spec.SetField(chat.FieldUserId, field.TypeUUID, value)
	}
	if value, ok := cu.mutation.Models(); ok {
		_spec.SetField(chat.FieldModels, field.TypeJSON, value)
	}
	if value, ok := cu.mutation.AppendedModels(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldModels, value)
		})
	}
	if value, ok := cu.mutation.Tags(); ok {
		_spec.SetField(chat.FieldTags, field.TypeJSON, value)
	}
	if value, ok := cu.mutation.AppendedTags(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldTags, value)
		})
	}
	if value, ok := cu.mutation.History(); ok {
		_spec.SetField(chat.FieldHistory, field.TypeJSON, value)
	}
	if value, ok := cu.mutation.Messages(); ok {
		_spec.SetField(chat.FieldMessages, field.TypeJSON, value)
	}
	if value, ok := cu.mutation.AppendedMessages(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldMessages, value)
		})
	}
	if n, err = sqlgraph.UpdateNodes(ctx, cu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{chat.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	cu.mutation.done = true
	return n, nil
}

// ChatUpdateOne is the builder for updating a single Chat entity.
type ChatUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *ChatMutation
}

// SetTitle sets the "title" field.
func (cuo *ChatUpdateOne) SetTitle(s string) *ChatUpdateOne {
	cuo.mutation.SetTitle(s)
	return cuo
}

// SetNillableTitle sets the "title" field if the given value is not nil.
func (cuo *ChatUpdateOne) SetNillableTitle(s *string) *ChatUpdateOne {
	if s != nil {
		cuo.SetTitle(*s)
	}
	return cuo
}

// SetUserId sets the "userId" field.
func (cuo *ChatUpdateOne) SetUserId(u uuid.UUID) *ChatUpdateOne {
	cuo.mutation.SetUserId(u)
	return cuo
}

// SetNillableUserId sets the "userId" field if the given value is not nil.
func (cuo *ChatUpdateOne) SetNillableUserId(u *uuid.UUID) *ChatUpdateOne {
	if u != nil {
		cuo.SetUserId(*u)
	}
	return cuo
}

// SetModels sets the "models" field.
func (cuo *ChatUpdateOne) SetModels(s []string) *ChatUpdateOne {
	cuo.mutation.SetModels(s)
	return cuo
}

// AppendModels appends s to the "models" field.
func (cuo *ChatUpdateOne) AppendModels(s []string) *ChatUpdateOne {
	cuo.mutation.AppendModels(s)
	return cuo
}

// SetTags sets the "tags" field.
func (cuo *ChatUpdateOne) SetTags(s []string) *ChatUpdateOne {
	cuo.mutation.SetTags(s)
	return cuo
}

// AppendTags appends s to the "tags" field.
func (cuo *ChatUpdateOne) AppendTags(s []string) *ChatUpdateOne {
	cuo.mutation.AppendTags(s)
	return cuo
}

// SetHistory sets the "history" field.
func (cuo *ChatUpdateOne) SetHistory(v v1.History) *ChatUpdateOne {
	cuo.mutation.SetHistory(v)
	return cuo
}

// SetNillableHistory sets the "history" field if the given value is not nil.
func (cuo *ChatUpdateOne) SetNillableHistory(v *v1.History) *ChatUpdateOne {
	if v != nil {
		cuo.SetHistory(*v)
	}
	return cuo
}

// SetMessages sets the "messages" field.
func (cuo *ChatUpdateOne) SetMessages(v []v1.Message) *ChatUpdateOne {
	cuo.mutation.SetMessages(v)
	return cuo
}

// AppendMessages appends v to the "messages" field.
func (cuo *ChatUpdateOne) AppendMessages(v []v1.Message) *ChatUpdateOne {
	cuo.mutation.AppendMessages(v)
	return cuo
}

// Mutation returns the ChatMutation object of the builder.
func (cuo *ChatUpdateOne) Mutation() *ChatMutation {
	return cuo.mutation
}

// Where appends a list predicates to the ChatUpdate builder.
func (cuo *ChatUpdateOne) Where(ps ...predicate.Chat) *ChatUpdateOne {
	cuo.mutation.Where(ps...)
	return cuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (cuo *ChatUpdateOne) Select(field string, fields ...string) *ChatUpdateOne {
	cuo.fields = append([]string{field}, fields...)
	return cuo
}

// Save executes the query and returns the updated Chat entity.
func (cuo *ChatUpdateOne) Save(ctx context.Context) (*Chat, error) {
	return withHooks(ctx, cuo.sqlSave, cuo.mutation, cuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (cuo *ChatUpdateOne) SaveX(ctx context.Context) *Chat {
	node, err := cuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (cuo *ChatUpdateOne) Exec(ctx context.Context) error {
	_, err := cuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (cuo *ChatUpdateOne) ExecX(ctx context.Context) {
	if err := cuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (cuo *ChatUpdateOne) check() error {
	if v, ok := cuo.mutation.Title(); ok {
		if err := chat.TitleValidator(v); err != nil {
			return &ValidationError{Name: "title", err: fmt.Errorf(`ent: validator failed for field "Chat.title": %w`, err)}
		}
	}
	return nil
}

func (cuo *ChatUpdateOne) sqlSave(ctx context.Context) (_node *Chat, err error) {
	if err := cuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(chat.Table, chat.Columns, sqlgraph.NewFieldSpec(chat.FieldID, field.TypeUUID))
	id, ok := cuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Chat.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := cuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, chat.FieldID)
		for _, f := range fields {
			if !chat.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != chat.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := cuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := cuo.mutation.Title(); ok {
		_spec.SetField(chat.FieldTitle, field.TypeString, value)
	}
	if value, ok := cuo.mutation.UserId(); ok {
		_spec.SetField(chat.FieldUserId, field.TypeUUID, value)
	}
	if value, ok := cuo.mutation.Models(); ok {
		_spec.SetField(chat.FieldModels, field.TypeJSON, value)
	}
	if value, ok := cuo.mutation.AppendedModels(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldModels, value)
		})
	}
	if value, ok := cuo.mutation.Tags(); ok {
		_spec.SetField(chat.FieldTags, field.TypeJSON, value)
	}
	if value, ok := cuo.mutation.AppendedTags(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldTags, value)
		})
	}
	if value, ok := cuo.mutation.History(); ok {
		_spec.SetField(chat.FieldHistory, field.TypeJSON, value)
	}
	if value, ok := cuo.mutation.Messages(); ok {
		_spec.SetField(chat.FieldMessages, field.TypeJSON, value)
	}
	if value, ok := cuo.mutation.AppendedMessages(); ok {
		_spec.AddModifier(func(u *sql.UpdateBuilder) {
			sqljson.Append(u, chat.FieldMessages, value)
		})
	}
	_node = &Chat{config: cuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, cuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{chat.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	cuo.mutation.done = true
	return _node, nil
}
