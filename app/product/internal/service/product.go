package service

import (
	"context"
	"strconv"
	"strings"

	pb "yinni_backend/api/product/v1"
	"yinni_backend/app/product/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductService struct {
	pb.UnimplementedProductServer
	uc  *biz.ProductUsecase
	log *log.Helper
}

func NewProductService(uc *biz.ProductUsecase, logger log.Logger) *ProductService {
	return &ProductService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductInfo, error) {
	s.log.WithContext(ctx).Infof("GetProduct called with id: %d", req.Id)

	product, err := s.uc.GetProduct(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetProduct failed: %v", err)
		return nil, err
	}

	return s.convertToProductInfo(product), nil
}

func (s *ProductService) GetProductByPID(ctx context.Context, req *pb.GetProductByPIDRequest) (*pb.ProductInfo, error) {
	s.log.WithContext(ctx).Infof("GetProductByPID called with pid: %s", req.Pid)

	product, err := s.uc.GetProductByPID(ctx, req.Pid)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetProductByPID failed: %v", err)
		return nil, err
	}

	return s.convertToProductInfo(product), nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsReply, error) {
	s.log.WithContext(ctx).Infof("ListProducts called: page=%d, pageSize=%d", req.Page, req.PageSize)

	params := &biz.ListProductsParams{
		Page:        req.Page,
		PageSize:    req.PageSize,
		Category:    req.Category,
		Brand:       req.Brand,
		SubCategory: req.SubCategory,
		MinPrice:    req.MinPrice,
		MaxPrice:    req.MaxPrice,
		MinRating:   req.MinRating,
		InStock:     req.InStock,
		Featured:    req.FeaturedOnly,
		Seller:      req.Seller,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
		SearchQuery: req.SearchQuery,
	}

	products, total, err := s.uc.ListProducts(ctx, params)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListProducts failed: %v", err)
		return nil, err
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *ProductService) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.ListProductsReply, error) {
	s.log.WithContext(ctx).Infof("SearchProducts called: query=%s, limit=%d", req.Query, req.Limit)

	params := &biz.ListProductsParams{
		PageSize:    req.Limit,
		Category:    req.Category,
		SearchQuery: req.Query,
	}

	// Use PriceRange if provided
	if req.PriceRange != nil {
		params.MinPrice = req.PriceRange.Min
		params.MaxPrice = req.PriceRange.Max
	}

	products, total, err := s.uc.SearchProducts(ctx, req.Query, params)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SearchProducts failed: %v", err)
		return nil, err
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(total),
		PageSize: req.Limit,
	}, nil
}

func (s *ProductService) GetFeaturedProducts(ctx context.Context, req *pb.GetFeaturedProductsRequest) (*pb.ListProductsReply, error) {
	s.log.WithContext(ctx).Infof("GetFeaturedProducts called: limit=%d, category=%s", req.Limit, req.Category)

	products, err := s.uc.GetFeaturedProducts(ctx, int(req.Limit), req.Category)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetFeaturedProducts failed: %v", err)
		return nil, err
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(len(products)),
	}, nil
}

func (s *ProductService) GetSimilarProducts(ctx context.Context, req *pb.GetSimilarProductsRequest) (*pb.ListProductsReply, error) {
	s.log.WithContext(ctx).Infof("GetSimilarProducts called: id=%d, limit=%d", req.Id, req.Limit)

	products, err := s.uc.GetSimilarProducts(ctx, req.Id, int(req.Limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetSimilarProducts failed: %v", err)
		return nil, err
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(len(products)),
	}, nil
}

// Helper methods for conversion

func (s *ProductService) convertToProductInfo(p *biz.Product) *pb.ProductInfo {
	if p == nil {
		return nil
	}

	// Convert product details to map
	productDetails := make(map[string]string)
	for _, detail := range p.ProductDetails {
		for k, v := range detail {
			productDetails[k] = v
		}
	}

	// Get primary image (first image)
	primaryImage := ""
	if len(p.Images) > 0 {
		primaryImage = p.Images[0]
	}

	// Calculate discount percentage
	discountPct := s.calculateDiscountPercentage(p.ActualPrice, p.SellingPrice)

	return &pb.ProductInfo{
		Id:                 p.ID,
		OriginalId:         p.OriginalID,
		Title:              p.Title,
		Brand:              p.Brand,
		Description:        p.Description,
		ActualPrice:        p.ActualPrice,
		SellingPrice:       p.SellingPrice,
		Discount:           p.Discount,
		DiscountPercentage: float32(discountPct),
		PriceNumeric:       int32(p.PriceNumeric),
		Category:           p.Category,
		SubCategory:        p.SubCategory,
		OutOfStock:         p.OutOfStock,
		Seller:             p.Seller,
		AverageRating:      p.AverageRating,
		RatingNumeric:      float32(p.RatingNumeric),
		Images:             p.Images,
		PrimaryImage:       primaryImage,
		ProductDetails:     productDetails,
		Url:                p.URL,
		Pid:                p.PID,
		StyleCode:          p.StyleCode,
		CrawledAt:          timestamppb.New(p.CrawledAt),
		CreatedAt:          timestamppb.New(p.CreatedAt),
		UpdatedAt:          timestamppb.New(p.UpdatedAt),
		ViewCount:          int32(p.ViewCount),
		ClickCount:         int32(p.ClickCount),
		Featured:           p.Featured,
	}
}

func (s *ProductService) convertToProductList(products []*biz.Product) []*pb.ProductInfo {
	result := make([]*pb.ProductInfo, len(products))
	for i, p := range products {
		result[i] = s.convertToProductInfo(p)
	}
	return result
}

func (s *ProductService) calculateDiscountPercentage(actualPrice, sellingPrice string) float64 {
	if actualPrice == "" || sellingPrice == "" {
		return 0
	}

	// Parse prices - remove currency symbols and commas
	actual := s.parsePrice(actualPrice)
	selling := s.parsePrice(sellingPrice)

	if actual <= 0 || selling <= 0 || selling >= actual {
		return 0
	}

	discount := float64(actual-selling) / float64(actual) * 100
	return discount
}

func (s *ProductService) parsePrice(priceStr string) int {
	// Remove currency symbols, commas, and spaces
	cleaned := strings.ReplaceAll(priceStr, "₹", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")

	// Try to parse as integer
	if price, err := strconv.Atoi(cleaned); err == nil {
		return price
	}

	return 0
}
