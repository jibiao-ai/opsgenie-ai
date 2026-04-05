import { create } from 'zustand';

const useStore = create((set, get) => ({
  // Auth
  user: JSON.parse(localStorage.getItem('user') || 'null'),
  token: localStorage.getItem('token') || null,
  setAuth: (user, token) => {
    localStorage.setItem('user', JSON.stringify(user));
    localStorage.setItem('token', token);
    set({ user, token });
  },
  logout: () => {
    localStorage.removeItem('user');
    localStorage.removeItem('token');
    set({ user: null, token: null });
  },

  // Sidebar
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

  // Active page
  activePage: 'chat',
  setActivePage: (page) => set({ activePage: page }),

  // Theme: light / dark / blue
  theme: localStorage.getItem('theme') || 'light',
  setTheme: (theme) => {
    localStorage.setItem('theme', theme);
    document.documentElement.setAttribute('data-theme', theme);
    set({ theme });
  },

  // Agents
  agents: [],
  selectedAgent: null,
  setAgents: (agents) => set({ agents }),
  setSelectedAgent: (agent) => set({ selectedAgent: agent }),

  // Conversations
  conversations: [],
  currentConversation: null,
  setConversations: (conversations) => set({ conversations }),
  setCurrentConversation: (conv) => set({ currentConversation: conv }),
  addConversation: (conv) => set((s) => ({
    conversations: [conv, ...s.conversations],
  })),
  removeConversation: (id) => set((s) => ({
    conversations: s.conversations.filter((c) => c.id !== id),
    currentConversation: s.currentConversation?.id === id ? null : s.currentConversation,
  })),

  // Messages
  messages: [],
  setMessages: (messages) => set({ messages }),
  addMessage: (msg) => set((s) => ({ messages: [...s.messages, msg] })),

  // Loading states
  isSending: false,
  setIsSending: (v) => set({ isSending: v }),

  // Mode: 'agent' or 'workflow'
  mode: 'agent',
  setMode: (mode) => set({ mode }),
}));

export default useStore;
