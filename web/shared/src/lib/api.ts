import axios, { AxiosInstance } from 'axios';

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

// Create axios instance with defaults
const axiosInstance: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Auth interceptor
axiosInstance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling
axiosInstance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Types
export interface Process {
  id: string;
  key: string;
  name: string;
  version: number;
  deploymentTime?: string;
}

export interface ProcessInstance {
  id: string;
  processDefinitionId: string;
  processDefinitionKey: string;
  status: 'active' | 'completed' | 'terminated' | 'suspended';
  startTime: string;
  endTime?: string;
  variables: Record<string, any>;
}

export interface Task {
  id: string;
  name: string;
  processInstanceId: string;
  processDefinitionId: string;
  assignee: string | null;
  createdAt: string;
  dueDate?: string;
  priority?: number;
  variables: Record<string, any>;
  formSchema?: any;
}

export interface ServiceInfo {
  name: string;
  version: string;
  actions: ServiceAction[];
}

export interface ServiceAction {
  name: string;
  description?: string;
  schema?: any;
}

// API Client
export const api = {
  processes: {
    list: (): Promise<Process[]> => 
      axiosInstance.get('/processes').then(r => r.data),
    
    deploy: (xml: string): Promise<{ id: string }> => 
      axiosInstance.post('/processes/deploy', { bpmn_xml: xml }).then(r => r.data),
    
    getXml: (id: string): Promise<string> => 
      axiosInstance.get(`/processes/${id}/xml`).then(r => r.data.bpmn_xml),
    
    start: (id: string, variables?: Record<string, any>): Promise<{ instanceId: string }> => 
      axiosInstance.post(`/processes/${id}/start`, { variables }).then(r => r.data),
    
    get: (id: string): Promise<Process> => 
      axiosInstance.get(`/processes/${id}`).then(r => r.data),
    
    delete: (id: string): Promise<void> => 
      axiosInstance.delete(`/processes/${id}`).then(() => {}),
  },

  instances: {
    list: (params?: any): Promise<ProcessInstance[]> => 
      axiosInstance.get('/instances', { params }).then(r => r.data),
    
    get: (id: string): Promise<ProcessInstance> => 
      axiosInstance.get(`/instances/${id}`).then(r => r.data),
    
    suspend: (id: string): Promise<void> => 
      axiosInstance.post(`/instances/${id}/suspend`).then(() => {}),
    
    resume: (id: string): Promise<void> => 
      axiosInstance.post(`/instances/${id}/resume`).then(() => {}),
    
    terminate: (id: string): Promise<void> => 
      axiosInstance.post(`/instances/${id}/terminate`).then(() => {}),
    
    getTokens: (id: string): Promise<any[]> => 
      axiosInstance.get(`/instances/${id}/tokens`).then(r => r.data),
    
    getEvents: (id: string): Promise<any[]> => 
      axiosInstance.get(`/instances/${id}/events`).then(r => r.data),
    
    wsUrl: (id: string): string => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      return `${protocol}//${host}/api/v1/ws/instances/${id}`;
    },
  },

  tasks: {
    list: (params?: any): Promise<Task[]> => 
      axiosInstance.get('/tasks', { params }).then(r => r.data),
    
    getForm: (instanceId: string, taskId: string): Promise<any> => 
      axiosInstance.get(`/instances/${instanceId}/tasks/${taskId}/form`).then(r => r.data),
    
    complete: (instanceId: string, taskId: string, data: any): Promise<void> => 
      axiosInstance.post(`/instances/${instanceId}/tasks/${taskId}/complete`, { variables: data }).then(() => {}),
    
    claim: (taskId: string, userId?: string): Promise<void> => 
      axiosInstance.post(`/tasks/${taskId}/claim`, { user_id: userId || 'current_user' }).then(() => {}),
    
    unclaim: (taskId: string): Promise<void> => 
      axiosInstance.post(`/tasks/${taskId}/unclaim`).then(() => {}),
    
    sseUrl: (assignee: string): string => 
      `${BASE_URL}/sse/tasks?assignee=${encodeURIComponent(assignee)}`,
  },

  registry: {
    listServices: (): Promise<ServiceInfo[]> => 
      axiosInstance.get('/registry/services').then(r => r.data),
    
    getServiceActions: (name: string): Promise<{ actions: ServiceAction[] }> => 
      axiosInstance.get(`/registry/services/${name}/actions`).then(r => r.data),
    
    heartbeat: (data: any): Promise<void> => 
      axiosInstance.post('/registry/heartbeat', data).then(() => {}),
  },
};

export default api;
