package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

/*
|--------------------------------------------------------------------------
| TimeMixin
|--------------------------------------------------------------------------
*/
type TimeMixin struct {
	mixin.Schema
}

func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

/*
|--------------------------------------------------------------------------
| User schema
|--------------------------------------------------------------------------
*/
type User struct {
	ent.Schema
}

// ✅ SINGULAR — required for ent v0.14.x
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("age").Optional(),
		field.String("name").NotEmpty(),
		field.String("email").NotEmpty().Unique(),
		field.String("username").Optional(),
		field.String("phone").Optional(),
		field.String("image").Optional(),
		field.String("password").NotEmpty(),
	}
}

func (User) Edges() []ent.Edge {
	return nil
}
