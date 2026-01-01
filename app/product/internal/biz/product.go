package biz

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	v1 "yinni_backend/api/product/v1"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrProductNotFound      = errors.NotFound(v1.ErrorReason_PRODUCT_NOT_FOUND.String(), "product not found")
	ErrInvalidProductID     = errors.BadRequest(v1.ErrorReason_INVALID_PRODUCT_ID.String(), "invalid product id")
	ErrInvalidPriceRange    = errors.BadRequest(v1.ErrorReason_INVALID_PRICE_RANGE.String(), "invalid price range")
	ErrInvalidParameters    = errors.BadRequest(v1.ErrorReason_INVALID_PARAMETERS.String(), "invalid parameters")
	ErrDatabaseError        = errors.InternalServer(v1.ErrorReason_DATABASE_ERROR.String(), "database error")
	ErrSearchFailed         = errors.InternalServer(v1.ErrorReason_SEARCH_FAILED.String(), "search failed")
	ErrEmbeddingsNotEnabled = errors.InternalServer(v1.ErrorReason_EMBEDDING_IS_NOT_ENABLED.String(), "embeddings not enabled")
)

// Product is a Product model.
type Product struct {
	ID             int64
	OriginalID     string
	Title          string
	Brand          string
	Description    string
	ActualPrice    string
	SellingPrice   string
	Discount       string
	PriceNumeric   int
	Category       string
	SubCategory    string
	OutOfStock     bool
	Seller         string
	AverageRating  string
	RatingNumeric  float32
	Images         []string
	ProductDetails []map[string]string
	URL            string
	PID            string
	StyleCode      string
	CrawledAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ViewCount      int
	ClickCount     int
	Featured       bool
	Embedding      []float32 // Add this field
	SearchKeywords []string  // Add this field
}

// ProductListItem is a lightweight version for lists
type ProductListItem struct {
	ID            int64
	Title         string
	Brand         string
	SellingPrice  string
	ActualPrice   string
	Discount      string
	Category      string
	SubCategory   string
	OutOfStock    bool
	AverageRating string
	RatingNumeric float32
	Images        []string
	Featured      bool
}

// ProductRepo is a Product repository interface.
type ProductRepo interface {
	// Basic CRUD
	Create(context.Context, *Product) (*Product, error)
	Update(context.Context, *Product) (*Product, error)
	Delete(context.Context, int64) (*Product, error)
	GetProduct(context.Context, int64) (*Product, error)
	GetProductByPID(context.Context, string) (*Product, error)

	// List operations
	ListAllProducts(context.Context) ([]*Product, error)
	ListProducts(context.Context, *ListProductsParams) ([]*Product, int64, error)

	// Search
	SearchProducts(context.Context, string, *ListProductsParams) ([]*Product, int64, error)

	// Special queries
	GetFeaturedProducts(context.Context, int, string) ([]*Product, error)
	GetSimilarProducts(context.Context, int64, int) ([]*Product, error)

	// Analytics
	IncrementViewCount(context.Context, int64) error
	IncrementClickCount(context.Context, int64) error

	// Embedding operations
	GenerateEmbedding(ctx context.Context, product *Product) ([]float32, error)
	SearchSimilarProducts(ctx context.Context, queryEmbedding []float32, limit int, category string, priceRange *PriceRange) ([]*Product, error)
	UpdateProductEmbedding(ctx context.Context, id int64, embedding []float32) error
	BatchUpdateEmbeddings(ctx context.Context, productEmbeddings map[int64][]float32) error
	GetProductsWithoutEmbeddings(ctx context.Context, limit int) ([]*Product, error)
	GetProductsWithEmbeddings(ctx context.Context, limit int) ([]*Product, error)
}

// ListProductsParams defines parameters for listing products
type ListProductsParams struct {
	Page        int32
	PageSize    int32
	Category    string
	Brand       string
	SubCategory string
	MinPrice    int32
	MaxPrice    int32
	MinRating   float32
	InStock     bool
	Featured    bool
	Seller      string
	SortBy      string
	SortOrder   string
	SearchQuery string
}

// Validate validates the ListProductsParams
func (p *ListProductsParams) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	if p.MinPrice < 0 {
		p.MinPrice = 0
	}
	if p.MaxPrice < 0 {
		p.MaxPrice = 0
	}
	if p.MinPrice > p.MaxPrice && p.MaxPrice > 0 {
		return ErrInvalidPriceRange
	}
	return nil
}

// PriceRange for embedding search
type PriceRange struct {
	Min int32
	Max int32
}

// EmbeddingConfig for AI features
type EmbeddingConfig struct {
	ApiKey     string
	Model      string
	BatchSize  int32
	BaseUrl    string
	Timeout    int32
	MaxRetries int32
	Enabled    bool
}

// ProductUsecase is a Product usecase.
type ProductUsecase struct {
	repo     ProductRepo
	log      *log.Helper
	embedCfg *EmbeddingConfig
}

// NewProductUsecase creates a new ProductUsecase.
func NewProductUsecase(repo ProductRepo, conf *conf.Embeddings, logger log.Logger) *ProductUsecase {
	return &ProductUsecase{
		repo: repo,
		embedCfg: &EmbeddingConfig{
			ApiKey:     conf.ApiKey,
			Model:      conf.Model,
			BatchSize:  conf.BatchSize,
			BaseUrl:    conf.BaseUrl,
			Timeout:    1000,
			MaxRetries: conf.MaxRetries,
			Enabled:    true,
		},
		log: log.NewHelper(logger),
	}
}

// ========== BASIC CRUD OPERATIONS ==========

// CreateProduct creates a new Product.
func (uc *ProductUsecase) CreateProduct(ctx context.Context, p *Product) (*Product, error) {
	uc.log.Infof("CreateProduct: %v", p.Title)
	return uc.repo.Create(ctx, p)
}

// UpdateProduct updates an existing Product.
func (uc *ProductUsecase) UpdateProduct(ctx context.Context, p *Product) (*Product, error) {
	uc.log.Infof("UpdateProduct: %v", p.ID)
	return uc.repo.Update(ctx, p)
}

// DeleteProduct deletes a Product.
func (uc *ProductUsecase) DeleteProduct(ctx context.Context, id int64) (*Product, error) {
	uc.log.Infof("DeleteProduct: %v", id)
	return uc.repo.Delete(ctx, id)
}

// GetProduct retrieves a Product by ID.
func (uc *ProductUsecase) GetProduct(ctx context.Context, id int64) (*Product, error) {
	uc.log.Infof("GetProduct: %v", id)

	if id <= 0 {
		return nil, ErrInvalidProductID
	}

	product, err := uc.repo.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	// Increment view count asynchronously
	go func() {
		_ = uc.repo.IncrementViewCount(context.Background(), id)
	}()

	return product, nil
}

// GetProductByPID retrieves a Product by Flipkart PID.
func (uc *ProductUsecase) GetProductByPID(ctx context.Context, pid string) (*Product, error) {
	uc.log.Infof("GetProductByPID: %v", pid)

	if pid == "" {
		return nil, ErrInvalidParameters
	}

	return uc.repo.GetProductByPID(ctx, pid)
}

// ListAllProducts retrieves all Products.
func (uc *ProductUsecase) ListAllProducts(ctx context.Context) ([]*Product, error) {
	uc.log.Info("ListAllProducts")
	return uc.repo.ListAllProducts(ctx)
}

// ListProducts retrieves Products with filtering and pagination.
func (uc *ProductUsecase) ListProducts(ctx context.Context, params *ListProductsParams) ([]*Product, int64, error) {
	uc.log.Infof("ListProducts: page=%d, pageSize=%d", params.Page, params.PageSize)

	if err := params.Validate(); err != nil {
		return nil, 0, err
	}

	return uc.repo.ListProducts(ctx, params)
}

// SearchProducts searches for Products.
func (uc *ProductUsecase) SearchProducts(ctx context.Context, query string, params *ListProductsParams) ([]*Product, int64, error) {
	uc.log.Infof("SearchProducts: %v", query)

	if query == "" {
		return uc.ListProducts(ctx, params)
	}

	if params == nil {
		params = &ListProductsParams{
			Page:     1,
			PageSize: 20,
		}
	}

	return uc.repo.SearchProducts(ctx, query, params)
}

// GetFeaturedProducts retrieves featured Products.
func (uc *ProductUsecase) GetFeaturedProducts(ctx context.Context, limit int, category string) ([]*Product, error) {
	uc.log.Infof("GetFeaturedProducts: limit=%d, category=%s", limit, category)

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	return uc.repo.GetFeaturedProducts(ctx, limit, category)
}

// GetSimilarProducts retrieves similar Products.
func (uc *ProductUsecase) GetSimilarProducts(ctx context.Context, id int64, limit int) ([]*Product, error) {
	uc.log.Infof("GetSimilarProducts: id=%d, limit=%d", id, limit)

	if id <= 0 {
		return nil, ErrInvalidProductID
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	return uc.repo.GetSimilarProducts(ctx, id, limit)
}

// RecordProductClick records a click on a Product.
func (uc *ProductUsecase) RecordProductClick(ctx context.Context, id int64) error {
	uc.log.Infof("RecordProductClick: %v", id)

	if id <= 0 {
		return ErrInvalidProductID
	}

	return uc.repo.IncrementClickCount(ctx, id)
}

// ========== EMBEDDING & AI SEARCH OPERATIONS ==========

// GenerateProductText generates text representation for embedding
func (uc *ProductUsecase) GenerateProductText(product *Product) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Title: %s\n", product.Title))
	sb.WriteString(fmt.Sprintf("Brand: %s\n", product.Brand))
	sb.WriteString(fmt.Sprintf("Category: %s - %s\n", product.Category, product.SubCategory))

	if product.Description != "" {
		desc := strings.TrimSpace(product.Description)
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("Description: %s\n", desc))
	}

	if len(product.ProductDetails) > 0 {
		sb.WriteString("Details: ")
		for _, detail := range product.ProductDetails {
			for k, v := range detail {
				sb.WriteString(fmt.Sprintf("%s: %s, ", k, v))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Price: %s (Discounted: %s)\n", product.ActualPrice, product.SellingPrice))
	sb.WriteString(fmt.Sprintf("Rating: %s\n", product.AverageRating))
	sb.WriteString(fmt.Sprintf("Seller: %s\n", product.Seller))

	return sb.String()
}

// GenerateEmbedding generates embedding for a product
func (uc *ProductUsecase) GenerateEmbedding(ctx context.Context, product *Product) ([]float32, error) {
	if !uc.embeddingsEnabled() {
		return nil, ErrEmbeddingsNotEnabled
	}

	return uc.repo.GenerateEmbedding(ctx, product)
}

// SearchWithEmbeddings searches products using vector similarity
func (uc *ProductUsecase) SearchWithEmbeddings(ctx context.Context, query string, limit int, category string, priceRange *PriceRange) ([]*Product, error) {
	if !uc.embeddingsEnabled() {
		return nil, ErrEmbeddingsNotEnabled
	}

	// Generate embedding for the query
	queryEmbedding, err := uc.repo.GenerateEmbedding(ctx, &Product{
		Title:       query,
		Description: query,
	})
	if err != nil {
		return nil, err
	}

	// Search similar products
	return uc.repo.SearchSimilarProducts(ctx, queryEmbedding, limit, category, priceRange)
}

// RAGSearch performs RAG-based semantic search
func (uc *ProductUsecase) RAGSearch(ctx context.Context, prompt string, limit int, category string, priceRange *PriceRange) ([]*Product, error) {
	if !uc.embeddingsEnabled() {
		return nil, ErrEmbeddingsNotEnabled
	}

	// First, find similar products based on the prompt
	products, err := uc.SearchWithEmbeddings(ctx, prompt, limit*2, category, priceRange)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	if len(products) == 0 {
		return products, nil
	}

	// Prepare context for LLM
	contextText := uc.buildContextFromProducts(products[:min(5, len(products))])

	// Note: This would require an LLM client. For now, we'll return vector results.
	// In a real implementation, you would call an LLM API here.
	uc.log.Infof("LLM context would be sent for query: %s", prompt)
	uc.log.Debugf("Context text: %s", contextText)

	// For now, return the top vector search results
	return products[:min(limit, len(products))], nil
}

// GenerateAllEmbeddings generates embeddings for all products
func (uc *ProductUsecase) GenerateAllEmbeddings(ctx context.Context, batchSize int) error {
	if !uc.embeddingsEnabled() {
		return ErrEmbeddingsNotEnabled
	}

	uc.log.Info("Starting embedding generation for all products")

	page := 1
	for {
		// Get products without embeddings
		params := &ListProductsParams{
			Page:     int32(page),
			PageSize: int32(batchSize),
		}

		products, _, err := uc.ListProducts(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to list products: %w", err)
		}

		if len(products) == 0 {
			break
		}

		// Process batch
		productEmbeddings := make(map[int64][]float32)
		for _, product := range products {
			// Skip if already has embedding
			if len(product.Embedding) > 0 {
				continue
			}

			embedding, err := uc.GenerateEmbedding(ctx, product)
			if err != nil {
				uc.log.Errorf("Failed to generate embedding for product %d: %v", product.ID, err)
				continue
			}

			productEmbeddings[product.ID] = embedding
			uc.log.Infof("Generated embedding for product %d: %s", product.ID, product.Title)

			// Rate limiting
			time.Sleep(100 * time.Millisecond)
		}

		// Batch update embeddings
		if len(productEmbeddings) > 0 {
			if err := uc.repo.BatchUpdateEmbeddings(ctx, productEmbeddings); err != nil {
				uc.log.Errorf("Failed to batch update embeddings: %v", err)
			}
		}

		page++
	}

	uc.log.Info("Embedding generation completed")
	return nil
}

// GenerateEmbeddingsForMissing generates embeddings only for products without them
func (uc *ProductUsecase) GenerateEmbeddingsForMissing(ctx context.Context, batchSize int) error {
	if !uc.embeddingsEnabled() {
		return ErrEmbeddingsNotEnabled
	}

	uc.log.Info("Generating embeddings for products without them")

	for {
		// Get products without embeddings
		products, err := uc.repo.GetProductsWithoutEmbeddings(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get products without embeddings: %w", err)
		}

		if len(products) == 0 {
			break
		}

		// Process batch
		productEmbeddings := make(map[int64][]float32)
		for _, product := range products {
			embedding, err := uc.GenerateEmbedding(ctx, product)
			if err != nil {
				uc.log.Errorf("Failed to generate embedding for product %d: %v", product.ID, err)
				continue
			}

			productEmbeddings[product.ID] = embedding
			uc.log.Infof("Generated embedding for product %d: %s", product.ID, product.Title)

			// Rate limiting
			time.Sleep(100 * time.Millisecond)
		}

		// Batch update embeddings
		if len(productEmbeddings) > 0 {
			if err := uc.repo.BatchUpdateEmbeddings(ctx, productEmbeddings); err != nil {
				uc.log.Errorf("Failed to batch update embeddings: %v", err)
			}
		}
	}

	uc.log.Info("Missing embeddings generation completed")
	return nil
}

// Helper function to build context from products
func (uc *ProductUsecase) buildContextFromProducts(products []*Product) string {
	var sb strings.Builder

	for i, product := range products {
		sb.WriteString(fmt.Sprintf("Product %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("PID: %s\n", product.PID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", product.Title))
		sb.WriteString(fmt.Sprintf("Brand: %s\n", product.Brand))
		sb.WriteString(fmt.Sprintf("Category: %s - %s\n", product.Category, product.SubCategory))
		sb.WriteString(fmt.Sprintf("Price: %s (Discounted from %s)\n", product.SellingPrice, product.ActualPrice))
		sb.WriteString(fmt.Sprintf("Rating: %s\n", product.AverageRating))
		if product.Description != "" {
			desc := product.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("Description: %s\n", desc))
		}
		sb.WriteString("---\n")
	}

	return sb.String()
}

// Check if embeddings are enabled
func (uc *ProductUsecase) embeddingsEnabled() bool {
	return uc.embedCfg != nil && uc.embedCfg.Enabled && uc.embedCfg.ApiKey != ""
}

// Cosine similarity calculation
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
