package data

import (
	"context"
	"fmt"
	"strings"

	"yinni_backend/app/product/internal/biz"
	"yinni_backend/ent"
	"yinni_backend/ent/product"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	openai "github.com/sashabaranov/go-openai"
)

type productRepo struct {
	data     *Data
	log      *log.Helper
	aiClient *openai.Client
}

// NewProductRepo creates a new Product repository.
func NewProductRepo(data *Data, cfg *conf.Embeddings, logger log.Logger) biz.ProductRepo {
	var aiClient *openai.Client

	// Initialize AI client if embeddings are enabled
	if cfg != nil && cfg.ApiKey != "" {
		openaiConfig := openai.DefaultConfig(cfg.ApiKey)
		if cfg.BaseUrl != "" {
			openaiConfig.BaseURL = cfg.BaseUrl
		}
		aiClient = openai.NewClientWithConfig(openaiConfig)
	}

	return &productRepo{
		data:     data,
		aiClient: aiClient,
		log:      log.NewHelper(logger),
	}
}

// ========== BASIC CRUD OPERATIONS ==========

func (r *productRepo) Create(ctx context.Context, p *biz.Product) (*biz.Product, error) {
	builder := r.data.ent.Product.Create().
		SetTitle(p.Title).
		SetBrand(p.Brand).
		SetCategory(p.Category).
		SetSubCategory(p.SubCategory).
		SetOutOfStock(p.OutOfStock)

	// Optional fields
	if p.Description != "" {
		builder.SetDescription(p.Description)
	}
	if p.ActualPrice != "" {
		builder.SetActualPrice(p.ActualPrice)
	}
	if p.SellingPrice != "" {
		builder.SetSellingPrice(p.SellingPrice)
	}
	if p.Discount != "" {
		builder.SetDiscount(p.Discount)
	}
	if p.PriceNumeric > 0 {
		builder.SetPriceNumeric(p.PriceNumeric)
	}
	if p.Seller != "" {
		builder.SetSeller(p.Seller)
	}
	if p.AverageRating != "" {
		builder.SetAverageRating(p.AverageRating)
	}
	if p.RatingNumeric > 0 {
		builder.SetRatingNumeric(float64(p.RatingNumeric))
	}
	if len(p.Images) > 0 {
		builder.SetImages(p.Images)
	}
	if len(p.ProductDetails) > 0 {
		builder.SetProductDetails(p.ProductDetails)
	}
	if p.URL != "" {
		builder.SetURL(p.URL)
	}
	if p.PID != "" {
		builder.SetPid(p.PID)
	}
	if p.OriginalID != "" {
		builder.SetOriginalID(p.OriginalID)
	}
	if p.StyleCode != "" {
		builder.SetStyleCode(p.StyleCode)
	}
	if !p.CrawledAt.IsZero() {
		builder.SetCrawledAt(p.CrawledAt)
	}
	if len(p.Embedding) > 0 {
		builder.SetEmbedding(p.Embedding)
	}
	if len(p.SearchKeywords) > 0 {
		builder.SetSearchKeywords(p.SearchKeywords)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	return convertEntToBiz(row), nil
}

func (r *productRepo) Update(ctx context.Context, p *biz.Product) (*biz.Product, error) {
	builder := r.data.ent.Product.UpdateOneID(int(p.ID))

	// Update fields if they have values
	if p.Title != "" {
		builder.SetTitle(p.Title)
	}
	if p.Brand != "" {
		builder.SetBrand(p.Brand)
	}
	if p.Category != "" {
		builder.SetCategory(p.Category)
	}
	if p.SubCategory != "" {
		builder.SetSubCategory(p.SubCategory)
	}
	builder.SetOutOfStock(p.OutOfStock)

	if p.Description != "" {
		builder.SetDescription(p.Description)
	}
	if p.ActualPrice != "" {
		builder.SetActualPrice(p.ActualPrice)
	}
	if p.SellingPrice != "" {
		builder.SetSellingPrice(p.SellingPrice)
	}
	if p.Discount != "" {
		builder.SetDiscount(p.Discount)
	}
	if p.PriceNumeric > 0 {
		builder.SetPriceNumeric(p.PriceNumeric)
	}
	if p.Seller != "" {
		builder.SetSeller(p.Seller)
	}
	if p.AverageRating != "" {
		builder.SetAverageRating(p.AverageRating)
	}
	if p.RatingNumeric >= 0 {
		builder.SetRatingNumeric(float64(p.RatingNumeric))
	}
	if p.Images != nil {
		builder.SetImages(p.Images)
	}
	if p.ProductDetails != nil {
		builder.SetProductDetails(p.ProductDetails)
	}
	if p.URL != "" {
		builder.SetURL(p.URL)
	}
	if p.PID != "" {
		builder.SetPid(p.PID)
	}
	if p.Featured {
		builder.SetFeatured(p.Featured)
	}
	if p.Embedding != nil {
		builder.SetEmbedding(p.Embedding)
	}
	if p.SearchKeywords != nil {
		builder.SetSearchKeywords(p.SearchKeywords)
	}

	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	return convertEntToBiz(row), nil
}

func (r *productRepo) Delete(ctx context.Context, id int64) (*biz.Product, error) {
	// Get product before deleting
	p, err := r.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	err = r.data.ent.Product.DeleteOneID(int(id)).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (r *productRepo) GetProduct(ctx context.Context, id int64) (*biz.Product, error) {
	row, err := r.data.ent.Product.Get(ctx, int(id))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrProductNotFound
		}
		return nil, err
	}

	return convertEntToBiz(row), nil
}

func (r *productRepo) GetProductByPID(ctx context.Context, pid string) (*biz.Product, error) {
	row, err := r.data.ent.Product.
		Query().
		Where(product.Pid(pid)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrProductNotFound
		}
		return nil, err
	}

	return convertEntToBiz(row), nil
}

func (r *productRepo) ListAllProducts(ctx context.Context) ([]*biz.Product, error) {
	rows, err := r.data.ent.Product.
		Query().
		Order(ent.Desc(product.FieldCreateTime)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	rv := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, convertEntToBiz(row))
	}
	return rv, nil
}

func (r *productRepo) ListProducts(ctx context.Context, params *biz.ListProductsParams) ([]*biz.Product, int64, error) {
	query := r.data.ent.Product.Query()

	// Apply filters
	if params.Category != "" {
		query = query.Where(product.Category(params.Category))
	}
	if params.SubCategory != "" {
		query = query.Where(product.SubCategory(params.SubCategory))
	}
	if params.Brand != "" {
		query = query.Where(product.Brand(params.Brand))
	}
	if params.Seller != "" {
		query = query.Where(product.Seller(params.Seller))
	}
	if params.MinPrice > 0 {
		query = query.Where(product.PriceNumericGTE(int(params.MinPrice)))
	}
	if params.MaxPrice > 0 {
		query = query.Where(product.PriceNumericLTE(int(params.MaxPrice)))
	}
	if params.MinRating > 0 {
		query = query.Where(product.RatingNumericGTE(float64(params.MinRating)))
	}
	if params.InStock {
		query = query.Where(product.OutOfStock(false))
	}
	if params.Featured {
		query = query.Where(product.Featured(true))
	}

	// Apply search query if provided
	if params.SearchQuery != "" {
		// Simple text search (could be enhanced with full-text search)
		query = query.Where(
			product.Or(
				product.TitleContains(params.SearchQuery),
				product.DescriptionContains(params.SearchQuery),
				product.BrandContains(params.SearchQuery),
			),
		)
	}

	// Apply sorting
	switch strings.ToLower(params.SortBy) {
	case "price":
		if strings.ToLower(params.SortOrder) == "asc" {
			query = query.Order(ent.Asc(product.FieldPriceNumeric))
		} else {
			query = query.Order(ent.Desc(product.FieldPriceNumeric))
		}
	case "rating":
		if strings.ToLower(params.SortOrder) == "asc" {
			query = query.Order(ent.Asc(product.FieldRatingNumeric))
		} else {
			query = query.Order(ent.Desc(product.FieldRatingNumeric))
		}
	case "newest":
		query = query.Order(ent.Desc(product.FieldCreateTime))
	case "popular":
		query = query.Order(ent.Desc(product.FieldViewCount))
	default:
		query = query.Order(ent.Desc(product.FieldCreateTime))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (int(params.Page) - 1) * int(params.PageSize)
	rows, err := query.
		Offset(offset).
		Limit(int(params.PageSize)).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	rv := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, convertEntToBiz(row))
	}
	return rv, int64(total), nil
}

func (r *productRepo) SearchProducts(ctx context.Context, queryStr string, params *biz.ListProductsParams) ([]*biz.Product, int64, error) {
	query := r.data.ent.Product.Query()

	// Apply search filters
	if queryStr != "" {
		// Search in title, description, brand, and category
		query = query.Where(
			product.Or(
				product.TitleContainsFold(queryStr),
				product.DescriptionContainsFold(queryStr),
				product.BrandContainsFold(queryStr),
				product.CategoryContainsFold(queryStr),
				product.SubCategoryContainsFold(queryStr),
			),
		)
	}

	// Apply additional filters from params
	if params != nil {
		if params.Category != "" {
			query = query.Where(product.Category(params.Category))
		}
		if params.Brand != "" {
			query = query.Where(product.Brand(params.Brand))
		}
		if params.MinPrice > 0 {
			query = query.Where(product.PriceNumericGTE(int(params.MinPrice)))
		}
		if params.MaxPrice > 0 {
			query = query.Where(product.PriceNumericLTE(int(params.MaxPrice)))
		}
		if params.InStock {
			query = query.Where(product.OutOfStock(false))
		}
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	limit := 20 // default
	if params != nil && params.PageSize > 0 {
		limit = int(params.PageSize)
	}

	rows, err := query.
		Order(ent.Desc(product.FieldCreateTime)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	rv := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, convertEntToBiz(row))
	}
	return rv, int64(total), nil
}

func (r *productRepo) GetFeaturedProducts(ctx context.Context, limit int, category string) ([]*biz.Product, error) {
	query := r.data.ent.Product.
		Query().
		Where(product.Featured(true))

	if category != "" {
		query = query.Where(product.Category(category))
	}

	rows, err := query.
		Order(ent.Desc(product.FieldCreateTime)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, err
	}

	rv := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, convertEntToBiz(row))
	}
	return rv, nil
}

func (r *productRepo) GetSimilarProducts(ctx context.Context, id int64, limit int) ([]*biz.Product, error) {
	// First get the target product
	target, err := r.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	// Find products in same category and brand
	rows, err := r.data.ent.Product.
		Query().
		Where(
			product.And(
				product.Category(target.Category),
				product.Brand(target.Brand),
				product.IDNEQ(int(id)),
				product.OutOfStock(false),
			),
		).
		Order(ent.Desc(product.FieldRatingNumeric)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, err
	}

	rv := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		rv = append(rv, convertEntToBiz(row))
	}
	return rv, nil
}

func (r *productRepo) IncrementViewCount(ctx context.Context, id int64) error {
	// Get current view count
	current, err := r.data.ent.Product.
		Query().
		Where(product.ID(int(id))).
		Select(product.FieldViewCount).
		Int(ctx)
	if err != nil {
		return err
	}

	// Increment and update
	_, err = r.data.ent.Product.
		UpdateOneID(int(id)).
		SetViewCount(current + 1).
		Save(ctx)

	return err
}

func (r *productRepo) IncrementClickCount(ctx context.Context, id int64) error {
	// Get current click count
	current, err := r.data.ent.Product.
		Query().
		Where(product.ID(int(id))).
		Select(product.FieldClickCount).
		Int(ctx)
	if err != nil {
		return err
	}

	// Increment and update
	_, err = r.data.ent.Product.
		UpdateOneID(int(id)).
		SetClickCount(current + 1).
		Save(ctx)

	return err
}

// ========== EMBEDDING OPERATIONS ==========

// GenerateEmbedding generates embedding for a product or query
func (r *productRepo) GenerateEmbedding(ctx context.Context, p *biz.Product) ([]float32, error) {
	if r.aiClient == nil {
		return nil, biz.ErrEmbeddingsNotEnabled
	}

	// Generate text representation
	text := ""
	if p.Description != "" {
		// For products with description
		text = fmt.Sprintf("%s %s %s %s", p.Title, p.Brand, p.Category, p.Description)
	} else {
		// For simple queries
		text = fmt.Sprintf("%s %s", p.Title, p.Brand)
	}

	if len(text) > 8000 {
		text = text[:8000]
	}

	// Call OpenAI/DeepSeek API
	resp, err := r.aiClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.AdaEmbeddingV2,
		Input: []string{text},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

// SearchSimilarProducts searches products using vector similarity
func (r *productRepo) SearchSimilarProducts(ctx context.Context, queryEmbedding []float32, limit int, category string, priceRange *biz.PriceRange) ([]*biz.Product, error) {
	if r.aiClient == nil {
		return nil, biz.ErrEmbeddingsNotEnabled
	}

	// Get all products with embeddings
	query := r.data.ent.Product.Query().
		Where(product.EmbeddingNotNil()).
		Limit(1000) // Limit for in-memory similarity calculation

	if category != "" {
		query = query.Where(product.Category(category))
	}

	if priceRange != nil {
		if priceRange.Min > 0 {
			query = query.Where(product.PriceNumericGTE(int(priceRange.Min)))
		}
		if priceRange.Max > 0 {
			query = query.Where(product.PriceNumericLTE(int(priceRange.Max)))
		}
	}

	products, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate similarity scores
	type scoredProduct struct {
		product *biz.Product
		score   float32
	}

	scoredProducts := make([]scoredProduct, 0, len(products))

	for _, p := range products {
		if len(p.Embedding) == 0 || len(p.Embedding) != len(queryEmbedding) {
			continue
		}

		// Convert to float32 slice
		embedding := make([]float32, len(p.Embedding))
		for i, v := range p.Embedding {
			embedding[i] = float32(v)
		}

		score := biz.CosineSimilarity(embedding, queryEmbedding)
		if score > 0.3 { // Threshold
			scoredProducts = append(scoredProducts, scoredProduct{
				product: convertEntToBiz(p),
				score:   score,
			})
		}
	}

	// Sort by similarity score (descending)
	for i := 0; i < len(scoredProducts); i++ {
		for j := i + 1; j < len(scoredProducts); j++ {
			if scoredProducts[i].score < scoredProducts[j].score {
				scoredProducts[i], scoredProducts[j] = scoredProducts[j], scoredProducts[i]
			}
		}
	}

	// Return top results
	resultCount := limit
	if len(scoredProducts) < limit {
		resultCount = len(scoredProducts)
	}

	results := make([]*biz.Product, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = scoredProducts[i].product
	}

	return results, nil
}

// UpdateProductEmbedding updates embedding for a single product
func (r *productRepo) UpdateProductEmbedding(ctx context.Context, id int64, embedding []float32) error {
	_, err := r.data.ent.Product.
		UpdateOneID(int(id)).
		SetEmbedding(embedding).
		Save(ctx)

	return err
}

// BatchUpdateEmbeddings updates embeddings for multiple products
func (r *productRepo) BatchUpdateEmbeddings(ctx context.Context, productEmbeddings map[int64][]float32) error {
	for productID, embedding := range productEmbeddings {
		if err := r.UpdateProductEmbedding(ctx, productID, embedding); err != nil {
			r.log.Errorf("Failed to update embedding for product %d: %v", productID, err)
			continue
		}
	}

	return nil
}

// GetProductsWithoutEmbeddings returns products that don't have embeddings
func (r *productRepo) GetProductsWithoutEmbeddings(ctx context.Context, limit int) ([]*biz.Product, error) {
	rows, err := r.data.ent.Product.
		Query().
		Where(product.EmbeddingIsNil()).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, err
	}

	products := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, convertEntToBiz(row))
	}

	return products, nil
}

// GetProductsWithEmbeddings returns products that have embeddings
func (r *productRepo) GetProductsWithEmbeddings(ctx context.Context, limit int) ([]*biz.Product, error) {
	rows, err := r.data.ent.Product.
		Query().
		Where(product.EmbeddingNotNil()).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, err
	}

	products := make([]*biz.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, convertEntToBiz(row))
	}

	return products, nil
}

// Helper function to convert ent.Product to biz.Product
func convertEntToBiz(p *ent.Product) *biz.Product {
	if p == nil {
		return nil
	}

	// Convert embedding from []float64 to []float32
	var embedding []float32
	if p.Embedding != nil {
		embedding = make([]float32, len(p.Embedding))
		for i, v := range p.Embedding {
			embedding[i] = float32(v)
		}
	}

	return &biz.Product{
		ID:             int64(p.ID),
		OriginalID:     p.OriginalID,
		Title:          p.Title,
		Brand:          p.Brand,
		Description:    p.Description,
		ActualPrice:    p.ActualPrice,
		SellingPrice:   p.SellingPrice,
		Discount:       p.Discount,
		PriceNumeric:   p.PriceNumeric,
		Category:       p.Category,
		SubCategory:    p.SubCategory,
		OutOfStock:     p.OutOfStock,
		Seller:         p.Seller,
		AverageRating:  p.AverageRating,
		RatingNumeric:  float32(p.RatingNumeric),
		Images:         p.Images,
		ProductDetails: p.ProductDetails,
		URL:            p.URL,
		PID:            p.Pid,
		StyleCode:      p.StyleCode,
		CrawledAt:      p.CrawledAt,
		CreatedAt:      p.CreateTime,
		UpdatedAt:      p.UpdateTime,
		ViewCount:      p.ViewCount,
		ClickCount:     p.ClickCount,
		Featured:       p.Featured,
		Embedding:      embedding,
		SearchKeywords: p.SearchKeywords,
	}
}
