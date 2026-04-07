import React, { useEffect, useState } from 'react';
import {
  Cloud,
  Plus,
  Trash2,
  Edit2,
  CheckCircle,
  XCircle,
  Loader2,
  Zap,
  AlertCircle,
  X,
  List,
  Server,
} from 'lucide-react';
import {
  getCloudPlatforms,
  createCloudPlatform,
  updateCloudPlatform,
  deleteCloudPlatform,
  testCloudPlatform,
} from '../services/api';
import toast from 'react-hot-toast';

const PLATFORM_TYPES = [
  { value: 'easystack', label: 'EasyStack', description: '私有云平台，基于OpenStack', emoji: '☁️', color: 'bg-blue-100', textColor: 'text-blue-700', badgeBg: 'bg-blue-50', badgeText: 'text-blue-600', badgeBorder: 'border-blue-200' },
  { value: 'zstack',    label: 'ZStack',    description: '企业级私有云平台',           emoji: '🖥️', color: 'bg-green-100', textColor: 'text-green-700', badgeBg: 'bg-green-50', badgeText: 'text-green-600', badgeBorder: 'border-green-200' },
];

const emptyEasyStack = {
  name: '', type: 'easystack',
  host_ip: '', base_domain: '',
  auth_url: '', username: '', password: '',
  domain_name: '', project_name: '', project_id: '',
  description: '',
};

const emptyZStack = {
  name: '', type: 'zstack',
  endpoint: '', access_key_id: '', access_key_secret: '',
  description: '',
};

// Tab 定义
const TABS = [
  { id: 'list', label: '已接入平台', icon: List },
  { id: 'add',  label: '添加平台',   icon: Plus },
];

function StatusDot({ status }) {
  if (status === 'connected') return (
    <span className="flex items-center gap-1.5 text-xs text-green-600">
      <span className="w-2 h-2 bg-green-500 rounded-full" />已连接
    </span>
  );
  if (status === 'failed') return (
    <span className="flex items-center gap-1.5 text-xs text-red-500">
      <span className="w-2 h-2 bg-red-500 rounded-full" />连接失败
    </span>
  );
  return (
    <span className="flex items-center gap-1.5 text-xs text-gray-400">
      <span className="w-2 h-2 bg-gray-300 rounded-full" />未测试
    </span>
  );
}

function getStatusBadge(status) {
  if (status === 'connected') return (
    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-green-50 text-green-600 rounded-full border border-green-200">
      <CheckCircle className="w-3 h-3" /> 已连接
    </span>
  );
  if (status === 'failed') return (
    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-red-50 text-red-600 rounded-full border border-red-200">
      <XCircle className="w-3 h-3" /> 连接失败
    </span>
  );
  return (
    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-gray-100 text-gray-400 rounded-full border border-gray-200">
      <AlertCircle className="w-3 h-3" /> 未测试
    </span>
  );
}

// 编辑/新增 Modal（保留原有逻辑）
function PlatformModal({ open, onClose, onSaved, editPlatform }) {
  const isEdit = !!editPlatform;
  const [form, setForm] = useState(editPlatform || { ...emptyEasyStack });
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);

  useEffect(() => {
    if (open) {
      setForm(editPlatform || { ...emptyEasyStack });
      setTestResult(null);
    }
  }, [open, editPlatform]);

  if (!open) return null;

  const handleTypeChange = (type) => {
    setForm(type === 'zstack' ? { ...emptyZStack } : { ...emptyEasyStack });
    setTestResult(null);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      let res;
      if (isEdit) {
        res = await updateCloudPlatform(editPlatform.id, form);
      } else {
        res = await createCloudPlatform(form);
      }
      if (res.code === 0) {
        toast.success(isEdit ? '平台已更新' : '平台已添加');
        onSaved();
        onClose();
      } else {
        toast.error(res.message || '操作失败');
      }
    } catch (err) {
      toast.error(err?.message || '操作失败');
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    if (!editPlatform?.id) {
      toast.error('请先保存平台配置，再进行连接测试');
      return;
    }
    setTesting(true);
    setTestResult(null);
    try {
      const res = await testCloudPlatform(editPlatform.id);
      if (res.code === 0) {
        setTestResult({ ok: true, message: res.data?.message || '连接成功' });
        toast.success('连接测试成功');
      } else {
        setTestResult({ ok: false, message: res.message || '连接失败' });
        toast.error(res.message || '连接失败');
      }
    } catch (err) {
      const msg = err?.message || '连接测试失败';
      setTestResult({ ok: false, message: msg });
      toast.error(msg);
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-lg font-semibold text-gray-800">
            {isEdit ? '编辑平台' : '接入平台'}
          </h2>
          <button onClick={onClose} className="p-1.5 text-gray-400 hover:text-gray-600 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {/* 平台类型 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-2">平台类型</label>
            <div className="grid grid-cols-2 gap-3">
              {PLATFORM_TYPES.map((pt) => (
                <button
                  key={pt.value}
                  type="button"
                  onClick={() => handleTypeChange(pt.value)}
                  className={`p-3 rounded-xl border-2 text-left transition ${
                    form.type === pt.value
                      ? 'border-[#513CC8] bg-[#EEE9FB]'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-lg">{pt.emoji}</span>
                    <span className={`font-semibold text-sm ${form.type === pt.value ? 'text-[#513CC8]' : 'text-gray-700'}`}>
                      {pt.label}
                    </span>
                  </div>
                  <div className="text-xs text-gray-400">{pt.description}</div>
                </button>
              ))}
            </div>
          </div>

          {/* 平台名称 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">平台名称 *</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="例如：生产环境-EasyStack"
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
              required
            />
          </div>

          {/* EasyStack 字段 */}
          {form.type === 'easystack' && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">平台 IP 地址 *</label>
                  <input
                    type="text"
                    value={form.host_ip}
                    onChange={(e) => setForm({ ...form, host_ip: e.target.value })}
                    placeholder="192.168.3.204"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">根域名 *</label>
                  <input
                    type="text"
                    value={form.base_domain}
                    onChange={(e) => setForm({ ...form, base_domain: e.target.value })}
                    placeholder="opsl2.svc.cluster.local"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required
                  />
                </div>
              </div>
              {form.host_ip && form.base_domain && (
                <div className="bg-blue-50 border border-blue-200 rounded-lg px-3 py-2 text-xs text-blue-700 space-y-0.5">
                  <p className="font-medium">自动解析的服务地址：</p>
                  <p>认证: keystone.{form.base_domain}</p>
                  <p>计算: nova.{form.base_domain}</p>
                  <p>监控: emla.{form.base_domain}</p>
                  <p className="text-blue-500">存储 / 网络 / 镜像等服务地址将自动拼接</p>
                </div>
              )}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">用户名 *</label>
                  <input
                    type="text"
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    placeholder="admin"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">密码 *</label>
                  <input
                    type="password"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    placeholder="••••••••"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required={!isEdit}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">域 (Domain)</label>
                  <input
                    type="text"
                    value={form.domain_name}
                    onChange={(e) => setForm({ ...form, domain_name: e.target.value })}
                    placeholder="Default"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">项目名称</label>
                  <input
                    type="text"
                    value={form.project_name}
                    onChange={(e) => setForm({ ...form, project_name: e.target.value })}
                    placeholder="admin"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">
                  Project ID
                  <span className="text-xs text-gray-400 ml-1">(测试连接时自动获取，也可手动填写)</span>
                </label>
                <input
                  type="text"
                  value={form.project_id}
                  onChange={(e) => setForm({ ...form, project_id: e.target.value })}
                  placeholder="测试连接后自动填入，如 4b3634c206414deb85e65c292b78951d"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none font-mono"
                />
              </div>
            </>
          )}

          {/* ZStack 字段 */}
          {form.type === 'zstack' && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">管理 URL (Endpoint) *</label>
                <input
                  type="text"
                  value={form.endpoint}
                  onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
                  placeholder="http://zstack-mn.example.com:8080"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">AccessKeyID *</label>
                  <input
                    type="text"
                    value={form.access_key_id}
                    onChange={(e) => setForm({ ...form, access_key_id: e.target.value })}
                    placeholder="AccessKeyID"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">AccessKeySecret *</label>
                  <input
                    type="password"
                    value={form.access_key_secret}
                    onChange={(e) => setForm({ ...form, access_key_secret: e.target.value })}
                    placeholder="••••••••"
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
                    required={!isEdit}
                  />
                </div>
              </div>
            </>
          )}

          {/* 描述 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1">描述</label>
            <input
              type="text"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="可选描述信息"
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
            />
          </div>

          {/* 测试结果 */}
          {testResult && (
            <div className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm ${
              testResult.ok ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'
            }`}>
              {testResult.ok ? <CheckCircle className="w-4 h-4 flex-shrink-0" /> : <XCircle className="w-4 h-4 flex-shrink-0" />}
              <span>{testResult.message}</span>
            </div>
          )}

          {/* 操作按钮 */}
          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={saving}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 text-white rounded-lg text-sm font-medium transition disabled:opacity-50"
              style={{ background: '#513CC8' }}
            >
              {saving && <Loader2 className="w-4 h-4 animate-spin" />}
              {isEdit ? '更新配置' : '添加平台'}
            </button>
            {isEdit && (
              <button
                type="button"
                onClick={handleTest}
                disabled={testing}
                className="flex items-center gap-2 px-4 py-2 bg-gray-100 hover:bg-gray-200 disabled:opacity-50 text-gray-700 rounded-lg text-sm font-medium transition"
              >
                {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
                测试连接
              </button>
            )}
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-200 text-gray-600 hover:bg-gray-50 rounded-lg text-sm transition"
            >
              取消
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function CloudPlatformPage() {
  const [platforms, setPlatforms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('list');
  const [modalOpen, setModalOpen] = useState(false);
  const [editPlatform, setEditPlatform] = useState(null);
  const [testing, setTesting] = useState({});

  // 添加平台 Tab：类型选择 + 表单
  const [addType, setAddType] = useState('easystack');

  useEffect(() => { loadPlatforms(); }, []);

  const loadPlatforms = async () => {
    try {
      const res = await getCloudPlatforms();
      if (res.code === 0) setPlatforms(res.data || []);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定删除该云平台？')) return;
    try {
      const res = await deleteCloudPlatform(id);
      if (res.code === 0) { toast.success('已删除'); loadPlatforms(); }
      else toast.error(res.message || '删除失败');
    } catch { toast.error('删除失败'); }
  };

  const handleTest = async (platform) => {
    setTesting((prev) => ({ ...prev, [platform.id]: true }));
    try {
      const res = await testCloudPlatform(platform.id);
      if (res.code === 0) {
        toast.success(res.data?.message || '连接成功');
        loadPlatforms();
      } else {
        toast.error(res.message || '连接失败');
        loadPlatforms();
      }
    } catch (err) {
      toast.error(err?.message || '连接测试失败');
      loadPlatforms();
    } finally {
      setTesting((prev) => ({ ...prev, [platform.id]: false }));
    }
  };

  const openAdd = (type) => {
    setAddType(type);
    setEditPlatform(null);
    setModalOpen(true);
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">

          {/* Tab 栏 */}
          <div className="border-b border-gray-100 px-6 pt-5">
            <div className="flex gap-0 items-end">
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
              {/* 平台数量 */}
              <div className="ml-auto pb-2.5 text-xs text-gray-400">
                共 <strong className="text-gray-600">{platforms.length}</strong> 个平台已接入
              </div>
            </div>
          </div>

          {/* Tab 内容 */}
          <div className="p-6">

            {/* === 已接入平台 Tab === */}
            {activeTab === 'list' && (
              <div>
                {loading ? (
                  <div className="flex items-center justify-center h-40">
                    <Loader2 className="w-6 h-6 animate-spin" style={{ color: '#513CC8' }} />
                  </div>
                ) : platforms.length === 0 ? (
                  <div className="text-center py-20">
                    <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                      <Cloud className="w-8 h-8 text-gray-300" />
                    </div>
                    <p className="text-base font-medium text-gray-500 mb-1">暂无云平台接入</p>
                    <p className="text-sm text-gray-400 mb-4">点击「添加平台」接入第一个云平台</p>
                    <button
                      onClick={() => setActiveTab('add')}
                      className="text-sm font-medium px-4 py-2 rounded-lg text-white transition-colors"
                      style={{ background: '#513CC8' }}
                    >
                      立即接入
                    </button>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {platforms.map((platform) => {
                      const ptMeta = PLATFORM_TYPES.find((t) => t.value === platform.type) || PLATFORM_TYPES[0];
                      return (
                        <div
                          key={platform.id}
                          className="bg-white rounded-xl border border-gray-200 hover:shadow-md transition-all p-5 flex flex-col gap-3"
                        >
                          {/* 卡片头 */}
                          <div className="flex items-start justify-between">
                            <div className="flex items-center gap-3">
                              <div className={`w-10 h-10 rounded-xl flex items-center justify-center text-xl ${ptMeta.color}`}>
                                {ptMeta.emoji}
                              </div>
                              <div>
                                <h3 className="font-semibold text-gray-800 text-sm">{platform.name}</h3>
                                <span className={`text-xs px-2 py-0.5 rounded-full border ${ptMeta.badgeBg} ${ptMeta.badgeText} ${ptMeta.badgeBorder}`}>
                                  {ptMeta.label}
                                </span>
                              </div>
                            </div>
                            {/* 状态指示灯 */}
                            <StatusDot status={platform.status} />
                          </div>

                          {/* 平台信息 */}
                          <div className="space-y-1 text-xs text-gray-500">
                            {platform.type === 'easystack' && platform.host_ip && (
                              <p className="truncate">
                                <span className="text-gray-400">IP: </span>{platform.host_ip}
                                {platform.base_domain && <span className="text-gray-400 ml-2">域: {platform.base_domain}</span>}
                              </p>
                            )}
                            {platform.type === 'easystack' && !platform.host_ip && platform.auth_url && (
                              <p className="truncate">
                                <span className="text-gray-400">URL: </span>{platform.auth_url}
                              </p>
                            )}
                            {platform.type === 'zstack' && platform.endpoint && (
                              <p className="truncate">
                                <span className="text-gray-400">Endpoint: </span>{platform.endpoint}
                              </p>
                            )}
                            {platform.username && (
                              <p><span className="text-gray-400">用户: </span>{platform.username}</p>
                            )}
                            {platform.description && (
                              <p className="text-gray-400 truncate">{platform.description}</p>
                            )}
                          </div>

                          {/* 最后测试时间 */}
                          {platform.last_tested_at && (
                            <p className="text-xs text-gray-400">
                              最后测试：{new Date(platform.last_tested_at).toLocaleString('zh-CN')}
                            </p>
                          )}

                          {/* 操作按钮 */}
                          <div className="flex items-center gap-2 pt-2 border-t border-gray-100">
                            <button
                              onClick={() => handleTest(platform)}
                              disabled={testing[platform.id]}
                              className="flex items-center gap-1.5 px-3 py-1.5 bg-gray-100 hover:bg-gray-200 disabled:opacity-50 text-gray-600 rounded-lg text-xs font-medium transition"
                            >
                              {testing[platform.id] ? (
                                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                              ) : (
                                <Zap className="w-3.5 h-3.5" />
                              )}
                              测试连接
                            </button>
                            <button
                              onClick={() => { setEditPlatform(platform); setModalOpen(true); }}
                              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition"
                              style={{ background: '#EEE9FB', color: '#513CC8' }}
                              onMouseEnter={e => { e.currentTarget.style.background = '#ddd5f7'; }}
                              onMouseLeave={e => { e.currentTarget.style.background = '#EEE9FB'; }}
                            >
                              <Edit2 className="w-3.5 h-3.5" />
                              编辑
                            </button>
                            <button
                              onClick={() => handleDelete(platform.id)}
                              className="ml-auto p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}

            {/* === 添加平台 Tab === */}
            {activeTab === 'add' && (
              <div className="space-y-6">
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 mb-3">选择云平台类型</h3>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                    {PLATFORM_TYPES.map((pt) => (
                      <button
                        key={pt.value}
                        onClick={() => openAdd(pt.value)}
                        className="flex items-center gap-4 p-5 rounded-xl border-2 border-gray-200 hover:border-[#513CC8] hover:bg-[#EEE9FB] transition-all text-left group"
                      >
                        <div className={`w-12 h-12 rounded-xl flex items-center justify-center text-2xl ${pt.color} flex-shrink-0`}>
                          {pt.emoji}
                        </div>
                        <div>
                          <p className="font-semibold text-gray-800 group-hover:text-[#513CC8] transition-colors">{pt.label}</p>
                          <p className="text-xs text-gray-400 mt-0.5">{pt.description}</p>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
                <p className="text-xs text-gray-400">
                  选择平台类型后，将弹出配置表单填写认证信息。
                </p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Modal（编辑 + 新增） */}
      <PlatformModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSaved={() => { loadPlatforms(); setActiveTab('list'); }}
        editPlatform={editPlatform}
      />
    </div>
  );
}
