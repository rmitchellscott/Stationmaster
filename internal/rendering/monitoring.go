package rendering

import (
	"context"
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// HealthStatus represents the health status of the worker pool
type HealthStatus struct {
	Status          string                 `json:"status"`           // "healthy", "degraded", "unhealthy"
	WorkerPool      WorkerPoolHealth       `json:"worker_pool"`
	Queue           QueueHealth            `json:"queue"`
	Performance     PerformanceMetrics     `json:"performance"`
	LastUpdated     time.Time              `json:"last_updated"`
	Recommendations []string               `json:"recommendations,omitempty"`
}

// WorkerPoolHealth represents worker pool specific health metrics
type WorkerPoolHealth struct {
	ActiveWorkers     int32   `json:"active_workers"`
	ExpectedWorkers   int     `json:"expected_workers"`
	WorkerUtilization float64 `json:"worker_utilization"` // % of workers currently processing
	ChannelCapacity   int     `json:"channel_capacity"`
	ChannelUtilization float64 `json:"channel_utilization"` // % of channel buffer used
}

// QueueHealth represents queue specific health metrics
type QueueHealth struct {
	PendingJobs        int64     `json:"pending_jobs"`
	ProcessingJobs     int64     `json:"processing_jobs"`
	FailedJobs         int64     `json:"failed_jobs"`
	OldestPendingJob   *time.Time `json:"oldest_pending_job,omitempty"`
	AverageWaitTime    *float64   `json:"average_wait_time_seconds,omitempty"`
}

// PerformanceMetrics represents performance statistics
type PerformanceMetrics struct {
	TotalJobs        int64   `json:"total_jobs"`
	SuccessRate      float64 `json:"success_rate"`
	JobsPerMinute    float64 `json:"jobs_per_minute"`
	AverageProcessingTime *float64 `json:"average_processing_time_seconds,omitempty"`
}

// MonitoringService provides health monitoring for the render system
type MonitoringService struct {
	workerPool   *RenderWorkerPool
	queueManager *QueueManager
	
	// Metrics collection
	startTime        time.Time
	lastMetricsReset time.Time
	lastJobCount     int64
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(workerPool *RenderWorkerPool, queueManager *QueueManager) *MonitoringService {
	now := time.Now()
	return &MonitoringService{
		workerPool:       workerPool,
		queueManager:     queueManager,
		startTime:        now,
		lastMetricsReset: now,
		lastJobCount:     0,
	}
}

// GetHealthStatus returns the current health status of the render system
func (m *MonitoringService) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	// Get worker pool metrics
	metrics := m.workerPool.GetMetrics()
	
	// Get queue statistics
	queueStats, err := m.queueManager.GetQueueStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	
	// Calculate worker utilization
	var workerUtilization float64
	if m.workerPool.workerCount > 0 {
		// Count how many workers are processing
		processingWorkers := 0
		for _, worker := range m.workerPool.workers {
			if worker.IsProcessing() {
				processingWorkers++
			}
		}
		workerUtilization = float64(processingWorkers) / float64(m.workerPool.workerCount) * 100
	}
	
	// Calculate channel utilization
	channelUtilization := float64(metrics.QueueLength) / float64(cap(m.workerPool.jobChan)) * 100
	
	// Build worker pool health
	workerPoolHealth := WorkerPoolHealth{
		ActiveWorkers:      metrics.ActiveWorkers,
		ExpectedWorkers:    m.workerPool.workerCount,
		WorkerUtilization:  workerUtilization,
		ChannelCapacity:    cap(m.workerPool.jobChan),
		ChannelUtilization: channelUtilization,
	}
	
	// Extract queue metrics
	statusCounts, ok := queueStats["status_counts"].(map[string]int64)
	if !ok {
		statusCounts = make(map[string]int64)
	}
	
	queueHealth := QueueHealth{
		PendingJobs:    statusCounts["pending"],
		ProcessingJobs: statusCounts["processing"],
		FailedJobs:     statusCounts["failed"],
	}
	
	// Add oldest pending job if available
	if oldestPending, ok := queueStats["oldest_pending"].(time.Time); ok {
		queueHealth.OldestPendingJob = &oldestPending
		
		// Calculate average wait time
		waitTime := time.Since(oldestPending).Seconds()
		queueHealth.AverageWaitTime = &waitTime
	}
	
	// Calculate performance metrics
	successRate := float64(0)
	if metrics.TotalJobs > 0 {
		successRate = float64(metrics.SuccessJobs) / float64(metrics.TotalJobs) * 100
	}
	
	// Calculate jobs per minute
	elapsed := time.Since(m.lastMetricsReset).Minutes()
	jobsPerMinute := float64(0)
	if elapsed > 0 {
		jobsSinceReset := metrics.TotalJobs - m.lastJobCount
		jobsPerMinute = float64(jobsSinceReset) / elapsed
	}
	
	performanceMetrics := PerformanceMetrics{
		TotalJobs:     metrics.TotalJobs,
		SuccessRate:   successRate,
		JobsPerMinute: jobsPerMinute,
	}
	
	// Determine overall health status and recommendations
	status, recommendations := m.determineHealthStatus(workerPoolHealth, queueHealth, performanceMetrics)
	
	healthStatus := &HealthStatus{
		Status:          status,
		WorkerPool:      workerPoolHealth,
		Queue:           queueHealth,
		Performance:     performanceMetrics,
		LastUpdated:     time.Now(),
		Recommendations: recommendations,
	}
	
	return healthStatus, nil
}

// determineHealthStatus analyzes metrics and determines overall health
func (m *MonitoringService) determineHealthStatus(
	workerPool WorkerPoolHealth, 
	queue QueueHealth, 
	performance PerformanceMetrics,
) (string, []string) {
	var recommendations []string
	
	// Check for unhealthy conditions
	unhealthyConditions := 0
	degradedConditions := 0
	
	// Worker pool checks
	if workerPool.ActiveWorkers < int32(workerPool.ExpectedWorkers) {
		unhealthyConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("Worker pool degraded: %d/%d workers active", 
				workerPool.ActiveWorkers, workerPool.ExpectedWorkers))
	}
	
	if workerPool.ChannelUtilization > 90 {
		degradedConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("Job channel nearly full (%.1f%% utilized) - consider increasing buffer size", 
				workerPool.ChannelUtilization))
	}
	
	if workerPool.WorkerUtilization > 95 {
		degradedConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("Workers heavily loaded (%.1f%% utilized) - consider adding more workers", 
				workerPool.WorkerUtilization))
	}
	
	// Queue checks
	if queue.PendingJobs > 50 {
		degradedConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("High queue backlog: %d pending jobs", queue.PendingJobs))
	}
	
	if queue.FailedJobs > 0 && performance.SuccessRate < 90 {
		degradedConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("High failure rate: %.1f%% success rate", performance.SuccessRate))
	}
	
	// Wait time checks
	if queue.AverageWaitTime != nil && *queue.AverageWaitTime > 300 { // 5 minutes
		degradedConditions++
		recommendations = append(recommendations, 
			fmt.Sprintf("Long queue wait times: %.0f seconds average", *queue.AverageWaitTime))
	}
	
	// Determine status
	if unhealthyConditions > 0 {
		return "unhealthy", recommendations
	} else if degradedConditions > 2 {
		return "degraded", recommendations
	} else if degradedConditions > 0 {
		recommendations = append(recommendations, "System operating with minor issues")
		return "degraded", recommendations
	}
	
	return "healthy", recommendations
}

// ResetMetrics resets the performance metrics counters
func (m *MonitoringService) ResetMetrics() {
	m.lastMetricsReset = time.Now()
	m.lastJobCount = m.workerPool.GetMetrics().TotalJobs
	logging.Info("[MONITORING] Reset performance metrics counters")
}

// LogHealthSummary logs a summary of the current health status
func (m *MonitoringService) LogHealthSummary(ctx context.Context) {
	health, err := m.GetHealthStatus(ctx)
	if err != nil {
		logging.Error("[MONITORING] Failed to get health status", "error", err)
		return
	}
	
	logging.Info("[MONITORING] Health Summary", 
		"status", health.Status,
		"active_workers", health.WorkerPool.ActiveWorkers,
		"worker_utilization", fmt.Sprintf("%.1f%%", health.WorkerPool.WorkerUtilization),
		"channel_utilization", fmt.Sprintf("%.1f%%", health.WorkerPool.ChannelUtilization),
		"pending_jobs", health.Queue.PendingJobs,
		"success_rate", fmt.Sprintf("%.1f%%", health.Performance.SuccessRate),
		"jobs_per_minute", fmt.Sprintf("%.1f", health.Performance.JobsPerMinute))
	
	if len(health.Recommendations) > 0 {
		for _, rec := range health.Recommendations {
			logging.Warn("[MONITORING] Recommendation", "message", rec)
		}
	}
}

// GetBufferHealthAlerts returns alerts if channel buffers are unhealthy
func (m *MonitoringService) GetBufferHealthAlerts() []string {
	var alerts []string
	metrics := m.workerPool.GetMetrics()
	
	// Check job channel utilization
	channelUtilization := float64(metrics.QueueLength) / float64(cap(m.workerPool.jobChan)) * 100
	if channelUtilization > 95 {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Job channel %.1f%% full", channelUtilization))
	} else if channelUtilization > 80 {
		alerts = append(alerts, fmt.Sprintf("WARNING: Job channel %.1f%% full", channelUtilization))
	}
	
	// Check result channel utilization
	resultUtilization := float64(len(m.workerPool.resultChan)) / float64(cap(m.workerPool.resultChan)) * 100
	if resultUtilization > 95 {
		alerts = append(alerts, fmt.Sprintf("CRITICAL: Result channel %.1f%% full", resultUtilization))
	} else if resultUtilization > 80 {
		alerts = append(alerts, fmt.Sprintf("WARNING: Result channel %.1f%% full", resultUtilization))
	}
	
	return alerts
}