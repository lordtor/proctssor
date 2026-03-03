package nats

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TimerType represents the type of timer
type TimerType string

const (
	// TimerTypeDate - execute at specific date/time
	TimerTypeDate TimerType = "date"
	// TimerTypeDuration - execute after duration
	TimerTypeDuration TimerType = "duration"
	// TimerTypeCycle - execute on a cycle
	TimerTypeCycle TimerType = "cycle"
	// TimerTypeISO8601 - ISO 8601 format
	TimerTypeISO8601 TimerType = "iso8601"
)

// TimerJobStatus represents the status of a timer job
type TimerJobStatus string

const (
	// TimerJobStatusPending - timer job is pending
	TimerJobStatusPending TimerJobStatus = "pending"
	// TimerJobStatusRunning - timer job is running
	TimerJobStatusRunning TimerJobStatus = "running"
	// TimerJobStatusCompleted - timer job completed
	TimerJobStatusCompleted TimerJobStatus = "completed"
	// TimerJobStatusFailed - timer job failed
	TimerJobStatusFailed TimerJobStatus = "failed"
	// TimerJobStatusCancelled - timer job was cancelled
	TimerJobStatusCancelled TimerJobStatus = "cancelled"
)

// TimerJob represents a timer job in the database
type TimerJob struct {
	ID              uuid.UUID       `json:"id"`
	InstanceID      uuid.UUID       `json:"instance_id"`
	NodeID          string          `json:"node_id"`
	TokenID         string          `json:"token_id,omitempty"`
	TimerType       TimerType       `json:"timer_type"`
	DueDate         time.Time       `json:"due_date"`
	RepeatInterval  *time.Duration  `json:"repeat_interval,omitempty"`
	MaxAttempts     int             `json:"max_attempts"`
	AttemptCount    int             `json:"attempt_count"`
	HandlerConfig   json.RawMessage `json:"handler_config"`
	Status          TimerJobStatus  `json:"status"`
	LastExecutedAt  *time.Time      `json:"last_executed_at,omitempty"`
	NextExecutionAt *time.Time      `json:"next_execution_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TimerManager manages timer jobs and their execution
type TimerManager struct {
	db        *sql.DB
	publisher *Publisher
	interval  time.Duration
}

// TimerManagerConfig holds configuration for the timer manager
type TimerManagerConfig struct {
	DB        *sql.DB
	Publisher *Publisher
	Interval  time.Duration
}

// NewTimerManager creates a new timer manager
func NewTimerManager(cfg TimerManagerConfig) (*TimerManager, error) {
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Second
	}

	return &TimerManager{
		db:        cfg.DB,
		publisher: cfg.Publisher,
		interval:  cfg.Interval,
	}, nil
}

// ScheduleTimer creates a new timer job
func (tm *TimerManager) ScheduleTimer(ctx context.Context, instanceID uuid.UUID, nodeID, tokenID string, timerType TimerType, duration time.Duration) (*TimerJob, error) {
	// Calculate due date based on timer type
	var dueDate time.Time
	var repeatInterval *time.Duration

	switch timerType {
	case TimerTypeDuration:
		dueDate = time.Now().Add(duration)
	case TimerTypeDate:
		dueDate = time.Now().Add(duration) // duration here is actually the target time
	case TimerTypeCycle:
		dueDate = time.Now().Add(duration)
		repeatInterval = &duration
	case TimerTypeISO8601:
		// Parse ISO8601 duration
		dueDate = time.Now().Add(duration)
	default:
		dueDate = time.Now().Add(duration)
	}

	job := &TimerJob{
		ID:              uuid.New(),
		InstanceID:      instanceID,
		NodeID:          nodeID,
		TokenID:         tokenID,
		TimerType:       timerType,
		DueDate:         dueDate,
		RepeatInterval:  repeatInterval,
		MaxAttempts:     3,
		AttemptCount:    0,
		HandlerConfig:   json.RawMessage("{}"),
		Status:          TimerJobStatusPending,
		NextExecutionAt: &dueDate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO timer_jobs (
			id, instance_id, node_id, token_id, timer_type, due_date,
			repeat_interval, max_attempts, attempt_count, handler_config,
			status, next_execution_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := tm.db.ExecContext(ctx, query,
		job.ID, job.InstanceID, job.NodeID, job.TokenID, job.TimerType,
		job.DueDate, job.RepeatInterval, job.MaxAttempts, job.AttemptCount,
		job.HandlerConfig, job.Status, job.NextExecutionAt, job.CreatedAt, job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert timer job: %w", err)
	}

	return job, nil
}

// ScheduleTimerISO8601 schedules a timer using ISO8601 format
func (tm *TimerManager) ScheduleTimerISO8601(ctx context.Context, instanceID uuid.UUID, nodeID, tokenID, iso8601Duration string) (*TimerJob, error) {
	// For simplicity, using duration parsing
	// In production, implement full ISO8601 parsing
	d, err := time.ParseDuration(iso8601Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ISO8601 duration: %w", err)
	}

	return tm.ScheduleTimer(ctx, instanceID, nodeID, tokenID, TimerTypeISO8601, d)
}

// CancelTimer cancels a timer job
func (tm *TimerManager) CancelTimer(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE timer_jobs
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = $4
	`
	_, err := tm.db.ExecContext(ctx, query,
		TimerJobStatusCancelled, time.Now(), jobID, TimerJobStatusPending,
	)
	return err
}

// Start starts the timer manager to process due timers
func (tm *TimerManager) Start(ctx context.Context) error {
	ticker := time.NewTicker(tm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := tm.processDueTimers(ctx); err != nil {
				fmt.Printf("Error processing due timers: %v\n", err)
			}
		}
	}
}

// processDueTimers processes timers that are due
func (tm *TimerManager) processDueTimers(ctx context.Context) error {
	// Get pending timers that are due
	query := `
		SELECT id, instance_id, node_id, token_id, timer_type, due_date,
		       repeat_interval, max_attempts, attempt_count, handler_config,
		       status, last_executed_at, next_execution_at, created_at, updated_at
		FROM timer_jobs
		WHERE status = 'pending'
		  AND due_date <= NOW()
		ORDER BY due_date ASC
		LIMIT 100
	`

	rows, err := tm.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query due timers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var job TimerJob
		err := rows.Scan(
			&job.ID, &job.InstanceID, &job.NodeID, &job.TokenID, &job.TimerType,
			&job.DueDate, &job.RepeatInterval, &job.MaxAttempts, &job.AttemptCount,
			&job.HandlerConfig, &job.Status, &job.LastExecutedAt, &job.NextExecutionAt,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Failed to scan timer job: %v\n", err)
			continue
		}

		if err := tm.executeTimer(ctx, &job); err != nil {
			fmt.Printf("Failed to execute timer job %s: %v\n", job.ID, err)
		}
	}

	return rows.Err()
}

// executeTimer executes a timer job
func (tm *TimerManager) executeTimer(ctx context.Context, job *TimerJob) error {
	// Use transaction to prevent race conditions
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark as running
	updateQuery := `
		UPDATE timer_jobs
		SET status = $1, attempt_count = attempt_count + 1,
		    last_executed_at = $2, updated_at = $2
		WHERE id = $3
	`
	_, err = tx.ExecContext(ctx, updateQuery,
		TimerJobStatusRunning, time.Now(), job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update timer job status: %w", err)
	}

	// Publish timer command to NATS
	if tm.publisher != nil {
		variables := map[string]interface{}{
			"timer_job_id": job.ID.String(),
			"node_id":      job.NodeID,
			"timer_type":   string(job.TimerType),
			"due_date":     job.DueDate.Format(time.RFC3339),
		}

		err = tm.publisher.PublishTimerCommand(ctx, job.InstanceID.String(), job.TokenID, job.NodeID, variables)
		if err != nil {
			// Mark as failed
			tm.markTimerJobFailed(ctx, job.ID, err.Error())
			return fmt.Errorf("failed to publish timer command: %w", err)
		}
	}

	// Get the updated attempt count after increment
	var newAttemptCount int
	err = tx.QueryRowContext(ctx, `SELECT attempt_count FROM timer_jobs WHERE id = $1`, job.ID).Scan(&newAttemptCount)
	if err != nil {
		return fmt.Errorf("failed to get attempt count: %w", err)
	}

	// Handle repeat interval
	if job.RepeatInterval != nil && newAttemptCount < job.MaxAttempts {
		nextDue := time.Now().Add(*job.RepeatInterval)
		_, err = tx.ExecContext(ctx, `
			UPDATE timer_jobs
			SET status = $1, due_date = $2, next_execution_at = $2, updated_at = $2
			WHERE id = $3
		`, TimerJobStatusPending, nextDue, job.ID)
	} else {
		// Mark as completed
		_, err = tx.ExecContext(ctx, `
			UPDATE timer_jobs
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, TimerJobStatusCompleted, time.Now(), job.ID)
	}

	if err != nil {
		return fmt.Errorf("failed to update timer job final status: %w", err)
	}

	return tx.Commit()
}

// markTimerJobFailed marks a timer job as failed
func (tm *TimerManager) markTimerJobFailed(ctx context.Context, jobID uuid.UUID, errorMsg string) {
	_, _ = tm.db.ExecContext(ctx, `
		UPDATE timer_jobs
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, TimerJobStatusFailed, time.Now(), jobID)
}

// GetPendingTimers returns pending timers for an instance
func (tm *TimerManager) GetPendingTimers(ctx context.Context, instanceID uuid.UUID) ([]TimerJob, error) {
	query := `
		SELECT id, instance_id, node_id, token_id, timer_type, due_date,
		       repeat_interval, max_attempts, attempt_count, handler_config,
		       status, last_executed_at, next_execution_at, created_at, updated_at
		FROM timer_jobs
		WHERE instance_id = $1 AND status = 'pending'
		ORDER BY due_date ASC
	`

	rows, err := tm.db.QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []TimerJob
	for rows.Next() {
		var job TimerJob
		err := rows.Scan(
			&job.ID, &job.InstanceID, &job.NodeID, &job.TokenID, &job.TimerType,
			&job.DueDate, &job.RepeatInterval, &job.MaxAttempts, &job.AttemptCount,
			&job.HandlerConfig, &job.Status, &job.LastExecutedAt, &job.NextExecutionAt,
			&job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// TimerFired marks a timer as fired (called after successful processing)
func (tm *TimerManager) TimerFired(ctx context.Context, jobID uuid.UUID) error {
	// Get the job to check if it should repeat
	var job TimerJob
	err := tm.db.QueryRowContext(ctx, `
		SELECT id, instance_id, node_id, token_id, timer_type, due_date,
		       repeat_interval, max_attempts, attempt_count, handler_config,
		       status, last_executed_at, next_execution_at, created_at, updated_at
		FROM timer_jobs WHERE id = $1
	`, jobID).Scan(
		&job.ID, &job.InstanceID, &job.NodeID, &job.TokenID, &job.TimerType,
		&job.DueDate, &job.RepeatInterval, &job.MaxAttempts, &job.AttemptCount,
		&job.HandlerConfig, &job.Status, &job.LastExecutedAt, &job.NextExecutionAt,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// If repeat interval and not exceeded max attempts, reschedule
	if job.RepeatInterval != nil && job.AttemptCount < job.MaxAttempts {
		nextDue := time.Now().Add(*job.RepeatInterval)
		_, err = tm.db.ExecContext(ctx, `
			UPDATE timer_jobs
			SET status = $1, due_date = $2, next_execution_at = $2, updated_at = $2
			WHERE id = $3
		`, TimerJobStatusPending, nextDue, job.ID)
	} else {
		// Mark as completed
		_, err = tm.db.ExecContext(ctx, `
			UPDATE timer_jobs
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, TimerJobStatusCompleted, time.Now(), job.ID)
	}

	return err
}

// CleanupTimers removes completed and failed timer jobs older than specified duration
func (tm *TimerManager) CleanupTimers(ctx context.Context, olderThan time.Duration) error {
	query := `
		DELETE FROM timer_jobs
		WHERE status IN ('completed', 'failed', 'cancelled')
		  AND updated_at < NOW() - $1
	`
	_, err := tm.db.ExecContext(ctx, query, olderThan)
	return err
}
