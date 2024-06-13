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

package migrate

import (
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// ChatsColumns holds the columns for the "chats" table.
	ChatsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeUUID, Unique: true},
		{Name: "title", Type: field.TypeString},
		{Name: "user_id", Type: field.TypeUUID},
		{Name: "models", Type: field.TypeJSON},
		{Name: "tags", Type: field.TypeJSON},
		{Name: "history", Type: field.TypeJSON},
		{Name: "messages", Type: field.TypeJSON},
		{Name: "created_at", Type: field.TypeTime},
	}
	// ChatsTable holds the schema information for the "chats" table.
	ChatsTable = &schema.Table{
		Name:       "chats",
		Columns:    ChatsColumns,
		PrimaryKey: []*schema.Column{ChatsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "chat_user_id",
				Unique:  false,
				Columns: []*schema.Column{ChatsColumns[2]},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		ChatsTable,
	}
)

func init() {
}
