import React, { useEffect, useState } from 'react';
import { Bot, Plus, Edit2, Trash2, CheckCircle, XCircle, X, Loader2 } from 'lucide-react';
import { getAgents, createAgent, updateAgent, deleteAgent } from '../services/api';
import toast from 'react-hot-toast';

export default function AgentsPage() {
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editAgent, setEditAgent] = useState(null);
  const [form, setForm] = useState({
    name: '', description: '', system_prompt: '', model: 'gpt-4',
    temperature: 0.7, max_tokens: 4096, is_active: true,
  });

  useEffect(() => { loadAgents(); }, []);

  const loadAgents = async () => {
    try {
      const res = await getAgents();
      if (res.code === 0) setAgents(res.data || []);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      if (editAgent) {
        await updateAgent(editAgent.id, form);
        toast.success('智能体已更新');
      } else {
        await createAgent(form);
        toast.success('智能体已创建');
      }
      loadAgents();
      resetForm();
    } catch (err) { toast.error('操作失败'); }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定删除该智能体?')) return;
    try {
      await deleteAgent(id);
      toast.success('已删除');
      loadAgents();
    } catch (err) { toast.error('删除失败'); }
  };

  const handleEdit = (agent) => {
    setEditAgent(agent);
    setForm({
      name: agent.name, description: agent.description,
      system_prompt: agent.system_prompt, model: agent.model,
      temperature: agent.temperature, max_tokens: agent.max_tokens,
      is_active: agent.is_active,
    });
    setShowForm(true);
  };

  const resetForm = () => {
    setShowForm(false);
    setEditAgent(null);
    setForm({ name: '', description: '', system_prompt: '', model: 'gpt-4', temperature: 0.7, max_tokens: 4096, is_active: true });
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-5xl">
        {/* 页面头部卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">智能体管理</h1>
              <p className="text-sm text-gray-400 mt-0.5">配置和管理 AI 运维智能体</p>
            </div>
            <button
              onClick={() => { resetForm(); setShowForm(true); }}
              className="flex items-center gap-2 bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            >
              <Plus className="w-4 h-4" /> 新建智能体
            </button>
          </div>
        </div>

        {/* 新建/编辑表单卡片 */}
        {showForm && (
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
              <h3 className="text-base font-semibold text-gray-800">{editAgent ? '编辑智能体' : '新建智能体'}</h3>
              <button onClick={resetForm} className="p-1.5 text-gray-400 hover:text-gray-600 rounded">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-6">
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-600 mb-1.5">名称</label>
                    <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" required />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-600 mb-1.5">模型</label>
                    <select value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none bg-white">
                      <option value="gpt-4">GPT-4</option>
                      <option value="gpt-4o">GPT-4o</option>
                      <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                      <option value="deepseek-chat">DeepSeek Chat</option>
                      <option value="qwen-plus">Qwen Plus</option>
                    </select>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1.5">描述</label>
                  <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1.5">系统提示词</label>
                  <textarea value={form.system_prompt} onChange={(e) => setForm({ ...form, system_prompt: e.target.value })}
                    rows={6} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none resize-none" />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-600 mb-1.5">Temperature</label>
                    <input type="number" step="0.1" min="0" max="2" value={form.temperature}
                      onChange={(e) => setForm({ ...form, temperature: parseFloat(e.target.value) })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-600 mb-1.5">Max Tokens</label>
                    <input type="number" min="256" max="128000" value={form.max_tokens}
                      onChange={(e) => setForm({ ...form, max_tokens: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
                  </div>
                </div>
                <div className="flex gap-3 pt-2">
                  <button type="submit"
                    className="bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
                    {editAgent ? '更新智能体' : '创建智能体'}
                  </button>
                  <button type="button" onClick={resetForm}
                    className="border border-gray-200 text-gray-600 hover:bg-gray-50 px-4 py-2 rounded-lg text-sm transition-colors">
                    取消
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* 智能体列表卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin text-[#513CC8]" />
            </div>
          ) : agents.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Bot className="w-10 h-10 mx-auto mb-2 opacity-30" />
              <p className="text-sm">暂无智能体，点击"新建智能体"创建第一个</p>
            </div>
          ) : (
            <div className="grid gap-4">
              {agents.map((agent) => (
                <div key={agent.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-[#EEE9FB] rounded-lg flex items-center justify-center mt-0.5">
                        <Bot className="w-5 h-5 text-[#513CC8]" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-800">{agent.name}</h3>
                          <span className="text-xs px-2 py-0.5 bg-gray-100 text-gray-500 rounded">#{agent.id}</span>
                          {agent.is_active ? (
                            <span className="flex items-center gap-1 text-xs text-green-600"><CheckCircle className="w-3 h-3" />活跃</span>
                          ) : (
                            <span className="flex items-center gap-1 text-xs text-gray-400"><XCircle className="w-3 h-3" />停用</span>
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">{agent.description}</p>
                        <div className="flex gap-4 mt-2 text-xs text-gray-400">
                          <span>模型: {agent.model}</span>
                          <span>温度: {agent.temperature}</span>
                          <span>最大Token: {agent.max_tokens}</span>
                        </div>
                      </div>
                    </div>
                    <div className="flex gap-1">
                      <button onClick={() => handleEdit(agent)}
                        className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors">
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button onClick={() => handleDelete(agent.id)}
                        className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
