-- BPMN Workflow Platform - Functions and Triggers
-- Part 3: NOTIFY Functions and Event Triggers

-- =============================================================================
-- Function: Notify Instance Change
-- =============================================================================
CREATE OR REPLACE FUNCTION notify_instance_change()
RETURNS TRIGGER AS $$
DECLARE
    notification_payload JSONB;
BEGIN
    -- Build notification payload
    notification_payload := jsonb_build_object(
        'instance_id', NEW.id,
        'process_definition_id', NEW.process_definition_id,
        'status', NEW.status,
        'current_node', NEW.current_node,
        'business_key', NEW.business_key,
        'updated_at', NEW.updated_at
    );

    -- Add old status if update
    IF TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status THEN
        notification_payload := notification_payload || jsonb_build_object(
            'old_status', OLD.status
        );
    END IF;

    -- Notify subscribers
    PERFORM pg_notify('instance_changes', notification_payload::text);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Notify Registry Change
-- =============================================================================
CREATE OR REPLACE FUNCTION notify_registry_change()
RETURNS TRIGGER AS $$
DECLARE
    notification_payload JSONB;
BEGIN
    -- Build notification payload
    notification_payload := jsonb_build_object(
        'service_id', NEW.id,
        'service_name', NEW.name,
        'version', NEW.version,
        'status', NEW.service_status,
        'updated_at', NEW.updated_at
    );

    -- Add old status if update
    IF TG_OP = 'UPDATE' AND OLD.service_status IS DISTINCT FROM NEW.service_status THEN
        notification_payload := notification_payload || jsonb_build_object(
            'old_status', OLD.service_status
        );
    END IF;

    -- Notify subscribers
    PERFORM pg_notify('registry_changes', notification_payload::text);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Process Event Logger
-- =============================================================================
CREATE OR REPLACE FUNCTION log_process_event(
    p_instance_id UUID,
    p_process_definition_id INTEGER,
    p_node_id VARCHAR,
    p_node_name VARCHAR,
    p_node_type node_type,
    p_action node_action,
    p_payload JSONB DEFAULT '{}',
    p_trace_id VARCHAR DEFAULT NULL,
    p_user_id VARCHAR DEFAULT NULL
) RETURNS BIGINT AS $$
DECLARE
    v_event_id BIGINT;
BEGIN
    INSERT INTO process_events (
        instance_id,
        process_definition_id,
        node_id,
        node_name,
        node_type,
        action,
        payload,
        trace_id,
        user_id
    ) VALUES (
        p_instance_id,
        p_process_definition_id,
        p_node_id,
        p_node_name,
        p_node_type,
        p_action,
        p_payload,
        p_trace_id,
        p_user_id
    )
    RETURNING id INTO v_event_id;

    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Heartbeat Handler
-- =============================================================================
CREATE OR REPLACE FUNCTION handle_heartbeat(
    p_service_name VARCHAR,
    p_version VARCHAR DEFAULT NULL
) RETURNS VOID AS $$
BEGIN
    UPDATE service_registry
    SET last_heartbeat = CURRENT_TIMESTAMP,
        service_status = 'healthy',
        updated_at = CURRENT_TIMESTAMP
    WHERE name = p_service_name 
      AND (p_version IS NULL OR version = p_version);
    
    -- If no rows updated, service doesn't exist
    IF NOT FOUND THEN
        RAISE NOTICE 'Service % not found for heartbeat', p_service_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Complete Process Instance
-- =============================================================================
CREATE OR REPLACE FUNCTION complete_process_instance(
    p_instance_id UUID,
    p_completed_variables JSONB DEFAULT NULL
) RETURNS VOID AS $$
BEGIN
    UPDATE process_instances
    SET status = 'completed',
        completed_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP,
        variables = COALESCE(p_completed_variables, variables) || jsonb_build_object(
            '_completed_at', CURRENT_TIMESTAMP
        )
    WHERE id = p_instance_id;

    -- Log completion event
    PERFORM log_process_event(
        p_instance_id => p_instance_id,
        p_process_definition_id => (SELECT process_definition_id FROM process_instances WHERE id = p_instance_id),
        p_node_id => NULL,
        p_node_name => NULL,
        p_node_type => NULL,
        p_action => 'workflow_completed',
        p_payload => '{}'
    );
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Suspend Process Instance
-- =============================================================================
CREATE OR REPLACE FUNCTION suspend_process_instance(
    p_instance_id UUID,
    p_until TIMESTAMP WITH TIME ZONE
) RETURNS VOID AS $$
BEGIN
    UPDATE process_instances
    SET status = 'suspended',
        suspended_until = p_until,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_instance_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Triggers for Notifications
-- =============================================================================
DROP TRIGGER IF EXISTS trigger_instance_change ON process_instances;
CREATE TRIGGER trigger_instance_change
    AFTER INSERT OR UPDATE ON process_instances
    FOR EACH ROW
    EXECUTE FUNCTION notify_instance_change();

DROP TRIGGER IF EXISTS trigger_registry_change ON service_registry;
CREATE TRIGGER trigger_registry_change
    AFTER INSERT OR UPDATE ON service_registry
    FOR EACH ROW
    EXECUTE FUNCTION notify_registry_change();

-- =============================================================================
-- Function: Get Active Instance Count
-- =============================================================================
CREATE OR REPLACE FUNCTION get_active_instance_count(
    p_definition_id INTEGER DEFAULT NULL
) RETURNS BIGINT AS $$
DECLARE
    v_count BIGINT;
BEGIN
    IF p_definition_id IS NULL THEN
        SELECT COUNT(*) INTO v_count
        FROM process_instances
        WHERE status IN ('active', 'pending');
    ELSE
        SELECT COUNT(*) INTO v_count
        FROM process_instances
        WHERE process_definition_id = p_definition_id
          AND status IN ('active', 'pending');
    END IF;
    
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Function: Get Task Summary
-- =============================================================================
CREATE OR REPLACE FUNCTION get_task_summary(
    p_assignee VARCHAR DEFAULT NULL
) RETURNS TABLE (
    status VARCHAR,
    count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ut.status,
        COUNT(*)::BIGINT
    FROM user_tasks ut
    WHERE p_assignee IS NULL OR ut.assignee = p_assignee
    GROUP BY ut.status;
END;
$$ LANGUAGE plpgsql;
