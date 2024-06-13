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

package chat

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldLTE(FieldID, id))
}

// Title applies equality check predicate on the "title" field. It's identical to TitleEQ.
func Title(v string) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldTitle, v))
}

// UserId applies equality check predicate on the "userId" field. It's identical to UserIdEQ.
func UserId(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldUserId, v))
}

// CreatedAt applies equality check predicate on the "createdAt" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldCreatedAt, v))
}

// TitleEQ applies the EQ predicate on the "title" field.
func TitleEQ(v string) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldTitle, v))
}

// TitleNEQ applies the NEQ predicate on the "title" field.
func TitleNEQ(v string) predicate.Chat {
	return predicate.Chat(sql.FieldNEQ(FieldTitle, v))
}

// TitleIn applies the In predicate on the "title" field.
func TitleIn(vs ...string) predicate.Chat {
	return predicate.Chat(sql.FieldIn(FieldTitle, vs...))
}

// TitleNotIn applies the NotIn predicate on the "title" field.
func TitleNotIn(vs ...string) predicate.Chat {
	return predicate.Chat(sql.FieldNotIn(FieldTitle, vs...))
}

// TitleGT applies the GT predicate on the "title" field.
func TitleGT(v string) predicate.Chat {
	return predicate.Chat(sql.FieldGT(FieldTitle, v))
}

// TitleGTE applies the GTE predicate on the "title" field.
func TitleGTE(v string) predicate.Chat {
	return predicate.Chat(sql.FieldGTE(FieldTitle, v))
}

// TitleLT applies the LT predicate on the "title" field.
func TitleLT(v string) predicate.Chat {
	return predicate.Chat(sql.FieldLT(FieldTitle, v))
}

// TitleLTE applies the LTE predicate on the "title" field.
func TitleLTE(v string) predicate.Chat {
	return predicate.Chat(sql.FieldLTE(FieldTitle, v))
}

// TitleContains applies the Contains predicate on the "title" field.
func TitleContains(v string) predicate.Chat {
	return predicate.Chat(sql.FieldContains(FieldTitle, v))
}

// TitleHasPrefix applies the HasPrefix predicate on the "title" field.
func TitleHasPrefix(v string) predicate.Chat {
	return predicate.Chat(sql.FieldHasPrefix(FieldTitle, v))
}

// TitleHasSuffix applies the HasSuffix predicate on the "title" field.
func TitleHasSuffix(v string) predicate.Chat {
	return predicate.Chat(sql.FieldHasSuffix(FieldTitle, v))
}

// TitleEqualFold applies the EqualFold predicate on the "title" field.
func TitleEqualFold(v string) predicate.Chat {
	return predicate.Chat(sql.FieldEqualFold(FieldTitle, v))
}

// TitleContainsFold applies the ContainsFold predicate on the "title" field.
func TitleContainsFold(v string) predicate.Chat {
	return predicate.Chat(sql.FieldContainsFold(FieldTitle, v))
}

// UserIdEQ applies the EQ predicate on the "userId" field.
func UserIdEQ(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldUserId, v))
}

// UserIdNEQ applies the NEQ predicate on the "userId" field.
func UserIdNEQ(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldNEQ(FieldUserId, v))
}

// UserIdIn applies the In predicate on the "userId" field.
func UserIdIn(vs ...uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldIn(FieldUserId, vs...))
}

// UserIdNotIn applies the NotIn predicate on the "userId" field.
func UserIdNotIn(vs ...uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldNotIn(FieldUserId, vs...))
}

// UserIdGT applies the GT predicate on the "userId" field.
func UserIdGT(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldGT(FieldUserId, v))
}

// UserIdGTE applies the GTE predicate on the "userId" field.
func UserIdGTE(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldGTE(FieldUserId, v))
}

// UserIdLT applies the LT predicate on the "userId" field.
func UserIdLT(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldLT(FieldUserId, v))
}

// UserIdLTE applies the LTE predicate on the "userId" field.
func UserIdLTE(v uuid.UUID) predicate.Chat {
	return predicate.Chat(sql.FieldLTE(FieldUserId, v))
}

// CreatedAtEQ applies the EQ predicate on the "createdAt" field.
func CreatedAtEQ(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "createdAt" field.
func CreatedAtNEQ(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "createdAt" field.
func CreatedAtIn(vs ...time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "createdAt" field.
func CreatedAtNotIn(vs ...time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "createdAt" field.
func CreatedAtGT(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "createdAt" field.
func CreatedAtGTE(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "createdAt" field.
func CreatedAtLT(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "createdAt" field.
func CreatedAtLTE(v time.Time) predicate.Chat {
	return predicate.Chat(sql.FieldLTE(FieldCreatedAt, v))
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Chat) predicate.Chat {
	return predicate.Chat(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Chat) predicate.Chat {
	return predicate.Chat(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Chat) predicate.Chat {
	return predicate.Chat(sql.NotPredicates(p))
}
