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
  activePage: 'dashboard',
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

  // Messages — per-conversation message cache
  // { [conversationId]: Message[] }
  messagesByConversation: {},
  // Getter: messages for the current conversation
  messages: [],
  setMessages: (messagesOrFn) => {
    const state = get();
    const convId = state.currentConversation?.id;
    if (!convId) {
      // No conversation, just update the flat messages array
      if (typeof messagesOrFn === 'function') {
        set((s) => ({ messages: messagesOrFn(s.messages) }));
      } else {
        set({ messages: messagesOrFn });
      }
      return;
    }
    if (typeof messagesOrFn === 'function') {
      set((s) => {
        const prev = s.messagesByConversation[convId] || s.messages;
        const next = messagesOrFn(prev);
        return {
          messages: s.currentConversation?.id === convId ? next : s.messages,
          messagesByConversation: { ...s.messagesByConversation, [convId]: next },
        };
      });
    } else {
      set((s) => ({
        messages: s.currentConversation?.id === convId ? messagesOrFn : s.messages,
        messagesByConversation: { ...s.messagesByConversation, [convId]: messagesOrFn },
      }));
    }
  },
  addMessage: (msg) => {
    const state = get();
    const convId = state.currentConversation?.id;
    set((s) => {
      const newMessages = [...s.messages, msg];
      const newCache = convId
        ? { ...s.messagesByConversation, [convId]: newMessages }
        : s.messagesByConversation;
      return { messages: newMessages, messagesByConversation: newCache };
    });
  },
  // Add a message to a SPECIFIC conversation (may differ from current)
  addMessageToConversation: (targetConvId, msg) => {
    set((s) => {
      const prevMsgs = s.messagesByConversation[targetConvId] || [];
      const newMsgs = [...prevMsgs, msg];
      const isCurrent = s.currentConversation?.id === targetConvId;
      return {
        messagesByConversation: { ...s.messagesByConversation, [targetConvId]: newMsgs },
        messages: isCurrent ? newMsgs : s.messages,
      };
    });
  },
  // Set messages for a specific conversation (used when loading messages)
  setMessagesForConversation: (convId, msgs) => {
    set((s) => ({
      messagesByConversation: { ...s.messagesByConversation, [convId]: msgs },
      // If this is the current conversation, also update the flat messages
      messages: s.currentConversation?.id === convId ? msgs : s.messages,
    }));
  },
  // Functional update for messages of a specific conversation
  updateMessagesForConversation: (targetConvId, updaterFn) => {
    set((s) => {
      const prevMsgs = s.messagesByConversation[targetConvId] || [];
      const nextMsgs = updaterFn(prevMsgs);
      const isCurrent = s.currentConversation?.id === targetConvId;
      return {
        messagesByConversation: { ...s.messagesByConversation, [targetConvId]: nextMsgs },
        messages: isCurrent ? nextMsgs : s.messages,
      };
    });
  },
  // Switch to cached messages for a conversation
  switchToConversationMessages: (convId) => {
    set((s) => {
      const cached = s.messagesByConversation[convId];
      return cached !== undefined ? { messages: cached } : {};
    });
  },

  // Loading states — per conversation
  // { [conversationId]: boolean }
  sendingByConversation: {},
  // Legacy single flag (derived from current conversation)
  isSending: false,
  setIsSending: (v) => {
    const state = get();
    const convId = state.currentConversation?.id;
    set((s) => ({
      isSending: v,
      sendingByConversation: convId
        ? { ...s.sendingByConversation, [convId]: v }
        : s.sendingByConversation,
    }));
  },
  setIsSendingForConversation: (convId, v) => {
    set((s) => ({
      sendingByConversation: { ...s.sendingByConversation, [convId]: v },
      // Update the flat flag if this is the current conversation
      isSending: s.currentConversation?.id === convId ? v : s.isSending,
    }));
  },
  isSendingForConversation: (convId) => {
    return get().sendingByConversation[convId] || false;
  },

  // Mode: 'agent' or 'workflow'
  mode: 'agent',
  setMode: (mode) => set({ mode }),
}));

export default useStore;
