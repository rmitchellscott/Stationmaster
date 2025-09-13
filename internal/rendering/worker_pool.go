package rendering

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/sse"
)

// RenderJob represents a job to be processed by the worker pool
type RenderJob struct {
	ID               uuid.UUID
	PluginInstanceID uuid.UUID
	Priority         int
	ScheduledFor     time.Time
	Attempts         int
	Context          context.Context
}

// JobResult represents the result of a render job
type JobResult struct {
	JobID      uuid.UUID
	Success    bool
	Error      error
	Message    string
	DurationMs int // Render duration in milliseconds
}

// WorkerMetrics tracks worker pool performance
type WorkerMetrics struct {
	TotalJobs     int64
	SuccessJobs   int64
	FailedJobs    int64
	ActiveWorkers int32
	QueueLength   int32
}

// RenderWorkerPool manages a pool of workers processing render jobs via channels
type RenderWorkerPool struct {
	workerCount     int
	workers         []*Worker
	jobChan         chan RenderJob
	resultChan      chan JobResult
	quitChan        chan struct{}
	wg              sync.WaitGroup
	db              *gorm.DB
	renderWorker    *RenderWorker
	queueManager    *QueueManager
	sseService      *sse.Service
	metrics         *WorkerMetrics
	monitoringService *MonitoringService
	
	// Cleanup timer for periodic maintenance
	cleanupTicker   *time.Ticker
	monitoringTicker *time.Ticker
	
	// Running state
	mu      sync.RWMutex
	running bool
}

// Worker represents a single worker goroutine
type Worker struct {
	id           int
	pool         *RenderWorkerPool
	jobChan      <-chan RenderJob
	resultChan   chan<- JobResult
	quitChan     <-chan struct{}
	isProcessing int32 // atomic flag
}

// NewRenderWorkerPool creates a new render worker pool
func NewRenderWorkerPool(db *gorm.DB, staticDir string, workerCount int, bufferSize int) (*RenderWorkerPool, error) {
	if workerCount <= 0 {
		workerCount = 3 // Default worker count
	}
	if bufferSize <= 0 {
		bufferSize = 100 // Default buffer size
	}
	
	renderWorker, err := NewRenderWorker(db, staticDir)
	if err != nil {
		return nil, err
	}
	
	queueManager := NewQueueManager(db)
	sseService := sse.GetSSEService()
	
	pool := &RenderWorkerPool{
		workerCount:  workerCount,
		workers:      make([]*Worker, workerCount),
		jobChan:      make(chan RenderJob, bufferSize),
		resultChan:   make(chan JobResult, bufferSize),
		quitChan:     make(chan struct{}),
		db:           db,
		renderWorker: renderWorker,
		queueManager: queueManager,
		sseService:   sseService,
		metrics: &WorkerMetrics{
			TotalJobs:     0,
			SuccessJobs:   0,
			FailedJobs:    0,
			ActiveWorkers: 0,
			QueueLength:   0,
		},
		cleanupTicker:    time.NewTicker(5 * time.Minute),  // Cleanup every 5 minutes
		monitoringTicker: time.NewTicker(30 * time.Second), // Monitor every 30 seconds
	}
	
	// Initialize monitoring service
	pool.monitoringService = NewMonitoringService(pool, queueManager)
	
	// Set up bidirectional relationship with queue manager
	queueManager.SetWorkerPool(pool)
	
	// Create workers
	for i := 0; i < workerCount; i++ {
		pool.workers[i] = &Worker{
			id:         i,
			pool:       pool,
			jobChan:    pool.jobChan,
			resultChan: pool.resultChan,
			quitChan:   pool.quitChan,
		}
	}
	
	return pool, nil
}

// Start initializes and starts the worker pool
func (p *RenderWorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		return nil
	}
	
	logging.Info("[WORKER_POOL] Starting render worker pool", "workers", p.workerCount)
	
	p.running = true
	
	// Start workers
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.workers[i].start()
	}
	
	// Start result processor
	p.wg.Add(1)
	go p.processResults(ctx)
	
	// Start cleanup routine
	p.wg.Add(1)
	go p.cleanupRoutine(ctx)
	
	// Start monitoring routine
	p.wg.Add(1)
	go p.monitoringRoutine(ctx)
	
	// Start job feeder that pulls from database
	p.wg.Add(1)
	go p.feedJobs(ctx)
	
	logging.Info("[WORKER_POOL] Worker pool started successfully")
	return nil
}

// Stop gracefully stops the worker pool
func (p *RenderWorkerPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.running {
		return nil
	}
	
	logging.Info("[WORKER_POOL] Stopping worker pool...")
	
	p.running = false
	p.cleanupTicker.Stop()
	p.monitoringTicker.Stop()
	close(p.quitChan)
	
	// Close job channel after a grace period to let pending jobs finish
	go func() {
		time.Sleep(5 * time.Second)
		close(p.jobChan)
	}()
	
	p.wg.Wait()
	close(p.resultChan)
	
	logging.Info("[WORKER_POOL] Worker pool stopped")
	return nil
}

// SubmitJob submits a job directly to the worker pool
func (p *RenderWorkerPool) SubmitJob(job RenderJob) bool {
	select {
	case p.jobChan <- job:
		atomic.AddInt32(&p.metrics.QueueLength, 1)
		return true
	default:
		logging.Warn("[WORKER_POOL] Job channel full, dropping job", "job_id", job.ID)
		return false
	}
}

// GetMetrics returns current worker pool metrics
func (p *RenderWorkerPool) GetMetrics() WorkerMetrics {
	return WorkerMetrics{
		TotalJobs:     atomic.LoadInt64(&p.metrics.TotalJobs),
		SuccessJobs:   atomic.LoadInt64(&p.metrics.SuccessJobs),
		FailedJobs:    atomic.LoadInt64(&p.metrics.FailedJobs),
		ActiveWorkers: atomic.LoadInt32(&p.metrics.ActiveWorkers),
		QueueLength:   int32(len(p.jobChan)),
	}
}

// feedJobs continuously feeds jobs from the database to the worker channel
func (p *RenderWorkerPool) feedJobs(ctx context.Context) {
	defer p.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second) // Check for jobs every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.quitChan:
			return
		case <-ticker.C:
			if err := p.loadPendingJobs(ctx); err != nil {
				logging.Error("[WORKER_POOL] Failed to load pending jobs", "error", err)
			}
		}
	}
}

// loadPendingJobs loads pending render jobs from the database and submits them to workers
func (p *RenderWorkerPool) loadPendingJobs(ctx context.Context) error {
	// Don't overload if channel is nearly full
	if len(p.jobChan) > cap(p.jobChan)*8/10 {
		return nil
	}
	
	type JobID struct {
		ID uuid.UUID
	}
	
	var jobIDs []JobID
	err := p.db.WithContext(ctx).Raw(`
		SELECT id FROM render_queues rq1
		WHERE status = ? AND scheduled_for <= ?
		AND id = (
			SELECT id FROM render_queues rq2
			WHERE rq2.plugin_instance_id = rq1.plugin_instance_id
			AND rq2.status = ? AND rq2.scheduled_for <= ?
			ORDER BY priority DESC, scheduled_for ASC
			LIMIT 1
		)
		GROUP BY plugin_instance_id, id
		LIMIT 20
	`, "pending", time.Now(), "pending", time.Now()).Scan(&jobIDs).Error

	if err != nil {
		return err
	}

	if len(jobIDs) == 0 {
		return nil
	}

	// Extract IDs for the main query
	ids := make([]uuid.UUID, len(jobIDs))
	for i, jobID := range jobIDs {
		ids[i] = jobID.ID
	}

	// Fetch actual jobs
	var dbJobs []database.RenderQueue
	err = p.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("priority DESC, scheduled_for ASC").
		Find(&dbJobs).Error

	if err != nil {
		return err
	}
	
	submitted := 0
	for _, dbJob := range dbJobs {
		job := RenderJob{
			ID:               dbJob.ID,
			PluginInstanceID: dbJob.PluginInstanceID,
			Priority:         dbJob.Priority,
			ScheduledFor:     dbJob.ScheduledFor,
			Attempts:         dbJob.Attempts,
			Context:          ctx,
		}
		
		if p.SubmitJob(job) {
			submitted++
		}
	}
	
	if submitted > 0 {
		logging.Debug("[WORKER_POOL] Submitted jobs to workers", "count", submitted)
	}
	
	return nil
}

// processResults processes job results from workers
func (p *RenderWorkerPool) processResults(ctx context.Context) {
	defer p.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.quitChan:
			return
		case result, ok := <-p.resultChan:
			if !ok {
				return
			}
			
			p.handleResult(ctx, result)
		}
	}
}

// handleResult processes a single job result
func (p *RenderWorkerPool) handleResult(ctx context.Context, result JobResult) {
	if result.Success {
		atomic.AddInt64(&p.metrics.SuccessJobs, 1)
		
		// Update the database with render duration
		err := p.db.WithContext(ctx).Model(&database.RenderQueue{}).
			Where("id = ?", result.JobID).
			Update("render_duration_ms", result.DurationMs).Error
		if err != nil {
			logging.Error("[WORKER_POOL] Failed to update render duration", "job_id", result.JobID, "error", err)
		}
		
		// Load plugin name for logging
		var job database.RenderQueue
		err = p.db.WithContext(ctx).
			Preload("PluginInstance").
			First(&job, result.JobID).Error
		
		// Broadcast success via SSE if available
		p.broadcastJobUpdate(ctx, result.JobID, "completed", result.Message, nil)
		
		// Log with duration in seconds and plugin name
		durationSeconds := float64(result.DurationMs) / 1000.0
		if err == nil {
			logging.Debug("[WORKER_POOL] Render job completed", 
				"job_id", result.JobID, 
				"plugin_name", job.PluginInstance.Name,
				"duration_s", durationSeconds)
		} else {
			logging.Debug("[WORKER_POOL] Render job completed", 
				"job_id", result.JobID, 
				"duration_s", durationSeconds)
		}
	} else {
		atomic.AddInt64(&p.metrics.FailedJobs, 1)
		
		// Broadcast failure via SSE
		p.broadcastJobUpdate(ctx, result.JobID, "failed", result.Message, result.Error)
		
		logging.Error("[WORKER_POOL] Render job failed", "job_id", result.JobID, "error", result.Error)
	}
}

// GetMonitoringService returns the monitoring service instance
func (p *RenderWorkerPool) GetMonitoringService() *MonitoringService {
	return p.monitoringService
}

// monitoringRoutine performs periodic health monitoring
func (p *RenderWorkerPool) monitoringRoutine(ctx context.Context) {
	defer p.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.quitChan:
			return
		case <-p.monitoringTicker.C:
			// Check for critical buffer alerts only
			alerts := p.monitoringService.GetBufferHealthAlerts()
			for _, alert := range alerts {
				logging.Warn("[WORKER_POOL] Buffer Health Alert", "message", alert)
			}
		}
	}
}

// broadcastJobUpdate broadcasts job status updates via SSE
func (p *RenderWorkerPool) broadcastJobUpdate(ctx context.Context, jobID uuid.UUID, status string, message string, err error) {
	if p.sseService == nil {
		return
	}
	
	// Get user context from job to determine who to notify
	var job database.RenderQueue
	dbErr := p.db.WithContext(ctx).
		Preload("PluginInstance.User").
		First(&job, jobID).Error
	if dbErr != nil {
		logging.Error("[WORKER_POOL] Failed to load job for SSE broadcast", "job_id", jobID, "error", dbErr)
		return
	}
	
	// Prepare event data
	eventData := map[string]interface{}{
		"job_id":           jobID.String(),
		"plugin_instance_id": job.PluginInstanceID.String(),
		"status":           status,
		"message":          message,
		"timestamp":        time.Now().UTC(),
	}
	
	if err != nil {
		eventData["error"] = err.Error()
	}
	
	// Broadcast to the user who owns this plugin
	if job.PluginInstance.UserID != uuid.Nil {
		event := sse.Event{
			Type: "render_job_update",
			Data: eventData,
		}
		
		p.sseService.BroadcastToUser(job.PluginInstance.UserID, event)
		
		logging.Debug("[WORKER_POOL] Broadcasted job update via SSE", 
			"job_id", jobID, 
			"user_id", job.PluginInstance.UserID,
			"plugin_name", job.PluginInstance.Name,
			"username", job.PluginInstance.User.Username,
			"status", status)
	}
}

// cleanupRoutine performs periodic maintenance
func (p *RenderWorkerPool) cleanupRoutine(ctx context.Context) {
	defer p.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.quitChan:
			return
		case <-p.cleanupTicker.C:
			// Clean up old render jobs
			if err := p.queueManager.CleanupOldJobs(ctx, 24*time.Hour); err != nil {
				logging.Error("[WORKER_POOL] Failed to cleanup old jobs", "error", err)
			}
			
			// Content cleanup is now handled by render function after successful renders
			// This prevents timing race conditions and ensures hash comparison always works
			
			// Clean up orphaned files
			if err := p.renderWorker.CleanupOrphanedFiles(ctx); err != nil {
				logging.Error("[WORKER_POOL] Failed to cleanup orphaned files", "error", err)
			}
		}
	}
}

// start runs a single worker
func (w *Worker) start() {
	defer w.pool.wg.Done()
	
	logging.Debug("[WORKER] Starting worker", "id", w.id)
	atomic.AddInt32(&w.pool.metrics.ActiveWorkers, 1)
	defer atomic.AddInt32(&w.pool.metrics.ActiveWorkers, -1)
	
	for {
		select {
		case <-w.quitChan:
			logging.Debug("[WORKER] Worker stopping", "id", w.id)
			return
		case job, ok := <-w.jobChan:
			if !ok {
				logging.Debug("[WORKER] Job channel closed", "id", w.id)
				return
			}
			
			w.processJob(job)
		}
	}
}

// processJob processes a single render job
func (w *Worker) processJob(job RenderJob) {
	atomic.StoreInt32(&w.isProcessing, 1)
	defer atomic.StoreInt32(&w.isProcessing, 0)
	
	atomic.AddInt64(&w.pool.metrics.TotalJobs, 1)
	atomic.AddInt32(&w.pool.metrics.QueueLength, -1)
	
	// Load plugin instance to get name and user context for better logging
	var pluginInstance database.PluginInstance
	pluginErr := w.pool.db.WithContext(job.Context).
		Preload("User").
		Preload("PluginDefinition").
		First(&pluginInstance, job.PluginInstanceID).Error
	
	if pluginErr != nil {
		logging.Debug("[WORKER] Processing job", "worker_id", w.id, "job_id", job.ID, "plugin_id", job.PluginInstanceID, "error", "failed to load plugin context")
	} else {
		logging.Debug("[WORKER] Processing job", 
			"worker_id", w.id, 
			"job_id", job.ID, 
			"plugin_id", job.PluginInstanceID,
			"plugin_name", pluginInstance.Name,
			"plugin_type", pluginInstance.PluginDefinition.PluginType,
			"username", pluginInstance.User.Username)
	}
	
	// Mark job as processing in database
	now := time.Now()
	err := w.pool.db.WithContext(job.Context).Model(&database.RenderQueue{}).
		Where("id = ?", job.ID).
		Updates(database.RenderQueue{
			Status:      "processing",
			LastAttempt: &now,
			Attempts:    job.Attempts + 1,
		}).Error
	
	if err != nil {
		w.resultChan <- JobResult{
			JobID:   job.ID,
			Success: false,
			Error:   err,
			Message: "Failed to update job status",
		}
		return
	}
	
	// Broadcast job start via SSE
	w.pool.broadcastJobUpdate(job.Context, job.ID, "processing", 
		fmt.Sprintf("Job started by worker %d", w.id), nil)
	
	// Load the database record to get full job details
	var dbJob database.RenderQueue
	err = w.pool.db.WithContext(job.Context).First(&dbJob, job.ID).Error
	if err != nil {
		w.resultChan <- JobResult{
			JobID:   job.ID,
			Success: false,
			Error:   err,
			Message: "Failed to load job from database",
		}
		return
	}
	
	// Process the job using existing render worker logic
	startTime := time.Now()
	err = w.pool.renderWorker.processRenderJob(job.Context, dbJob)
	processingDuration := time.Since(startTime)
	durationMs := int(processingDuration.Milliseconds())
	
	result := JobResult{
		JobID:      job.ID,
		Success:    err == nil,
		Error:      err,
		DurationMs: durationMs,
	}
	
	if err == nil {
		result.Message = fmt.Sprintf("Job completed successfully in %v", processingDuration)
	} else {
		result.Message = fmt.Sprintf("Job failed after %v: %v", processingDuration, err.Error())
	}
	
	w.resultChan <- result
}

// IsProcessing returns true if the worker is currently processing a job
func (w *Worker) IsProcessing() bool {
	return atomic.LoadInt32(&w.isProcessing) == 1
}