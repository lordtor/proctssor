-- BPMN Workflow Platform - Database Schemas
-- Part 1: Core Tables

-- Drop existing tables for clean initialization
DROP TABLE IF EXISTS timer_jobs CASCADE;
DROP TABLE IF EXISTS user_tasks CASCADE;
DROP TABLE IF EXISTS process_tokens CASCADE;
DROP TABLE IF EXISTS service_registry CASCADE;
DROP TABLE IF EXISTS process_events CASCADE;
DROP TABLE IF EXISTS process_instances CASCADE;
DROP TABLE IF EXISTS process_definitions CASCADE;

-- Drop types
DROP TYPE IF EXISTS instance_status CASCADE;
DROP TYPE IF EXISTS node_action CASCADE;
DROP TYPE IF EXISTS node_type CASCADE;
DROP TYPE IF EXISTS service_status CASCADE;
DROP TYPE IF EXISTS token_status CASCADE;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =============================================================================
-- Process Definitions (BPMN Process Templates)
-- =============================================================================
CREATE TABLE process_definitions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    bpmn_xml TEXT NOT NULL,
    version VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    category VARCHAR(128),
    tags JSONB DEFAULT '[]',
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_pd_name_version ON process_definitions(name, version);

-- =============================================================================
-- Process Instances (Running Workflows)
-- =============================================================================
CREATE TYPE instance_status AS ENUM (
    'pending', 
    'active', 
    'completed', 
    'error', 
    'suspended',
    'terminated',
    'cancelled'
);

CREATE TABLE process_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    process_definition_id SERIAL NOT NULL REFERENCES process_definitions(id) ON DELETE RESTRICT,
    process_key VARCHAR(255) NOT NULL DEFAULT '',
    status instance_status NOT NULL DEFAULT 'pending',
    current_node VARCHAR(255),
    previous_node VARCHAR(255),
    variables JSONB DEFAULT '{}',
    business_key VARCHAR(255),
    parent_instance_id UUID REFERENCES process_instances(id) ON DELETE SET NULL,
    root_instance_id UUID REFERENCES process_instances(id) ON DELETE SET NULL,
    version INTEGER NOT NULL DEFAULT 1,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_message TEXT,
    error_details JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    suspended_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(128)
);

-- =============================================================================
-- Process Events (Audit Trail)
-- =============================================================================
CREATE TYPE node_action AS ENUM (
    'node_entered',
    'node_exited',
    'workflow_started',
    'workflow_completed',
    'workflow_error',
    'workflow_terminated',
    'task_completed',
    'task_skipped',
    'timer_fired',
    'signal_received',
    'message_received',
    'variable_changed',
    'subprocess_started',
    'subprocess_completed'
);

CREATE TYPE node_type AS ENUM (
    'startEvent',
    'endEvent',
    'userTask',
    'serviceTask',
    'scriptTask',
    'manualTask',
    'exclusiveGateway',
    'parallelGateway',
    'inclusiveGateway',
    'intermediateCatchEvent',
    'intermediateThrowEvent',
    'subProcess',
    'callActivity'
);

CREATE TABLE process_events (
    id BIGSERIAL PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
    process_definition_id SERIAL NOT NULL REFERENCES process_definitions(id) ON DELETE CASCADE,
    node_id VARCHAR(255),
    node_name VARCHAR(255),
    node_type node_type,
    action node_action NOT NULL,
    payload JSONB DEFAULT '{}',
    previous_payload JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    trace_id VARCHAR(256),
    span_id VARCHAR(256),
    correlation_id VARCHAR(256),
    user_id VARCHAR(255),
    session_id VARCHAR(256),
    ip_address INET,
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- Service Registry (Microservices)
-- =============================================================================
CREATE TYPE service_status AS ENUM (
    'registered',
    'healthy',
    'unhealthy',
    'unresponsive',
    'deregistered'
);

CREATE TABLE service_registry (
    id SERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    version VARCHAR(32),
    description TEXT,
    openapi_url TEXT,
    health_url TEXT,
    nats_subject VARCHAR(128),
    actions JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    service_status service_status DEFAULT 'registered',
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    heartbeat_timeout INTERVAL DEFAULT interval '2 minutes',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    registered_by VARCHAR(255)
);

CREATE UNIQUE INDEX idx_sr_name_version ON service_registry(name, version);

-- =============================================================================
-- Process Tokens (Tracking)
-- =============================================================================
CREATE TYPE token_status AS ENUM (
    'waiting',
    'active',
    'consumed',
    'expired',
    'cancelled'
);

CREATE TABLE process_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    node_name VARCHAR(255),
    token_key VARCHAR(128) NOT NULL,
    token_value TEXT,
    status token_status DEFAULT 'waiting',
    expires_at TIMESTAMP WITH TIME ZONE,
    consumed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(instance_id, node_id, token_key)
);

-- =============================================================================
-- User Tasks
-- =============================================================================
CREATE TABLE user_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES process_instances(id) ON DELETE CASCADE,
    task_id VARCHAR(255) NOT NULL,
    task_name VARCHAR(255),
    task_type VARCHAR(64),
    assignee VARCHAR(255),
    candidate_users JSONB DEFAULT '[]',
    candidate_groups JSONB DEFAULT '[]',
    status VARCHAR(32) DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    form_data JSONB DEFAULT '{}',
    form_schema JSONB,
    due_date TIMESTAMP WITH TIME ZONE,
    follow_up_date TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(instance_id, task_id)
);

-- =============================================================================
-- Timer Jobs
-- =============================================================================
CREATE TABLE timer_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES process_instances(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    token_id VARCHAR(255),
    timer_type VARCHAR(64) NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    repeat_interval INTERVAL,
    max_attempts INTEGER DEFAULT 1,
    attempt_count INTEGER DEFAULT 0,
    handler_config JSONB DEFAULT '{}',
    status VARCHAR(32) DEFAULT 'pending',
    last_executed_at TIMESTAMP WITH TIME ZONE,
    next_execution_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- Triggers for updated_at
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_process_definitions_updated_at 
    BEFORE UPDATE ON process_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_process_instances_updated_at 
    BEFORE UPDATE ON process_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_service_registry_updated_at 
    BEFORE UPDATE ON service_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_process_tokens_updated_at 
    BEFORE UPDATE ON process_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_tasks_updated_at 
    BEFORE UPDATE ON user_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_timer_jobs_updated_at 
    BEFORE UPDATE ON timer_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
