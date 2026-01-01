package service

import (
	"context"

	pb "yinni_backend/api/product/v1"
)

type ProductService struct {
	pb.UnimplementedProductServer
}

func NewProductService() *ProductService {
	return &ProductService{}
}

func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductInfo, error) {
    return &pb.ProductInfo{}, nil
}
func (s *ProductService) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsReply, error) {
    return &pb.ListProductsReply{}, nil
}
func (s *ProductService) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.ListProductsReply, error) {
    return &pb.ListProductsReply{}, nil
}
