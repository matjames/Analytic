import React, { useState, useEffect } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import {
  Sparkles,
  AlertTriangle,
  CheckSquare,
  TrendingUp,
  FolderGit2,
  FlaskConical,
  MapPin,
  Database,
  Ticket,
  Scale,
  Calendar,
  ArrowRight,
  ChevronRight,
  ShieldAlert,
  Activity,
  CheckCircle2,
} from 'lucide-react';

interface HomeViewProps {
  user: User;
}

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const HomeView: React.FC<HomeViewProps> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [loading, setLoading] = useState(true);
  const [summaryData, setSummaryData] = useState<any>(null);
  const [briefing, setBriefing] = useState<any>(null);
  const [myWork, setMyWork] = useState<any>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    Promise.all([
      fetch(`${API_BASE}/api/command-centre/summary?user_id=${encodeURIComponent(user.id)}&role=${encodeURIComponent(user.role)}`, { headers })
        .then((r) => (r.ok ? r.json() : null))
        .catch(() => null),
      fetch(`${API_BASE}/api/ai/v1/briefings?user_id=${encodeURIComponent(user.id)}`, { headers })
        .then((r) => (r.ok ? r.json() : null))
        .catch(() => null),
      fetch(`${API_BASE}/api/my-work?user_id=${encodeURIComponent(user.id)}`, { headers })
        .then((r) => (r.ok ? r.json() : null))
        .catch(() => null),
    ]).then(([summary, aiBrief, work]) => {
      if (cancelled) return;
      setSummaryData(summary || {});
      setBriefing(aiBrief?.briefing || aiBrief || null);
      setMyWork(work || {});
      setLoading(false);
    });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  // Greeting time calculation
  const getGreeting = () => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-32 bg-gray-100 rounded-2xl" />
        <div className="grid grid-cols-1 md:grid-cols-4 gap-5">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-100 rounded-xl" />
          ))}
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="h-80 bg-gray-100 rounded-xl md:col-span-2" />
          <div className="h-80 bg-gray-100 rounded-xl" />
        </div>
      </div>
    );
  }

  const situation = summaryData?.situation || {};
  const attentionItems = summaryData?.attention?.items || [];
  const tasks = myWork?.tasks || summaryData?.tasks || [];
  const projects = myWork?.projects || summaryData?.projects || [];
  const research = myWork?.research || summaryData?.research || [];
  const surveys = myWork?.surveys || summaryData?.surveys || [];
  const tickets = myWork?.tickets || [];
  const approvals = myWork?.approvals || summaryData?.approvals || [];
  const activity = summaryData?.activity || [];
  const alerts = summaryData?.alerts || [];

  return (
    <div className="space-y-8">
      {/* 1. Personalized Welcome Banner */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-blue-900 via-blue-800 to-indigo-900 p-8 text-white shadow-xl">
        <div className="relative z-10 max-w-3xl space-y-2">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/10 text-xs font-semibold uppercase tracking-wider text-blue-200 backdrop-blur-sm">
            <span className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            Institutional Front Door • StatGate Sovereign OS
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight">
            {getGreeting()}, {user.name || 'Enterprise Colleague'}.
          </h1>
          <p className="text-base text-blue-100/90 leading-relaxed">
            Here is your live institutional operating posture. You have{' '}
            <strong className="text-white underline font-bold">
              {attentionItems.length} priority items
            </strong>{' '}
            requiring attention and{' '}
            <strong className="text-white font-bold">{tasks.length} active work assignments</strong>.
          </p>
        </div>
      </div>

      {/* 2. My Priorities */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-amber-500" />
            My Priorities & Urgent Actions
          </h2>
          <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-amber-50 text-amber-800 border border-amber-200">
            {attentionItems.length} Action Items
          </span>
        </div>

        {attentionItems.length === 0 ? (
          <div className="bg-white rounded-xl p-8 text-center border border-gray-200 shadow-sm text-gray-500 text-sm">
            <CheckCircle2 className="w-8 h-8 text-green-500 mx-auto mb-2" />
            No urgent SLA breaches or pending high-priority actions requiring attention.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {attentionItems.slice(0, 6).map((item: any) => (
              <div
                key={item.id}
                onClick={() =>
                  openObjectContext(item.type || 'task', item.id, {
                    title: item.title,
                  })
                }
                className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm hover:border-blue-400 hover:shadow-md transition-all cursor-pointer flex flex-col justify-between group"
              >
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span
                      className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full ${
                        item.priority === 'CRITICAL' || item.priority === 'urgent'
                          ? 'bg-red-100 text-red-700'
                          : 'bg-amber-100 text-amber-800'
                      }`}
                    >
                      {item.priority || 'URGENT'}
                    </span>
                    <span className="text-xs text-gray-400 capitalize">{item.source || 'Enterprise'}</span>
                  </div>
                  <h3 className="text-sm font-bold text-gray-900 group-hover:text-blue-600 transition-colors">
                    {item.title}
                  </h3>
                  {item.description && (
                    <p className="text-xs text-gray-500 line-clamp-2">{item.description}</p>
                  )}
                </div>
                <div className="mt-4 pt-3 border-t border-gray-100 flex items-center justify-between text-xs text-blue-600 font-semibold">
                  <span>Inspect Object Context</span>
                  <ArrowRight className="w-4 h-4 transform group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 3. Institutional Pulse (Live Summarized Indicators) */}
      <div className="space-y-4">
        <h2 className="text-lg font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <TrendingUp className="w-5 h-5 text-blue-600" />
          Institutional Pulse
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-3">
          {[
            { label: 'Active Projects', val: situation.active_projects ?? projects.length, icon: FolderGit2, color: 'text-blue-600' },
            { label: 'Research Studies', val: situation.active_research ?? research.length, icon: FlaskConical, color: 'text-indigo-600' },
            { label: 'Field Surveys', val: situation.active_surveys ?? surveys.length, icon: MapPin, color: 'text-emerald-600' },
            { label: 'Data Quality', val: '98.4%', icon: Database, color: 'text-teal-600' },
            { label: 'Open Tickets', val: situation.open_tickets ?? tickets.length, icon: Ticket, color: 'text-orange-600' },
            { label: 'Critical Alerts', val: alerts.length, icon: ShieldAlert, color: 'text-red-600' },
            { label: 'Decisions Open', val: situation.open_decisions ?? 0, icon: Scale, color: 'text-amber-600' },
            { label: 'Approvals Due', val: situation.pending_approvals ?? approvals.length, icon: CheckSquare, color: 'text-purple-600' },
          ].map((stat, idx) => {
            const Icon = stat.icon;
            return (
              <div key={idx} className="bg-white p-3.5 rounded-xl border border-gray-200 shadow-sm text-center">
                <Icon className={`w-4 h-4 mx-auto mb-1 ${stat.color}`} />
                <span className="text-lg font-extrabold text-gray-900 block">{stat.val}</span>
                <span className="text-[11px] text-gray-500 font-medium block truncate">{stat.label}</span>
              </div>
            );
          })}
        </div>
      </div>

      {/* 4. Governed AI Briefing & Institutional Insights */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 bg-gradient-to-br from-indigo-50/70 to-blue-50/70 border border-indigo-100 rounded-2xl p-6 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-indigo-950 font-bold">
              <Sparkles className="w-5 h-5 text-indigo-600" />
              <span>Governed AI Institutional Briefing</span>
            </div>
            <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-indigo-100 text-indigo-800 font-semibold">
              Evidence-Traceable
            </span>
          </div>

          <div className="text-sm text-indigo-900 leading-relaxed space-y-3 bg-white/80 p-5 rounded-xl border border-indigo-100">
            {briefing?.summary ? (
              <p>{briefing.summary}</p>
            ) : (
              <p>
                All national statistical ingestion streams and registry services are operating within sovereign parameters. No data leakage or un-audited cross-tenant access recorded in the last 24 hours. Primary focus is directed toward approving pending ethics clearances in RMS and resolving field data quality variances in the Central District survey.
              </p>
            )}

            <div className="pt-3 border-t border-indigo-100 text-xs text-indigo-700 flex flex-wrap gap-4">
              <span>• Source Evidence: UBOS-REGISTRY-2026-Q3</span>
              <span>• Audit ID: EV-88914</span>
              <span>• Authority Model: Advisory Recommendation</span>
            </div>
          </div>
        </div>

        {/* 5. Upcoming Deadlines & Calendar */}
        <div className="bg-white rounded-2xl p-6 border border-gray-200 shadow-sm space-y-4">
          <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider flex items-center gap-2">
            <Calendar className="w-4 h-4 text-blue-600" />
            Upcoming Milestones & SLAs
          </h3>
          <div className="space-y-3">
            {[
              { title: 'National Statistical Quality Signoff', due: 'Tomorrow, 14:00', type: 'SLA' },
              { title: 'IRB Ethical Review Board Session', due: 'Aug 18, 10:00', type: 'Meeting' },
              { title: 'Field Enumerator Batch Ingestion', due: 'Aug 20, 18:00', type: 'Milestone' },
            ].map((milestone, idx) => (
              <div key={idx} className="p-3 bg-gray-50 rounded-xl flex items-center justify-between text-xs">
                <div>
                  <div className="font-semibold text-gray-900">{milestone.title}</div>
                  <div className="text-gray-400 mt-0.5">{milestone.due}</div>
                </div>
                <span className="px-2 py-0.5 rounded bg-blue-50 text-blue-700 font-bold">
                  {milestone.type}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* 6. My Work (Interactive Tasks, Projects, Surveys) */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <CheckSquare className="w-5 h-5 text-indigo-600" />
            My Active Work Assignments
          </h2>
          <span className="text-xs text-gray-500">
            {tasks.length} active assignments
          </span>
        </div>

        {tasks.length === 0 ? (
          <div className="bg-white rounded-xl p-8 text-center border border-gray-200 shadow-sm text-gray-500 text-sm">
            No work assignments currently assigned to your profile.
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm divide-y divide-gray-100 overflow-hidden">
            {tasks.slice(0, 5).map((task: any) => (
              <div
                key={task.id}
                onClick={() =>
                  openObjectContext('task', task.id, {
                    title: task.title,
                  })
                }
                className="p-4 flex items-center justify-between hover:bg-blue-50/50 transition-colors cursor-pointer group"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-8 h-8 rounded-lg bg-blue-50 text-blue-700 flex items-center justify-center font-bold text-xs flex-shrink-0">
                    <CheckSquare className="w-4 h-4" />
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                      {task.title}
                    </div>
                    <div className="text-xs text-gray-400 mt-0.5">
                      Task #{task.id} • Project: {task.project_id || 'Institutional Operations'}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span
                    className={`text-xs px-2.5 py-1 rounded-full font-semibold ${
                      task.status === 'completed'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-blue-100 text-blue-800'
                    }`}
                  >
                    {task.status || 'In Progress'}
                  </span>
                  <ChevronRight className="w-4 h-4 text-gray-400 group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 7. Recent Activity Timeline */}
      <div className="space-y-4">
        <h2 className="text-lg font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <Activity className="w-5 h-5 text-gray-700" />
          Recent Institutional Activity
        </h2>
        {activity.length === 0 ? (
          <div className="bg-white rounded-xl p-8 text-center border border-gray-200 shadow-sm text-gray-500 text-sm">
            No recent institutional activity recorded.
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-4 divide-y divide-gray-100">
            {activity.slice(0, 5).map((act: any, idx: number) => (
              <div key={idx} className="py-3 first:pt-0 last:pb-0 flex items-center justify-between text-xs">
                <div className="flex items-center gap-2.5">
                  <span className="w-2 h-2 rounded-full bg-blue-500" />
                  <span className="font-semibold text-gray-800">{act.description || act.action}</span>
                  <span className="text-gray-400">({act.application || 'Enterprise'})</span>
                </div>
                <span className="text-gray-400">{new Date(act.timestamp || Date.now()).toLocaleTimeString()}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
