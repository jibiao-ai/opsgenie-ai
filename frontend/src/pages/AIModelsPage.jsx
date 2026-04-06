import React, { useEffect, useState } from 'react';
import {
  Cpu,
  Eye,
  EyeOff,
  CheckCircle,
  AlertCircle,
  Loader2,
  Star,
  Zap,
  Save,
  Plus,
  List,
  Wifi,
  Pencil,
  Trash2,
  X,
  Send,
} from 'lucide-react';
import { getAIProviders, updateAIProvider, testAIProvider } from '../services/api';
import toast from 'react-hot-toast';

// Provider brand colors, icons, and common model hints
const PROVIDER_META = {
  openai:      { color: 'bg-emerald-100', textColor: 'text-emerald-700', borderColor: 'border-emerald-200', abbr: 'OAI', emoji: '🤖', commonModels: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo'] },
  deepseek:    { color: 'bg-blue-100',    textColor: 'text-blue-700',    borderColor: 'border-blue-200',    abbr: 'DS',  emoji: '🔍', commonModels: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-coder'] },
  qwen:        { color: 'bg-orange-100',  textColor: 'text-orange-700',  borderColor: 'border-orange-200',  abbr: 'QW',  emoji: '☁️', commonModels: ['qwen-plus', 'qwen-max', 'qwen-turbo', 'qwen-long'] },
  glm:         { color: 'bg-purple-100',  textColor: 'text-purple-700',  borderColor: 'border-purple-200',  abbr: 'GLM', emoji: '🧠', commonModels: ['glm-4', 'glm-4-flash', 'glm-4-plus', 'glm-4-air'] },
  minimax:     { color: 'bg-pink-100',    textColor: 'text-pink-700',    borderColor: 'border-pink-200',    abbr: 'MM',  emoji: '✨', commonModels: ['abab6.5s-chat', 'abab6.5-chat', 'abab5.5-chat'] },
  siliconflow: { color: 'bg-cyan-100',    textColor: 'text-cyan-700',    borderColor: 'border-cyan-200',    abbr: 'SF',  emoji: '💎', commonModels: ['Qwen/Qwen2.5-7B-Instruct', 'deepseek-ai/DeepSeek-V2.5', 'THUDM/glm-4-9b-chat'] },
  moonshot:    { color: 'bg-indigo-100',  textColor: 'text-indigo-700',  borderColor: 'border-indigo-200',  abbr: 'KM',  emoji: '🌙', commonModels: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k'] },
  baidu:       { color: 'bg-red-100',     textColor: 'text-red-700',     borderColor: 'border-red-200',     abbr: 'BD',  emoji: '🐦', commonModels: ['ernie-4.5-8k', 'ernie-4.0-8k', 'ernie-speed-8k', 'ernie-lite-8k'] },
  zhipu:       { color: 'bg-violet-100',  textColor: 'text-violet-700',  borderColor: 'border-violet-200',  abbr: 'ZP',  emoji: '🎯', commonModels: ['glm-4-flash', 'glm-4', 'glm-4-plus', 'glm-4-air'] },
  volcengine:  { color: 'bg-yellow-100',  textColor: 'text-yellow-700',  borderColor: 'border-yellow-200',  abbr: 'VE',  emoji: '🌋', commonModels: ['doubao-pro-4k', 'doubao-pro-32k', 'doubao-lite-4k', 'doubao-lite-32k'] },
  hunyuan:     { color: 'bg-teal-100',    textColor: 'text-teal-700',    borderColor: 'border-teal-200',    abbr: 'HY',  emoji: '🌊', commonModels: ['hunyuan-pro', 'hunyuan-standard', 'hunyuan-lite'] },
  baichuan:    { color: 'bg-amber-100',   textColor: 'text-amber-700',   borderColor: 'border-amber-200',   abbr: 'BC',  emoji: '🏔️', commonModels: ['Baichuan4', 'Baichuan3-Turbo', 'Baichuan2-Turbo'] },
  anthropic:   { color: 'bg-stone-100',   textColor: 'text-stone-700',   borderColor: 'border-stone-200',   abbr: 'AN',  emoji: '🔮', commonModels: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022', 'claude-3-opus-20240229'] },
  gemini:      { color: 'bg-sky-100',     textColor: 'text-sky-700',     borderColor: 'border-sky-200',     abbr: 'GM',  emoji: '💫', commonModels: ['gemini-2.0-flash', 'gemini-1.5-pro', 'gemini-1.5-flash'] },
};

function getProviderMeta(name) {
  return PROVIDER_META[name] || {
    color: 'bg-gray-100', textColor: 'text-gray-700', borderColor: 'border-gray-200',
    abbr: name.slice(0, 3).toUpperCase(), emoji: '🤖', commonModels: [],
  };
}

// Tab 定义
const TABS = [
  { id: 'list',   label: '已配置模型', icon: List },
  { id: 'add',    label: '添加模型',   icon: Plus },
  { id: 'test',   label: '连通测试',   icon: Wifi },
];

export default function AIModelsPage() {
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('list');

  // 编辑弹窗状态
  const [editProvider, setEditProvider] = useState(null);
  const [editForm, setEditForm] = useState({});
  const [showKey, setShowKey] = useState(false);
  const [saving, setSaving] = useState(false);

  // 添加模型 Tab 状态
  const [addSelected, setAddSelected] = useState(null);
  const [addForm, setAddForm] = useState({ api_key: '', base_url: '', model: '', is_enabled: true, is_default: false });
  const [addShowKey, setAddShowKey] = useState(false);
  const [addSaving, setAddSaving] = useState(false);

  // 连通测试 Tab 状态
  const [testProviderId, setTestProviderId] = useState('');
  const [testMsg, setTestMsg] = useState('你好，请做个简单自我介绍。');
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);

  // 列表中的即时测试
  const [listTesting, setListTesting] = useState({});
  const [listTestResult, setListTestResult] = useState({});

  useEffect(() => {
    loadProviders();
  }, []);

  const loadProviders = async () => {
    try {
      const res = await getAIProviders();
      if (res.code === 0) {
        setProviders(res.data || []);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // 打开编辑
  const openEdit = (provider) => {
    setEditProvider(provider);
    setEditForm({
      api_key: '',
      base_url: provider.base_url || '',
      model: provider.model || '',
      is_default: provider.is_default,
      is_enabled: provider.is_enabled,
    });
    setShowKey(false);
  };

  // 保存编辑
  const handleSave = async () => {
    if (!editProvider) return;
    setSaving(true);
    try {
      const payload = {
        base_url: editForm.base_url,
        model: editForm.model,
        is_default: editForm.is_default,
        is_enabled: editForm.is_enabled,
      };
      if (editForm.api_key.trim() !== '') {
        payload.api_key = editForm.api_key.trim();
      }
      const res = await updateAIProvider(editProvider.id, payload);
      if (res.code === 0) {
        toast.success(`${editProvider.label} 配置已保存`);
        setEditProvider(null);
        loadProviders();
      } else {
        toast.error(res.message || '保存失败');
      }
    } catch (err) {
      toast.error(err?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  // 设为默认
  const handleSetDefault = async (provider) => {
    try {
      const res = await updateAIProvider(provider.id, {
        base_url: provider.base_url,
        model: provider.model,
        is_default: true,
        is_enabled: provider.is_enabled,
      });
      if (res.code === 0) {
        toast.success(`${provider.label} 已设为默认`);
        loadProviders();
      } else {
        toast.error(res.message || '设置失败');
      }
    } catch (err) {
      toast.error(err?.message || '设置失败');
    }
  };

  // 切换启用状态
  const handleToggleEnabled = async (provider) => {
    try {
      const res = await updateAIProvider(provider.id, {
        base_url: provider.base_url,
        model: provider.model,
        is_default: provider.is_default,
        is_enabled: !provider.is_enabled,
      });
      if (res.code === 0) {
        toast.success(provider.is_enabled ? '已禁用' : '已启用');
        loadProviders();
      } else {
        toast.error(res.message || '操作失败');
      }
    } catch (err) {
      toast.error(err?.message || '操作失败');
    }
  };

  // 添加模型提交
  const handleAddSave = async () => {
    if (!addSelected) { toast.error('请先选择厂商'); return; }
    const provider = providers.find((p) => p.name === addSelected);
    if (!provider) { toast.error('未找到该提供商'); return; }
    if (!addForm.api_key.trim()) { toast.error('请输入 API Key'); return; }
    setAddSaving(true);
    try {
      const payload = {
        api_key: addForm.api_key.trim(),
        base_url: addForm.base_url,
        model: addForm.model,
        is_default: addForm.is_default,
        is_enabled: addForm.is_enabled,
      };
      const res = await updateAIProvider(provider.id, payload);
      if (res.code === 0) {
        toast.success(`${provider.label} 配置已保存`);
        setAddForm({ api_key: '', base_url: '', model: '', is_enabled: true, is_default: false });
        setAddSelected(null);
        loadProviders();
        setActiveTab('list');
      } else {
        toast.error(res.message || '保存失败');
      }
    } catch (err) {
      toast.error(err?.message || '保存失败');
    } finally {
      setAddSaving(false);
    }
  };

  // 选中厂商时预填信息
  const handleSelectVendor = (name) => {
    const provider = providers.find((p) => p.name === name);
    setAddSelected(name);
    setAddForm({
      api_key: '',
      base_url: provider?.base_url || '',
      model: provider?.model || getProviderMeta(name).commonModels?.[0] || '',
      is_enabled: true,
      is_default: false,
    });
  };

  // 提取错误消息的辅助函数（兼容多种后端返回格式）
  const extractErrorMessage = (err, fallback = '连接失败，请检查 API Key 和网络') => {
    if (!err) return fallback;
    // 后端 response.BadRequest 返回 {code: -1, message: "xxx"}
    if (typeof err === 'object' && err.message && typeof err.message === 'string') return err.message;
    // Axios 原始错误
    if (err.response?.data?.message) return err.response.data.message;
    if (typeof err === 'string') return err;
    return fallback;
  };

  // 连通测试
  const handleTest = async () => {
    if (!testProviderId) { toast.error('请选择要测试的模型'); return; }
    setTesting(true);
    setTestResult(null);
    try {
      const res = await testAIProvider(testProviderId);
      if (res.code === 0) {
        setTestResult({ ok: true, message: res.data?.message || '连接成功' });
        toast.success('连接测试成功');
      } else {
        const msg = res.message || '连接失败';
        setTestResult({ ok: false, message: msg });
        toast.error(msg);
      }
    } catch (err) {
      const msg = extractErrorMessage(err);
      setTestResult({ ok: false, message: msg });
      toast.error(msg);
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="w-6 h-6 animate-spin" style={{ color: '#513CC8' }} />
      </div>
    );
  }

  const configuredProviders = providers.filter((p) => p.configured);

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">

        {/* 页面卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">

          {/* Tab 栏 */}
          <div className="border-b border-gray-100 px-6 pt-5">
            <div className="flex gap-0">
              {TABS.map((tab) => {
                const Icon = tab.icon;
                const isActive = activeTab === tab.id;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-all mr-1 ${
                      isActive
                        ? 'border-[#513CC8] text-[#513CC8]'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    {tab.label}
                  </button>
                );
              })}
              {/* 统计信息 */}
              <div className="ml-auto flex items-center gap-3 pb-2.5 text-xs text-gray-400">
                <span>共 <strong className="text-gray-600">{providers.length}</strong> 个提供商</span>
                <span className="text-green-600 font-medium">
                  {configuredProviders.length} 个已配置
                </span>
                <span>默认：<strong style={{ color: '#513CC8' }}>
                  {providers.find((p) => p.is_default)?.label || '未设置'}
                </strong></span>
              </div>
            </div>
          </div>

          {/* Tab 内容 */}
          <div className="p-6">

            {/* === 已配置模型 Tab === */}
            {activeTab === 'list' && (
              <div>
                {configuredProviders.length === 0 ? (
                  <div className="text-center py-16">
                    <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                      <Cpu className="w-8 h-8 text-gray-300" />
                    </div>
                    <p className="text-gray-400 mb-3">暂无已配置的模型</p>
                    <button
                      onClick={() => setActiveTab('add')}
                      className="text-sm font-medium px-4 py-2 rounded-lg text-white transition-colors"
                      style={{ background: '#513CC8' }}
                    >
                      立即添加模型
                    </button>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {configuredProviders.map((provider) => {
                      const meta = getProviderMeta(provider.name);
                      const isListTesting = listTesting[provider.id];
                      const listResult = listTestResult[provider.id];
                      return (
                        <div
                          key={provider.id}
                          className="flex items-center gap-4 px-5 py-4 rounded-xl border border-gray-100 hover:border-gray-200 hover:shadow-sm transition-all bg-white"
                        >
                          {/* 图标 */}
                          <div className={`w-10 h-10 ${meta.color} rounded-xl flex items-center justify-center text-lg flex-shrink-0`}>
                            {meta.emoji}
                          </div>

                          {/* 信息 */}
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="font-semibold text-gray-800 text-sm">{provider.label}</span>
                              {provider.is_default && (
                                <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border" style={{ background: '#EEE9FB', color: '#513CC8', borderColor: '#c4b5fd' }}>
                                  <Star className="w-3 h-3" /> 默认
                                </span>
                              )}
                            </div>
                            {provider.base_url && (
                              <p className="text-xs text-gray-400 truncate mt-0.5 max-w-xs">{provider.base_url}</p>
                            )}
                            {provider.model && (
                              <p className="text-xs text-gray-400 mt-0.5">模型：<span className="text-gray-600">{provider.model}</span></p>
                            )}
                          </div>

                          {/* 测试结果 */}
                          {listResult && (
                            <div className={`text-xs px-2 py-1 rounded-lg flex items-center gap-1 ${
                              listResult.ok ? 'bg-green-50 text-green-600' : 'bg-red-50 text-red-600'
                            }`}>
                              {listResult.ok ? <CheckCircle className="w-3 h-3" /> : <AlertCircle className="w-3 h-3" />}
                              {listResult.ok ? '连接正常' : '连接失败'}
                            </div>
                          )}

                          {/* 启用开关 */}
                          <button
                            onClick={() => handleToggleEnabled(provider)}
                            className={`relative w-10 h-5 rounded-full transition-colors flex-shrink-0 ${
                              provider.is_enabled ? 'bg-[#513CC8]' : 'bg-gray-200'
                            }`}
                            title={provider.is_enabled ? '点击禁用' : '点击启用'}
                          >
                            <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-all ${
                              provider.is_enabled ? 'left-5' : 'left-0.5'
                            }`} />
                          </button>

                          {/* 操作按钮 */}
                          <div className="flex items-center gap-1 flex-shrink-0">
                            {!provider.is_default && (
                              <button
                                onClick={() => handleSetDefault(provider)}
                                className="p-1.5 text-gray-400 hover:text-amber-500 hover:bg-amber-50 rounded-lg transition-colors"
                                title="设为默认"
                              >
                                <Star className="w-4 h-4" />
                              </button>
                            )}
                            <button
                              onClick={() => {
                                setListTesting((prev) => ({ ...prev, [provider.id]: true }));
                                setListTestResult((prev) => { const n = { ...prev }; delete n[provider.id]; return n; });
                                testAIProvider(provider.id).then((res) => {
                                  setListTestResult((prev) => ({
                                    ...prev,
                                    [provider.id]: { ok: res.code === 0, message: res.data?.message || res.message },
                                  }));
                                  if (res.code === 0) toast.success('连接正常');
                                  else toast.error(res.message || '连接失败');
                                }).catch((err) => {
                                  const errMsg = extractErrorMessage(err);
                                  setListTestResult((prev) => ({ ...prev, [provider.id]: { ok: false, message: errMsg } }));
                                  toast.error(errMsg);
                                }).finally(() => {
                                  setListTesting((prev) => ({ ...prev, [provider.id]: false }));
                                });
                              }}
                              disabled={isListTesting}
                              className="p-1.5 text-gray-400 hover:text-blue-500 hover:bg-blue-50 rounded-lg transition-colors"
                              title="测试连接"
                            >
                              {isListTesting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wifi className="w-4 h-4" />}
                            </button>
                            <button
                              onClick={() => openEdit(provider)}
                              className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors"
                              title="编辑配置"
                            >
                              <Pencil className="w-4 h-4" />
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* 未配置的提供商（折叠显示） */}
                {providers.filter((p) => !p.configured).length > 0 && (
                  <div className="mt-6">
                    <p className="text-xs text-gray-400 uppercase tracking-wider mb-3">未配置的提供商</p>
                    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
                      {providers.filter((p) => !p.configured).map((provider) => {
                        const meta = getProviderMeta(provider.name);
                        return (
                          <button
                            key={provider.id}
                            onClick={() => { handleSelectVendor(provider.name); setActiveTab('add'); }}
                            className="flex items-center gap-2 px-3 py-2.5 rounded-xl border border-dashed border-gray-200 hover:border-[#513CC8] hover:bg-[#EEE9FB] transition-all text-left group"
                          >
                            <span className="text-base flex-shrink-0">{meta.emoji}</span>
                            <span className="text-sm text-gray-500 group-hover:text-[#513CC8] truncate">{provider.label}</span>
                            <Plus className="w-3.5 h-3.5 text-gray-300 group-hover:text-[#513CC8] ml-auto flex-shrink-0" />
                          </button>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* === 添加模型 Tab === */}
            {activeTab === 'add' && (
              <div className="space-y-6">
                {/* 厂商选择网格 */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 mb-3">选择 AI 提供商</h3>
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
                    {providers.map((provider) => {
                      const meta = getProviderMeta(provider.name);
                      const isSelected = addSelected === provider.name;
                      return (
                        <button
                          key={provider.id}
                          onClick={() => handleSelectVendor(provider.name)}
                          className={`flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl border-2 transition-all ${
                            isSelected
                              ? 'border-[#513CC8] bg-[#EEE9FB]'
                              : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                          }`}
                        >
                          <span className="text-xl">{meta.emoji}</span>
                          <span className={`text-xs font-medium truncate w-full text-center ${isSelected ? 'text-[#513CC8]' : 'text-gray-600'}`}>
                            {provider.label}
                          </span>
                          {provider.configured && (
                            <span className="text-xs text-green-500">✓ 已配置</span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* 配置表单 */}
                {addSelected && (() => {
                  const provider = providers.find((p) => p.name === addSelected);
                  const meta = getProviderMeta(addSelected);
                  return (
                    <div className="border border-gray-100 rounded-xl p-5 space-y-4 bg-gray-50">
                      <div className="flex items-center gap-3 mb-2">
                        <div className={`w-10 h-10 ${meta.color} rounded-xl flex items-center justify-center text-xl`}>
                          {meta.emoji}
                        </div>
                        <div>
                          <h3 className="font-semibold text-gray-800">{provider?.label}</h3>
                          <p className="text-xs text-gray-400">{provider?.description}</p>
                        </div>
                      </div>

                      {/* API Key */}
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1.5">
                          API Key
                          {provider?.configured && (
                            <span className="ml-2 text-xs text-gray-400 font-normal">（当前：{provider.api_key} — 留空则不修改）</span>
                          )}
                        </label>
                        <div className="relative">
                          <input
                            type={addShowKey ? 'text' : 'password'}
                            value={addForm.api_key}
                            onChange={(e) => setAddForm({ ...addForm, api_key: e.target.value })}
                            placeholder={provider?.configured ? '留空保持不变，输入新值则更新' : '请输入 API Key'}
                            className="w-full px-3 py-2 pr-10 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none font-mono bg-white"
                            style={{ '--tw-ring-color': '#513CC8' }}
                          />
                          <button
                            type="button"
                            onClick={() => setAddShowKey(!addShowKey)}
                            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                          >
                            {addShowKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                          </button>
                        </div>
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1.5">Base URL</label>
                          <input
                            type="text"
                            value={addForm.base_url}
                            onChange={(e) => setAddForm({ ...addForm, base_url: e.target.value })}
                            placeholder="https://api.example.com/v1"
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
                          />
                          <p className="text-xs text-gray-400 mt-1">自定义接口地址，默认官方</p>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1.5">默认模型</label>
                          <input
                            type="text"
                            value={addForm.model}
                            onChange={(e) => setAddForm({ ...addForm, model: e.target.value })}
                            placeholder={meta.commonModels?.[0] || 'e.g. gpt-4o'}
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
                          />
                          {meta.commonModels?.length > 0 && (
                            <div className="flex flex-wrap gap-1 mt-1.5">
                              {meta.commonModels.slice(0, 3).map((m) => (
                                <button
                                  key={m}
                                  type="button"
                                  onClick={() => setAddForm({ ...addForm, model: m })}
                                  className="text-xs px-2 py-0.5 bg-gray-200 hover:bg-[#EEE9FB] hover:text-[#513CC8] text-gray-500 rounded-full transition"
                                >
                                  {m}
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-4 pt-1">
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={addForm.is_default}
                            onChange={(e) => setAddForm({ ...addForm, is_default: e.target.checked })}
                            className="w-4 h-4 rounded"
                          />
                          <span className="text-sm text-gray-600">设为默认模型</span>
                        </label>
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={addForm.is_enabled}
                            onChange={(e) => setAddForm({ ...addForm, is_enabled: e.target.checked })}
                            className="w-4 h-4 rounded"
                          />
                          <span className="text-sm text-gray-600">启用</span>
                        </label>
                      </div>

                      <div className="flex gap-3 pt-2">
                        <button
                          onClick={handleAddSave}
                          disabled={addSaving}
                          className="flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
                          style={{ background: '#513CC8' }}
                        >
                          {addSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                          保存配置
                        </button>
                        <button
                          type="button"
                          onClick={() => { setAddSelected(null); setAddForm({ api_key: '', base_url: '', model: '', is_enabled: true, is_default: false }); }}
                          className="px-4 py-2 border border-gray-200 text-gray-600 hover:bg-gray-50 rounded-lg text-sm transition-colors"
                        >
                          重置
                        </button>
                      </div>
                    </div>
                  );
                })()}

                {!addSelected && (
                  <div className="text-center py-8 text-gray-400">
                    <p className="text-sm">请先选择上方的 AI 提供商</p>
                  </div>
                )}
              </div>
            )}

            {/* === 连通测试 Tab === */}
            {activeTab === 'test' && (
              <div className="space-y-5">
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">选择测试模型</label>
                  <select
                    value={testProviderId}
                    onChange={(e) => { setTestProviderId(e.target.value); setTestResult(null); }}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
                  >
                    <option value="">-- 请选择 --</option>
                    {configuredProviders.map((p) => (
                      <option key={p.id} value={p.id}>
                        {getProviderMeta(p.name).emoji} {p.label} {p.model ? `(${p.model})` : ''}
                      </option>
                    ))}
                  </select>
                  {configuredProviders.length === 0 && (
                    <p className="text-xs text-gray-400 mt-1">暂无已配置的模型，请先前往「添加模型」配置。</p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">测试消息</label>
                  <textarea
                    value={testMsg}
                    onChange={(e) => setTestMsg(e.target.value)}
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none resize-none"
                    placeholder="输入测试消息..."
                  />
                </div>

                <button
                  onClick={handleTest}
                  disabled={testing || !testProviderId}
                  className="flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50"
                  style={{ background: '#513CC8' }}
                >
                  {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                  {testing ? '测试中...' : '发送测试'}
                </button>

                {testResult && (
                  <div className={`rounded-xl border p-4 ${
                    testResult.ok
                      ? 'bg-green-50 border-green-200 text-green-700'
                      : 'bg-red-50 border-red-200 text-red-700'
                  }`}>
                    <div className="flex items-center gap-2 mb-2 font-medium">
                      {testResult.ok
                        ? <><CheckCircle className="w-4 h-4" /> 连接成功</>
                        : <><AlertCircle className="w-4 h-4" /> 连接失败</>
                      }
                    </div>
                    <p className="text-sm">{testResult.message}</p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 编辑弹窗 */}
      {editProvider && (
        <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <div className="flex items-center gap-3">
                <span className="text-xl">{getProviderMeta(editProvider.name).emoji}</span>
                <h2 className="text-lg font-semibold text-gray-800">编辑 {editProvider.label}</h2>
              </div>
              <button onClick={() => setEditProvider(null)} className="p-1.5 text-gray-400 hover:text-gray-600 rounded">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1.5">
                  API Key
                  {editProvider.configured && (
                    <span className="ml-2 text-xs text-gray-400 font-normal">（当前：{editProvider.api_key} — 留空则不修改）</span>
                  )}
                </label>
                <div className="relative">
                  <input
                    type={showKey ? 'text' : 'password'}
                    value={editForm.api_key || ''}
                    onChange={(e) => setEditForm({ ...editForm, api_key: e.target.value })}
                    placeholder={editProvider.configured ? '留空保持不变' : '请输入 API Key'}
                    className="w-full px-3 py-2 pr-10 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => setShowKey(!showKey)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  >
                    {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1.5">Base URL</label>
                  <input
                    type="text"
                    value={editForm.base_url || ''}
                    onChange={(e) => setEditForm({ ...editForm, base_url: e.target.value })}
                    placeholder="https://api.example.com/v1"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1.5">默认模型</label>
                  <input
                    type="text"
                    value={editForm.model || ''}
                    onChange={(e) => setEditForm({ ...editForm, model: e.target.value })}
                    placeholder={getProviderMeta(editProvider.name).commonModels?.[0] || ''}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                  />
                </div>
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={editForm.is_default || false}
                    onChange={(e) => setEditForm({ ...editForm, is_default: e.target.checked })}
                    className="w-4 h-4 rounded"
                  />
                  <span className="text-sm text-gray-600">设为默认</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={editForm.is_enabled ?? true}
                    onChange={(e) => setEditForm({ ...editForm, is_enabled: e.target.checked })}
                    className="w-4 h-4 rounded"
                  />
                  <span className="text-sm text-gray-600">启用</span>
                </label>
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-medium text-white transition-colors disabled:opacity-50 flex-1 justify-center"
                  style={{ background: '#513CC8' }}
                >
                  {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  保存
                </button>
                <button
                  onClick={() => setEditProvider(null)}
                  className="px-4 py-2 border border-gray-200 text-gray-600 hover:bg-gray-50 rounded-lg text-sm transition-colors"
                >
                  取消
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
