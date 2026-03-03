import { useEffect, useRef, useCallback } from 'react';

interface UseSSEOptions {
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
  onMessage?: (data: any, event: MessageEvent) => void;
}

/**
 * Custom hook for Server-Sent Events (SSE) connection
 */
export function useSSE(
  url: string | null,
  onMessage: (data: any) => void,
  options: UseSSEOptions = {}
) {
  const { onOpen, onClose, onError, onMessage: onMessageCallback } = options;
  
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = 5;
  const reconnectInterval = 3000;

  const connect = useCallback(() => {
    if (!url) return;

    try {
      // If previous connection exists, close it first
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }

      const eventSource = new EventSource(url);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        reconnectAttemptsRef.current = 0;
        onOpen?.();
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onMessage(data);
          onMessageCallback?.(data, event);
        } catch (err) {
          // If not JSON, pass as-is
          onMessage(event.data);
          onMessageCallback?.(event.data, event);
        }
      };

      eventSource.onerror = (error) => {
        onError?.(error);
        eventSource.close();
        
        // Auto-reconnect logic
        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++;
            connect();
          }, reconnectInterval);
        }
      };

      // Named event handlers
      eventSource.addEventListener('task_created', (event) => {
        const data = JSON.parse(event.data);
        onMessage({ type: 'task_created', ...data });
      });

      eventSource.addEventListener('task_updated', (event) => {
        const data = JSON.parse(event.data);
        onMessage({ type: 'task_updated', ...data });
      });

      eventSource.addEventListener('instance_update', (event) => {
        const data = JSON.parse(event.data);
        onMessage({ type: 'instance_update', ...data });
      });

      eventSource.addEventListener('token_update', (event) => {
        const data = JSON.parse(event.data);
        onMessage({ type: 'token_update', ...data });
      });

    } catch (err) {
      console.error('SSE connection error:', err);
    }
  }, [url, onMessage, onOpen, onError, onMessageCallback]);

  const close = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    reconnectAttemptsRef.current = maxReconnectAttempts; // Prevent reconnection
    eventSourceRef.current?.close();
    eventSourceRef.current = null;
  }, []);

  useEffect(() => {
    connect();

    return () => {
      close();
    };
  }, [connect, close]);

  return {
    close,
    reconnect: connect,
  };
}

export default useSSE;
