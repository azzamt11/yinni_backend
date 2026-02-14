package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "yinni_backend/api/product/v1"
	"yinni_backend/app/product/internal/biz"
	"yinni_backend/pkg/middleware"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductService struct {
	pb.UnimplementedProductServer
	uc         *biz.ProductUsecase
	log        *log.Helper
	jobMgr     *EmbeddingJobManager
	seedJobMgr *ProductSeedJobManager
}

// EmbeddingJobManager manages embedding generation jobs
type EmbeddingJobManager struct {
	mu        sync.RWMutex
	activeJob *EmbeddingJob
	log       *log.Helper
	uc        *biz.ProductUsecase
}

// EmbeddingJob represents an embedding generation job
type EmbeddingJob struct {
	ID          string
	Status      string // "pending", "running", "completed", "failed", "cancelled"
	Processed   int
	Total       int
	Error       string
	StartedAt   time.Time
	LastUpdated time.Time
	Request     *pb.GenerateEmbeddingsRequest
	CreatedBy   int64 // User ID who started the job
}

type ProductSeedJobManager struct {
	mu        sync.RWMutex
	activeJob *ProductSeedJob
	log       *log.Helper
	uc        *biz.ProductUsecase
}

type ProductSeedJob struct {
	ID          string
	Status      string // "pending", "running", "completed", "failed", "cancelled"
	Processed   int
	Total       int
	Seeded      int
	Skipped     int
	Error       string
	StartedAt   time.Time
	LastUpdated time.Time
	Request     *pb.StartProductSeedJobRequest
	CreatedBy   int64
}

func NewProductService(uc *biz.ProductUsecase, logger log.Logger) *ProductService {
	return &ProductService{
		uc:  uc,
		log: log.NewHelper(logger),
		jobMgr: &EmbeddingJobManager{
			log: log.NewHelper(log.With(logger, "module", "embedding-job")),
			uc:  uc,
		},
		seedJobMgr: &ProductSeedJobManager{
			log: log.NewHelper(log.With(logger, "module", "product-seed-job")),
			uc:  uc,
		},
	}
}

// ===================== EXISTING METHODS =====================

func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductInfo, error) {
	s.log.WithContext(ctx).Infof("GetProduct called with id: %d", req.Id)

	product, err := s.uc.GetProduct(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetProduct failed: %v", err)
		return nil, err
	}

	return s.convertToProductInfo(product)
}

func (s *ProductService) GetProductByPID(ctx context.Context, req *pb.GetProductByPIDRequest) (*pb.ProductInfo, error) {
	s.log.WithContext(ctx).Infof("GetProductByPID called with pid: %s", req.Pid)

	product, err := s.uc.GetProductByPID(ctx, req.Pid)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetProductByPID failed: %v", err)
		return nil, err
	}

	return s.convertToProductInfo(product)
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

	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var products []*biz.Product
	var err error

	// Convert PriceRange
	var priceRange *biz.PriceRange
	if req.PriceRange != nil {
		priceRange = &biz.PriceRange{
			Min: req.PriceRange.Min,
			Max: req.PriceRange.Max,
		}
	}

	// Try RAG search if embeddings are enabled
	products, err = s.uc.RAGSearch(ctx, req.Query, limit, req.Category, priceRange)
	if err != nil {
		s.log.WithContext(ctx).Warnf("RAG search failed: %v, falling back to traditional search", err)
		// Fallback to traditional search
		params := &biz.ListProductsParams{
			PageSize:    int32(limit),
			SearchQuery: req.Query,
		}

		if priceRange != nil {
			params.MinPrice = priceRange.Min
			params.MaxPrice = priceRange.Max
		}

		products, _, err = s.uc.SearchProducts(ctx, req.Query, params)
		if err != nil {
			s.log.WithContext(ctx).Errorf("Traditional search failed: %v", err)
			return nil, err
		}
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(len(products)),
		PageSize: req.Limit,
	}, nil
}

func (s *ProductService) GetFeaturedProducts(ctx context.Context, req *pb.GetFeaturedProductsRequest) (*pb.ListProductsReply, error) {
	s.log.WithContext(ctx).Infof("GetFeaturedProducts called: limit=%d, category=%s", req.Limit, req.Category)

	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	products, err := s.uc.GetFeaturedProducts(ctx, limit, req.Category)
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

	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	products, err := s.uc.GetSimilarProducts(ctx, req.Id, limit)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetSimilarProducts failed: %v", err)
		return nil, err
	}

	return &pb.ListProductsReply{
		Products: s.convertToProductList(products),
		Total:    int32(len(products)),
	}, nil
}

// ===================== ADMIN EMBEDDING METHODS =====================

// GenerateEmbeddings - Admin only endpoint to generate embeddings
func (s *ProductService) GenerateEmbeddings(ctx context.Context, req *pb.GenerateEmbeddingsRequest) (*pb.GenerateEmbeddingsResponse, error) {
	// Get user info from context (set by middleware)
	claims, err := middleware.ExtractClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to extract user claims: %v", err)
		return nil, errors.Unauthorized("UNAUTHORIZED", "authentication required")
	}

	s.log.WithContext(ctx).Infof("Admin %d generating embeddings: regenerate_all=%v, batch_size=%d, product_ids=%d",
		claims.UserID, req.RegenerateAll, req.BatchSize, len(req.ProductIds))

	// Start embedding generation
	job, err := s.jobMgr.StartJob(ctx, req, claims.UserID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to start embedding job: %v", err)
		return nil, err
	}

	// Estimate time (rough estimate: 2 seconds per product)
	estimatedTime := int32(job.Total * 2)

	return &pb.GenerateEmbeddingsResponse{
		JobId:                job.ID,
		Status:               job.Status,
		EstimatedTimeSeconds: estimatedTime,
		TotalProducts:        int32(job.Total),
	}, nil
}

// GetEmbeddingStatus - Admin only endpoint to check embedding generation status
func (s *ProductService) GetEmbeddingStatus(ctx context.Context, req *pb.GetEmbeddingStatusRequest) (*pb.GetEmbeddingStatusResponse, error) {
	s.log.WithContext(ctx).Infof("GetEmbeddingStatus called: job_id=%s", req.JobId)

	job := s.jobMgr.GetJob(req.JobId)
	if job == nil {
		return nil, errors.NotFound("EMBEDDING_JOB_NOT_FOUND", "embedding job not found")
	}

	progress := float32(0)
	if job.Total > 0 {
		progress = float32(job.Processed) / float32(job.Total)
	}

	// Estimate completion time
	var estimatedCompletion *timestamppb.Timestamp
	if job.Status == "running" && job.Processed > 0 {
		elapsed := time.Since(job.StartedAt).Seconds()
		rate := float64(job.Processed) / elapsed
		if rate > 0 {
			remaining := float64(job.Total-job.Processed) / rate
			estTime := time.Now().Add(time.Duration(remaining) * time.Second)
			estimatedCompletion = timestamppb.New(estTime)
		}
	}

	return &pb.GetEmbeddingStatusResponse{
		JobId:               job.ID,
		Status:              job.Status,
		Processed:           int32(job.Processed),
		Total:               int32(job.Total),
		Progress:            progress,
		ErrorMessage:        job.Error,
		StartedAt:           timestamppb.New(job.StartedAt),
		LastUpdated:         timestamppb.New(job.LastUpdated),
		EstimatedCompletion: estimatedCompletion,
	}, nil
}

// CancelEmbeddingGeneration - Admin only endpoint to cancel embedding generation
func (s *ProductService) CancelEmbeddingGeneration(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.log.WithContext(ctx).Info("CancelEmbeddingGeneration called")

	if err := s.jobMgr.CancelJob(); err != nil {
		s.log.WithContext(ctx).Errorf("Failed to cancel embedding job: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ProductService) StartProductSeedJob(ctx context.Context, req *pb.StartProductSeedJobRequest) (*pb.StartProductSeedJobResponse, error) {
	claims, err := middleware.ExtractClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to extract user claims: %v", err)
		return nil, errors.Unauthorized("UNAUTHORIZED", "authentication required")
	}

	s.log.WithContext(ctx).Infof("Admin %d requested product seed job: clear_existing=%v, max_products=%d, batch_size=%d",
		claims.UserID, req.ClearExisting, req.MaxProducts, req.BatchSize)

	job, err := s.seedJobMgr.StartJob(req, claims.UserID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to start product seed job: %v", err)
		return nil, err
	}

	estimated := int32(job.Total / 75)
	if estimated < 60 {
		estimated = 60
	}

	return &pb.StartProductSeedJobResponse{
		JobId:                job.ID,
		Status:               job.Status,
		TotalProducts:        int32(job.Total),
		EstimatedTimeSeconds: estimated,
	}, nil
}

func (s *ProductService) GetProductSeedJobStatus(ctx context.Context, req *pb.GetProductSeedJobStatusRequest) (*pb.GetProductSeedJobStatusResponse, error) {
	s.log.WithContext(ctx).Infof("GetProductSeedJobStatus called: job_id=%s", req.JobId)

	job := s.seedJobMgr.GetJob(req.JobId)
	if job == nil {
		s.log.WithContext(ctx).Warnf("GetProductSeedJobStatus: job not found for job_id=%s", req.JobId)
		return nil, errors.NotFound("PRODUCT_SEED_JOB_NOT_FOUND", "product seed job not found")
	}

	progress := float32(0)
	if job.Total > 0 {
		progress = float32(job.Processed) / float32(job.Total)
	}

	var estimatedCompletion *timestamppb.Timestamp
	if job.Status == "running" && job.Processed > 0 {
		elapsed := time.Since(job.StartedAt).Seconds()
		rate := float64(job.Processed) / elapsed
		if rate > 0 {
			remaining := float64(job.Total-job.Processed) / rate
			estTime := time.Now().Add(time.Duration(remaining) * time.Second)
			estimatedCompletion = timestamppb.New(estTime)
		}
	}

	return &pb.GetProductSeedJobStatusResponse{
		JobId:               job.ID,
		Status:              job.Status,
		Processed:           int32(job.Processed),
		Total:               int32(job.Total),
		Progress:            progress,
		Seeded:              int32(job.Seeded),
		Skipped:             int32(job.Skipped),
		ErrorMessage:        job.Error,
		StartedAt:           timestamppb.New(job.StartedAt),
		LastUpdated:         timestamppb.New(job.LastUpdated),
		EstimatedCompletion: estimatedCompletion,
	}, nil
}

// ===================== EMBEDDING JOB MANAGER METHODS =====================

// StartJob starts a new embedding generation job
func (m *EmbeddingJobManager) StartJob(ctx context.Context, req *pb.GenerateEmbeddingsRequest, userID int64) (*EmbeddingJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job is already running
	if m.activeJob != nil && (m.activeJob.Status == "running" || m.activeJob.Status == "pending") {
		return nil, errors.Conflict("EMBEDDING_JOB_ALREADY_RUNNING", "embedding generation already in progress")
	}

	// Generate job ID
	jobID := generateJobID()

	// Get total product count
	total := 0
	if req.RegenerateAll {
		// Count all products
		count, err := m.uc.CountProducts(ctx)
		if err != nil {
			return nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("failed to count products: %v", err))
		}
		total = count
	} else if len(req.ProductIds) > 0 {
		total = len(req.ProductIds)
	} else {
		// Count products without embeddings
		count, err := m.uc.CountProductsWithoutEmbeddings(ctx)
		if err != nil {
			return nil, errors.InternalServer("DATABASE_ERROR", fmt.Sprintf("failed to count products without embeddings: %v", err))
		}
		total = count
	}

	if total == 0 {
		return nil, errors.BadRequest("INVALID_PARAMETERS", "no products to process")
	}

	job := &EmbeddingJob{
		ID:          jobID,
		Status:      "pending",
		Total:       total,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
		Request:     req,
		CreatedBy:   userID,
	}

	m.activeJob = job
	m.log.Infof("Embedding job %s created for %d products (user: %d)", job.ID, job.Total, userID)

	// Start processing in background
	go m.processJob(ctx, job)

	return job, nil
}

// // processJob processes the embedding generation job in the background
// func (m *EmbeddingJobManager) processJob(ctx context.Context, job *EmbeddingJob) {
// 	m.mu.Lock()
// 	job.Status = "running"
// 	job.LastUpdated = time.Now()
// 	m.mu.Unlock()

// 	m.log.Infof("Starting embedding job %s for %d products", job.ID, job.Total)

// 	batchSize := 50
// 	if job.Request.BatchSize > 0 {
// 		batchSize = int(job.Request.BatchSize)
// 	}

// 	var err error
// 	if job.Request.RegenerateAll {
// 		err = m.uc.GenerateAllEmbeddings(ctx, batchSize, func(processed int) {
// 			m.mu.Lock()
// 			job.Processed = processed
// 			job.LastUpdated = time.Now()
// 			m.mu.Unlock()
// 		})
// 	} else if len(job.Request.ProductIds) > 0 {
// 		err = m.uc.GenerateEmbeddingsForProducts(ctx, job.Request.ProductIds)
// 		if err == nil {
// 			m.mu.Lock()
// 			job.Processed = job.Total
// 			m.mu.Unlock()
// 		}
// 	} else {
// 		err = m.uc.GenerateAllEmbeddings(ctx, batchSize, func(processed int) {
// 			m.mu.Lock()
// 			job.Processed = processed
// 			job.LastUpdated = time.Now()
// 			m.mu.Unlock()
// 		})
// 	}

// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if err != nil {
// 		job.Status = "failed"
// 		job.Error = err.Error()
// 		m.log.Errorf("Embedding job %s failed: %v", job.ID, err)
// 	} else {
// 		job.Status = "completed"
// 		job.Processed = job.Total
// 		m.log.Infof("Embedding job %s completed successfully", job.ID)
// 	}
// 	job.LastUpdated = time.Now()
// }

// processJob processes the embedding generation job in the background
func (m *EmbeddingJobManager) processJob(ctx context.Context, job *EmbeddingJob) {
	m.log.Infof("Process job started, original context err: %v", ctx.Err())

	m.mu.Lock()
	job.Status = "running"
	job.LastUpdated = time.Now()
	m.mu.Unlock()

	m.log.Infof("Starting embedding job %s for %d products", job.ID, job.Total)

	batchSize := 50
	if job.Request.BatchSize > 0 {
		batchSize = int(job.Request.BatchSize)
	}

	// Create a new background context (not tied to HTTP request)
	bgCtx := context.Background()

	// Optional: Add timeout (e.g., 2 hours for large batches)
	bgCtx, cancel := context.WithTimeout(bgCtx, 2*time.Hour)
	defer cancel()

	var err error
	if job.Request.RegenerateAll {
		err = m.uc.GenerateAllEmbeddings(bgCtx, batchSize, func(processed int) {
			m.mu.Lock()
			job.Processed = processed
			job.LastUpdated = time.Now()
			m.mu.Unlock()
		})
	} else if len(job.Request.ProductIds) > 0 {
		err = m.uc.GenerateEmbeddingsForProducts(bgCtx, job.Request.ProductIds)
		if err == nil {
			m.mu.Lock()
			job.Processed = job.Total
			m.mu.Unlock()
		}
	} else {
		err = m.uc.GenerateAllEmbeddings(bgCtx, batchSize, func(processed int) {
			m.mu.Lock()
			job.Processed = processed
			job.LastUpdated = time.Now()
			m.mu.Unlock()
		})
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		m.log.Errorf("Embedding job %s failed: %v", job.ID, err)
	} else {
		job.Status = "completed"
		job.Processed = job.Total
		m.log.Infof("Embedding job %s completed successfully", job.ID)
	}
	job.LastUpdated = time.Now()
}

// GetJob returns a job by ID
func (m *EmbeddingJobManager) GetJob(jobID string) *EmbeddingJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeJob != nil && m.activeJob.ID == jobID {
		return m.activeJob
	}
	return nil
}

// CancelJob cancels the active embedding job
func (m *EmbeddingJobManager) CancelJob() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeJob == nil || (m.activeJob.Status != "running" && m.activeJob.Status != "pending") {
		return errors.BadRequest("EMBEDDING_JOB_NOT_RUNNING", "no active embedding job to cancel")
	}

	m.activeJob.Status = "cancelled"
	m.activeJob.LastUpdated = time.Now()
	m.log.Infof("Embedding job %s cancelled", m.activeJob.ID)

	return nil
}

func (m *ProductSeedJobManager) StartJob(req *pb.StartProductSeedJobRequest, userID int64) (*ProductSeedJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeJob != nil && (m.activeJob.Status == "running" || m.activeJob.Status == "pending") {
		return nil, errors.Conflict("PRODUCT_SEED_JOB_ALREADY_RUNNING", "product seeding already in progress")
	}

	m.log.Infof("Loading dataset for seed job request: clear_existing=%v max_products=%d batch_size=%d user=%d",
		req.GetClearExisting(), req.GetMaxProducts(), req.GetBatchSize(), userID)
	dataset, err := loadProductDataset(m.log)
	if err != nil {
		m.log.Errorf("Failed to load dataset for seed job: %v", err)
		return nil, errors.InternalServer("DATASET_LOAD_FAILED", fmt.Sprintf("failed to load dataset: %v", err))
	}

	originalLen := len(dataset)
	if req.MaxProducts > 0 && int(req.MaxProducts) < len(dataset) {
		dataset = dataset[:req.MaxProducts]
	}
	m.log.Infof("Dataset loaded for seed job: original=%d effective=%d", originalLen, len(dataset))

	job := &ProductSeedJob{
		ID:          generateSeedJobID(),
		Status:      "pending",
		Total:       len(dataset),
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
		Request:     req,
		CreatedBy:   userID,
	}

	m.activeJob = job
	m.log.Infof("Product seed job %s created for %d products (user: %d)", job.ID, job.Total, userID)

	go m.processJob(job, dataset)
	return job, nil
}

func (m *ProductSeedJobManager) processJob(job *ProductSeedJob, dataset []map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			m.mu.Lock()
			job.Status = "failed"
			job.Error = fmt.Sprintf("panic in seed job: %v", r)
			job.LastUpdated = time.Now()
			m.mu.Unlock()
			m.log.Errorf("Product seed job %s panicked: %v\n%s", job.ID, r, string(debug.Stack()))
		}
	}()

	m.log.Infof("Product seed job %s started: total_dataset_rows=%d", job.ID, len(dataset))

	m.mu.Lock()
	job.Status = "running"
	job.LastUpdated = time.Now()
	m.mu.Unlock()

	bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	m.log.Infof("Product seed job %s context timeout set to 2h", job.ID)

	if job.Request.GetClearExisting() {
		m.log.Infof("Product seed job %s clearing existing products...", job.ID)
		if err := m.uc.ClearProducts(bgCtx); err != nil {
			m.mu.Lock()
			job.Status = "failed"
			job.Error = fmt.Sprintf("failed to clear products: %v", err)
			job.LastUpdated = time.Now()
			m.mu.Unlock()
			m.log.Errorf("Product seed job %s failed while clearing products: %v", job.ID, err)
			return
		}
		m.log.Infof("Product seed job %s existing products cleared", job.ID)
	}

	records := make([]*biz.Product, 0, len(dataset))
	skipped := 0
	skippedByReason := map[string]int{}
	seenPID := make(map[string]struct{}, len(dataset))
	seenOriginalID := make(map[string]struct{}, len(dataset))
	for i, data := range dataset {
		product, ok, reason := mapDatasetProduct(data, i)
		if !ok {
			skipped++
			skippedByReason[reason]++
			continue
		}
		if _, exists := seenPID[product.PID]; exists {
			skipped++
			skippedByReason["duplicate_pid_in_dataset"]++
			continue
		}
		if _, exists := seenOriginalID[product.OriginalID]; exists {
			skipped++
			skippedByReason["duplicate_original_id_in_dataset"]++
			continue
		}
		seenPID[product.PID] = struct{}{}
		seenOriginalID[product.OriginalID] = struct{}{}
		records = append(records, product)

		if (i+1)%5000 == 0 || i+1 == len(dataset) {
			m.log.Infof("Product seed job %s mapping progress: scanned=%d/%d valid=%d skipped=%d",
				job.ID, i+1, len(dataset), len(records), skipped)
		}
	}
	m.log.Infof("Product seed job %s mapping complete: valid=%d skipped=%d reasons=%v", job.ID, len(records), skipped, skippedByReason)

	if len(records) == 0 {
		m.mu.Lock()
		job.Status = "failed"
		job.Error = "no valid records after dataset mapping"
		job.Skipped = skipped
		job.Processed = skipped
		job.LastUpdated = time.Now()
		m.mu.Unlock()
		m.log.Errorf("Product seed job %s failed: no valid records after mapping", job.ID)
		return
	}

	m.mu.Lock()
	job.Skipped = skipped
	job.Processed = skipped
	job.LastUpdated = time.Now()
	m.mu.Unlock()

	batchSize := int(job.Request.GetBatchSize())
	if batchSize <= 0 {
		batchSize = 100
	}
	m.log.Infof("Product seed job %s bulk insert starting: records=%d batch_size=%d", job.ID, len(records), batchSize)
	seeded, err := m.uc.BulkCreateProducts(bgCtx, records, batchSize, func(processed int) {
		m.mu.Lock()
		job.Seeded = processed
		job.Processed = job.Skipped + processed
		if job.Processed > job.Total {
			job.Processed = job.Total
		}
		job.LastUpdated = time.Now()
		m.mu.Unlock()
		if processed%1000 == 0 || processed == len(records) {
			m.log.Infof("Product seed job %s progress: seeded=%d/%d skipped=%d processed=%d/%d",
				job.ID, processed, len(records), job.Skipped, job.Processed, job.Total)
		}
	})

	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		job.Seeded = seeded
		job.Processed = job.Skipped + seeded
		if job.Processed > job.Total {
			job.Processed = job.Total
		}
		job.LastUpdated = time.Now()
		m.log.Errorf("Product seed job %s failed: %v", job.ID, err)
		return
	}

	job.Status = "completed"
	job.Seeded = seeded
	job.Processed = job.Total
	job.LastUpdated = time.Now()
	m.log.Infof("Product seed job %s completed successfully: seeded=%d skipped=%d", job.ID, job.Seeded, job.Skipped)
}

func (m *ProductSeedJobManager) GetJob(jobID string) *ProductSeedJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeJob != nil && m.activeJob.ID == jobID {
		return m.activeJob
	}
	return nil
}

// ===================== HELPER METHODS =====================

// Helper function to generate job ID
func generateJobID() string {
	return "embed_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateSeedJobID() string {
	return "seed_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func loadProductDataset(logger *log.Helper) ([]map[string]interface{}, error) {
	paths := []string{
		"./product_dataset.json",
		"../product_dataset.json",
		"/data/product_dataset.json",
		"/app/product_dataset.json",
	}

	var file *os.File
	var err error
	for _, path := range paths {
		logger.Infof("Trying dataset path: %s", path)
		file, err = os.Open(path)
		if err == nil {
			defer file.Close()
			logger.Infof("Dataset opened from path: %s", path)
			break
		}
	}
	if err != nil {
		logger.Errorf("Could not open dataset from any known path: %v", err)
		return nil, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		logger.Errorf("Failed reading dataset content: %v", err)
		return nil, err
	}
	logger.Infof("Dataset size bytes: %d", len(content))

	var dataset []map[string]interface{}
	if err := json.Unmarshal(content, &dataset); err != nil {
		logger.Errorf("Failed unmarshalling dataset JSON: %v", err)
		return nil, err
	}
	logger.Infof("Dataset entries loaded: %d", len(dataset))

	return dataset, nil
}

func mapDatasetProduct(data map[string]interface{}, index int) (*biz.Product, bool, string) {
	pid, _ := data["pid"].(string)
	title, _ := data["title"].(string)
	brand, _ := data["brand"].(string)
	category, _ := data["category"].(string)
	subCategory, _ := data["sub_category"].(string)
	originalID, _ := data["_id"].(string)

	if pid == "" {
		return nil, false, "missing_pid"
	}
	if title == "" {
		return nil, false, "missing_title"
	}
	if brand == "" {
		return nil, false, "missing_brand"
	}
	if category == "" {
		return nil, false, "missing_category"
	}
	if subCategory == "" {
		return nil, false, "missing_sub_category"
	}
	if originalID == "" {
		return nil, false, "missing_original_id"
	}

	p := &biz.Product{
		PID:         pid,
		Title:       title,
		Brand:       brand,
		Category:    category,
		SubCategory: subCategory,
		OriginalID:  originalID,
	}

	if desc, ok := data["description"].(string); ok {
		p.Description = desc
	}
	if price, ok := data["selling_price"].(string); ok {
		p.SellingPrice = price
		p.PriceNumeric = parseDatasetPrice(price)
	}
	if actualPrice, ok := data["actual_price"].(string); ok {
		p.ActualPrice = actualPrice
	}
	if discount, ok := data["discount"].(string); ok {
		p.Discount = discount
	}
	if seller, ok := data["seller"].(string); ok {
		p.Seller = seller
	}
	if rating, ok := data["average_rating"].(string); ok {
		p.AverageRating = rating
		if ratingNum, err := strconv.ParseFloat(rating, 32); err == nil {
			p.RatingNumeric = float32(ratingNum)
		}
	}
	if outOfStock, ok := data["out_of_stock"].(bool); ok {
		p.OutOfStock = outOfStock
	}
	if url, ok := data["url"].(string); ok {
		p.URL = url
	}
	if styleCode, ok := data["style_code"].(string); ok {
		p.StyleCode = styleCode
	}

	if images, ok := data["images"].([]interface{}); ok {
		imageStrings := make([]string, 0, len(images))
		for _, img := range images {
			if str, ok := img.(string); ok {
				imageStrings = append(imageStrings, str)
			}
		}
		p.Images = imageStrings
	}

	if details, ok := data["product_details"].([]interface{}); ok {
		productDetails := make([]map[string]string, 0, len(details))
		for _, detail := range details {
			if detailMap, ok := detail.(map[string]interface{}); ok {
				strMap := make(map[string]string)
				for k, v := range detailMap {
					if str, ok := v.(string); ok {
						strMap[k] = str
					}
				}
				if len(strMap) > 0 {
					productDetails = append(productDetails, strMap)
				}
			}
		}
		p.ProductDetails = productDetails
	}

	if crawledAt, ok := data["crawled_at"].(string); ok {
		p.CrawledAt = parseDatasetTime(crawledAt)
	}

	if (index+1)%20 == 0 {
		p.Featured = true
	}

	return p, true, ""
}

func parseDatasetPrice(priceStr string) int {
	if priceStr == "" {
		return 0
	}

	cleaned := strings.ReplaceAll(priceStr, "â‚¹", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	if price, err := strconv.Atoi(cleaned); err == nil {
		return price
	}
	return 0
}

func parseDatasetTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	layouts := []string{
		"02/01/2006, 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, timeStr)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}

// ===================== EXISTING HELPER METHODS =====================

func (s *ProductService) convertToProductInfo(p *biz.Product) (*pb.ProductInfo, error) {
	if p == nil {
		return nil, nil
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

	// Create timestamps
	var crawledAt, createdAt, updatedAt *timestamppb.Timestamp
	if !p.CrawledAt.IsZero() {
		crawledAt = timestamppb.New(p.CrawledAt)
	}
	if !p.CreatedAt.IsZero() {
		createdAt = timestamppb.New(p.CreatedAt)
	}
	if !p.UpdatedAt.IsZero() {
		updatedAt = timestamppb.New(p.UpdatedAt)
	}

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
		CrawledAt:          crawledAt,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ViewCount:          int32(p.ViewCount),
		ClickCount:         int32(p.ClickCount),
		Featured:           p.Featured,
	}, nil
}

func (s *ProductService) convertToProductList(products []*biz.Product) []*pb.ProductInfo {
	result := make([]*pb.ProductInfo, len(products))
	for i, p := range products {
		info, err := s.convertToProductInfo(p)
		if err != nil {
			s.log.Errorf("Failed to convert product %d: %v", p.ID, err)
			continue
		}
		result[i] = info
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
