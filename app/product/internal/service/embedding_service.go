package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"yinni_backend/app/product/internal/biz"
	"yinni_backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	openai "github.com/sashabaranov/go-openai"
)

type EmbeddingService struct {
	client    *openai.Client
	log       *log.Helper
	productUC *biz.ProductUsecase
	conf      *conf.Embeddings
}

func NewEmbeddingService(conf *conf.Embeddings, productUC *biz.ProductUsecase, logger log.Logger) *EmbeddingService {
	var client *openai.Client

	if conf != nil && conf.ApiKey != "" {
		openaiConfig := openai.DefaultConfig(conf.ApiKey)
		if conf.BaseUrl != "" {
			openaiConfig.BaseURL = conf.BaseUrl
		}
		client = openai.NewClientWithConfig(openaiConfig)
	}

	return &EmbeddingService{
		client:    client,
		productUC: productUC,
		log:       log.NewHelper(logger),
		conf:      conf,
	}
}

// Generate product text for embedding
func (s *EmbeddingService) generateProductText(product *biz.Product) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Title: %s\n", product.Title))
	sb.WriteString(fmt.Sprintf("Brand: %s\n", product.Brand))
	sb.WriteString(fmt.Sprintf("Category: %s - %s\n", product.Category, product.SubCategory))

	if product.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", product.Description))
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
	sb.WriteString(fmt.Sprintf("Seller: %s\n", product.Seller))

	return sb.String()
}

// Generate embedding for a product
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, product *biz.Product) ([]float32, error) {
	if s.client == nil {
		return nil, fmt.Errorf("embeddings service not configured")
	}

	text := s.generateProductText(product)

	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: s.getModel(),
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

// Helper to get the model
func (s *EmbeddingService) getModel() openai.EmbeddingModel {
	if s.conf != nil && s.conf.Model != "" {
		return openai.EmbeddingModel(s.conf.Model)
	}
	return openai.AdaEmbeddingV2
}

// Generate embeddings for all products
func (s *EmbeddingService) GenerateAllEmbeddings(ctx context.Context, batchSize int) error {
	if s.client == nil {
		return fmt.Errorf("embeddings service not configured")
	}

	page := int32(1)
	pageSize := int32(batchSize)

	for {
		params := &biz.ListProductsParams{
			Page:     page,
			PageSize: pageSize,
		}

		products, _, err := s.productUC.ListProducts(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to list products: %w", err)
		}

		for _, product := range products {
			// Skip if already has embedding
			if len(product.Embedding) > 0 {
				continue
			}

			embedding, err := s.GenerateEmbedding(ctx, product)
			if err != nil {
				s.log.Errorf("Failed to generate embedding for product %d: %v", product.ID, err)
				continue
			}

			// Update product with embedding
			product.Embedding = embedding
			if _, err := s.productUC.UpdateProduct(ctx, product); err != nil {
				s.log.Errorf("Failed to update product with embedding %d: %v", product.ID, err)
			}

			s.log.Infof("Generated embedding for product %d: %s", product.ID, product.Title)

			// Rate limiting
			time.Sleep(100 * time.Millisecond)
		}

		if int64(len(products)) < int64(pageSize) {
			break
		}

		page++
	}

	return nil
}

// PriceRange struct for search
type PriceRange struct {
	Min int32
	Max int32
}

// Search products using vector similarity
func (s *EmbeddingService) SearchSimilarProducts(ctx context.Context, query string, limit int, category string, priceRange *PriceRange) ([]*biz.Product, error) {
	if s.client == nil {
		return nil, fmt.Errorf("embeddings service not configured")
	}

	// Generate embedding for the query
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: s.getModel(),
		Input: []string{query},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned for query")
	}

	queryEmbedding := resp.Data[0].Embedding

	// Get all products and calculate similarity
	params := &biz.ListProductsParams{
		Page:     1,
		PageSize: 1000, // Get a large batch for similarity calculation
		Category: category,
	}

	products, _, err := s.productUC.ListProducts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	// Filter and score products
	var scoredProducts []struct {
		product *biz.Product
		score   float32
	}

	for _, product := range products {
		// Apply price filter
		if priceRange != nil {
			if priceRange.Min > 0 && int32(product.PriceNumeric) < priceRange.Min {
				continue
			}
			if priceRange.Max > 0 && int32(product.PriceNumeric) > priceRange.Max {
				continue
			}
		}

		// Skip products without embeddings
		if len(product.Embedding) == 0 {
			continue
		}

		// Calculate similarity
		score := cosineSimilarity(product.Embedding, queryEmbedding)
		if score > 0.3 { // Threshold
			scoredProducts = append(scoredProducts, struct {
				product *biz.Product
				score   float32
			}{product, score})
		}
	}

	// Simple sort by score (for production, use proper sorting)
	// Sort in descending order of similarity
	for i := 0; i < len(scoredProducts); i++ {
		for j := i + 1; j < len(scoredProducts); j++ {
			if scoredProducts[i].score < scoredProducts[j].score {
				scoredProducts[i], scoredProducts[j] = scoredProducts[j], scoredProducts[i]
			}
		}
	}

	// Return top results
	var results []*biz.Product
	for i := 0; i < min(limit, len(scoredProducts)); i++ {
		results = append(results, scoredProducts[i].product)
	}

	return results, nil
}

// RAG-based search with DeepSeek
func (s *EmbeddingService) RAGSearch(ctx context.Context, prompt string, limit int) ([]*biz.Product, error) {
	if s.client == nil {
		return nil, fmt.Errorf("embeddings service not configured")
	}

	// First, find similar products based on the prompt
	products, err := s.SearchSimilarProducts(ctx, prompt, limit*2, "", nil)
	if err != nil {
		return nil, err
	}

	if len(products) == 0 {
		return products, nil
	}

	// Prepare context for LLM
	contextText := s.buildContextFromProducts(products[:min(5, len(products))])

	// Query LLM to refine results
	systemPrompt := `You are an e-commerce product search assistant. Given a user query and product context, 
	return a JSON array of product IDs that best match the query. Consider:
	1. Relevance to user intent
	2. Product quality and rating
	3. Value for money
	4. Availability
	
	Return only JSON array like: ["pid1", "pid2", "pid3"]`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("User query: %s\n\nAvailable products:\n%s\n\nReturn top %d relevant product PIDs:", prompt, contextText, limit),
		},
	}

	completion, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   500,
	})

	if err != nil {
		// If LLM fails, return the vector search results
		s.log.Errorf("LLM completion failed: %v, returning vector search results", err)
		return products[:min(limit, len(products))], nil
	}

	// Parse LLM response
	var pids []string
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &pids); err != nil {
		s.log.Errorf("Failed to parse LLM response: %v, returning vector search results", err)
		return products[:min(limit, len(products))], nil
	}

	// Fetch final products by PID
	var finalProducts []*biz.Product
	for _, pid := range pids {
		if len(finalProducts) >= limit {
			break
		}

		product, err := s.productUC.GetProductByPID(ctx, pid)
		if err == nil && product != nil {
			finalProducts = append(finalProducts, product)
		}
	}

	return finalProducts, nil
}

func (s *EmbeddingService) buildContextFromProducts(products []*biz.Product) string {
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

// Helper function for cosine similarity
func cosineSimilarity(a, b []float32) float32 {
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

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// Simple square root implementation
func sqrt(x float32) float32 {
	// Using float64 for better precision
	return float32(math.Sqrt(float64(x)))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
