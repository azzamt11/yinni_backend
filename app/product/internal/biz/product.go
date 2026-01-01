package biz

import (
	"context"
	"time"

	v1 "yinni_backend/api/product/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrProductNotFound   = errors.NotFound(v1.ErrorReason_PRODUCT_NOT_FOUND.String(), "product not found")
	ErrInvalidProductID  = errors.BadRequest(v1.ErrorReason_INVALID_PRODUCT_ID.String(), "invalid product id")
	ErrInvalidPriceRange = errors.BadRequest(v1.ErrorReason_INVALID_PRICE_RANGE.String(), "invalid price range")
	ErrInvalidParameters = errors.BadRequest(v1.ErrorReason_INVALID_PARAMETERS.String(), "invalid parameters")
	ErrDatabaseError     = errors.InternalServer(v1.ErrorReason_DATABASE_ERROR.String(), "database error")
	ErrSearchFailed      = errors.InternalServer(v1.ErrorReason_SEARCH_FAILED.String(), "search failed")
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

// ProductUsecase is a Product usecase.
type ProductUsecase struct {
	repo ProductRepo
	log  *log.Helper
}

// NewProductUsecase creates a new ProductUsecase.
func NewProductUsecase(repo ProductRepo, logger log.Logger) *ProductUsecase {
	return &ProductUsecase{repo: repo, log: log.NewHelper(logger)}
}

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
