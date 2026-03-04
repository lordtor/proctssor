import { create } from 'zustand';
import axios from 'axios';

export interface Task {
  id: string;
  name: string;
  processInstanceId: string;
  processDefinitionId: string;
  assignee: string | null;
  createdAt: string;
  dueDate: string | null;
  variables: Record<string, any>;
  formSchema?: any;
  completedAt?: string;
}

interface TaskState {
  tasks: Task[];
  historyTasks: Task[];
  selectedTask: Task | null;
  loading: boolean;
  error: string | null;
  filter: 'all' | 'assigned' | 'unassigned';
  viewMode: 'active' | 'history';
  fetchTasks: () => Promise<void>;
  fetchHistory: () => Promise<void>;
  selectTask: (task: Task | null) => void;
  completeTask: (taskId: string, variables: Record<string, any>) => Promise<void>;
  claimTask: (taskId: string) => Promise<void>;
  unclaimTask: (taskId: string) => Promise<void>;
  delegateTask: (taskId: string, userId: string) => Promise<void>;
  setFilter: (filter: 'all' | 'assigned' | 'unassigned') => void;
  setViewMode: (mode: 'active' | 'history') => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  historyTasks: [],
  selectedTask: null,
  loading: false,
  error: null,
  filter: 'all',
  viewMode: 'active',

  fetchTasks: async () => {
    set({ loading: true, error: null });
    try {
      const { filter } = get();
      // Use correct API endpoint - filter by assignee
      const assignee = filter === 'assigned' ? 'current_user' : (filter === 'unassigned' ? '' : undefined);
      const params = assignee !== undefined ? { assignee } : {};
      
      const response = await axios.get('/api/v1/tasks', { params });
      set({ tasks: response.data, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  fetchHistory: async () => {
    set({ loading: true, error: null });
    try {
      const response = await axios.get('/api/v1/tasks/history', { params: { limit: 100 } });
      set({ historyTasks: response.data, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  selectTask: (task) => {
    set({ selectedTask: task });
  },

  completeTask: async (taskId, variables) => {
    set({ loading: true, error: null });
    try {
      // Find the task to get its instance ID
      const task = get().tasks.find(t => t.id === taskId);
      if (task) {
        await axios.post(`/api/v1/instances/${task.processInstanceId}/tasks/${taskId}/complete`, { variables });
      }
      await get().fetchTasks();
      set({ selectedTask: null, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  claimTask: async (taskId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/tasks/${taskId}/claim`, { user_id: 'current_user' });
      await get().fetchTasks();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  unclaimTask: async (taskId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/tasks/${taskId}/unclaim`);
      await get().fetchTasks();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  delegateTask: async (taskId, userId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/tasks/${taskId}/delegate`, { user_id: userId });
      await get().fetchTasks();
      set({ loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  setFilter: (filter) => {
    set({ filter });
    get().fetchTasks();
  },

  setViewMode: (mode) => {
    set({ viewMode: mode });
    if (mode === 'history') {
      get().fetchHistory();
    }
  },
}));
