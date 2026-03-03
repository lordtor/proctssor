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

interface MonitorState {
  instances: ProcessInstance[];
  selectedInstance: ProcessInstance | null;
  tokens: Token[];
  events: ProcessEvent[];
  loading: boolean;
  error: string | null;
  wsConnected: boolean;
  
  fetchInstances: () => Promise<void>;
  selectInstance: (instance: ProcessInstance | null) => void;
  fetchInstanceDetails: (instanceId: string) => Promise<void>;
  terminateInstance: (instanceId: string) => Promise<void>;
  suspendInstance: (instanceId: string) => Promise<void>;
  resumeInstance: (instanceId: string) => Promise<void>;
  connectWebSocket: () => void;
  disconnectWebSocket: () => void;
}

let ws: WebSocket | null = null;

export const useMonitorStore = create<MonitorState>((set, get) => ({
  instances: [],
  selectedInstance: null,
  tokens: [],
  events: [],
  loading: false,
  error: null,
  wsConnected: false,

  fetchInstances: async () => {
    set({ loading: true, error: null });
    try {
      const response = await axios.get('/api/instances');
      set({ instances: response.data, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  selectInstance: (instance) => {
    set({ selectedInstance: instance });
    if (instance) {
      get().fetchInstanceDetails(instance.id);
    }
  },

  fetchInstanceDetails: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      const [tokensRes, eventsRes] = await Promise.all([
        axios.get(`/api/instances/${instanceId}/tokens`),
        axios.get(`/api/instances/${instanceId}/events`),
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
      await axios.post(`/api/instances/${instanceId}/terminate`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  suspendInstance: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/instances/${instanceId}/suspend`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  resumeInstance: async (instanceId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/instances/${instanceId}/resume`);
      await get().fetchInstances();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  connectWebSocket: () => {
    if (ws) return;
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
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
}));
