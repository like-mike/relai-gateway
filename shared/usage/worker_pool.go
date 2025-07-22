package usage

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/like-mike/relai-gateway/shared/db"
	"github.com/like-mike/relai-gateway/shared/models"
)

// UsageLogJob represents a usage logging job
type UsageLogJob struct {
	OrganizationID string
	APIKeyID       string
	ModelID        string
	Provider       string
	Endpoint       string
	RequestID      *string
	ResponseStatus int
	ResponseTimeMS *int
	Usage          *models.AIProviderUsage
	Cost           *float64
	Metadata       map[string]interface{}
	RetryCount     int
	CreatedAt      time.Time
}

// UsageWorkerPool manages background workers for processing usage logs
type UsageWorkerPool struct {
	workers  int
	jobQueue chan *UsageLogJob
	db       *sql.DB
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	config   *WorkerConfig
}

// WorkerConfig configures the worker pool behavior
type WorkerConfig struct {
	WorkerCount    int           `json:"worker_count"`
	QueueSize      int           `json:"queue_size"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
	BatchSize      int           `json:"batch_size"`
	BatchTimeout   time.Duration `json:"batch_timeout"`
	EnableBatching bool          `json:"enable_batching"`
}

// DefaultWorkerConfig returns a sensible default configuration
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		WorkerCount:    5,
		QueueSize:      1000,
		MaxRetries:     3,
		RetryDelay:     time.Second * 2,
		BatchSize:      10,
		BatchTimeout:   time.Second * 5,
		EnableBatching: false, // Start with simple single inserts
	}
}

// NewUsageWorkerPool creates a new worker pool for usage logging
func NewUsageWorkerPool(database *sql.DB, config *WorkerConfig) *UsageWorkerPool {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &UsageWorkerPool{
		workers:  config.WorkerCount,
		jobQueue: make(chan *UsageLogJob, config.QueueSize),
		db:       database,
		ctx:      ctx,
		cancel:   cancel,
		config:   config,
	}

	return pool
}

// Start begins processing usage logging jobs
func (p *UsageWorkerPool) Start() {
	log.Printf("Starting usage worker pool with %d workers", p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (p *UsageWorkerPool) Stop() {
	log.Println("Stopping usage worker pool...")
	p.cancel()
	close(p.jobQueue)
	p.wg.Wait()
	log.Println("Usage worker pool stopped")
}

// SubmitJob submits a usage logging job to the worker pool
func (p *UsageWorkerPool) SubmitJob(job *UsageLogJob) bool {
	select {
	case p.jobQueue <- job:
		return true
	default:
		log.Printf("Usage worker pool queue is full, dropping job for org %s", job.OrganizationID)
		return false
	}
}

// SubmitUsage is a convenience method to submit usage data
func (p *UsageWorkerPool) SubmitUsage(
	orgID, apiKeyID, modelID, provider, endpoint string,
	requestID *string, responseStatus int, responseTimeMS *int,
	usage *models.AIProviderUsage, cost *float64,
	metadata map[string]interface{},
) bool {
	job := &UsageLogJob{
		OrganizationID: orgID,
		APIKeyID:       apiKeyID,
		ModelID:        modelID,
		Provider:       provider,
		Endpoint:       endpoint,
		RequestID:      requestID,
		ResponseStatus: responseStatus,
		ResponseTimeMS: responseTimeMS,
		Usage:          usage,
		Cost:           cost,
		Metadata:       metadata,
		CreatedAt:      time.Now(),
	}

	return p.SubmitJob(job)
}

// worker processes jobs from the queue
func (p *UsageWorkerPool) worker(workerID int) {
	defer p.wg.Done()

	log.Printf("Usage worker %d started", workerID)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Usage worker %d stopping", workerID)
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				log.Printf("Usage worker %d: job queue closed", workerID)
				return
			}

			p.processJob(workerID, job)
		}
	}
}

// processJob processes a single usage logging job
func (p *UsageWorkerPool) processJob(workerID int, job *UsageLogJob) {
	if job.Usage == nil {
		log.Printf("Worker %d: skipping job with nil usage data", workerID)
		return
	}

	// Create usage log request
	usageReq := db.CreateUsageLogRequest{
		OrganizationID:   job.OrganizationID,
		APIKeyID:         job.APIKeyID,
		ModelID:          job.ModelID,
		Endpoint:         job.Endpoint,
		PromptTokens:     job.Usage.PromptTokens,
		CompletionTokens: job.Usage.CompletionTokens,
		TotalTokens:      job.Usage.TotalTokens,
		RequestID:        job.RequestID,
		ResponseStatus:   job.ResponseStatus,
		ResponseTimeMS:   job.ResponseTimeMS,
		CostUSD:          job.Cost,
		Metadata:         job.Metadata,
	}

	// Attempt to log usage
	if err := db.CreateUsageLog(p.db, usageReq); err != nil {
		log.Printf("Worker %d: failed to create usage log: %v", workerID, err)

		// Retry logic
		if job.RetryCount < p.config.MaxRetries {
			job.RetryCount++
			log.Printf("Worker %d: retrying job (attempt %d/%d)", workerID, job.RetryCount, p.config.MaxRetries)

			// Schedule retry with delay
			go func() {
				time.Sleep(p.config.RetryDelay * time.Duration(job.RetryCount))
				p.SubmitJob(job)
			}()
		} else {
			log.Printf("Worker %d: max retries exceeded for usage log, dropping job", workerID)
		}
		return
	}

	// Update organization quota
	if err := db.UpdateOrganizationUsage(p.db, job.OrganizationID, job.Usage.TotalTokens); err != nil {
		log.Printf("Worker %d: failed to update organization usage: %v", workerID, err)
		// Note: We don't retry quota updates to avoid duplicate increments
	}

	log.Printf("Worker %d: successfully logged usage: %d tokens for org %s",
		workerID, job.Usage.TotalTokens, job.OrganizationID)
}

// GetQueueSize returns the current number of jobs in the queue
func (p *UsageWorkerPool) GetQueueSize() int {
	return len(p.jobQueue)
}

// GetStats returns usage statistics for the worker pool
func (p *UsageWorkerPool) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		WorkerCount:      p.workers,
		QueueSize:        len(p.jobQueue),
		QueueCapacity:    cap(p.jobQueue),
		QueueUtilization: float64(len(p.jobQueue)) / float64(cap(p.jobQueue)) * 100,
	}
}

// WorkerPoolStats contains statistics about the worker pool
type WorkerPoolStats struct {
	WorkerCount      int     `json:"worker_count"`
	QueueSize        int     `json:"queue_size"`
	QueueCapacity    int     `json:"queue_capacity"`
	QueueUtilization float64 `json:"queue_utilization_percent"`
}

// Global worker pool instance
var globalWorkerPool *UsageWorkerPool
var workerPoolOnce sync.Once

// InitGlobalWorkerPool initializes the global worker pool
func InitGlobalWorkerPool(database *sql.DB, config *WorkerConfig) {
	workerPoolOnce.Do(func() {
		globalWorkerPool = NewUsageWorkerPool(database, config)
		globalWorkerPool.Start()
	})
}

// GetGlobalWorkerPool returns the global worker pool instance
func GetGlobalWorkerPool() *UsageWorkerPool {
	return globalWorkerPool
}

// StopGlobalWorkerPool stops the global worker pool
func StopGlobalWorkerPool() {
	if globalWorkerPool != nil {
		globalWorkerPool.Stop()
	}
}
