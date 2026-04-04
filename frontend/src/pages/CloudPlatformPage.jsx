import React, { useEffect, useState } from 'react';
import {
  Server,
  Plus,
  Edit2,
  Trash2,
  Zap,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  X,
} from 'lucide-react';
import {
  getCloudPlatforms,
  createCloudPlatform,
  updateCloudPlatform,
  deleteCloudPlatform,
  testCloudPlatform,
} from '../services/api';
import toast from 'react-hot-toast';

const EMPTY_FORM = {
  name: '',
  type: 'easystack',
  description: '',
  // EasyStack
  auth_url: '',
  username: '',
  password: '',
  domain_name: '',
  project_name: '',
  project_id: '',
  // ZStack
  endpoint: '',
  access_key_id: '',
  access_key_secret: '',
};

function StatusBadge({ status }) {
  if (status === 'connected') {
    return (
      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-50 text-green-600 border border-green-200">
        <CheckCircle className="w-3 h-3" />
        已连接
      </span>
    );
  }
  if (status === 'failed') {
    return (
      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-red-50 text-red-600 border border-red-200">
        <XCircle className="w-3 h-3" />
        连接失败
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-500 border border-gray-200">
      <Clock className="w-3 h-3" />
      未测试
    </span>
  );
}

function TypeBadge({ type }) {
  if (type === 'easystack') {
    return (
      <span className="inline-flex items-center text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 border border-blue-200 font-medium">
        EasyStack
      </span>
    );
  }
  return (
    <span className="inline-flex items-center text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 border border-green-200 font-medium">
      ZStack
    </span>
  );
}

export default function CloudPlatformPage() {
  const [platforms, setPlatforms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editId, setEditId] = useState(null);
  const [form, setForm] = useState(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [testingId, setTestingId] = useState(null);

  useEffect(() => {
    loadPlatforms();
  }, []);

  const loadPlatforms = async () => {
    try {
      const res = await getCloudPlatforms();
      if (res.code === 0) setPlatforms(res.data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const openAdd = () => {
    setEditId(null);
    setForm(EMPTY_FORM);
    setShowModal(true);
  };

  const openEdit = (p) => {
    setEditId(p.id);
    setForm({
      name: p.name || '',
      type: p.type || 'easystack',
      description: p.description || '',
      auth_url: p.auth_url || '',
      username: p.username || '',
      password: '',
      domain_name: p.domain_name || '',
      project_name: p.project_name || '',
      project_id: p.project_id || '',
      endpoint: p.endpoint || '',
      access_key_id: p.access_key_id || '',
      access_key_secret: '',
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (editId) {
        const res = await updateCloudPlatform(editId, form);
        if (res.code === 0) {
          toast.success('平台已更新');
          setShowModal(false);
          loadPlatforms();
        } else {
          toast.error(res.message || '更新失败');
        }
      } else {
        const res = await createCloudPlatform(form);
        if (res.code === 0) {
          toast.success('平台已添加');
          setShowModal(false);
          loadPlatforms();
        } else {
          toast.error(res.message || '添加失败');
        }
      }
    } catch (err) {
      toast.error(err?.message || '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定删除该云平台?')) return;
    try {
      await deleteCloudPlatform(id);
      toast.success('已删除');
      loadPlatforms();
    } catch {
      toast.error('删除失败');
    }
  };

  const handleTest = async (id) => {
    setTestingId(id);
    try {
      const res = await testCloudPlatform(id);
      if (res.code === 0) {
        toast.success(res.data?.message || '连接成功');
      } else {
        toast.error(res.message || '连接失败');
      }
      loadPlatforms();
    } catch (err) {
      toast.error(err?.message || '连接失败');
      loadPlatforms();
    } finally {
      setTestingId(null);
    }
  };

  const setField = (k, v) => setForm((prev) => ({ ...prev, [k]: v }));

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      {/* Page Header */}
      <div className="bg-gradient-to-r from-primary to-primary-700 px-8 py-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-white/20 rounded-xl flex items-center justify-center">
              <Server className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-white">接入云平台</h1>
              <p className="text-primary-100 text-sm mt-0.5">管理 EasyStack 和 ZStack 云平台连接</p>
            </div>
          </div>
          <button
            onClick={openAdd}
            className="flex items-center gap-2 px-4 py-2 bg-white/20 hover:bg-white/30 text-white rounded-lg text-sm transition"
          >
            <Plus className="w-4 h-4" />
            添加平台
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {platforms.length === 0 ? (
          <div className="text-center py-20 text-gray-400">
            <Server className="w-12 h-12 mx-auto mb-3 opacity-30" />
            <p className="text-sm">暂无接入的云平台</p>
            <button
              onClick={openAdd}
              className="mt-4 px-4 py-2 bg-primary hover:bg-primary-600 text-white rounded-lg text-sm transition"
            >
              添加第一个平台
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {platforms.map((p) => (
              <div key={p.id} className="bg-white rounded-xl border border-gray-200 hover:border-primary-300 hover:shadow-sm transition-all">
                {/* Card Header */}
                <div className="px-5 py-4 border-b border-gray-100">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap mb-1">
                        <span className="font-semibold text-gray-800 truncate">{p.name}</span>
                        <TypeBadge type={p.type} />
                      </div>
                      {p.description && (
                        <p className="text-xs text-gray-400 truncate">{p.description}</p>
                      )}
                    </div>
                    <StatusBadge status={p.status} />
                  </div>
                </div>

                {/* Card Details */}
                <div className="px-5 py-3 space-y-1.5 text-xs text-gray-500">
                  {p.type === 'easystack' && (
                    <>
                      {p.auth_url && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">Keystone:</span>
                          <span className="truncate text-gray-600">{p.auth_url}</span>
                        </div>
                      )}
                      {p.username && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">用户名:</span>
                          <span className="text-gray-600">{p.username}</span>
                        </div>
                      )}
                      {p.domain_name && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">域:</span>
                          <span className="text-gray-600">{p.domain_name}</span>
                        </div>
                      )}
                      {p.project_name && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">项目:</span>
                          <span className="text-gray-600">{p.project_name}</span>
                        </div>
                      )}
                    </>
                  )}
                  {p.type === 'zstack' && (
                    <>
                      {p.endpoint && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">Endpoint:</span>
                          <span className="truncate text-gray-600">{p.endpoint}</span>
                        </div>
                      )}
                      {p.access_key_id && (
                        <div className="flex gap-2">
                          <span className="text-gray-400 w-16 flex-shrink-0">AccessKey:</span>
                          <span className="text-gray-600">{p.access_key_id}</span>
                        </div>
                      )}
                    </>
                  )}
                  {p.last_tested_at && (
                    <div className="flex gap-2 pt-1">
                      <span className="text-gray-400 w-16 flex-shrink-0">上次测试:</span>
                      <span className="text-gray-500">
                        {new Date(p.last_tested_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                </div>

                {/* Card Actions */}
                <div className="px-5 py-3 border-t border-gray-100 flex items-center gap-2">
                  <button
                    onClick={() => handleTest(p.id)}
                    disabled={testingId === p.id}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-50 hover:bg-primary-100 text-primary rounded-lg text-xs font-medium transition disabled:opacity-50"
                  >
                    {testingId === p.id ? (
                      <Loader2 className="w-3 h-3 animate-spin" />
                    ) : (
                      <Zap className="w-3 h-3" />
                    )}
                    测试连接
                  </button>
                  <button
                    onClick={() => openEdit(p)}
                    className="flex items-center gap-1 px-3 py-1.5 text-gray-500 hover:text-primary hover:bg-gray-50 rounded-lg text-xs transition"
                  >
                    <Edit2 className="w-3 h-3" />
                    编辑
                  </button>
                  <button
                    onClick={() => handleDelete(p.id)}
                    className="flex items-center gap-1 px-3 py-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg text-xs transition ml-auto"
                  >
                    <Trash2 className="w-3 h-3" />
                    删除
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-800">
                {editId ? '编辑云平台' : '添加云平台'}
              </h3>
              <button
                onClick={() => setShowModal(false)}
                className="p-1 text-gray-400 hover:text-gray-600 rounded"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
              {/* Platform Type Selector */}
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-2">平台类型</label>
                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={() => setField('type', 'easystack')}
                    className={`flex-1 py-3 rounded-xl border-2 font-medium text-sm transition ${
                      form.type === 'easystack'
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200 text-gray-500 hover:border-gray-300'
                    }`}
                  >
                    ☁️ EasyStack
                  </button>
                  <button
                    type="button"
                    onClick={() => setField('type', 'zstack')}
                    className={`flex-1 py-3 rounded-xl border-2 font-medium text-sm transition ${
                      form.type === 'zstack'
                        ? 'border-green-500 bg-green-50 text-green-700'
                        : 'border-gray-200 text-gray-500 hover:border-gray-300'
                    }`}
                  >
                    🖥️ ZStack
                  </button>
                </div>
              </div>

              {/* Common Fields */}
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">平台名称 *</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setField('name', e.target.value)}
                  required
                  placeholder="例如：生产环境 EasyStack"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">描述</label>
                <input
                  type="text"
                  value={form.description}
                  onChange={(e) => setField('description', e.target.value)}
                  placeholder="可选描述信息"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                />
              </div>

              {/* EasyStack Fields */}
              {form.type === 'easystack' && (
                <>
                  <div className="border-t border-gray-100 pt-4">
                    <p className="text-xs font-semibold text-blue-600 uppercase tracking-wider mb-3">
                      EasyStack Keystone 认证
                    </p>
                    <div className="space-y-3">
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">
                          Keystone 地址 (AuthURL) *
                        </label>
                        <input
                          type="url"
                          value={form.auth_url}
                          onChange={(e) => setField('auth_url', e.target.value)}
                          required
                          placeholder="http://keystone.example.com:5000"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                        />
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1">用户名 *</label>
                          <input
                            type="text"
                            value={form.username}
                            onChange={(e) => setField('username', e.target.value)}
                            required
                            placeholder="admin"
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1">
                            密码{editId && ' (留空不修改)'}
                          </label>
                          <input
                            type="password"
                            value={form.password}
                            onChange={(e) => setField('password', e.target.value)}
                            required={!editId}
                            placeholder="密码"
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                          />
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1">域名 (Domain) *</label>
                          <input
                            type="text"
                            value={form.domain_name}
                            onChange={(e) => setField('domain_name', e.target.value)}
                            required
                            placeholder="Default"
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-600 mb-1">项目名称 *</label>
                          <input
                            type="text"
                            value={form.project_name}
                            onChange={(e) => setField('project_name', e.target.value)}
                            required
                            placeholder="admin"
                            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </>
              )}

              {/* ZStack Fields */}
              {form.type === 'zstack' && (
                <>
                  <div className="border-t border-gray-100 pt-4">
                    <p className="text-xs font-semibold text-green-600 uppercase tracking-wider mb-3">
                      ZStack AccessKey 认证
                    </p>
                    <div className="space-y-3">
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">
                          管理端地址 (Endpoint) *
                        </label>
                        <input
                          type="url"
                          value={form.endpoint}
                          onChange={(e) => setField('endpoint', e.target.value)}
                          required
                          placeholder="http://zstack.example.com:8080"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">AccessKeyID</label>
                        <input
                          type="text"
                          value={form.access_key_id}
                          onChange={(e) => setField('access_key_id', e.target.value)}
                          placeholder="Access Key ID"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-600 mb-1">
                          AccessKeySecret / 管理员密码{editId && ' (留空不修改)'}
                        </label>
                        <input
                          type="password"
                          value={form.access_key_secret}
                          onChange={(e) => setField('access_key_secret', e.target.value)}
                          required={!editId}
                          placeholder="Secret Key 或管理员密码"
                          className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary outline-none"
                        />
                      </div>
                    </div>
                  </div>
                </>
              )}

              {/* Actions */}
              <div className="flex items-center gap-3 pt-2 border-t border-gray-100">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-gray-500 bg-gray-100 hover:bg-gray-200 rounded-lg text-sm transition"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex items-center gap-2 px-5 py-2 bg-primary hover:bg-primary-600 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition ml-auto"
                >
                  {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
                  {editId ? '更新' : '保存'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
