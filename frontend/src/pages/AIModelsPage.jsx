import React, { useEffect, useState } from 'react';
import {
  Cpu,
  ChevronDown,
  ChevronUp,
  Eye,
  EyeOff,
  CheckCircle,
  AlertCircle,
  Loader2,
  Star,
  Zap,
  Save,
} from 'lucide-react';
import { getAIProviders, updateAIProvider, testAIProvider } from '../services/api';
import toast from 'react-hot-toast';

// Provider brand colors and icons (text-based since we can't load external images reliably)
const PROVIDER_META = {
  openai:   { color: 'bg-emerald-100', textColor: 'text-emerald-700', borderColor: 'border-emerald-200', abbr: 'OAI' },
  deepseek: { color: 'bg-blue-100',    textColor: 'text-blue-700',    borderColor: 'border-blue-200',    abbr: 'DS'  },
  qwen:     { color: 'bg-orange-100',  textColor: 'text-orange-700',  borderColor: 'border-orange-200',  abbr: 'QW'  },
  glm:      { color: 'bg-purple-100',  textColor: 'text-purple-700',  borderColor: 'border-purple-200',  abbr: 'GLM' },
  minimax:  { color: 'bg-pink-100',    textColor: 'text-pink-700',    borderColor: 'border-pink-200',    abbr: 'MM'  },
};

function getProviderMeta(name) {
  return PROVIDER_META[name] || { color: 'bg-gray-100', textColor: 'text-gray-700', borderColor: 'border-gray-200', abbr: name.slice(0, 3).toUpperCase() };
}

export default function AIModelsPage() {
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  // expandedId: which provider card is expanded
  const [expandedId, setExpandedId] = useState(null);
  // forms: { [id]: { api_key, base_url, model, is_default, is_enabled } }
  const [forms, setForms] = useState({});
  // showKey: { [id]: bool } - toggle API key visibility
  const [showKey, setShowKey] = useState({});
  // saving: { [id]: bool }
  const [saving, setSaving] = useState({});
  // testing: { [id]: bool }
  const [testing, setTesting] = useState({});
  // testResult: { [id]: { ok: bool, message: string } }
  const [testResult, setTestResult] = useState({});

  useEffect(() => {
    loadProviders();
  }, []);

  const loadProviders = async () => {
    try {
      const res = await getAIProviders();
      if (res.code === 0) {
        const data = res.data || [];
        setProviders(data);
        // Init forms
        const fs = {};
        data.forEach((p) => {
          fs[p.id] = {
            api_key:    '',  // always start blank so user must type a new key to change
            base_url:   p.base_url  || '',
            model:      p.model     || '',
            is_default: p.is_default,
            is_enabled: p.is_enabled,
          };
        });
        setForms(fs);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleExpand = (id) => {
    setExpandedId((prev) => (prev === id ? null : id));
    // Clear test result when collapsing
    if (expandedId === id) {
      setTestResult((prev) => { const n = { ...prev }; delete n[id]; return n; });
    }
  };

  const handleFormChange = (id, field, value) => {
    setForms((prev) => ({ ...prev, [id]: { ...prev[id], [field]: value } }));
  };

  const handleSave = async (provider) => {
    const form = forms[provider.id];
    if (!form) return;

    setSaving((prev) => ({ ...prev, [provider.id]: true }));
    try {
      const payload = {
        base_url:   form.base_url,
        model:      form.model,
        is_default: form.is_default,
        is_enabled: form.is_enabled,
      };
      // Only send api_key if user entered something new
      if (form.api_key.trim() !== '') {
        payload.api_key = form.api_key.trim();
      }

      const res = await updateAIProvider(provider.id, payload);
      if (res.code === 0) {
        toast.success(`${provider.label} 配置已保存`);
        // Clear the api_key field after save
        setForms((prev) => ({ ...prev, [provider.id]: { ...prev[provider.id], api_key: '' } }));
        loadProviders();
      } else {
        toast.error(res.message || '保存失败');
      }
    } catch (err) {
      toast.error(err?.message || '保存失败');
    } finally {
      setSaving((prev) => ({ ...prev, [provider.id]: false }));
    }
  };

  const handleSetDefault = async (provider) => {
    setSaving((prev) => ({ ...prev, [provider.id]: true }));
    try {
      const form = forms[provider.id];
      const payload = {
        base_url:   form?.base_url   || provider.base_url,
        model:      form?.model      || provider.model,
        is_default: true,
        is_enabled: form?.is_enabled ?? provider.is_enabled,
      };
      const res = await updateAIProvider(provider.id, payload);
      if (res.code === 0) {
        toast.success(`${provider.label} 已设为默认`);
        loadProviders();
      } else {
        toast.error(res.message || '设置失败');
      }
    } catch (err) {
      toast.error(err?.message || '设置失败');
    } finally {
      setSaving((prev) => ({ ...prev, [provider.id]: false }));
    }
  };

  const handleTest = async (provider) => {
    setTesting((prev) => ({ ...prev, [provider.id]: true }));
    setTestResult((prev) => { const n = { ...prev }; delete n[provider.id]; return n; });
    try {
      const res = await testAIProvider(provider.id);
      if (res.code === 0) {
        setTestResult((prev) => ({ ...prev, [provider.id]: { ok: true, message: res.data?.message || '连接成功' } }));
        toast.success('连接测试成功');
      } else {
        setTestResult((prev) => ({ ...prev, [provider.id]: { ok: false, message: res.message || '连接失败' } }));
        toast.error(res.message || '连接失败');
      }
    } catch (err) {
      const msg = err?.message || '连接失败，请检查 API Key 和网络';
      setTestResult((prev) => ({ ...prev, [provider.id]: { ok: false, message: msg } }));
      toast.error(msg);
    } finally {
      setTesting((prev) => ({ ...prev, [provider.id]: false }));
    }
  };

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      {/* Page Header */}
      <div className="bg-gradient-to-r from-blue-600 to-cyan-600 px-8 py-6">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-white/20 rounded-xl flex items-center justify-center">
            <Cpu className="w-6 h-6 text-white" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">AI 模型配置</h1>
            <p className="text-blue-100 text-sm mt-0.5">配置主流 AI 服务提供商的 API Key 和模型参数</p>
          </div>
        </div>
      </div>

      {/* Provider Cards */}
      <div className="p-6 max-w-4xl mx-auto space-y-4">
        {/* Summary bar */}
        <div className="flex items-center gap-4 text-sm text-gray-500 mb-2">
          <span>共 {providers.length} 个提供商</span>
          <span>·</span>
          <span className="text-green-600 font-medium">
            {providers.filter((p) => p.configured).length} 个已配置
          </span>
          <span>·</span>
          <span className="text-blue-600 font-medium">
            {providers.find((p) => p.is_default)?.label || '未设置'} 为默认
          </span>
        </div>

        {providers.map((provider) => {
          const meta = getProviderMeta(provider.name);
          const isExpanded = expandedId === provider.id;
          const form = forms[provider.id] || {};
          const isSaving = saving[provider.id];
          const isTesting = testing[provider.id];
          const result = testResult[provider.id];

          return (
            <div
              key={provider.id}
              className={`bg-white rounded-xl border transition-all duration-200 ${
                isExpanded ? 'border-blue-300 shadow-md' : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
              }`}
            >
              {/* Card Header */}
              <div
                className="flex items-center px-5 py-4 cursor-pointer select-none"
                onClick={() => handleToggleExpand(provider.id)}
              >
                {/* Provider Logo */}
                <div className={`w-11 h-11 ${meta.color} rounded-xl flex items-center justify-center font-bold text-sm ${meta.textColor} mr-4 flex-shrink-0`}>
                  {meta.abbr}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-gray-800">{provider.label}</span>
                    {provider.is_default && (
                      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-blue-50 text-blue-600 rounded-full border border-blue-200">
                        <Star className="w-3 h-3" />
                        默认
                      </span>
                    )}
                    {provider.configured ? (
                      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-green-50 text-green-600 rounded-full border border-green-200">
                        <CheckCircle className="w-3 h-3" />
                        已配置
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-gray-100 text-gray-400 rounded-full border border-gray-200">
                        <AlertCircle className="w-3 h-3" />
                        未配置
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-400 mt-0.5 truncate">{provider.description}</p>
                  {!isExpanded && provider.model && (
                    <p className="text-xs text-gray-400 mt-0.5">
                      模型: <span className="text-gray-600">{provider.model}</span>
                    </p>
                  )}
                </div>

                {/* Expand icon */}
                <div className="ml-3 text-gray-400 flex-shrink-0">
                  {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                </div>
              </div>

              {/* Expandable Form */}
              {isExpanded && (
                <div className="px-5 pb-5 border-t border-gray-100">
                  <div className="pt-4 space-y-4">
                    {/* API Key */}
                    <div>
                      <label className="block text-sm font-medium text-gray-600 mb-1">
                        API Key
                        {provider.configured && (
                          <span className="ml-2 text-xs text-gray-400 font-normal">
                            (当前: {provider.api_key} — 留空则不修改)
                          </span>
                        )}
                      </label>
                      <div className="relative">
                        <input
                          type={showKey[provider.id] ? 'text' : 'password'}
                          value={form.api_key || ''}
                          onChange={(e) => handleFormChange(provider.id, 'api_key', e.target.value)}
                          placeholder={provider.configured ? '留空保持不变，输入新值则更新' : '请输入 API Key'}
                          className="w-full px-3 py-2 pr-10 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono"
                        />
                        <button
                          type="button"
                          onClick={() => setShowKey((prev) => ({ ...prev, [provider.id]: !prev[provider.id] }))}
                          className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                        >
                          {showKey[provider.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </button>
                      </div>
                    </div>

                    {/* Base URL + Model in a row */}
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">Base URL</label>
                        <input
                          type="text"
                          value={form.base_url || ''}
                          onChange={(e) => handleFormChange(provider.id, 'base_url', e.target.value)}
                          placeholder="https://api.example.com/v1"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">默认模型</label>
                        <input
                          type="text"
                          value={form.model || ''}
                          onChange={(e) => handleFormChange(provider.id, 'model', e.target.value)}
                          placeholder="e.g. gpt-4o"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                        />
                      </div>
                    </div>

                    {/* Test result banner */}
                    {result && (
                      <div
                        className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm ${
                          result.ok
                            ? 'bg-green-50 text-green-700 border border-green-200'
                            : 'bg-red-50 text-red-700 border border-red-200'
                        }`}
                      >
                        {result.ok ? (
                          <CheckCircle className="w-4 h-4 flex-shrink-0" />
                        ) : (
                          <AlertCircle className="w-4 h-4 flex-shrink-0" />
                        )}
                        <span>{result.message}</span>
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex items-center gap-3 pt-1">
                      {/* Save */}
                      <button
                        onClick={() => handleSave(provider)}
                        disabled={isSaving}
                        className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition"
                      >
                        {isSaving ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Save className="w-4 h-4" />
                        )}
                        保存配置
                      </button>

                      {/* Test Connection */}
                      <button
                        onClick={() => handleTest(provider)}
                        disabled={isTesting || !provider.configured}
                        title={!provider.configured ? '请先保存 API Key 才能测试连接' : ''}
                        className="flex items-center gap-1.5 px-4 py-2 bg-gray-100 hover:bg-gray-200 disabled:opacity-40 text-gray-700 rounded-lg text-sm font-medium transition"
                      >
                        {isTesting ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Zap className="w-4 h-4" />
                        )}
                        测试连接
                      </button>

                      {/* Set as Default */}
                      {!provider.is_default && (
                        <button
                          onClick={() => handleSetDefault(provider)}
                          disabled={isSaving}
                          className="flex items-center gap-1.5 px-4 py-2 border border-gray-200 hover:border-blue-300 hover:bg-blue-50 disabled:opacity-40 text-gray-600 hover:text-blue-600 rounded-lg text-sm font-medium transition ml-auto"
                        >
                          <Star className="w-4 h-4" />
                          设为默认
                        </button>
                      )}
                      {provider.is_default && (
                        <span className="ml-auto flex items-center gap-1 text-xs text-blue-600 bg-blue-50 px-3 py-2 rounded-lg border border-blue-200">
                          <Star className="w-3.5 h-3.5" />
                          当前默认
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {providers.length === 0 && (
          <div className="text-center py-16 text-gray-400">
            <Cpu className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p>暂无 AI 提供商配置</p>
          </div>
        )}
      </div>
    </div>
  );
}
