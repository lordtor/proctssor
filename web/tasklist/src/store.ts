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
}

interface TaskState {
  tasks: Task[];
  selectedTask: Task | null;
  loading: boolean;
  error: string | null;
  filter: 'all' | 'assigned' | 'unassigned';
  fetchTasks: () => Promise<void>;
  selectTask: (task: Task | null) => void;
  completeTask: (taskId: string, variables: Record<string, any>) => Promise<void>;
  claimTask: (taskId: string) => Promise<void>;
  unclaimTask: (taskId: string) => Promise<void>;
  setFilter: (filter: 'all' | 'assigned' | 'unassigned') => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  selectedTask: null,
  loading: false,
  error: null,
  filter: 'all',

  fetchTasks: async () => {
    set({ loading: true, error: null });
    try {
      const { filter } = get();
      let url = '/api/v1/tasks';
      if (filter === 'assigned') url += '?assigned=true';
      if (filter === 'unassigned') url += '?assigned=false';
      
      const response = await axios.get(url);
      set({ tasks: response.data, loading: false });
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
      await axios.post(`/api/v1/tasks/${taskId}/complete`, variables);
      await get().fetchTasks();
      set({ selectedTask: null, loading: false });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  claimTask: async (taskId) => {
    set({ loading: true, error: null });
    try {
      await axios.post(`/api/v1/tasks/${taskId}/claim`);
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

  setFilter: (filter) => {
    set({ filter });
    get().fetchTasks();
  },
}));
