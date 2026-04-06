import axios from 'axios';

const API_BASE = '/api';

const api = axios.create({
  baseURL: API_BASE,
  timeout: 120000,
  headers: { 'Content-Type': 'application/json' },
});

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error.response?.data || error);
  }
);

// Auth
export const login = (username, password) =>
  api.post('/login', { username, password });

export const getProfile = () => api.get('/profile');

// Dashboard
export const getDashboard = () => api.get('/dashboard');

// Agents
export const getAgents = () => api.get('/agents');
export const getAgent = (id) => api.get(`/agents/${id}`);
export const createAgent = (data) => api.post('/agents', data);
export const updateAgent = (id, data) => api.put(`/agents/${id}`, data);
export const deleteAgent = (id) => api.delete(`/agents/${id}`);

// Conversations
export const getConversations = () => api.get('/conversations');
export const createConversation = (agentId, title) =>
  api.post('/conversations', { agent_id: agentId, title });
export const deleteConversation = (id) => api.delete(`/conversations/${id}`);

// Messages
export const getMessages = (conversationId) =>
  api.get(`/conversations/${conversationId}/messages`);
export const sendMessage = (conversationId, content, attachments = []) =>
  api.post(`/conversations/${conversationId}/messages`, { content, attachments });

// File Upload
export const uploadFile = (file) => {
  const formData = new FormData();
  formData.append('file', file);
  return api.post('/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  });
};

// Skills
export const getSkills = () => api.get('/skills');
export const getAgentSkills = (agentId) => api.get(`/agents/${agentId}/skills`);

// Workflows
export const getWorkflows = () => api.get('/workflows');
export const createWorkflow = (data) => api.post('/workflows', data);

// Scheduled Tasks
export const getScheduledTasks = () => api.get('/scheduled-tasks');
export const createScheduledTask = (data) => api.post('/scheduled-tasks', data);
export const updateScheduledTask = (id, data) => api.put(`/scheduled-tasks/${id}`, data);
export const deleteScheduledTask = (id) => api.delete(`/scheduled-tasks/${id}`);

// Users (Admin)
export const getUsers = () => api.get('/users');
export const createUser = (data) => api.post('/users', data);
export const updateUser = (id, data) => api.put(`/users/${id}`, data);
export const deleteUser = (id) => api.delete(`/users/${id}`);

// Task Logs
export const getTaskLogs = () => api.get('/task-logs');

// AI Providers
export const getAIProviders = () => api.get('/ai-providers');
export const updateAIProvider = (id, data) => api.put(`/ai-providers/${id}`, data);
export const testAIProvider = (id) => api.post(`/ai-providers/${id}/test`);

// Resource Monitor (big-screen)
export const getResourceMonitor = () => api.get('/resource-monitor');

// Cloud Platforms
export const getCloudPlatforms = () => api.get('/cloud-platforms');
export const createCloudPlatform = (data) => api.post('/cloud-platforms', data);
export const updateCloudPlatform = (id, data) => api.put(`/cloud-platforms/${id}`, data);
export const deleteCloudPlatform = (id) => api.delete(`/cloud-platforms/${id}`);
export const testCloudPlatform = (id) => api.post(`/cloud-platforms/${id}/test`);

// Operation Logs (Admin)
export const getOperationLogs = (params) => api.get('/operation-logs', { params });

export default api;
