package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/export"
	"github.com/rmitchellscott/stationmaster/internal/storage"
	"gorm.io/gorm"
)

type Worker struct {
	db             *gorm.DB
	dataDir        string
	mu             sync.RWMutex
	running        bool
	quit           chan struct{}
	emptyPollCount int
}

// Global worker instance for on-demand management
var globalWorker *Worker
var globalWorkerMu sync.Mutex

func NewWorker(db *gorm.DB) *Worker {
	dataDir := config.Get("DATA_DIR", "")
	if dataDir == "" {
		dataDir = "/data"
	}

	return &Worker{
		db:      db,
		dataDir: dataDir,
		quit:    make(chan struct{}),
	}
}

func (w *Worker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.run()
}

func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	close(w.quit)
}

func (w *Worker) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.quit:
			return
		case <-ticker.C:
			w.processPendingJobs()
		}
	}
}

func (w *Worker) processPendingJobs() {
	var jobs []database.BackupJob
	if err := w.db.Where("status = ?", "pending").Order("created_at ASC").Find(&jobs).Error; err != nil {
		return
	}

	if len(jobs) == 0 {
		// No jobs found, increment empty poll counter
		w.mu.Lock()
		w.emptyPollCount++
		emptyPolls := w.emptyPollCount
		w.mu.Unlock()

		// Auto-shutdown after 6 empty polls (30 seconds with 5s interval)
		if emptyPolls >= 6 {
			fmt.Printf("[BACKUP] Backup worker shutting down after %d empty polls\n", emptyPolls)
			w.Stop()
			return
		}
		return
	}

	// Reset empty poll counter when jobs are found
	w.mu.Lock()
	w.emptyPollCount = 0
	w.mu.Unlock()

	for _, job := range jobs {
		w.processJob(job)
	}
}

func (w *Worker) processJob(job database.BackupJob) {
	now := time.Now()
	job.Status = "running"
	job.StartedAt = &now
	job.Progress = 0

	if err := w.db.Save(&job).Error; err != nil {
		return
	}

	tempDir, err := os.MkdirTemp("", "stationmaster-backup-*")
	if err != nil {
		w.failJob(job, fmt.Sprintf("Failed to create temp directory: %v", err))
		return
	}
	defer os.RemoveAll(tempDir)

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("stationmaster_backup_%s.tar.gz", timestamp)
	tempBackupPath := filepath.Join(tempDir, filename)

	exporter := export.NewExporter(w.db, w.dataDir)

	var userIDs []uuid.UUID
	if job.UserIDs != "" {
		for _, idStr := range strings.Split(job.UserIDs, ",") {
			if id, err := uuid.Parse(strings.TrimSpace(idStr)); err == nil {
				userIDs = append(userIDs, id)
			}
		}
	}

	exportOptions := export.ExportOptions{
		IncludeDatabase: true,
		IncludeFiles:    job.IncludeFiles,
		IncludeConfigs:  job.IncludeConfigs,
		UserIDs:         userIDs,
	}

	job.Progress = 50
	w.db.Save(&job)

	if err := exporter.Export(tempBackupPath, exportOptions); err != nil {
		w.failJob(job, fmt.Sprintf("Export failed: %v", err))
		return
	}

	ctx := context.Background()
	backend := storage.GetStorageBackend()
	storageKey := fmt.Sprintf("backups/%s", filename)

	backupFile, err := os.Open(tempBackupPath)
	if err != nil {
		w.failJob(job, fmt.Sprintf("Failed to open backup file: %v", err))
		return
	}
	defer backupFile.Close()

	if err := backend.Put(ctx, storageKey, backupFile); err != nil {
		w.failJob(job, fmt.Sprintf("Failed to store backup: %v", err))
		return
	}

	stat, err := os.Stat(tempBackupPath)
	if err != nil {
		w.failJob(job, fmt.Sprintf("Failed to get backup file info: %v", err))
		return
	}

	completedAt := time.Now()

	job.Status = "completed"
	job.Progress = 100
	job.FilePath = storageKey
	job.Filename = filename
	job.FileSize = stat.Size()
	job.CompletedAt = &completedAt

	w.db.Save(&job)
}

func (w *Worker) failJob(job database.BackupJob, errorMsg string) {
	now := time.Now()
	job.Status = "failed"
	job.ErrorMessage = errorMsg
	job.CompletedAt = &now
	w.db.Save(&job)
}

func CreateBackupJob(db *gorm.DB, adminUserID uuid.UUID, includeFiles, includeConfigs bool, userIDs []uuid.UUID) (*database.BackupJob, error) {
	var userIDsStr string
	if len(userIDs) > 0 {
		var strs []string
		for _, id := range userIDs {
			strs = append(strs, id.String())
		}
		userIDsStr = strings.Join(strs, ",")
	}

	job := database.BackupJob{
		AdminUserID:    adminUserID,
		Status:         "pending",
		IncludeFiles:   includeFiles,
		IncludeConfigs: includeConfigs,
		UserIDs:        userIDsStr,
	}

	if err := db.Create(&job).Error; err != nil {
		return nil, err
	}

	return &job, nil
}

func GetBackupJobs(db *gorm.DB, adminUserID uuid.UUID) ([]database.BackupJob, error) {
	var jobs []database.BackupJob
	err := db.Where("admin_user_id = ?", adminUserID).
		Order("created_at DESC").
		Limit(10).
		Find(&jobs).Error
	return jobs, err
}

func GetBackupJob(db *gorm.DB, jobID uuid.UUID, adminUserID uuid.UUID) (*database.BackupJob, error) {
	var job database.BackupJob
	err := db.Where("id = ? AND admin_user_id = ?", jobID, adminUserID).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func CleanupExpiredBackups(db *gorm.DB) error {
	var expiredJobs []database.BackupJob
	if err := db.Where("status = ? AND completed_at < ?", "completed", time.Now().Add(-24*time.Hour)).Find(&expiredJobs).Error; err != nil {
		return err
	}

	ctx := context.Background()
	backend := storage.GetStorageBackend()

	for _, job := range expiredJobs {
		if job.FilePath != "" {
			if err := backend.Delete(ctx, job.FilePath); err != nil {
				fmt.Printf("[BACKUP] Warning: failed to delete backup %s: %v\n", job.FilePath, err)
			}
		}
		db.Delete(&job)
	}

	return nil
}

func DeleteBackupJob(db *gorm.DB, jobID uuid.UUID, adminUserID uuid.UUID) error {
	var job database.BackupJob
	if err := db.Where("id = ? AND admin_user_id = ?", jobID, adminUserID).First(&job).Error; err != nil {
		return err
	}

	// Delete the backup file if it exists
	if job.FilePath != "" {
		ctx := context.Background()
		backend := storage.GetStorageBackend()
		if err := backend.Delete(ctx, job.FilePath); err != nil {
			fmt.Printf("[BACKUP] Warning: failed to delete backup %s: %v\n", job.FilePath, err)
		}
	}

	// Delete the job record
	return db.Delete(&job).Error
}

// EnsureWorkerRunning starts the backup worker if it's not already running
func EnsureWorkerRunning(db *gorm.DB) {
	globalWorkerMu.Lock()
	defer globalWorkerMu.Unlock()

	// If worker exists and is running, nothing to do
	if globalWorker != nil && globalWorker.IsRunning() {
		return
	}

	// Create and start new worker
	globalWorker = NewWorker(db)
	globalWorker.Start()
	fmt.Printf("[BACKUP] Backup worker started on-demand\n")
}

// IsRunning returns true if the worker is currently running
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetWorkerStatus returns the current status of the global worker for debugging
func GetWorkerStatus() map[string]interface{} {
	globalWorkerMu.Lock()
	defer globalWorkerMu.Unlock()

	if globalWorker == nil {
		return map[string]interface{}{
			"exists":  false,
			"running": false,
		}
	}

	globalWorker.mu.RLock()
	status := map[string]interface{}{
		"exists":           true,
		"running":          globalWorker.running,
		"empty_poll_count": globalWorker.emptyPollCount,
	}
	globalWorker.mu.RUnlock()

	return status
}
