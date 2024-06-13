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
	"time"

	"github.com/google/uuid"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent/chat"
	v1 "github.com/llmos-ai/llmos-controller/pkg/types/v1"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	chatFields := v1.Chat{}.Fields()
	_ = chatFields
	// chatDescTitle is the schema descriptor for title field.
	chatDescTitle := chatFields[1].Descriptor()
	// chat.TitleValidator is a validator for the "title" field. It is called by the builders before save.
	chat.TitleValidator = chatDescTitle.Validators[0].(func(string) error)
	// chatDescCreatedAt is the schema descriptor for createdAt field.
	chatDescCreatedAt := chatFields[7].Descriptor()
	// chat.DefaultCreatedAt holds the default value on creation for the createdAt field.
	chat.DefaultCreatedAt = chatDescCreatedAt.Default.(time.Time)
	// chatDescID is the schema descriptor for id field.
	chatDescID := chatFields[0].Descriptor()
	// chat.DefaultID holds the default value on creation for the id field.
	chat.DefaultID = chatDescID.Default.(func() uuid.UUID)
}
