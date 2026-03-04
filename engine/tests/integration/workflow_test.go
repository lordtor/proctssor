package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/workflow-engine/v2/internal/api/websocket"
	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/executor"
	natsClient "github.com/workflow-engine/v2/internal/integration/nats"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"github.com/workflow-engine/v2/internal/integration/registry"
	"github.com/workflow-engine/v2/internal/service"
	"go.uber.org/zap"
)

// WorkflowTestSuite интеграционные тесты для workflow engine
type WorkflowTestSuite struct {
	suite.Suite
	ctx             context.Context
	postgresC       testcontainers.Container
	natsC           testcontainers.Container
	db              *postgres.DB
	processRepo     *postgres.PostgresProcessRepository
	instanceRepo    *postgres.PostgresInstanceRepository
	eventRepo       *postgres.PostgresEventRepository
	natsConn        *nats.Conn
	js              nats.JetStreamContext
	publisher       *natsClient.Publisher
	registryRepo    registry.RegistryRepository
	cache           *registry.LRUCache
	instanceService *service.InstanceService
	executor        *executor.DefaultExecutor
	wsHub           *websocket.Hub
	logger          *zap.Logger
}

// TestWorkflowTestSuite запускает все тесты
func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

// SetupSuite настраивает тестовое окружение
func (s *WorkflowTestSuite) SetupSuite() {
	s.ctx = context.Background()

	var err error
	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	// Запускаем PostgreSQL
	s.setupPostgres()

	// Запускаем NATS
	s.setupNATS()

	// Инициализация компонентов
	s.setupComponents()
}

func (s *WorkflowTestSuite) setupPostgres() {
	s.T().Log("Starting PostgreSQL container...")

	postgresC, err := tcpostgres.Run(
		s.ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("workflow_test"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	s.Require().NoError(err)
	s.postgresC = postgresC

	// Получаем mapped port для подключения
	host, err := postgresC.Host(s.ctx)
	s.Require().NoError(err)
	port, err := postgresC.MappedPort(s.ctx, "5432")
	s.Require().NoError(err)

	s.T().Logf("PostgreSQL host: %s, port: %s", host, port.Port())

	// Создаем подключение к БД с правильным портом
	s.db, err = postgres.NewDB(postgres.Config{
		Host:     host,
		Port:     port.Int(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "workflow_test",
		SSLMode:  "disable",
	})
	s.Require().NoError(err)

	// Запускаем миграции
	err = runMigrations(s.db.GetDB())
	s.Require().NoError(err)
}

func (s *WorkflowTestSuite) setupNATS() {
	s.T().Log("Starting NATS container...")

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp", "8222/tcp"},
		Cmd:          []string{"--js", "--store_dir", "/data/jetstream"},
		WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(30 * time.Second),
	}

	natsC, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	s.Require().NoError(err)
	s.natsC = natsC

	// Получаем endpoint
	host, err := natsC.Host(s.ctx)
	s.Require().NoError(err)
	port, err := natsC.MappedPort(s.ctx, "4222")
	s.Require().NoError(err)

	natsURL := fmt.Sprintf("nats://%s:%s", host, port.Port())
	s.T().Logf("NATS URL: %s", natsURL)

	// Подключаемся к NATS
	s.natsConn, err = nats.Connect(natsURL)
	s.Require().NoError(err)

	// Получаем JetStream context
	s.js, err = s.natsConn.JetStream()
	s.Require().NoError(err)

	// Создаем streams
	err = s.setupNATSStreams()
	s.Require().NoError(err)

	// Создаем publisher
	s.publisher, err = natsClient.NewPublisher(natsClient.PublisherConfig{
		URL:     natsURL,
		Timeout: 30 * time.Second,
	})
	s.Require().NoError(err)
}

func (s *WorkflowTestSuite) setupNATSStreams() error {
	// Создаем command stream
	_, err := s.js.AddStream(&nats.StreamConfig{
		Name:      "WORKFLOW_CMD",
		Subjects:  []string{"wf.cmd.>"},
		Storage:   nats.FileStorage,
		MaxMsgs:   10000,
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return err
	}

	// Создаем response stream
	_, err = s.js.AddStream(&nats.StreamConfig{
		Name:      "WORKFLOW_RESP",
		Subjects:  []string{"wf.resp.>"},
		Storage:   nats.FileStorage,
		MaxMsgs:   10000,
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return err
	}

	// Создаем DLQ stream
	_, err = s.js.AddStream(&nats.StreamConfig{
		Name:      "WORKFLOW_DLQ",
		Subjects:  []string{"wf.dlq.>"},
		Storage:   nats.FileStorage,
		MaxMsgs:   10000,
		Retention: nats.WorkQueuePolicy,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return err
	}

	return nil
}

func (s *WorkflowTestSuite) setupComponents() {
	// Repositories
	s.processRepo = postgres.NewProcessRepository(s.db)
	s.instanceRepo = postgres.NewInstanceRepository(s.db)
	s.eventRepo = postgres.NewEventRepository(s.db)

	// Registry
	s.registryRepo = registry.NewRegistryRepository(s.db)

	// Cache
	s.cache = registry.NewLRUCache(1000, 5*time.Minute)

	// WebSocket Hub
	s.wsHub = websocket.NewHub()
	go s.wsHub.Run()

	// Executor
	s.executor = executor.NewExecutor(s.cache, s.publisher, s.logger)

	// Instance Service
	s.instanceService = service.NewInstanceService(
		s.processRepo,
		s.instanceRepo,
		s.eventRepo,
		s.executor,
		s.publisher,
		s.wsHub,
		s.logger,
	)
}

// TearDownSuite очищает ресурсы
func (s *WorkflowTestSuite) TearDownSuite() {
	if s.publisher != nil {
		s.publisher.Close()
	}
	if s.natsConn != nil {
		s.natsConn.Close()
	}
	if s.natsC != nil {
		s.natsC.Terminate(s.ctx)
	}
	if s.postgresC != nil {
		s.postgresC.Terminate(s.ctx)
	}
}

// SetupTest выполняется перед каждым тестом
func (s *WorkflowTestSuite) SetupTest() {
	// Очистка данных перед каждым тестом
	s.cleanupDatabase()
}

func (s *WorkflowTestSuite) cleanupDatabase() {
	tables := []string{
		"process_instances",
		"process_tokens",
		"process_events",
		"user_tasks",
		"service_registry",
		"process_definitions",
	}

	for _, table := range tables {
		_, err := s.db.GetDB().ExecContext(s.ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			s.T().Logf("Failed to truncate %s: %v", table, err)
		}
	}
}

// TestRegistryHeartbeat тест heartbeat registry
func (s *WorkflowTestSuite) TestRegistryHeartbeat() {
	// Регистрируем сервис
	service := &registry.Service{
		ID:       "test-service-1",
		Name:     "test-service",
		Type:     "handler",
		Endpoint: "http://localhost:8080",
		Metadata: map[string]string{"key": "value"},
	}

	err := s.registryRepo.Register(s.ctx, service)
	s.Require().NoError(err)

	// Проверяем что сервис доступен
	services, err := s.registryRepo.Discover(s.ctx, "handler")
	s.Require().NoError(err)
	s.Len(services, 1)
	s.Equal("test-service", services[0].Name)

	// Обновляем heartbeat
	err = s.registryRepo.Heartbeat(s.ctx, service.ID)
	s.Require().NoError(err)
}

// TestCacheOperations тест операций с кэшем
func (s *WorkflowTestSuite) TestCacheOperations() {
	// Set value in cache
	s.cache.Set("test-key", "test-value")

	// Get value from cache
	val, ok := s.cache.Get(s.ctx, "test-key")
	s.True(ok)
	s.Equal("test-value", val)

	// Delete from cache
	s.cache.Delete("test-key")

	// Verify deleted
	_, ok = s.cache.Get(s.ctx, "test-key")
	s.False(ok)
}

// TestNATSStreams тест создания streams
func (s *WorkflowTestSuite) TestNATSStreams() {
	// Проверяем что streams созданы
	streams := []string{"WORKFLOW_CMD", "WORKFLOW_RESP", "WORKFLOW_DLQ"}
	for _, streamName := range streams {
		info, err := s.js.StreamInfo(streamName)
		s.Require().NoError(err, "Stream %s should exist", streamName)
		s.NotNil(info)
	}
}

// TestNatsPublisher тест публикации сообщений
func (s *WorkflowTestSuite) TestNatsPublisher() {
	// Публикуем тестовое сообщение
	cmd := &natsClient.WorkflowCommand{
		CommandType: natsClient.CommandTypeServiceTask,
		InstanceID:  "test-instance",
		NodeID:      "test-node",
		ServiceName: "test-service",
		Operation:   "test",
		InputVariables: map[string]interface{}{
			"key": "value",
		},
		MaxRetries: 3,
	}

	err := s.publisher.PublishCommand(s.ctx, cmd)
	s.Require().NoError(err)
}

// TestBPMNParse тест парсинга BPMN
func (s *WorkflowTestSuite) TestBPMNParse() {
	bpmnXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="testProcess" name="Test Process" isExecutable="true">
    <startEvent id="start" name="Start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="userTask"/>
    <userTask id="userTask" name="Review Task"/>
    <sequenceFlow id="flow2" sourceRef="userTask" targetRef="end"/>
    <endEvent id="end" name="End"/>
  </process>
</definitions>`)

	process, err := bpmn.Parse(bpmnXML)
	s.Require().NoError(err)
	s.NotNil(process)
	s.Equal("testProcess", process.ID)
	s.Equal("Test Process", process.Name)
}

// runMigrations выполняет SQL миграции
func runMigrations(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS process_definitions (
	id TEXT PRIMARY KEY,
	process_key TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	bpmn_xml TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'draft',
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS process_instances (
	id TEXT PRIMARY KEY,
	process_id TEXT NOT NULL,
	business_key TEXT,
	status TEXT NOT NULL,
	variables JSONB DEFAULT '{}',
	current_node_id TEXT,
	started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	completed_at TIMESTAMP WITH TIME ZONE,
	suspended_at TIMESTAMP WITH TIME ZONE,
	initiator TEXT
);

CREATE TABLE IF NOT EXISTS process_tokens (
	id TEXT PRIMARY KEY,
	instance_id TEXT NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
	node_id TEXT NOT NULL,
	status TEXT NOT NULL,
	variables JSONB DEFAULT '{}',
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS process_events (
	id TEXT PRIMARY KEY,
	instance_id TEXT NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
	event_type TEXT NOT NULL,
	node_id TEXT,
	payload JSONB,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_tasks (
	id TEXT PRIMARY KEY,
	instance_id TEXT NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
	node_id TEXT NOT NULL,
	name TEXT NOT NULL,
	assignee TEXT,
	candidate_users TEXT[],
	candidate_groups TEXT[],
	form_key TEXT,
	variables JSONB DEFAULT '{}',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	completed_at TIMESTAMP WITH TIME ZONE,
	completed_by TEXT
);

CREATE TABLE IF NOT EXISTS service_registry (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	endpoint TEXT NOT NULL,
	metadata JSONB DEFAULT '{}',
	status TEXT NOT NULL DEFAULT 'active',
	heartbeat_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
	UNIQUE(name, type)
);
`

	_, err := db.Exec(schema)
	return err
}
