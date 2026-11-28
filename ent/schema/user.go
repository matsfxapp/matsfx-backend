// Package schema declares the database tables
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newID).Immutable(),
		field.String("username").Unique(),
		field.String("display_name"),
		field.String("password_hash").Sensitive(),
		field.String("bio").Default(""),
		field.String("avatar_key").Optional().Nillable(),
		field.String("banner_key").Optional().Nillable(),
		field.Enum("role").Values("user", "admin").Default("user"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func newID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
