import React, { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import {
  Send,
  Plus,
  Paperclip,
  Bot,
  User,
  Trash2,
  ChevronDown,
  Loader2,
  MessageSquare,
} from 'lucide-react';
import useStore from '../store/useStore';
import {
  getAgents,
  getConversations,
  createConversation,
  deleteConversation,
  getMessages,
  sendMessage,
} from '../services/api';
import toast from 'react-hot-toast';

export default function ChatPage() {
  const {
    agents, setAgents, selectedAgent, setSelectedAgent,
    conversations, setConversations, currentConversation, setCurrentConversation,
    addConversation, removeConversation,
    messages, setMessages, addMessage,
    isSending, setIsSending,
    mode, setMode,
  } = useStore();

  const [input, setInput] = useState('');
  const [showAgentDropdown, setShowAgentDropdown] = useState(false);
  const messagesEndRef = useRef(null);
  const inputRef = useRef(null);

  // Load agents and conversations on mount
  useEffect(() => {
    loadAgents();
    loadConversations();
  }, []);

  // Scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Load messages when conversation changes
  useEffect(() => {
    if (currentConversation) {
      loadMessages(currentConversation.id);
    } else {
      setMessages([]);
    }
  }, [currentConversation?.id]);

  const loadAgents = async () => {
    try {
      const res = await getAgents();
      if (res.code === 0 && res.data) {
        setAgents(res.data);
        if (!selectedAgent && res.data.length > 0) {
          setSelectedAgent(res.data[0]);
        }
      }
    } catch (err) {
      console.error('Failed to load agents:', err);
    }
  };

  const loadConversations = async () => {
    try {
      const res = await getConversations();
      if (res.code === 0) {
        setConversations(res.data || []);
      }
    } catch (err) {
      console.error('Failed to load conversations:', err);
    }
  };

  const loadMessages = async (convId) => {
    try {
      const res = await getMessages(convId);
      if (res.code === 0) {
        setMessages(res.data || []);
      }
    } catch (err) {
      console.error('Failed to load messages:', err);
    }
  };

  const handleNewConversation = async () => {
    if (!selectedAgent) {
      toast.error('请先选择一个智能体');
      return;
    }
    try {
      const res = await createConversation(selectedAgent.id, '新会话');
      if (res.code === 0) {
        addConversation(res.data);
        setCurrentConversation(res.data);
        setMessages([]);
        toast.success('新会话已创建');
      }
    } catch (err) {
      toast.error('创建会话失败');
    }
  };

  const handleDeleteConversation = async (id, e) => {
    e.stopPropagation();
    try {
      await deleteConversation(id);
      removeConversation(id);
      toast.success('会话已删除');
    } catch (err) {
      toast.error('删除失败');
    }
  };

  const handleSend = async () => {
    if (!input.trim() || isSending) return;

    let convId = currentConversation?.id;

    // Auto-create conversation if none selected
    if (!convId) {
      if (!selectedAgent) {
        toast.error('请先选择一个智能体');
        return;
      }
      try {
        const res = await createConversation(selectedAgent.id, input.slice(0, 30));
        if (res.code === 0) {
          addConversation(res.data);
          setCurrentConversation(res.data);
          convId = res.data.id;
        } else {
          toast.error('创建会话失败');
          return;
        }
      } catch (err) {
        toast.error('创建会话失败');
        return;
      }
    }

    const userContent = input.trim();
    setInput('');
    setIsSending(true);

    // Optimistic update - add user message immediately
    const tempUserMsg = {
      id: Date.now(),
      role: 'user',
      content: userContent,
      created_at: new Date().toISOString(),
    };
    addMessage(tempUserMsg);

    try {
      const res = await sendMessage(convId, userContent);
      if (res.code === 0) {
        // Replace temp message with real one and add assistant response
        setMessages((prev) => {
          const filtered = prev.filter((m) => m.id !== tempUserMsg.id);
          return [...filtered, res.data.user_message, res.data.assistant_message];
        });
        // Refresh conversation list to update titles
        loadConversations();
      } else {
        toast.error(res.message || '发送失败');
      }
    } catch (err) {
      toast.error('发送消息失败，请重试');
      // Add error message
      addMessage({
        id: Date.now() + 1,
        role: 'assistant',
        content: '抱歉，处理请求时出现错误。请检查网络连接后重试。',
        created_at: new Date().toISOString(),
      });
    } finally {
      setIsSending(false);
      inputRef.current?.focus();
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="h-full flex flex-col p-4 gap-4 overflow-hidden">
      {/* 顶部工具栏 */}
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm px-4 py-3 flex items-center gap-4 flex-shrink-0">
        {/* 模式切换 */}
        <div className="flex bg-gray-100 rounded-lg p-0.5">
          <button
            onClick={() => setMode('agent')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition ${
              mode === 'agent' ? 'bg-[#513CC8] text-white' : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            智能体
          </button>
          <button
            onClick={() => setMode('workflow')}
            className={`px-3 py-1 text-xs font-medium rounded-md transition ${
              mode === 'workflow' ? 'bg-[#513CC8] text-white' : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            工作流
          </button>
        </div>

        {/* 智能体选择器 */}
        <div className="relative">
          <button
            onClick={() => setShowAgentDropdown(!showAgentDropdown)}
            className="flex items-center gap-2 px-3 py-1.5 bg-white border border-gray-200 rounded-lg text-sm hover:border-[#513CC8] transition-colors"
          >
            <Bot className="w-4 h-4 text-[#513CC8]" />
            <span className="text-gray-700 font-medium">
              {selectedAgent ? selectedAgent.name : '请选择智能体'}
            </span>
            <ChevronDown className="w-3 h-3 text-gray-400" />
          </button>
          {showAgentDropdown && (
            <div className="absolute top-full mt-1 left-0 w-72 bg-white border border-gray-200 rounded-xl shadow-lg z-50 py-1">
              {agents.map((agent) => (
                <button
                  key={agent.id}
                  onClick={() => {
                    setSelectedAgent(agent);
                    setShowAgentDropdown(false);
                  }}
                  className={`w-full text-left px-4 py-2.5 hover:bg-[#EEE9FB] transition ${
                    selectedAgent?.id === agent.id ? 'bg-[#EEE9FB] text-[#513CC8]' : ''
                  }`}
                >
                  <div className="font-medium text-sm">{agent.name}</div>
                  <div className="text-xs text-gray-400 mt-0.5">{agent.description?.slice(0, 50)}</div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="ml-auto">
          <button
            onClick={handleNewConversation}
            className="flex items-center gap-1.5 border border-gray-200 text-gray-600 hover:bg-gray-50 px-4 py-2 rounded-lg text-sm transition-colors"
          >
            <Plus className="w-4 h-4" />
            新会话
          </button>
        </div>
      </div>

      {/* 主体区域：左侧会话列表 + 右侧对话区 */}
      <div className="flex flex-1 gap-4 overflow-hidden min-h-0">
        {/* 左侧会话列表 */}
        <div className="w-72 bg-white rounded-xl border border-gray-200 shadow-sm flex flex-col overflow-hidden hidden lg:flex flex-shrink-0">
          <div className="px-4 py-3 border-b border-gray-100">
            <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">历史会话</h3>
          </div>
          <div className="flex-1 overflow-y-auto py-1">
            {conversations.length === 0 ? (
              <div className="text-center py-8 text-gray-400 text-sm">
                <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>暂无会话</p>
                <p className="text-xs mt-1">点击"新会话"开始</p>
              </div>
            ) : (
              conversations.map((conv) => (
                <button
                  key={conv.id}
                  onClick={() => setCurrentConversation(conv)}
                  className={`w-full flex items-center justify-between px-3 py-2.5 text-left transition group ${
                    currentConversation?.id === conv.id
                      ? 'bg-[#EEE9FB] border-r-2 border-[#513CC8]'
                      : 'hover:bg-gray-50'
                  }`}
                >
                  <div className="flex-1 min-w-0">
                    <p className={`text-sm font-medium truncate ${currentConversation?.id === conv.id ? 'text-[#513CC8]' : 'text-gray-700'}`}>
                      {conv.title || '新会话'}
                    </p>
                    <p className="text-xs text-gray-400 mt-0.5">
                      {conv.agent?.name || '智能体'}
                    </p>
                  </div>
                  <button
                    onClick={(e) => handleDeleteConversation(conv.id, e)}
                    className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-red-500 transition"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </button>
              ))
            )}
          </div>
        </div>

        {/* 右侧对话区 */}
        <div className="flex-1 bg-white rounded-xl border border-gray-200 shadow-sm flex flex-col overflow-hidden">
          {/* 消息区 */}
          <div className="flex-1 overflow-y-auto px-4 py-6">
            {messages.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-gray-400">
                <div className="w-16 h-16 bg-[#EEE9FB] rounded-2xl flex items-center justify-center mb-4">
                  <Bot className="w-8 h-8 text-[#513CC8]" />
                </div>
                <h3 className="text-lg font-medium text-gray-600 mb-2">
                  {selectedAgent?.name || 'AI 运维助手'}
                </h3>
                <p className="text-sm text-center max-w-md text-gray-400">
                  {selectedAgent?.description || '我可以帮您管理 EasyStack 云平台的各种资源，包括云主机、云硬盘、网络、监控告警等。请直接告诉我您需要什么帮助。'}
                </p>
                <div className="grid grid-cols-2 gap-3 mt-6 max-w-lg">
                  {[
                    '帮我查看所有云主机的运行状态',
                    '列出当前所有的告警信息',
                    '查询最近一小时的CPU使用率',
                    '创建一个新的安全组并添加SSH规则',
                  ].map((suggestion, i) => (
                    <button
                      key={i}
                      onClick={() => {
                        setInput(suggestion);
                        inputRef.current?.focus();
                      }}
                      className="text-left px-4 py-3 bg-gray-50 rounded-xl border border-gray-200 text-sm text-gray-600 hover:border-[#513CC8] hover:bg-[#EEE9FB] transition-colors"
                    >
                      {suggestion}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="max-w-4xl mx-auto space-y-4">
                {messages.map((msg, idx) => (
                  <div
                    key={msg.id || idx}
                    className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                  >
                    <div className={`max-w-[80%] ${msg.role === 'user' ? 'order-2' : 'order-1'}`}>
                      {msg.role !== 'user' && (
                        <div className="flex items-center gap-1.5 mb-1 ml-1">
                          <Bot className="w-3.5 h-3.5 text-[#513CC8]" />
                          <span className="text-xs text-gray-400">智能体</span>
                        </div>
                      )}
                      <div
                        className={`rounded-2xl px-4 py-3 text-sm leading-relaxed ${
                          msg.role === 'user'
                            ? 'bg-[#513CC8] text-white rounded-tr-sm'
                            : 'bg-gray-100 text-gray-800 rounded-tl-sm'
                        }`}
                      >
                        {msg.role === 'user' ? (
                          <p className="whitespace-pre-wrap">{msg.content}</p>
                        ) : (
                          <div className="chat-message">
                            <ReactMarkdown>{msg.content}</ReactMarkdown>
                          </div>
                        )}
                      </div>
                      {msg.role === 'user' && (
                        <div className="flex items-center gap-1.5 mt-1 mr-1 justify-end">
                          <span className="text-xs text-gray-400">我</span>
                          <User className="w-3.5 h-3.5 text-gray-400" />
                        </div>
                      )}
                    </div>
                  </div>
                ))}

                {/* Typing indicator */}
                {isSending && (
                  <div className="flex justify-start">
                    <div>
                      <div className="flex items-center gap-1.5 mb-1 ml-1">
                        <Bot className="w-3.5 h-3.5 text-[#513CC8]" />
                        <span className="text-xs text-gray-400">智能体</span>
                      </div>
                      <div className="bg-gray-100 rounded-2xl rounded-tl-sm px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <Loader2 className="w-4 h-4 animate-spin text-[#513CC8]" />
                          <span className="text-sm text-gray-400">正在思考...</span>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                <div ref={messagesEndRef} />
              </div>
            )}
          </div>

          {/* 输入区 */}
          <div className="border-t border-gray-100 p-4">
            <div className="max-w-4xl mx-auto flex items-end gap-3">
              <button className="p-2 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors">
                <Paperclip className="w-5 h-5" />
              </button>
              <div className="flex-1 relative">
                <textarea
                  ref={inputRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="输入消息... (Enter 发送，Shift+Enter 换行)"
                  rows={1}
                  className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl resize-none focus:ring-2 focus:ring-[#513CC8] focus:border-transparent outline-none text-sm transition-all"
                  style={{ minHeight: '42px', maxHeight: '120px' }}
                  onInput={(e) => {
                    e.target.style.height = 'auto';
                    e.target.style.height = Math.min(e.target.scrollHeight, 120) + 'px';
                  }}
                />
              </div>
              <button
                onClick={handleSend}
                disabled={!input.trim() || isSending}
                className="px-5 py-2.5 bg-[#513CC8] hover:bg-[#4230A6] text-white rounded-xl text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
              >
                {isSending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Send className="w-4 h-4" />
                )}
                发送
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
