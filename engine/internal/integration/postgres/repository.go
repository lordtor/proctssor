package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/statemachine"
)

// DB represents a PostgreSQL database connection
type DB struct {
	db *sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewDB creates a new database connection
func NewDB(cfg Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{db: db}, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}

// BeginTx starts a new transaction
func (d *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}

// GetDB returns the underlying sql.DB
func (d *DB) GetDB() *sql.DB {
	return d.db
}

// ProcessRepository handles process definition persistence
type ProcessRepository interface {
	DeployProcess(ctx context.Context, process *bpmn.Process, definition []byte) error
	GetProcessByID(ctx context.Context, id string) (*bpmn.Process, []byte, error)
	GetProcessByKey(ctx context.Context, key string) (*bpmn.Process, []byte, error)
	ListProcesses(ctx context.Context) ([]ProcessInfo, error)
	DeleteProcess(ctx context.Context, id string) error
}

// ProcessInfo holds process metadata
type ProcessInfo struct {
	ID             string    `json:"id"`
	ProcessKey     string    `json:"process_key"`
	Version        int       `json:"version"`
	Name           string    `json:"name"`
	DeploymentTime time.Time `json:"deployment_time"`
}

// PostgresProcessRepository implements ProcessRepository
type PostgresProcessRepository struct {
	db *DB
}

// NewProcessRepository creates a new process repository
func NewProcessRepository(db *DB) *PostgresProcessRepository {
	return &PostgresProcessRepository{db: db}
}

// DeployProcess deploys a new process version
func (r *PostgresProcessRepository) DeployProcess(ctx context.Context, process *bpmn.Process, definition []byte) error {
	if process == nil || process.ID == "" {
		return fmt.Errorf("process is nil or has no ID")
	}

	// Extract process key from ID (format: processKey_version)
	var processKey string
	var version int
	fmt.Sscanf(process.ID, "%s_%d", &processKey, &version)
	if processKey == "" {
		processKey = process.ID
		version = 1
	}

	query := `
		INSERT INTO process_definitions (id, process_key, version, name, definition, deployed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (process_key, version) 
		DO UPDATE SET definition = $5, deployed_at = NOW()
		RETURNING id
	`

	var id string
	err := r.db.GetDB().QueryRowContext(ctx, query,
		process.ID, processKey, version, process.Name, definition,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to deploy process: %w", err)
	}

	return nil
}

// GetProcessByID gets a process by ID
func (r *PostgresProcessRepository) GetProcessByID(ctx context.Context, id string) (*bpmn.Process, []byte, error) {
	query := `SELECT id, process_key, version, name, definition FROM process_definitions WHERE id = $1`

	var processKey, name string
	var version int
	var definition []byte

	err := r.db.GetDB().QueryRowContext(ctx, query, id).Scan(
		&id, &processKey, &version, &name, &definition,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("process not found: %s", id)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get process: %w", err)
	}

	// Parse BPMN definition
	process, err := bpmn.Parse(definition)
	if err != nil {
		return nil, definition, fmt.Errorf("failed to parse process definition: %w", err)
	}

	return process, definition, nil
}

// GetProcessByKey gets the latest process by key
func (r *PostgresProcessRepository) GetProcessByKey(ctx context.Context, key string) (*bpmn.Process, []byte, error) {
	query := `
		SELECT id, process_key, version, name, definition 
		FROM process_definitions 
		WHERE process_key = $1 
		ORDER BY version DESC 
		LIMIT 1
	`

	var id, processKey, name string
	var version int
	var definition []byte

	err := r.db.GetDB().QueryRowContext(ctx, query, key).Scan(
		&id, &processKey, &version, &name, &definition,
	)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("process not found: %s", key)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get process: %w", err)
	}

	process, err := bpmn.Parse(definition)
	if err != nil {
		return nil, definition, fmt.Errorf("failed to parse process definition: %w", err)
	}

	return process, definition, nil
}

// ListProcesses lists all process definitions
func (r *PostgresProcessRepository) ListProcesses(ctx context.Context) ([]ProcessInfo, error) {
	query := `
		SELECT pd.id, pd.process_key, pd.version, pd.name, pd.deployed_at
		FROM process_definitions pd
		INNER JOIN (
			SELECT process_key, MAX(version) as max_version
			FROM process_definitions
			GROUP BY process_key
		) latest ON pd.process_key = latest.process_key AND pd.version = latest.max_version
		ORDER BY pd.deployed_at DESC
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	defer rows.Close()

	var processes []ProcessInfo
	for rows.Next() {
		var p ProcessInfo
		if err := rows.Scan(&p.ID, &p.ProcessKey, &p.Version, &p.Name, &p.DeploymentTime); err != nil {
			return nil, fmt.Errorf("failed to scan process: %w", err)
		}
		processes = append(processes, p)
	}

	return processes, nil
}

// DeleteProcess deletes a process definition
func (r *PostgresProcessRepository) DeleteProcess(ctx context.Context, id string) error {
	query := `DELETE FROM process_definitions WHERE id = $1`
	_, err := r.db.GetDB().ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete process: %w", err)
	}
	return nil
}

// InstanceRepository handles process instance persistence
type InstanceRepository interface {
	CreateInstance(ctx context.Context, instance *statemachine.ProcessInstance) error
	GetInstanceByID(ctx context.Context, id string) (*statemachine.ProcessInstance, error)
	UpdateInstance(ctx context.Context, instance *statemachine.ProcessInstance) error
	UpdateInstanceWithTx(ctx context.Context, tx *sql.Tx, instance *statemachine.ProcessInstance) error
	ListInstances(ctx context.Context, filter InstanceFilter) ([]*statemachine.ProcessInstance, error)
}

// InstanceFilter filters process instances
type InstanceFilter struct {
	ProcessKey string
	Status     string
	FromTime   *time.Time
	ToTime     *time.Time
	Limit      int
	Offset     int
}

// TaskFilter filters user tasks
type TaskFilter struct {
	Assignee   string
	InstanceID string
	Status     string
}

// UserTask represents a user task
type UserTask struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	InstanceID   string                 `json:"instance_id"`
	ProcessDefID string                 `json:"process_definition_id"`
	Assignee     string                 `json:"assignee"`
	CreatedAt    time.Time              `json:"created_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	DueDate      *time.Time             `json:"due_date"`
	Variables    map[string]interface{} `json:"variables"`
}

// PostgresInstanceRepository implements InstanceRepository
type PostgresInstanceRepository struct {
	db *DB
}

// NewInstanceRepository creates a new instance repository
func NewInstanceRepository(db *DB) *PostgresInstanceRepository {
	return &PostgresInstanceRepository{db: db}
}

// CreateInstance creates a new process instance
func (r *PostgresInstanceRepository) CreateInstance(ctx context.Context, instance *statemachine.ProcessInstance) error {
	if instance == nil || instance.ID == "" {
		return fmt.Errorf("instance is nil or has no ID")
	}

	variablesJSON, err := json.Marshal(instance.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `
		INSERT INTO process_instances (id, process_key, status, variables, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id string
	err = r.db.GetDB().QueryRowContext(ctx, query,
		instance.ID,
		instance.ProcessKey,
		instance.Status,
		variablesJSON,
		instance.CreatedAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// GetInstanceByID gets a process instance by ID
func (r *PostgresInstanceRepository) GetInstanceByID(ctx context.Context, id string) (*statemachine.ProcessInstance, error) {
	query := `
		SELECT id, process_key, status, variables, created_at, started_at, completed_at, completed_by, error
		FROM process_instances 
		WHERE id = $1
	`

	var instance statemachine.ProcessInstance
	var variablesJSON []byte
	var startedAt, completedAt sql.NullTime
	var completedBy sql.NullString
	var errorJSON []byte

	err := r.db.GetDB().QueryRowContext(ctx, query, id).Scan(
		&instance.ID,
		&instance.ProcessKey,
		&instance.Status,
		&variablesJSON,
		&instance.CreatedAt,
		&startedAt,
		&completedAt,
		&completedBy,
		&errorJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Parse variables
	if err := json.Unmarshal(variablesJSON, &instance.Variables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	// Set nullable fields
	if startedAt.Valid {
		instance.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		instance.CompletedAt = &completedAt.Time
	}
	if completedBy.Valid {
		instance.CompletedBy = completedBy.String
	}

	return &instance, nil
}

// UpdateInstance updates a process instance (with optimistic locking)
func (r *PostgresInstanceRepository) UpdateInstance(ctx context.Context, instance *statemachine.ProcessInstance) error {
	tx, err := r.db.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := r.UpdateInstanceWithTx(ctx, tx, instance); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateInstanceWithTx updates a process instance within a transaction
func (r *PostgresInstanceRepository) UpdateInstanceWithTx(ctx context.Context, tx *sql.Tx, instance *statemachine.ProcessInstance) error {
	if instance == nil || instance.ID == "" {
		return fmt.Errorf("instance is nil or has no ID")
	}

	variablesJSON, err := json.Marshal(instance.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `
		UPDATE process_instances 
		SET status = $1, 
		    variables = $2, 
		    updated_at = NOW(),
		    version = version + 1
		WHERE id = $3
		RETURNING id
	`

	var id string
	err = tx.QueryRowContext(ctx, query,
		instance.Status,
		variablesJSON,
		instance.ID,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return fmt.Errorf("instance not found or version conflict: %s", instance.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

// ListInstances lists process instances with filters
func (r *PostgresInstanceRepository) ListInstances(ctx context.Context, filter InstanceFilter) ([]*statemachine.ProcessInstance, error) {
	query := `
		SELECT id, process_key, status, variables, created_at, started_at, completed_at
		FROM process_instances 
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.ProcessKey != "" {
		query += fmt.Sprintf(" AND process_key = $%d", argNum)
		args = append(args, filter.ProcessKey)
		argNum++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*statemachine.ProcessInstance
	for rows.Next() {
		var instance statemachine.ProcessInstance
		var variablesJSON []byte
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&instance.ID,
			&instance.ProcessKey,
			&instance.Status,
			&variablesJSON,
			&instance.CreatedAt,
			&startedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}

		json.Unmarshal(variablesJSON, &instance.Variables)
		if startedAt.Valid {
			instance.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			instance.CompletedAt = &completedAt.Time
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

// GetTasks gets user tasks based on filter
func (r *PostgresInstanceRepository) GetTasks(ctx context.Context, filter TaskFilter) ([]UserTask, error) {
	// Build a single optimized query using JOIN
	query := `
		SELECT ut.id, ut.name, ut.instance_id, ut.process_definition_id, ut.assignee, ut.created_at, ut.due_date, ut.variables
		FROM user_tasks ut
		INNER JOIN process_instances pi ON ut.instance_id = pi.id
		WHERE pi.status = 'active'
	`
	args := []interface{}{}
	argNum := 1

	if filter.Assignee != "" {
		query += fmt.Sprintf(" AND ut.assignee = $%d", argNum)
		args = append(args, filter.Assignee)
		argNum++
	}
	if filter.InstanceID != "" {
		query += fmt.Sprintf(" AND ut.instance_id = $%d", argNum)
		args = append(args, filter.InstanceID)
		argNum++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND ut.status = $%d", argNum)
		args = append(args, filter.Status)
	}

	query += " ORDER BY ut.created_at DESC LIMIT 100"

	rows, err := r.db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		// Log the error for debugging but don't fail completely
		log.Printf("GetTasks query error: %v", err)
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer rows.Close()

	var tasks []UserTask
	for rows.Next() {
		var task UserTask
		var dueDate sql.NullTime
		var variablesJSON []byte

		err := rows.Scan(&task.ID, &task.Name, &task.InstanceID, &task.ProcessDefID, &task.Assignee, &task.CreatedAt, &dueDate, &variablesJSON)
		if err != nil {
			log.Printf("GetTasks scan error: %v", err)
			continue
		}

		if dueDate.Valid {
			task.DueDate = &dueDate.Time
		}
		if err := json.Unmarshal(variablesJSON, &task.Variables); err != nil {
			log.Printf("GetTasks unmarshal variables error for task %s: %v", task.ID, err)
			task.Variables = make(map[string]interface{})
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetTasks rows error: %v", err)
	}

	return tasks, nil
}

// DelegateTask delegates a task to another user
func (r *PostgresInstanceRepository) DelegateTask(ctx context.Context, taskID, userID string) error {
	query := `
		UPDATE user_tasks
		SET assignee = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.GetDB().ExecContext(ctx, query, userID, taskID)
	if err != nil {
		return fmt.Errorf("failed to delegate task: %w", err)
	}

	return nil
}

// GetTaskHistory returns completed tasks
func (r *PostgresInstanceRepository) GetTaskHistory(ctx context.Context, limit int) ([]UserTask, error) {
	query := `
		SELECT ut.id, ut.name, ut.instance_id, ut.process_definition_id, ut.assignee, ut.created_at, ut.completed_at, ut.due_date, ut.variables
		FROM user_tasks ut
		WHERE ut.completed_at IS NOT NULL
		ORDER BY ut.completed_at DESC
		LIMIT $1
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query, limit)
	if err != nil {
		log.Printf("GetTaskHistory query error: %v", err)
		return nil, fmt.Errorf("failed to fetch task history: %w", err)
	}
	defer rows.Close()

	var tasks []UserTask
	for rows.Next() {
		var task UserTask
		var dueDate, completedAt sql.NullTime
		var variablesJSON []byte

		err := rows.Scan(&task.ID, &task.Name, &task.InstanceID, &task.ProcessDefID, &task.Assignee, &task.CreatedAt, &completedAt, &dueDate, &variablesJSON)
		if err != nil {
			log.Printf("GetTaskHistory scan error: %v", err)
			continue
		}

		if dueDate.Valid {
			task.DueDate = &dueDate.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		if err := json.Unmarshal(variablesJSON, &task.Variables); err != nil {
			log.Printf("GetTaskHistory unmarshal variables error for task %s: %v", task.ID, err)
			task.Variables = make(map[string]interface{})
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetTaskHistory rows error: %v", err)
	}

	return tasks, nil
}

// ClaimTask assigns a task to a user
func (r *PostgresInstanceRepository) ClaimTask(ctx context.Context, taskID, userID string) error {
	query := `
		UPDATE user_tasks 
		SET assignee = $1, status = 'claimed', updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.GetDB().ExecContext(ctx, query, userID, taskID)
	if err != nil {
		return fmt.Errorf("failed to claim task: %w", err)
	}
	return nil
}

// UnclaimTask removes assignment from a task
func (r *PostgresInstanceRepository) UnclaimTask(ctx context.Context, taskID string) error {
	query := `
		UPDATE user_tasks 
		SET assignee = NULL, status = 'pending', updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.GetDB().ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to unclaim task: %w", err)
	}
	return nil
}

// GetTokens returns tokens for an instance
func (r *PostgresInstanceRepository) GetTokens(ctx context.Context, instanceID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, node_id, node_name, token_key, token_value, status, created_at
		FROM process_tokens 
		WHERE instance_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tokens: %w", err)
	}
	defer rows.Close()

	var tokens []map[string]interface{}
	for rows.Next() {
		var id, nodeID, nodeName, tokenKey, tokenValue, status string
		var createdAt time.Time

		err := rows.Scan(&id, &nodeID, &nodeName, &tokenKey, &tokenValue, &status, &createdAt)
		if err != nil {
			log.Printf("GetTokens scan error: %v", err)
			continue
		}

		tokens = append(tokens, map[string]interface{}{
			"id":          id,
			"node_id":     nodeID,
			"node_name":   nodeName,
			"token_key":   tokenKey,
			"token_value": tokenValue,
			"status":      status,
			"created_at":  createdAt,
		})
	}

	return tokens, nil
}

// EventRepository handles process event persistence
type EventRepository interface {
	AppendEvent(ctx context.Context, event *ProcessEvent) error
	ListByInstance(ctx context.Context, instanceID string) ([]ProcessEvent, error)
}

// ProcessEvent represents a process event
type ProcessEvent struct {
	ID                  string          `json:"id"`
	InstanceID          string          `json:"instance_id"`
	ProcessDefinitionID int             `json:"process_definition_id"`
	NodeID              string          `json:"node_id,omitempty"`
	NodeName            string          `json:"node_name,omitempty"`
	NodeType            string          `json:"node_type,omitempty"`
	Action              string          `json:"action"`
	Payload             json.RawMessage `json:"payload,omitempty"`
	OccurredAt          time.Time       `json:"occurred_at"`
}

// PostgresEventRepository implements EventRepository
type PostgresEventRepository struct {
	db *DB
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *DB) *PostgresEventRepository {
	return &PostgresEventRepository{db: db}
}

// AppendEvent appends a new event
func (r *PostgresEventRepository) AppendEvent(ctx context.Context, event *ProcessEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}

	payload, _ := json.Marshal(event.Payload)

	query := `
		INSERT INTO process_events (id, instance_id, process_definition_id, node_id, node_name, node_type, action, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.GetDB().ExecContext(ctx, query,
		event.ID,
		event.InstanceID,
		event.ProcessDefinitionID,
		event.NodeID,
		event.NodeName,
		event.NodeType,
		event.Action,
		payload,
		event.OccurredAt,
	)

	if err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	return nil
}

// ListByInstance lists events for an instance
func (r *PostgresEventRepository) ListByInstance(ctx context.Context, instanceID string) ([]ProcessEvent, error) {
	query := `
		SELECT id, instance_id, node_id, node_name, node_type, action, payload, occurred_at
		FROM process_events 
		WHERE instance_id = $1
		ORDER BY occurred_at ASC
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []ProcessEvent
	for rows.Next() {
		var event ProcessEvent
		var payload []byte
		var occurredAt time.Time
		if err := rows.Scan(
			&event.ID,
			&event.InstanceID,
			&event.NodeID,
			&event.NodeName,
			&event.NodeType,
			&event.Action,
			&payload,
			&occurredAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		event.OccurredAt = occurredAt
		_ = json.Unmarshal(payload, &event.Payload)
		events = append(events, event)
	}

	return events, nil
}

// GetEvents returns events for an instance (alias for compatibility)
func (r *PostgresEventRepository) GetEvents(ctx context.Context, instanceID string) ([]map[string]interface{}, error) {
	events, err := r.ListByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(events))
	for i, e := range events {
		result[i] = map[string]interface{}{
			"id":          e.ID,
			"instance_id": e.InstanceID,
			"node_id":     e.NodeID,
			"node_name":   e.NodeName,
			"node_type":   e.NodeType,
			"action":      e.Action,
			"payload":     e.Payload,
			"occurred_at": e.OccurredAt,
		}
	}

	return result, nil
}
