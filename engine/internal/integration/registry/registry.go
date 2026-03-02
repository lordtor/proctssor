package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workflow-engine/v2/internal/integration/postgres"
)

// Service represents a registered service in the registry
type Service struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"` // worker, handler, external
	Endpoint     string            `json:"endpoint"`
	Metadata     map[string]string `json:"metadata"`
	Status       string            `json:"status"` // active, inactive, failed
	HeartbeatAt  time.Time         `json:"heartbeat_at"`
	RegisteredAt time.Time         `json:"registered_at"`
}

// RegistryRepository handles service registry persistence
type RegistryRepository interface {
	Register(ctx context.Context, service *Service) error
	Heartbeat(ctx context.Context, serviceID string) error
	Discover(ctx context.Context, serviceType string) ([]*Service, error)
	DiscoverByName(ctx context.Context, name string) (*Service, error)
	ListAll(ctx context.Context) ([]*Service, error)
	Unregister(ctx context.Context, serviceID string) error
}

// PostgresRegistryRepository implements RegistryRepository
type PostgresRegistryRepository struct {
	db *postgres.DB
}

// NewRegistryRepository creates a new registry repository
func NewRegistryRepository(db *postgres.DB) *PostgresRegistryRepository {
	return &PostgresRegistryRepository{db: db}
}

// Register registers a new service
func (r *PostgresRegistryRepository) Register(ctx context.Context, service *Service) error {
	if service.ID == "" {
		service.ID = uuid.New().String()
	}
	if service.Status == "" {
		service.Status = "active"
	}
	if service.RegisteredAt.IsZero() {
		service.RegisteredAt = time.Now()
	}
	if service.HeartbeatAt.IsZero() {
		service.HeartbeatAt = service.RegisteredAt
	}

	metadataJSON, err := json.Marshal(service.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO service_registry (id, name, type, endpoint, metadata, status, heartbeat_at, registered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (name, type) 
		DO UPDATE SET endpoint = $4, metadata = $5, status = $6, heartbeat_at = $7
		RETURNING id
	`

	var id string
	err = r.db.GetDB().QueryRowContext(ctx, query,
		service.ID,
		service.Name,
		service.Type,
		service.Endpoint,
		metadataJSON,
		service.Status,
		service.HeartbeatAt,
		service.RegisteredAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	return nil
}

// Heartbeat updates the heartbeat timestamp for a service
func (r *PostgresRegistryRepository) Heartbeat(ctx context.Context, serviceID string) error {
	query := `
		UPDATE service_registry 
		SET heartbeat_at = NOW(), status = 'active'
		WHERE id = $1
	`

	result, err := r.db.GetDB().ExecContext(ctx, query, serviceID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	return nil
}

// Discover finds active services by type
func (r *PostgresRegistryRepository) Discover(ctx context.Context, serviceType string) ([]*Service, error) {
	query := `
		SELECT id, name, type, endpoint, metadata, status, heartbeat_at, registered_at
		FROM service_registry 
		WHERE type = $1 
		  AND status = 'active'
		  AND heartbeat_at > NOW() - INTERVAL '30 seconds'
		ORDER BY registered_at ASC
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query, serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var service Service
		var metadataJSON []byte

		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Type,
			&service.Endpoint,
			&metadataJSON,
			&service.Status,
			&service.HeartbeatAt,
			&service.RegisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}

		json.Unmarshal(metadataJSON, &service.Metadata)
		services = append(services, &service)
	}

	return services, nil
}

// DiscoverByName finds a service by name
func (r *PostgresRegistryRepository) DiscoverByName(ctx context.Context, name string) (*Service, error) {
	query := `
		SELECT id, name, type, endpoint, metadata, status, heartbeat_at, registered_at
		FROM service_registry 
		WHERE name = $1
		  AND status = 'active'
		  AND heartbeat_at > NOW() - INTERVAL '30 seconds'
		LIMIT 1
	`

	var service Service
	var metadataJSON []byte

	err := r.db.GetDB().QueryRowContext(ctx, query, name).Scan(
		&service.ID,
		&service.Name,
		&service.Type,
		&service.Endpoint,
		&metadataJSON,
		&service.Status,
		&service.HeartbeatAt,
		&service.RegisteredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("service not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	json.Unmarshal(metadataJSON, &service.Metadata)
	return &service, nil
}

// ListAll lists all registered services
func (r *PostgresRegistryRepository) ListAll(ctx context.Context) ([]*Service, error) {
	query := `
		SELECT id, name, type, endpoint, metadata, status, heartbeat_at, registered_at
		FROM service_registry 
		ORDER BY registered_at DESC
	`

	rows, err := r.db.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var service Service
		var metadataJSON []byte

		err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Type,
			&service.Endpoint,
			&metadataJSON,
			&service.Status,
			&service.HeartbeatAt,
			&service.RegisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}

		json.Unmarshal(metadataJSON, &service.Metadata)
		services = append(services, &service)
	}

	return services, nil
}

// Unregister removes a service from the registry
func (r *PostgresRegistryRepository) Unregister(ctx context.Context, serviceID string) error {
	query := `DELETE FROM service_registry WHERE id = $1`

	_, err := r.db.GetDB().ExecContext(ctx, query, serviceID)
	if err != nil {
		return fmt.Errorf("failed to unregister service: %w", err)
	}

	return nil
}

// CleanStale removes stale service registrations
func (r *PostgresRegistryRepository) CleanStale(ctx context.Context, threshold time.Duration) error {
	query := `
		UPDATE service_registry 
		SET status = 'inactive'
		WHERE heartbeat_at < NOW() - $1
		  AND status = 'active'
	`

	_, err := r.db.GetDB().ExecContext(ctx, query, threshold)
	if err != nil {
		return fmt.Errorf("failed to clean stale services: %w", err)
	}

	return nil
}
