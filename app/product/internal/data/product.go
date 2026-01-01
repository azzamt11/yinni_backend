package data

import (
	"context"
	"strings"

	"yinni_backend/app/product/internal/biz"
	"yinni_backend/ent"
	"yinni_backend/ent/product"

	"github.com/go-kratos/kratos/v2/log"
)

type productRepo struct {
	data *Data
	log  *log.Helper
}

// NewProductRepo creates a new Product repository.
func NewProductRepo(data *Data, logger log.Logger) biz.ProductRepo {
	return &productRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

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

// Helper function to convert ent.Product to biz.Product
func convertEntToBiz(p *ent.Product) *biz.Product {
	if p == nil {
		return nil
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
		Embedding:      p.Embedding,
		SearchKeywords: p.SearchKeywords,
	}
}
