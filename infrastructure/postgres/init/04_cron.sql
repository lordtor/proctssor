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

-- =============================================================================
-- Job 7: Process Timer Jobs via pg_cron
-- =============================================================================
CREATE OR REPLACE FUNCTION process_due_timer_jobs() RETURNS VOID AS $$
DECLARE
    job_record RECORD;
BEGIN
    -- Get pending timers that are due
    FOR job_record IN
        SELECT id, instance_id, node_id, token_id, timer_type, due_date,
               repeat_interval, max_attempts, attempt_count
        FROM timer_jobs
        WHERE status = 'pending'
          AND due_date <= NOW()
        ORDER BY due_date ASC
        LIMIT 100
    LOOP
        -- Mark as running
        UPDATE timer_jobs
        SET status = 'running',
            attempt_count = attempt_count + 1,
            last_executed_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = job_record.id;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Function to complete timer jobs (called by external trigger)
CREATE OR REPLACE FUNCTION complete_timer_job(
    p_job_id UUID,
    p_status VARCHAR DEFAULT 'completed'
) RETURNS VOID AS $$
DECLARE
    job_record RECORD;
    next_due TIMESTAMP;
BEGIN
    -- Get the job
    SELECT * INTO job_record
    FROM timer_jobs
    WHERE id = p_job_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- If repeat interval and not exceeded max attempts, reschedule
    IF job_record.repeat_interval IS NOT NULL 
       AND job_record.attempt_count < job_record.max_attempts THEN
        next_due := CURRENT_TIMESTAMP + job_record.repeat_interval;
        UPDATE timer_jobs
        SET status = 'pending',
            due_date = next_due,
            next_execution_at = next_due,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = p_job_id;
    ELSE
        -- Mark as completed or failed
        UPDATE timer_jobs
        SET status = p_status,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = p_job_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to schedule a new timer job
CREATE OR REPLACE FUNCTION schedule_timer_job(
    p_instance_id UUID,
    p_node_id VARCHAR,
    p_token_id VARCHAR,
    p_timer_type VARCHAR,
    p_due_date TIMESTAMP WITH TIME ZONE,
    p_repeat_interval INTERVAL DEFAULT NULL,
    p_max_attempts INTEGER DEFAULT 3
) RETURNS UUID AS $$
DECLARE
    v_job_id UUID;
BEGIN
    v_job_id := gen_random_uuid();

    INSERT INTO timer_jobs (
        id, instance_id, node_id, token_id, timer_type, due_date,
        repeat_interval, max_attempts, attempt_count, handler_config,
        status, next_execution_at, created_at, updated_at
    ) VALUES (
        v_job_id, p_instance_id, p_node_id, p_token_id, p_timer_type,
        p_due_date, p_repeat_interval, p_max_attempts, 0, '{}',
        'pending', p_due_date, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
    );

    RETURN v_job_id;
END;
$$ LANGUAGE plpgsql;

-- Function to cancel a timer job
CREATE OR REPLACE FUNCTION cancel_timer_job(p_job_id UUID) RETURNS VOID AS $$
BEGIN
    UPDATE timer_jobs
    SET status = 'cancelled',
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_job_id
      AND status = 'pending';
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions
GRANT EXECUTE ON FUNCTION process_due_timer_jobs() TO workflow_user;
GRANT EXECUTE ON FUNCTION complete_timer_job(UUID, VARCHAR) TO workflow_user;
GRANT EXECUTE ON FUNCTION schedule_timer_job(UUID, VARCHAR, VARCHAR, VARCHAR, TIMESTAMP WITH TIME ZONE, INTERVAL, INTEGER) TO workflow_user;
GRANT EXECUTE ON FUNCTION cancel_timer_job(UUID) TO workflow_user;
