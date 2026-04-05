import React, { useEffect, useState } from 'react';
import { Zap, Server, HardDrive, Network, Activity, ShieldCheck, Loader2 } from 'lucide-react';
import { getSkills } from '../services/api';

const iconMap = {
  compute: Server,
  storage: HardDrive,
  network: Network,
  monitor: Activity,
  loadbalancer: ShieldCheck,
};

export default function SkillsPage() {
  const [skills, setSkills] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const res = await getSkills();
        if (res.code === 0) setSkills(res.data || []);
      } catch (err) { console.error(err); }
      finally { setLoading(false); }
    })();
  }, []);

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-5xl">
        {/* 页面头部卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">技能管理</h1>
              <p className="text-sm text-gray-400 mt-0.5">管理智能体可调用的 EasyStack API 技能</p>
            </div>
          </div>
        </div>

        {/* 技能列表卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin text-[#513CC8]" />
            </div>
          ) : skills.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Zap className="w-10 h-10 mx-auto mb-2 opacity-30" />
              <p className="text-sm">暂无技能数据</p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {skills.map((skill) => {
                let config = {};
                try { config = JSON.parse(skill.config || '{}'); } catch {}
                const Icon = iconMap[config.service] || Zap;
                return (
                  <div key={skill.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-[#EEE9FB] rounded-lg flex items-center justify-center">
                        <Icon className="w-5 h-5 text-[#513CC8]" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-800">{skill.name}</h3>
                          <span className="text-xs px-2 py-0.5 bg-purple-50 text-purple-600 rounded">{skill.type}</span>
                        </div>
                        <p className="text-sm text-gray-500 mt-1">{skill.description}</p>
                        {config.capabilities && (
                          <div className="flex flex-wrap gap-1.5 mt-3">
                            {config.capabilities.map((cap, i) => (
                              <span key={i} className="text-xs px-2 py-0.5 bg-gray-50 text-gray-500 rounded-full border border-gray-100">
                                {cap}
                              </span>
                            ))}
                          </div>
                        )}
                        {config.api_version && (
                          <p className="text-xs text-gray-400 mt-2">API版本: {config.api_version}</p>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
