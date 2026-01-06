package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("age").
			Optional(),
		field.String("name").
			NotEmpty(),
		field.String("email").
			NotEmpty().
			Unique(),
		field.String("phone").
			Optional(),
		field.String("username").
			Optional(),
		field.String("password").
			NotEmpty(),
		// Simple optional fields - NO defaults, NO mixins
		field.Time("created_at").
			Optional(),
		field.Time("updated_at").
			Optional(),
	}
}

func (User) Edges() []ent.Edge {
	return nil
}
