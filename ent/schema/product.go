package schema

import (
	"context"
	"strconv"
	"strings"
	"time"
	"yinni_backend/ent/hook"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// Product holds the schema definition for the Product entity.
type Product struct {
	ent.Schema
}

// Mixins for Product
func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		// Original ID from dataset
		field.String("original_id").
			Unique().
			Comment("Original _id from dataset"),

		// Basic product info
		field.String("title").
			NotEmpty().
			Comment("Product title"),
		field.String("brand").
			NotEmpty().
			Comment("Brand name"),
		field.Text("description").
			Optional().
			Comment("Product description"),

		// Pricing
		field.String("actual_price").
			Optional().
			Comment("Original price"),
		field.String("selling_price").
			Optional().
			Comment("Discounted price"),
		field.String("discount").
			Optional().
			Comment("Discount percentage"),

		// Categorization
		field.String("category").
			NotEmpty().
			Comment("Main category"),
		field.String("sub_category").
			NotEmpty().
			Comment("Sub category"),

		// Stock & Seller
		field.Bool("out_of_stock").
			Default(false),
		field.String("seller").
			Optional().
			Comment("Seller name"),

		// Ratings
		field.String("average_rating").
			Optional().
			Comment("Average rating"),

		// Images
		field.JSON("images", []string{}).
			Optional().
			Comment("Product image URLs"),

		// Product details (dynamic JSON)
		field.JSON("product_details", []map[string]string{}).
			Optional().
			Comment("Dynamic product attributes"),

		// URLs and identifiers
		field.String("url").
			Optional().
			Comment("Product URL"),
		field.String("pid").
			Unique().
			Comment("Flipkart product ID"),
		field.String("style_code").
			Optional().
			Comment("Style/sku code"),

		// Crawled timestamp
		field.Time("crawled_at").
			Optional().
			Comment("When the data was crawled"),

		// Embeddings for vector search
		field.JSON("embedding", []float32{}).
			Optional().
			Comment("Vector embeddings for semantic search"),

		// Search index fields
		field.JSON("search_keywords", []string{}).
			Optional().
			Comment("Keywords for full-text search"),

		// Metadata
		field.Bool("featured").
			Default(false).
			Comment("Featured product"),
		field.Int("view_count").
			Default(0).
			Comment("Number of views"),
		field.Int("click_count").
			Default(0).
			Comment("Number of clicks"),

		// Price as integer for sorting/filtering
		field.Int("price_numeric").
			Optional().
			Min(0).
			Comment("Price as integer for filtering"),

		// Rating as float for sorting
		field.Float("rating_numeric").
			Optional().
			Min(0).
			Max(5).
			Comment("Rating as float for sorting"),
	}
}

// Edges of the Product.
func (Product) Edges() []ent.Edge {
	return []ent.Edge{
		// Add relationships here if needed
		// For example:
		// edge.To("category", Category.Type),
		// edge.To("reviews", Review.Type),
		// edge.To("similar_products", Product.Type),
	}
}

// Indexes for Product
func (Product) Indexes() []ent.Index {
	return []ent.Index{
		// Primary indexes
		index.Fields("pid").Unique(),
		index.Fields("original_id").Unique(),

		// Search indexes
		index.Fields("brand"),
		index.Fields("category"),
		index.Fields("sub_category"),

		// Performance indexes
		index.Fields("price_numeric"),
		index.Fields("rating_numeric"),
		index.Fields("out_of_stock"),
		index.Fields("featured"),

		// Composite indexes for common queries
		index.Fields("category", "sub_category"),
		index.Fields("brand", "category"),
		index.Fields("category", "price_numeric"),
		index.Fields("category", "rating_numeric"),

		// Full-text search (if supported by your DB)
		// index.Fields("title", "description", "search_keywords").StorageKey("product_search_idx"),
	}
}

// Hooks for Product
func (Product) Hooks() []ent.Hook {
	return []ent.Hook{
		// Pre-create hook to set numeric fields
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
					// Convert price string to numeric
					if m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne) {
						if price, ok := m.Field("selling_price"); ok {
							priceStr, _ := price.(string)
							numericPrice := extractPriceNumber(priceStr)
							m.SetField("price_numeric", numericPrice)
						}

						if rating, ok := m.Field("average_rating"); ok {
							ratingStr, _ := rating.(string)
							numericRating := extractRatingNumber(ratingStr)
							m.SetField("rating_numeric", numericRating)
						}

						// Set crawled_at if empty
						if crawledAt, ok := m.Field("crawled_at"); !ok || crawledAt.(time.Time).IsZero() {
							m.SetField("crawled_at", time.Now())
						}
					}

					return next.Mutate(ctx, m)
				})
			},
			ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne,
		),

		// Generate search keywords
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
					if m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne) {
						var keywords []string

						if title, ok := m.Field("title"); ok {
							titleStr, _ := title.(string)
							keywords = append(keywords, generateKeywords(titleStr)...)
						}

						if brand, ok := m.Field("brand"); ok {
							brandStr, _ := brand.(string)
							keywords = append(keywords, brandStr)
						}

						if category, ok := m.Field("category"); ok {
							categoryStr, _ := category.(string)
							keywords = append(keywords, categoryStr)
						}

						m.SetField("search_keywords", keywords)
					}

					return next.Mutate(ctx, m)
				})
			},
			ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne,
		),
	}
}

// Helper functions (add these in a separate helper file)
func extractPriceNumber(priceStr string) int {
	// Remove commas, ₹, $, etc. and convert to integer
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.ReplaceAll(priceStr, "₹", "")
	priceStr = strings.ReplaceAll(priceStr, "$", "")
	priceStr = strings.ReplaceAll(priceStr, " ", "")

	if price, err := strconv.Atoi(priceStr); err == nil {
		return price
	}
	return 0
}

func extractRatingNumber(ratingStr string) float64 {
	if rating, err := strconv.ParseFloat(ratingStr, 64); err == nil {
		return rating
	}
	return 0.0
}

func generateKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))

	// Common words to exclude
	stopWords := map[string]bool{
		"and": true, "or": true, "the": true, "a": true, "an": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"with": true, "by": true, "of": true, "men": true, "women": true,
	}

	var keywords []string
	for _, word := range words {
		if !stopWords[word] && len(word) > 2 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// package schema

// import "entgo.io/ent"

// // Product holds the schema definition for the Product entity.
// type Product struct {
// 	ent.Schema
// }

// // Fields of the Product.
// func (Product) Fields() []ent.Field {
// 	return nil
// }

// // Edges of the Product.
// func (Product) Edges() []ent.Edge {
// 	return nil
// }
