-- BPMN Workflow Platform - Cron Jobs
-- Part 4: Scheduled Background Jobs
-- NOTE: pg_cron extension is not installed in the base PostgreSQL image
-- These jobs should be implemented at the application level or via external scheduler

-- =============================================================================
-- Job 1: Cleanup Dead Services
-- =============================================================================
-- Remove services that haven't sent heartbeat for more than 2 minutes
-- Implementation: Run via application scheduler every 2 minutes
/*
    UPDATE service_registry
    SET service_status = 'unresponsive'
    WHERE service_status NOT IN ('unhealthy', 'deregistered')
      AND last_heartbeat < NOW() - INTERVAL '2 minutes';
    
    UPDATE service_registry
    SET service_status = 'unhealthy'
    WHERE service_status = 'unresponsive'
      AND last_heartbeat < NOW() - INTERVAL '5 minutes';
    
    UPDATE service_registry
    SET service_status = 'deregistered'
    WHERE service_status = 'unhealthy'
      AND last_heartbeat < NOW() - INTERVAL '10 minutes';
*/

-- =============================================================================
-- Job 2: Timeout Stuck Instances
-- =============================================================================
-- Mark instances as error if stuck for more than 1 hour
/*
    UPDATE process_instances
    SET status = 'error',
        error_message = 'Process timeout - no activity for 1 hour',
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'active'
      AND updated_at < NOW() - INTERVAL '1 hour'
      AND retry_count >= max_retries;
*/

-- =============================================================================
-- Job 3: Resume Suspended Instances
-- =============================================================================
-- Resume instances when suspension time expires
/*
    UPDATE process_instances
    SET status = 'active',
        suspended_until = NULL,
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'suspended'
      AND suspended_until IS NOT NULL
      AND suspended_until <= NOW();
*/

-- =============================================================================
-- Job 4: Process Timer Jobs
-- =============================================================================
-- Execute pending timers that are due
/*
    UPDATE timer_jobs
    SET status = 'running',
        attempt_count = attempt_count + 1,
        last_executed_at = CURRENT_TIMESTAMP
    WHERE status = 'pending'
      AND due_date <= NOW();
*/

-- =============================================================================
-- Job 5: Cleanup Expired Tokens
-- =============================================================================
-- Mark expired tokens as expired
/*
    UPDATE process_tokens
    SET status = 'expired',
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'waiting'
      AND expires_at IS NOT NULL
      AND expires_at <= NOW();
*/

-- =============================================================================
-- Job 6: Auto-complete Timed-out Tasks
-- =============================================================================
-- Complete tasks that are past due date
/*
    UPDATE user_tasks
    SET status = 'error',
        error_message = 'Task auto-timeout: past due date',
        completed_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'pending'
      AND due_date IS NOT NULL
      AND due_date <= NOW() - INTERVAL '24 hours';
*/

-- =============================================================================
-- Helper Function: Manual Job Execution
-- =============================================================================
-- Run cleanup manually
CREATE OR REPLACE FUNCTION run_cleanup_dead_services() RETURNS VOID AS $$
BEGIN
    UPDATE service_registry
    SET service_status = 'unresponsive'
    WHERE service_status NOT IN ('unhealthy', 'deregistered')
      AND last_heartbeat < NOW() - INTERVAL '2 minutes';
    
    UPDATE service_registry
    SET service_status = 'unhealthy'
    WHERE service_status = 'unresponsive'
      AND last_heartbeat < NOW() - INTERVAL '5 minutes';
    
    UPDATE service_registry
    SET service_status = 'deregistered'
    WHERE service_status = 'unhealthy'
      AND last_heartbeat < NOW() - INTERVAL '10 minutes';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION run_timeout_stuck_instances() RETURNS VOID AS $$
BEGIN
    UPDATE process_instances
    SET status = 'error',
        error_message = 'Process timeout - no activity for 1 hour',
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'active'
      AND updated_at < NOW() - INTERVAL '1 hour'
      AND retry_count >= max_retries;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION run_resume_suspended_instances() RETURNS VOID AS $$
BEGIN
    UPDATE process_instances
    SET status = 'active',
        suspended_until = NULL,
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'suspended'
      AND suspended_until IS NOT NULL
      AND suspended_until <= NOW();
END;
$$ LANGUAGE plpgsql;
