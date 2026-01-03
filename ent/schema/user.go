package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Mixin defines the mixins for the User entity.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
		// Alternatively, you can use:
		// mixin.CreateTime{},
		// mixin.UpdateTime{},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("age").
			Positive().
			Optional(), // Make optional since auth might not need age
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
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
