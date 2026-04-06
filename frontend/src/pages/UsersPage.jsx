import React, { useEffect, useState } from 'react';
import {
  Users,
  Plus,
  Edit2,
  Trash2,
  Shield,
  User,
  Search,
  CheckCircle,
  XCircle,
  X,
  Loader2,
} from 'lucide-react';
import { getUsers, createUser, updateUser, deleteUser } from '../services/api';
import toast from 'react-hot-toast';

// Password strength checker
function checkPasswordStrength(password) {
  if (!password) return { level: 0, label: '', color: '' };
  let score = 0;
  const checks = {
    length:   password.length >= 9,
    upper:    /[A-Z]/.test(password),
    lower:    /[a-z]/.test(password),
    digit:    /[0-9]/.test(password),
    special:  /[^A-Za-z0-9]/.test(password),
  };
  score = Object.values(checks).filter(Boolean).length;
  if (score <= 2) return { level: 1, label: '弱', color: 'bg-red-500', checks };
  if (score <= 3) return { level: 2, label: '中', color: 'bg-yellow-500', checks };
  if (score === 4) return { level: 3, label: '较强', color: 'bg-blue-500', checks };
  return { level: 4, label: '强', color: 'bg-green-500', checks };
}

function PasswordStrengthBar({ password }) {
  const strength = checkPasswordStrength(password);
  if (!password) return null;

  const requirements = [
    { key: 'length',  label: '至少9位字符' },
    { key: 'upper',   label: '包含大写字母' },
    { key: 'lower',   label: '包含小写字母' },
    { key: 'digit',   label: '包含数字' },
    { key: 'special', label: '包含特殊字符' },
  ];

  return (
    <div className="mt-2 space-y-2">
      <div className="flex items-center gap-2">
        <div className="flex gap-1 flex-1">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className={`h-1.5 flex-1 rounded-full transition-all ${
                i <= strength.level ? strength.color : 'bg-gray-200'
              }`}
            />
          ))}
        </div>
        <span className={`text-xs font-medium ${
          strength.level <= 1 ? 'text-red-500' :
          strength.level === 2 ? 'text-yellow-600' :
          strength.level === 3 ? 'text-blue-600' : 'text-green-600'
        }`}>
          {strength.label}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-0.5">
        {requirements.map((req) => (
          <div key={req.key} className="flex items-center gap-1 text-xs">
            {strength.checks?.[req.key] ? (
              <CheckCircle className="w-3 h-3 text-green-500 flex-shrink-0" />
            ) : (
              <XCircle className="w-3 h-3 text-gray-300 flex-shrink-0" />
            )}
            <span className={strength.checks?.[req.key] ? 'text-green-600' : 'text-gray-400'}>
              {req.label}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// 新增/编辑用户 Modal
function UserModal({ open, editUser, onClose, onSaved }) {
  const isEdit = !!editUser;
  const [form, setForm] = useState({ username: '', password: '', email: '', role: 'user' });
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      if (editUser) {
        setForm({ username: editUser.username, password: '', email: editUser.email || '', role: editUser.role });
      } else {
        setForm({ username: '', password: '', email: '', role: 'user' });
      }
    }
  }, [open, editUser]);

  if (!open) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (isEdit) {
        const res = await updateUser(editUser.id, form);
        if (res.code === 0) {
          toast.success('用户已更新');
          onSaved();
          onClose();
        } else {
          toast.error(res.message || '更新失败');
        }
      } else {
        const res = await createUser(form);
        if (res.code === 0) {
          toast.success('用户已创建');
          onSaved();
          onClose();
        } else {
          toast.error(res.message || '创建失败');
        }
      }
    } catch (err) {
      toast.error(err?.message || '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-lg font-semibold text-gray-800">{isEdit ? '编辑用户' : '新增用户'}</h2>
          <button onClick={onClose} className="p-1.5 text-gray-400 hover:text-gray-600 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {/* 用户名 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1.5">用户名</label>
            <input
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
              placeholder="请输入用户名"
              required
              disabled={isEdit}
            />
          </div>

          {/* 密码 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1.5">
              密码
              {isEdit && <span className="ml-1 text-xs text-gray-400 font-normal">（留空则不修改）</span>}
            </label>
            <input
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
              placeholder="≥9位，含大小写字母、数字和特殊字符"
              required={!isEdit}
            />
            {form.password && <PasswordStrengthBar password={form.password} />}
            {!form.password && !isEdit && (
              <p className="text-xs text-gray-400 mt-1">密码要求：至少9位字符，必须包含大写字母、小写字母、数字和特殊字符</p>
            )}
          </div>

          {/* 邮箱 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1.5">邮箱</label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none"
              placeholder="user@example.com（可选）"
            />
          </div>

          {/* 角色 */}
          <div>
            <label className="block text-sm font-medium text-gray-600 mb-1.5">角色</label>
            <select
              value={form.role}
              onChange={(e) => setForm({ ...form, role: e.target.value })}
              className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
            >
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
            </select>
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="submit"
              disabled={submitting}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 text-white rounded-lg text-sm font-medium transition disabled:opacity-50"
              style={{ background: '#513CC8' }}
            >
              {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
              {isEdit ? '更新用户' : '创建用户'}
            </button>
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

export default function UsersPage() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editUser, setEditUser] = useState(null);

  useEffect(() => { loadUsers(); }, []);

  const loadUsers = async () => {
    try {
      const res = await getUsers();
      if (res.code === 0) setUsers(res.data || []);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定删除该用户?')) return;
    try {
      await deleteUser(id);
      toast.success('已删除');
      loadUsers();
    } catch {
      toast.error('删除失败');
    }
  };

  const openCreate = () => {
    setEditUser(null);
    setModalOpen(true);
  };

  const openEdit = (user) => {
    setEditUser(user);
    setModalOpen(true);
  };

  // 过滤搜索
  const filteredUsers = users.filter((u) =>
    u.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (u.email || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">

          {/* 卡片头：搜索 + 新增按钮 */}
          <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <h2 className="text-base font-semibold text-gray-800">用户列表</h2>
              <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full">
                共 {users.length} 人
              </span>
            </div>
            <div className="flex items-center gap-3">
              {/* 搜索框 */}
              <div className="relative">
                <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="搜索用户名或邮箱..."
                  className="pl-9 pr-4 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none w-52"
                />
              </div>
              {/* 新增用户按钮 */}
              <button
                onClick={openCreate}
                className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white transition-colors"
                style={{ background: '#513CC8' }}
                onMouseEnter={e => e.currentTarget.style.background = '#4231a8'}
                onMouseLeave={e => e.currentTarget.style.background = '#513CC8'}
              >
                <Plus className="w-4 h-4" />
                新增用户
              </button>
            </div>
          </div>

          {/* 用户表格 */}
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin" style={{ color: '#513CC8' }} />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px]">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-100">
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">用户</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">邮箱</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">角色</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">创建时间</th>
                    <th className="text-right px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {filteredUsers.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="text-center py-12 text-gray-400">
                        <Users className="w-10 h-10 mx-auto mb-2 opacity-30" />
                        <p className="text-sm">{searchQuery ? '未找到匹配的用户' : '暂无用户'}</p>
                      </td>
                    </tr>
                  ) : (
                    filteredUsers.map((u) => (
                      <tr key={u.id} className="hover:bg-gray-50 transition-colors">
                        {/* 用户头像 + 名称 */}
                        <td className="px-6 py-3.5">
                          <div className="flex items-center gap-3">
                            <div
                              className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0"
                              style={{ background: u.role === 'admin' ? '#513CC8' : '#6b7280' }}
                            >
                              {(u.username || 'U').slice(0, 1).toUpperCase()}
                            </div>
                            <span className="text-sm font-medium text-gray-700">{u.username}</span>
                          </div>
                        </td>
                        {/* 邮箱 */}
                        <td className="px-6 py-3.5 text-sm text-gray-500">{u.email || <span className="text-gray-300">-</span>}</td>
                        {/* 角色徽章 */}
                        <td className="px-6 py-3.5">
                          {u.role === 'admin' ? (
                            <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-purple-100 text-purple-700 font-medium">
                              <Shield className="w-3 h-3" />
                              管理员
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                              <User className="w-3 h-3" />
                              用户
                            </span>
                          )}
                        </td>
                        {/* 创建时间 */}
                        <td className="px-6 py-3.5 text-sm text-gray-400">
                          {u.created_at
                            ? new Date(u.created_at).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
                            : <span className="text-gray-300">-</span>}
                        </td>
                        {/* 操作 */}
                        <td className="px-6 py-3.5">
                          <div className="flex items-center justify-end gap-1">
                            <button
                              onClick={() => openEdit(u)}
                              className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors"
                              title="编辑用户"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => handleDelete(u.id)}
                              className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                              title="删除用户"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* 新增/编辑 Modal */}
      <UserModal
        open={modalOpen}
        editUser={editUser}
        onClose={() => setModalOpen(false)}
        onSaved={loadUsers}
      />
    </div>
  );
}
