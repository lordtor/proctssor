import { create } from 'zustand';
import axios from 'axios';

export interface ProcessInstance {
  id: string;
  processDefinitionId: string;
  processDefinitionKey: string;
  status: 'active' | 'completed' | 'terminated' | 'suspended';
  startTime: string;
  endTime: string | null;
  variables: Record<string, any>;
}

export interface Token {
  id: string;
  processInstanceId: string;
  elementId: string;
  elementName: string;
  status: 'waiting' | 'active' | 'completed';
  arrivedAt: string;
}

export interface ProcessEvent {
  id: string;
  processInstanceId: string;
  eventType: string;
  elementId: string;
  timestamp: string;
  variables: Record<string, any>;
}

export interface Metrics {
  processCount: number;
  averageDuration: number;
  errorRate: number;
  completedCount: number;
  activeCount: number;
  terminatedCount: number;
  durationHistory: { date: string; value: number }[];
  errorHistory: { date: string; value: number }[];
}

interface MonitorState {
  instances: ProcessInstance[];
  selectedInstance: ProcessInstance | null;
  tokens: Token[];
  events: ProcessEvent[];
  metrics: Metrics | null;
  loading: boolean;
  error: string | null;
  wsConnected: boolean;
  
  // Drill-down navigation
  processStack: ProcessStackItem[];
  currentProcessDefinitionId: string | null;
  
  fetchInstances: () => Promise<void>;
  selectInstance: (instance: ProcessInstance | null) => void;
  fetchInstanceDetails: (instanceId: string) => Promise<void>;
  terminateInstance: (instanceId: string) => Promise<void>;
  suspendInstance: (instanceId: string) => Promise<void>;
  resumeInstance: (instanceId: string) => Promise<void>;
  fetchMetrics: () => Promise<void>;
  connectWebSocket: () => void;
  disconnectWebSocket: () => void;
  drillDown: (processDefinitionId: string, elementId?: string) => Promise<void>;
  drillUp: () => void;
  resetNavigation: () => void;
}

export interface ProcessStackItem {
  processDefinitionId: string;
  elementId?: string;
  elementName?: string;
}

let ws: WebSocket | null = null;

export const useMonitorStore = create<MonitorState>((set, get) => ({
  instances: [],
  selectedInstance: null,
  tokens: [],
  events: [],
  metrics: null,
  loading: false,
  error: null,
  wsConnected: false,
  
  // Drill-down navigation
  processStack: [],
  currentProcessDefinitionId: null,

  fetchInstances: async () => {
    set({ loading: true, error: null });
    try {
      const response = await axios.get('/api/v1/instances');
      set({ instances: response.data, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  selectInstance: (instance) => {
    set({ selectedInstance: instance, processStack: [], currentProcessDefinitionId: null });
    if (instance) {
      get().fetchInstanceDetails(instance.id);
    }
  },

  fetchInstanceDetails: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      const [tokensRes, eventsRes] = await Promise.all([
        axios.get(`/api/v1/instances/${instanceId}/tokens`),
        axios.get(`/api/v1/instances/${instanceId}/events`),
      ]);
      set({ 
        tokens: tokensRes.data, 
        events: eventsRes.data, 
        loading: false 
      });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  terminateInstance: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/instances/${instanceId}/terminate`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  suspendInstance: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/instances/${instanceId}/suspend`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  resumeInstance: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/instances/${instanceId}/resume`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  fetchMetrics: async () => {
    set({ loading: true, error: null });
    try {
      const response = await axios.get('/api/v1/metrics');
      set({ metrics: response.data, loading: false });
    } catch (err: any) {
      // Если API недоступен, используем моковые данные
      const mockMetrics = {
        processCount: 156,
        averageDuration: 2450,
        errorRate: 2.3,
        completedCount: 142,
        activeCount: 8,
        terminatedCount: 6,
        durationHistory: [
          { date: '2026-02-25', value: 2100 },
          { date: '2026-02-26', value: 1950 },
          { date: '2026-02-27', value: 2300 },
          { date: '2026-02-28', value: 1800 },
          { date: '2026-03-01', value: 2450 },
          { date: '2026-03-02', value: 2100 },
          { date: '2026-03-03', value: 2450 },
        ],
        errorHistory: [
          { date: '2026-02-25', value: 3.2 },
          { date: '2026-02-26', value: 1.8 },
          { date: '2026-02-27', value: 2.5 },
          { date: '2026-02-28', value: 1.2 },
          { date: '2026-03-01', value: 2.8 },
          { date: '2026-03-02', value: 1.5 },
          { date: '2026-03-03', value: 2.3 },
        ],
      };
      set({ metrics: mockMetrics, loading: false });
    }
  },

  connectWebSocket: () => {
    if (ws) return;
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/instances/${get().selectedInstance?.id}`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      set({ wsConnected: true });
    };
    
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        if (data.type === 'instance_update') {
          get().fetchInstances();
        } else if (data.type === 'token_update' && get().selectedInstance) {
          get().fetchInstanceDetails(get().selectedInstance!.id);
        }
      } catch (e) {
        console.error('WebSocket message parse error:', e);
      }
    };
    
    ws.onclose = () => {
      set({ wsConnected: false });
      ws = null;
    };
    
    ws.onerror = () => {
      set({ wsConnected: false });
    };
  },

  disconnectWebSocket: () => {
    if (ws) {
      ws.close();
      ws = null;
    }
  },

  drillDown: async (processDefinitionId: string, elementId?: string) => {
    const state = get();
    const elementName = elementId || 'Sub Process';
    
    // Add current process to stack if we have one
    if (state.currentProcessDefinitionId) {
      set({
        processStack: [
          ...state.processStack,
          {
            processDefinitionId: state.currentProcessDefinitionId,
          }
        ],
        currentProcessDefinitionId: processDefinitionId,
      });
    } else if (state.selectedInstance) {
      // First level - from main process
      set({
        processStack: [
          {
            processDefinitionId: state.selectedInstance.processDefinitionId,
          }
        ],
        currentProcessDefinitionId: processDefinitionId,
      });
    }
  },

  drillUp: () => {
    const state = get();
    if (state.processStack.length === 0) {
      // Go back to root process
      set({
        currentProcessDefinitionId: state.selectedInstance?.processDefinitionId || null,
      });
    } else {
      // Go back to previous process in stack
      const newStack = [...state.processStack];
      const previousProcess = newStack.pop();
      set({
        processStack: newStack,
        currentProcessDefinitionId: previousProcess?.processDefinitionId || null,
      });
    }
  },

  resetNavigation: () => {
    set({
      processStack: [],
      currentProcessDefinitionId: null,
    });
  },
}));
