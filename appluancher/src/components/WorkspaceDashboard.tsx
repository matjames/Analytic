import React, { useState, useEffect } from 'react';
import { User } from '@typings/index';
import { KnowledgePanel } from '@components/KnowledgePanel';
import { EnterpriseIntelligencePanel } from '@components/EnterpriseIntelligencePanel';

interface WorkspaceData {
  pms: any;
  rms: any;
  notifications: any[];
  chats: any[];
  tickets: any[];
  meetings: any[];
  surveys: any[];
  pendingApprovals: any[];
  assignments: any[];
  activity: any[];
}

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const WorkspaceDashboard: React.FC<{ user: User }> = ({ user }) => {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<WorkspaceData>({
    pms: null,
    rms: null,
    notifications: [],
    chats: [],
    tickets: [],
    meetings: [],
    surveys: [],
    pendingApprovals: [],
    assignments: [],
    activity: [],
  });

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    // Use the Enterprise Core workspace API for personalized data
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {};
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/workspace?user_id=${encodeURIComponent(user.id)}`, { headers })
      .then(r => r.ok ? r.json() : null)
      .catch(() => null)
      .then((ws: any) => {
        if (cancelled) return;
        if (!ws) {
          setLoading(false);
          return;
        }

        const projects = ws.my_projects || [];
        const research = ws.my_research || [];
        const meetings = ws.my_meetings || [];

        // Derive pending approvals from workspace data
        const pendingApprovals = ws.my_approvals || [];

        // Derive assigned tasks
        const assignments = (ws.my_tasks || []).map((t: any) => ({
          id: t.id,
          title: t.title,
          project: t.project_id || '',
          status: t.status,
          dueDate: t.due || '',
        }));

        setData({
          pms: { totalProjects: projects.length, projects },
          rms: { totalResearch: research.length, research },
          notifications: ws.my_notifications || [],
          chats: ws.my_messages || [],
          tickets: ws.my_tickets || [],
          meetings,
          surveys: ws.my_surveys || [],
          pendingApprovals,
          assignments,
          activity: ws.recent_activity || [],
        });
        setLoading(false);
      });
    return () => { cancelled = true; };
  }, [user.id]);

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[...Array(8)].map((_, i) => (
          <div key={i} className="animate-pulse h-24 bg-gray-100 rounded-xl" />
        ))}
      </div>
    );
  }

  const totalProjects = data.pms?.totalProjects ?? (data.pms?.projects || []).length ?? 0;
  const totalResearch = data.rms?.totalResearch ?? 0;

  const quickActions = [
    { label: 'New Project', href: '/pms/projects/new', icon: '📁' },
    { label: 'Create Survey', href: '/pms/surveys/new', icon: '📋' },
    { label: 'Log Issue', href: '/pms/issues/new', icon: '⚠️' },
    { label: 'Request Approval', href: '/pms/approvals/new', icon: '✅' },
    { label: 'HelpDesk', href: '/helpdesk', icon: '🎧' },
    { label: 'StatChat', href: '/chats', icon: '💬' },
  ];

  const frequentApps = [
    { name: 'PMS', url: 'http://localhost:3010', desc: 'Projects', icon: '🏗️' },
    { name: 'RMS', url: 'http://localhost:3011', desc: 'Research', icon: '🔬' },
    { name: 'StatChat', url: 'http://localhost:3009', desc: 'Messages', icon: '💬' },
    { name: 'HelpDesk', url: 'http://localhost:3005', desc: 'Support', icon: '🎫' },
    { name: 'Analytics', url: 'http://localhost:5000', desc: 'Dashboards', icon: '📊' },
    { name: 'Registry', url: 'http://localhost:3007', desc: 'Facilities', icon: '🏥' },
  ];

  return (
    <div className="space-y-8">
      {/* Personal KPIs */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="glass-panel stat-card">
          <span className="stat-val">{totalProjects}</span>
          <span className="stat-label">Assigned Projects</span>
        </div>
        <div className="glass-panel stat-card">
          <span className="stat-val">{totalResearch}</span>
          <span className="stat-label">Research Studies</span>
        </div>
        <div className="glass-panel stat-card">
          <span className="stat-val">{data.tickets.length}</span>
          <span className="stat-label">Open HelpDesk Tickets</span>
        </div>
        <div className="glass-panel stat-card">
          <span className="stat-val">{data.pendingApprovals.length}</span>
          <span className="stat-label">Pending Approvals</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column */}
        <div className="lg:col-span-2 space-y-8">
	      <EnterpriseIntelligencePanel user={user} />
          {/* Pending Approvals */}
          {data.pendingApprovals.length > 0 && (
            <div className="glass-panel">
              <h3 className="text-lg font-semibold mb-3">Pending Approvals</h3>
              <div className="space-y-2">
                {data.pendingApprovals.slice(0, 5).map(item => (
                  <div key={`${item.type}-${item.id}`} className="flex items-center justify-between p-3 rounded-lg border border-yellow-200 bg-yellow-50">
                    <div className="flex items-center gap-3">
                      <span className="text-xl">{item.type === 'project' ? '🏗️' : '🔬'}</span>
                      <div>
                        <div className="text-sm font-medium">{item.title}</div>
                        <div className="text-xs text-gray-500">{item.meta}</div>
                      </div>
                    </div>
                    <button className="text-xs px-3 py-1.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                      Review
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Quick Actions & Frequent Apps */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="glass-panel">
              <h3 className="text-lg font-semibold mb-3">Quick Actions</h3>
              <div className="grid grid-cols-2 gap-3">
                {quickActions.map(a => (
                  <a key={a.label} href={a.href} className="flex items-center gap-2 p-3 rounded border hover:bg-gray-50">
                    <span className="text-xl">{a.icon}</span>
                    <span className="text-sm font-medium">{a.label}</span>
                  </a>
                ))}
              </div>
            </div>
            <div className="glass-panel">
              <h3 className="text-lg font-semibold mb-3">Frequently Used</h3>
              <div className="space-y-2">
                {frequentApps.map(app => (
                  <a key={app.name} href={app.url} className="flex items-center justify-between p-3 rounded border hover:bg-gray-50">
                    <div className="flex items-center gap-2">
                      <span className="text-xl">{app.icon}</span>
                      <div>
                        <div className="text-sm font-medium">{app.name}</div>
                        <div className="text-xs text-gray-500">{app.desc}</div>
                      </div>
                    </div>
                    <span className="text-gray-400">→</span>
                  </a>
                ))}
              </div>
            </div>
          </div>

          {/* Activity Timeline */}
          <div className="glass-panel">
            <h3 className="text-lg font-semibold mb-3">Enterprise Activity</h3>
            {data.activity.length === 0 ? (
              <p className="text-sm text-gray-500">No recent activity</p>
            ) : (
              <div className="space-y-3">
                {data.activity.slice(0, 8).map((item: any, idx: number) => (
                  <div key={idx} className="flex items-start gap-3">
                    <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 text-sm font-semibold flex-shrink-0">
                      {item.source === 'pms' ? 'P' : 'R'}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-gray-900">
                        {item.action || item.title || 'Activity'}
                      </div>
                      <div className="flex items-center gap-2 text-xs text-gray-500">
                        <span className="capitalize">{item.source}</span>
                        {item.details && <span>•</span>}
                        <span className="truncate">{item.details}</span>
                      </div>
                    </div>
                    <span className="text-xs text-gray-400 flex-shrink-0">
                      {formatRelativeTime(item.timestamp || item.createdTime)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Recent Data Lists */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="glass-panel">
              <h3 className="text-lg font-semibold mb-3">Recent StatChat</h3>
              <ul className="space-y-2">
                {data.chats.length === 0 && <li className="text-sm text-gray-500">No recent messages</li>}
                {data.chats.slice(0, 5).map((c: any, i) => (
                  <li key={c.id || i} className="text-sm border-b pb-2">
                    <div className="font-medium">{c.sender || 'System'}</div>
                    <div className="text-gray-600 truncate">{c.message}</div>
                  </li>
                ))}
              </ul>
            </div>
            <div className="glass-panel">
              <h3 className="text-lg font-semibold mb-3">HelpDesk Tickets</h3>
              <ul className="space-y-2">
                {data.tickets.length === 0 && <li className="text-sm text-gray-500">No open tickets</li>}
                {data.tickets.slice(0, 5).map((t: any, i) => (
                  <li key={t.id || i} className="text-sm border-b pb-2">
                    <div className="font-medium">{t.title}</div>
                    <div className="text-gray-600">{t.status} • {t.priority}</div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>

        {/* Right Column */}
        <div className="space-y-6">
          <KnowledgePanel user={user} />
          <div className="glass-panel">
            <h3 className="text-lg font-semibold mb-3">Notifications</h3>
            <ul className="space-y-2">
              {data.notifications.length === 0 && <li className="text-sm text-gray-500">No notifications</li>}
              {data.notifications.slice(0, 8).map((n: any, i) => (
                <li key={n.id || i} className="text-sm border-b pb-2">
                  <div className="font-medium">{n.title}</div>
                  <div className="text-gray-600 truncate">{n.body}</div>
                </li>
              ))}
            </ul>
          </div>
          <div className="glass-panel">
            <h3 className="text-lg font-semibold mb-3">Upcoming Meetings</h3>
            <ul className="space-y-2">
              {data.meetings.length === 0 && <li className="text-sm text-gray-500">No upcoming meetings</li>}
              {data.meetings.slice(0, 6).map((m: any, i) => (
                <li key={m.id || i} className="text-sm border-b pb-2">
                  <div className="font-medium">{m.title}</div>
                  <div className="text-gray-600">{m.date_time || m.start_time}</div>
                </li>
              ))}
            </ul>
          </div>
          <div className="glass-panel">
            <h3 className="text-lg font-semibold mb-3">Survey Activity</h3>
            <ul className="space-y-2">
              {data.surveys.length === 0 && <li className="text-sm text-gray-500">No recent surveys</li>}
              {data.surveys.slice(0, 5).map((s: any, i) => (
                <li key={s.id || i} className="text-sm border-b pb-2">
                  <div className="font-medium">{s.name}</div>
                  <div className="text-gray-600">{s.status}</div>
                </li>
              ))}
            </ul>
          </div>
          <div className="glass-panel">
            <h3 className="text-lg font-semibold mb-3">My Assignments</h3>
            <ul className="space-y-2">
              {data.assignments.length === 0 && <li className="text-sm text-gray-500">No assigned tasks</li>}
              {data.assignments.slice(0, 5).map((a: any, i) => (
                <li key={i} className="text-sm border-b pb-2">
                  <div className="font-medium">{a.title}</div>
                  <div className="text-gray-600">{a.project} • {a.status}</div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
};

function formatRelativeTime(timestamp: string): string {
  if (!timestamp) return '';
  const date = new Date(timestamp);
  if (isNaN(date.getTime())) return '';
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return date.toLocaleDateString();
}

export default WorkspaceDashboard;
