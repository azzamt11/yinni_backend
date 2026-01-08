package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
		field.Bool("is_admin").Default(false).Comment("Admin users can access admin endpoints"),
		field.JSON("roles", []string{}).Default([]string{"user"}), // Simple array of role names
		field.JSON("permissions", []string{}).Optional(),          // Optional granular permissions
	}
}

func (User) Edges() []ent.Edge {
	return nil // No edges for simple implementation
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").Unique(),
		index.Fields("is_admin"), // For quick admin lookups
	}
}
