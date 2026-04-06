import React, { useEffect, useState } from 'react';
import { Zap, Server, HardDrive, Network, Activity, ShieldCheck, Loader2, Bot, Link2, Wrench } from 'lucide-react';
import { getSkills, getAgents } from '../services/api';

const iconMap = {
  compute: Server,
  storage: HardDrive,
  network: Network,
  monitor: Activity,
  loadbalancer: ShieldCheck,
};

export default function SkillsPage() {
  const [skills, setSkills] = useState([]);
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const [skillsRes, agentsRes] = await Promise.all([getSkills(), getAgents()]);
        if (skillsRes.code === 0) setSkills(skillsRes.data || []);
        if (agentsRes.code === 0) setAgents(agentsRes.data || []);
      } catch (err) { console.error(err); }
      finally { setLoading(false); }
    })();
  }, []);

  // For a given skill ID, find all agents that have it associated
  const getAssociatedAgents = (skillId) => {
    return agents.filter((agent) => {
      if (agent.agent_skills && agent.agent_skills.length > 0) {
        return agent.agent_skills.some((as) => as.skill_id === skillId || as.skill?.id === skillId);
      }
      return false;
    });
  };

  // Parse tool_defs to get function names
  const getToolNames = (toolDefs) => {
    if (!toolDefs) return [];
    try {
      const defs = JSON.parse(toolDefs);
      return defs.map((d) => d.function?.name).filter(Boolean);
    } catch {
      return [];
    }
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-5xl">
        {/* Header */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">技能中心</h1>
              <p className="text-sm text-gray-400 mt-0.5">管理智能体可调用的云平台 API 技能，通过 Function Calling 赋能智能体</p>
            </div>
          </div>
        </div>

        {/* Skills List */}
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
                const associatedAgents = getAssociatedAgents(skill.id);
                const toolNames = getToolNames(skill.tool_defs);
                return (
                  <div key={skill.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-[#EEE9FB] rounded-lg flex items-center justify-center flex-shrink-0">
                        <Icon className="w-5 h-5 text-[#513CC8]" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-800">{skill.name}</h3>
                          <span className="text-xs px-2 py-0.5 bg-purple-50 text-purple-600 rounded">{skill.type}</span>
                          {skill.is_active ? (
                            <span className="w-2 h-2 bg-green-400 rounded-full" title="已启用" />
                          ) : (
                            <span className="w-2 h-2 bg-gray-300 rounded-full" title="已停用" />
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">{skill.description}</p>

                        {/* Tool Functions */}
                        {toolNames.length > 0 && (
                          <div className="mt-2">
                            <div className="flex items-center gap-1 text-xs text-gray-400 mb-1">
                              <Wrench className="w-3 h-3" />
                              <span>Function Calling ({toolNames.length} 个工具)</span>
                            </div>
                            <div className="flex flex-wrap gap-1">
                              {toolNames.map((name, i) => (
                                <span key={i} className="text-xs px-1.5 py-0.5 bg-gray-50 text-gray-500 rounded border border-gray-100 font-mono">
                                  {name}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}

                        {/* Associated Agents */}
                        {associatedAgents.length > 0 && (
                          <div className="mt-2">
                            <div className="flex items-center gap-1 text-xs text-gray-400 mb-1">
                              <Link2 className="w-3 h-3" />
                              <span>已关联 {associatedAgents.length} 个智能体</span>
                            </div>
                            <div className="flex flex-wrap gap-1.5">
                              {associatedAgents.map((agent) => (
                                <span key={agent.id}
                                  className="inline-flex items-center gap-1 text-xs px-2 py-0.5 bg-blue-50 text-blue-600 rounded-full border border-blue-100">
                                  <Bot className="w-3 h-3" />{agent.name}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}
                        {associatedAgents.length === 0 && (
                          <p className="text-xs text-gray-300 mt-2">未被任何智能体关联</p>
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
