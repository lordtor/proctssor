-- BPMN Workflow Platform - Database Indexes
-- Part 2: Indexes for Performance

-- =============================================================================
-- Process Definitions Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_process_definitions_name ON process_definitions(name);
CREATE INDEX IF NOT EXISTS idx_process_definitions_category ON process_definitions(category);
CREATE INDEX IF NOT EXISTS idx_process_definitions_deployed ON process_definitions(deployed_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_definitions_created_by ON process_definitions(created_by);

-- Full text search on name and description
CREATE INDEX IF NOT EXISTS idx_process_definitions_fts ON process_definitions 
    USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- =============================================================================
-- Process Instances Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_process_instances_definition ON process_instances(process_definition_id);
CREATE INDEX IF NOT EXISTS idx_process_instances_status ON process_instances(status);
CREATE INDEX IF NOT EXISTS idx_process_instances_current_node ON process_instances(current_node);
CREATE INDEX IF NOT EXISTS idx_process_instances_business_key ON process_instances(business_key);
CREATE INDEX IF NOT EXISTS idx_process_instances_parent ON process_instances(parent_instance_id);
CREATE INDEX IF NOT EXISTS idx_process_instances_root ON process_instances(root_instance_id);
CREATE INDEX IF NOT EXISTS idx_process_instances_started ON process_instances(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_instances_completed ON process_instances(completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_instances_tenant ON process_instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_process_instances_created ON process_instances(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_instances_updated ON process_instances(updated_at DESC);

-- =============================================================================
-- Process Events Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_process_events_instance ON process_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_process_events_instance_time ON process_events(instance_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_events_definition ON process_events(process_definition_id);
CREATE INDEX IF NOT EXISTS idx_process_events_action ON process_events(action);
CREATE INDEX IF NOT EXISTS idx_process_events_node ON process_events(node_id);
CREATE INDEX IF NOT EXISTS idx_process_events_occurred ON process_events(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_events_trace ON process_events(trace_id);
CREATE INDEX IF NOT EXISTS idx_process_events_correlation ON process_events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_process_events_user ON process_events(user_id);

-- =============================================================================
-- Service Registry Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_service_registry_name ON service_registry(name);
CREATE INDEX IF NOT EXISTS idx_service_registry_status ON service_registry(service_status);
CREATE INDEX IF NOT EXISTS idx_service_registry_heartbeat ON service_registry(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_service_registry_nats_subject ON service_registry(nats_subject);

-- =============================================================================
-- Process Tokens Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_process_tokens_instance ON process_tokens(instance_id);
CREATE INDEX IF NOT EXISTS idx_process_tokens_node ON process_tokens(node_id);
CREATE INDEX IF NOT EXISTS idx_process_tokens_key ON process_tokens(token_key);
CREATE INDEX IF NOT EXISTS idx_process_tokens_status ON process_tokens(status);
CREATE INDEX IF NOT EXISTS idx_process_tokens_expires ON process_tokens(expires_at);

-- =============================================================================
-- User Tasks Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_user_tasks_instance ON user_tasks(instance_id);
CREATE INDEX IF NOT EXISTS idx_user_tasks_assignee ON user_tasks(assignee);
CREATE INDEX IF NOT EXISTS idx_user_tasks_status ON user_tasks(status);
CREATE INDEX IF NOT EXISTS idx_user_tasks_due_date ON user_tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_user_tasks_candidate_users ON user_tasks USING gin(candidate_users);
CREATE INDEX IF NOT EXISTS idx_user_tasks_candidate_groups ON user_tasks USING gin(candidate_groups);
CREATE INDEX IF NOT EXISTS idx_user_tasks_priority ON user_tasks(priority DESC);

-- Composite indexes
CREATE INDEX IF NOT EXISTS idx_user_tasks_assignee_status ON user_tasks(assignee, status);

-- =============================================================================
-- Timer Jobs Indexes
-- =============================================================================
CREATE INDEX IF NOT EXISTS idx_timer_jobs_instance ON timer_jobs(instance_id);
CREATE INDEX IF NOT EXISTS idx_timer_jobs_due_date ON timer_jobs(due_date ASC);
CREATE INDEX IF NOT EXISTS idx_timer_jobs_next_execution ON timer_jobs(next_execution_at ASC);
CREATE INDEX IF NOT EXISTS idx_timer_jobs_status ON timer_jobs(status);
CREATE INDEX IF NOT EXISTS idx_timer_jobs_type ON timer_jobs(timer_type);
