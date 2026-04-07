import React, { useEffect, useState } from 'react';
import { Bot, Plus, Edit2, Trash2, CheckCircle, XCircle, X, Loader2, Cpu, Thermometer, Hash, MessageSquare, Zap, Cloud, Link2, AlertTriangle } from 'lucide-react';
import { getAgents, createAgent, updateAgent, deleteAgent, getAIProviders, getSkills, getCloudPlatforms } from '../services/api';
import toast from 'react-hot-toast';

// =============== Agent Form Modal ===============
function AgentModal({ visible, onClose, onSubmit, editAgent, availableModels, allSkills, cloudPlatforms }) {
  const [form, setForm] = useState({
    name: '', description: '', system_prompt: '', model: '',
    temperature: 0.7, max_tokens: 4096, is_active: true,
    skill_ids: [], cloud_platform_id: null,
  });

  useEffect(() => {
    if (visible) {
      if (editAgent) {
        const skillIds = (editAgent.agent_skills || []).map((as) => as.skill_id || as.skill?.id).filter(Boolean);
        setForm({
          name: editAgent.name || '',
          description: editAgent.description || '',
          system_prompt: editAgent.system_prompt || '',
          model: editAgent.model || '',
          temperature: editAgent.temperature ?? 0.7,
          max_tokens: editAgent.max_tokens ?? 4096,
          is_active: editAgent.is_active !== false,
          skill_ids: skillIds,
          cloud_platform_id: editAgent.cloud_platform_id || null,
        });
      } else {
        const defaultModel = availableModels.find((m) => m.is_default) || availableModels[0];
        setForm({
          name: '', description: '', system_prompt: '', model: defaultModel?.model || '',
          temperature: 0.7, max_tokens: 4096, is_active: true, skill_ids: [], cloud_platform_id: null,
        });
      }
    }
  }, [visible, editAgent, availableModels]);

  const toggleSkill = (skillId) => {
    setForm((prev) => ({
      ...prev,
      skill_ids: prev.skill_ids.includes(skillId)
        ? prev.skill_ids.filter((id) => id !== skillId)
        : [...prev.skill_ids, skillId],
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit({
      name: form.name,
      description: form.description,
      system_prompt: form.system_prompt,
      model: form.model,
      temperature: parseFloat(form.temperature) || 0.7,
      max_tokens: parseInt(form.max_tokens) || 4096,
      is_active: form.is_active,
      skill_ids: form.skill_ids,
      cloud_platform_id: form.cloud_platform_id || null,
    });
  };

  if (!visible) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />

      {/* Modal */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-2xl mx-4 max-h-[90vh] flex flex-col animate-in fade-in zoom-in-95">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between flex-shrink-0">
          <h3 className="text-base font-semibold text-gray-800">
            {editAgent ? `编辑智能体 - ${editAgent.name}` : '新建智能体'}
          </h3>
          <button onClick={onClose} className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Body - scrollable */}
        <div className="p-6 overflow-y-auto flex-1">
          <form id="agent-form" onSubmit={handleSubmit} className="space-y-4">
            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">
                <span className="flex items-center gap-1.5"><Bot className="w-3.5 h-3.5" />名称</span>
              </label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="输入智能体名称，例如：EasyStack 运维助手"
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" required />
            </div>

            {/* Model */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">
                <span className="flex items-center gap-1.5"><Cpu className="w-3.5 h-3.5" />模型厂商 / 模型</span>
              </label>
              {availableModels.length > 0 ? (
                <select value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none bg-white">
                  <option value="">-- 请选择模型 --</option>
                  {availableModels.map((m) => (
                    <option key={m.name} value={m.model}>
                      {m.label} - {m.model}{m.is_default ? ' ★ 默认' : ''}
                    </option>
                  ))}
                </select>
              ) : (
                <div className="w-full px-3 py-2 border border-orange-200 bg-orange-50 rounded-lg text-sm text-orange-600">
                  暂无已配置的模型，请先前往「模型配置」页面添加
                </div>
              )}
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">描述</label>
              <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="输入智能体描述"
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
            </div>

            {/* System Prompt */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">
                <span className="flex items-center gap-1.5"><MessageSquare className="w-3.5 h-3.5" />系统提示词</span>
              </label>
              <textarea value={form.system_prompt} onChange={(e) => setForm({ ...form, system_prompt: e.target.value })}
                placeholder="输入系统提示词，定义智能体的角色和行为..."
                rows={6} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none resize-none" />
            </div>

            {/* Skill Association */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">
                <span className="flex items-center gap-1.5"><Zap className="w-3.5 h-3.5" />关联技能</span>
              </label>
              {allSkills.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {allSkills.map((skill) => {
                    const isSelected = form.skill_ids.includes(skill.id);
                    return (
                      <button
                        key={skill.id}
                        type="button"
                        onClick={() => toggleSkill(skill.id)}
                        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm border transition-all ${
                          isSelected
                            ? 'bg-[#EEE9FB] border-[#513CC8] text-[#513CC8] font-medium'
                            : 'bg-white border-gray-200 text-gray-500 hover:border-[#513CC8] hover:text-[#513CC8]'
                        }`}
                      >
                        <Zap className="w-3.5 h-3.5" />
                        {skill.name}
                        {isSelected && <CheckCircle className="w-3.5 h-3.5" />}
                      </button>
                    );
                  })}
                </div>
              ) : (
                <p className="text-sm text-gray-400">暂无可用技能</p>
              )}
              <p className="text-xs text-gray-400 mt-1">
                选择技能后，智能体在对话中可通过 Function Calling 调用相应的云平台 API
              </p>
            </div>

            {/* Cloud Platform Binding */}
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1.5">
                <span className="flex items-center gap-1.5"><Cloud className="w-3.5 h-3.5" />绑定云平台</span>
              </label>
              <select
                value={form.cloud_platform_id || ''}
                onChange={(e) => setForm({ ...form, cloud_platform_id: e.target.value ? parseInt(e.target.value) : null })}
                className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none bg-white"
              >
                <option value="">-- 自动选择（使用首个已连接平台）--</option>
                {cloudPlatforms.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} ({p.type}) {p.status === 'connected' ? '✓ 已连接' : p.status === 'failed' ? '✕ 连接失败' : ''}
                  </option>
                ))}
              </select>
              <p className="text-xs text-gray-400 mt-1">
                绑定云平台后，智能体执行技能时将通过该平台的 Token 认证进行 API 调用
              </p>
            </div>

            {/* Temperature / Max Tokens */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1.5">
                  <span className="flex items-center gap-1.5"><Thermometer className="w-3.5 h-3.5" />温度参数 (Temperature)</span>
                </label>
                <input type="number" step="0.1" min="0" max="2" value={form.temperature}
                  onChange={(e) => setForm({ ...form, temperature: parseFloat(e.target.value) || 0 })}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
                <p className="text-xs text-gray-400 mt-1">值越大回答越随机，推荐 0.1-0.5 用于精确任务</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1.5">
                  <span className="flex items-center gap-1.5"><Hash className="w-3.5 h-3.5" />最大令牌数 (Max Tokens)</span>
                </label>
                <input type="number" min="256" max="128000" value={form.max_tokens}
                  onChange={(e) => setForm({ ...form, max_tokens: parseInt(e.target.value) || 4096 })}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-[#513CC8] outline-none" />
                <p className="text-xs text-gray-400 mt-1">控制模型单次回复的最大长度，一般 4096 即可</p>
              </div>
            </div>

            {/* Active toggle */}
            <div className="flex items-center gap-3">
              <label className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" checked={form.is_active}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                  className="sr-only peer" />
                <div className="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#513CC8]"></div>
              </label>
              <span className="text-sm text-gray-600">{form.is_active ? '已启用' : '已停用'}</span>
            </div>
          </form>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-100 flex justify-end gap-3 flex-shrink-0">
          <button type="button" onClick={onClose}
            className="border border-gray-200 text-gray-600 hover:bg-gray-50 px-4 py-2 rounded-lg text-sm transition-colors">
            取消
          </button>
          <button type="submit" form="agent-form"
            className="bg-[#513CC8] hover:bg-[#4230A6] text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors">
            {editAgent ? '保存修改' : '创建智能体'}
          </button>
        </div>
      </div>
    </div>
  );
}

// =============== Delete Confirmation Modal ===============
function DeleteConfirmModal({ visible, onClose, onConfirm, agentName, loading }) {
  if (!visible) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />

      {/* Modal */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-md mx-4 animate-in fade-in zoom-in-95">
        {/* Body */}
        <div className="p-6 text-center">
          <div className="w-14 h-14 bg-red-50 rounded-full flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-7 h-7 text-red-500" />
          </div>
          <h3 className="text-lg font-semibold text-gray-800 mb-2">确认删除智能体</h3>
          <p className="text-sm text-gray-500 leading-relaxed">
            你确定要删除智能体 <span className="font-medium text-gray-700">"{agentName}"</span> 吗？
            <br />
            <span className="text-red-400">此操作不可撤销</span>，该智能体的所有配置、关联技能和会话记录都将被永久删除。
          </p>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-100 flex justify-end gap-3">
          <button
            onClick={onClose}
            disabled={loading}
            className="border border-gray-200 text-gray-600 hover:bg-gray-50 px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-50"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className="bg-red-500 hover:bg-red-600 text-white px-5 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 flex items-center gap-1.5"
          >
            {loading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
            {loading ? '删除中...' : '确认删除'}
          </button>
        </div>
      </div>
    </div>
  );
}

// =============== Main Page ===============
export default function AgentsPage() {
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalVisible, setModalVisible] = useState(false);
  const [editAgent, setEditAgent] = useState(null);
  const [availableModels, setAvailableModels] = useState([]);
  const [allSkills, setAllSkills] = useState([]);
  const [cloudPlatforms, setCloudPlatforms] = useState([]);
  const [deleteTarget, setDeleteTarget] = useState(null); // agent to delete
  const [deleteLoading, setDeleteLoading] = useState(false);

  useEffect(() => {
    loadAgents();
    loadModels();
    loadSkills();
    loadCloudPlatforms();
  }, []);

  const loadModels = async () => {
    try {
      const res = await getAIProviders();
      if (res.code === 0 && res.data) {
        const models = res.data
          .filter((p) => p.is_enabled && p.configured)
          .map((p) => ({
            name: p.name, label: p.label, model: p.model, is_default: p.is_default,
          }));
        setAvailableModels(models);
      }
    } catch (err) { console.error('Failed to load AI models:', err); }
  };

  const loadSkills = async () => {
    try {
      const res = await getSkills();
      if (res.code === 0) setAllSkills(res.data || []);
    } catch (err) { console.error('Failed to load skills:', err); }
  };

  const loadCloudPlatforms = async () => {
    try {
      const res = await getCloudPlatforms();
      if (res.code === 0) setCloudPlatforms(res.data || []);
    } catch (err) { console.error('Failed to load cloud platforms:', err); }
  };

  const loadAgents = async () => {
    try {
      const res = await getAgents();
      if (res.code === 0) setAgents(res.data || []);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const handleOpenCreate = () => {
    setEditAgent(null);
    setModalVisible(true);
  };

  const handleOpenEdit = (agent) => {
    setEditAgent(agent);
    setModalVisible(true);
  };

  const handleModalClose = () => {
    setModalVisible(false);
    setEditAgent(null);
  };

  const handleModalSubmit = async (payload) => {
    try {
      if (editAgent) {
        await updateAgent(editAgent.id, payload);
        toast.success('智能体已更新');
      } else {
        await createAgent(payload);
        toast.success('智能体已创建');
      }
      loadAgents();
      handleModalClose();
    } catch (err) {
      const msg = err?.message || err?.data?.message || '操作失败';
      toast.error(msg);
    }
  };

  const handleOpenDelete = (agent) => {
    setDeleteTarget(agent);
  };

  const handleCloseDelete = () => {
    if (!deleteLoading) {
      setDeleteTarget(null);
    }
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleteLoading(true);
    try {
      await deleteAgent(deleteTarget.id);
      toast.success(`智能体"${deleteTarget.name}"已删除`);
      setDeleteTarget(null);
      loadAgents();
    } catch (err) {
      toast.error('删除失败，请重试');
    } finally {
      setDeleteLoading(false);
    }
  };

  const getModelProviderLabel = (modelName) => {
    if (!modelName) return '未设置';
    const provider = availableModels.find(m => m.model === modelName);
    return provider ? `${provider.label} (${modelName})` : modelName;
  };

  const getAgentSkillNames = (agent) => {
    if (!agent.agent_skills || agent.agent_skills.length === 0) return [];
    return agent.agent_skills
      .map((as) => as.skill?.name || allSkills.find((s) => s.id === as.skill_id)?.name)
      .filter(Boolean);
  };

  const getAgentPlatformName = (agent) => {
    if (agent.cloud_platform) return agent.cloud_platform.name;
    if (agent.cloud_platform_id) {
      const p = cloudPlatforms.find((cp) => cp.id === agent.cloud_platform_id);
      return p?.name || `ID:${agent.cloud_platform_id}`;
    }
    return null;
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        {/* Header */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">智能体管理</h1>
              <p className="text-sm text-gray-400 mt-0.5">配置和管理 AI 运维智能体，关联技能与云平台</p>
            </div>
            <button onClick={handleOpenCreate}
              className="flex items-center gap-2 bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
              <Plus className="w-4 h-4" /> 新建智能体
            </button>
          </div>
        </div>

        {/* Agent List */}
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
              {agents.map((agent) => {
                const skillNames = getAgentSkillNames(agent);
                const platformName = getAgentPlatformName(agent);
                return (
                  <div key={agent.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow">
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-3">
                        <div className="w-10 h-10 bg-[#EEE9FB] rounded-lg flex items-center justify-center mt-0.5">
                          <Bot className="w-5 h-5 text-[#513CC8]" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="font-semibold text-gray-800">{agent.name}</h3>
                            <span className="text-xs px-2 py-0.5 bg-gray-100 text-gray-500 rounded">#{agent.id}</span>
                            {agent.is_active ? (
                              <span className="flex items-center gap-1 text-xs text-green-600"><CheckCircle className="w-3 h-3" />活跃</span>
                            ) : (
                              <span className="flex items-center gap-1 text-xs text-gray-400"><XCircle className="w-3 h-3" />停用</span>
                            )}
                          </div>
                          <p className="text-sm text-gray-500 mt-1">{agent.description}</p>
                          <div className="flex gap-4 mt-2 text-xs text-gray-400 flex-wrap">
                            <span className="flex items-center gap-1">
                              <Cpu className="w-3 h-3" /> 模型: {agent.model || '未设置'}
                            </span>
                            <span className="flex items-center gap-1">
                              <Thermometer className="w-3 h-3" /> 温度: {agent.temperature ?? '-'}
                            </span>
                            <span className="flex items-center gap-1">
                              <Hash className="w-3 h-3" /> 最大令牌: {agent.max_tokens ?? '-'}
                            </span>
                          </div>

                          {/* Skill and Platform tags */}
                          <div className="flex flex-wrap gap-1.5 mt-2">
                            {skillNames.map((name, i) => (
                              <span key={i} className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-purple-50 text-purple-600 rounded-full border border-purple-100">
                                <Zap className="w-3 h-3" />{name}
                              </span>
                            ))}
                            {platformName && (
                              <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-blue-50 text-blue-600 rounded-full border border-blue-100">
                                <Cloud className="w-3 h-3" />{platformName}
                              </span>
                            )}
                            {skillNames.length === 0 && !platformName && (
                              <span className="text-xs text-gray-300">未关联技能或云平台</span>
                            )}
                          </div>

                          {agent.system_prompt && (
                            <div className="mt-2 text-xs text-gray-400 bg-gray-50 rounded px-2 py-1 max-h-16 overflow-hidden">
                              <span className="font-medium">提示词: </span>
                              {agent.system_prompt.length > 100 ? agent.system_prompt.slice(0, 100) + '...' : agent.system_prompt}
                            </div>
                          )}
                        </div>
                      </div>
                      <div className="flex gap-1 flex-shrink-0 ml-2">
                        <button onClick={() => handleOpenEdit(agent)}
                          title="编辑智能体"
                          className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors">
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button onClick={() => handleOpenDelete(agent)}
                          title="删除智能体"
                          className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirm Modal */}
      <DeleteConfirmModal
        visible={!!deleteTarget}
        onClose={handleCloseDelete}
        onConfirm={handleConfirmDelete}
        agentName={deleteTarget?.name || ''}
        loading={deleteLoading}
      />

      {/* Agent Modal */}
      <AgentModal
        visible={modalVisible}
        onClose={handleModalClose}
        onSubmit={handleModalSubmit}
        editAgent={editAgent}
        availableModels={availableModels}
        allSkills={allSkills}
        cloudPlatforms={cloudPlatforms}
      />
    </div>
  );
}
