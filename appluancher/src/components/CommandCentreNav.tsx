import React, { useState } from 'react';
import { User, UserRole } from '@typings/index';
import { useAuth } from '@context/AuthContext';
import {
  LayoutDashboard,
  Sparkles,
  CheckSquare,
  AlertOctagon,
  TrendingUp,
  Sliders,
  MapPin,
  FolderGit2,
  FlaskConical,
  Database,
  ShieldCheck,
  Scale,
  Brain,
  History,
  Activity,
  CheckCircle2,
  LayoutGrid,
  Network,
  LogOut,
  Server,
  Lock,
  Flame,
  BookOpen,
  FileCheck2,
  RotateCcw,
  ShieldAlert,
  GitBranch,
  Target,
} from 'lucide-react';


export type CommandCentreTab =
  | 'home'
  | 'briefing'
  | 'my-work'
  | 'action-centre'
  | 'executive'
  | 'operations'
  | 'field'
  | 'projects'
  | 'research'
  | 'data-intelligence'
  | 'fabric'
  | 'governance'
  | 'decisions'
  | 'investigations'
  | 'builder'
  | 'activity'
  | 'service-health'
  | 'outcomes'
  | 'platform-health'
  | 'security'
  | 'resilience-operations'
  | 'incidents'
  | 'dr-drills'
  | 'backups'
  | 'integrity'
  | 'runbooks'
  | 'evidence'
  | 'institutional-condition'
  | 'knowledge-graph'
  | 'objectives-kpi'
  | 'risk-intelligence'
  | 'ai-recommendations'
  | 'data-lineage';

interface NavItem {
  id: CommandCentreTab;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  allowedRoles?: UserRole[];
  badgeKey?: string;
}

const NAV_GROUPS: { groupLabel: string; items: NavItem[] }[] = [
  {
    groupLabel: 'Personal Workspace',
    items: [
      { id: 'home', label: 'Command Home', icon: LayoutDashboard },
      { id: 'briefing', label: 'AI Briefing', icon: Sparkles },
      { id: 'my-work', label: 'My Work', icon: CheckSquare, badgeKey: 'tasksCount' },
      { id: 'action-centre', label: 'Action Centre', icon: AlertOctagon, badgeKey: 'actionCount' },
    ],
  },
  {
    groupLabel: 'Institutional Views',
    items: [
      {
        id: 'executive',
        label: 'Executive Command',
        icon: TrendingUp,
        allowedRoles: [UserRole.ADMIN, UserRole.MANAGER],
      },
      {
        id: 'operations',
        label: 'Operations Console',
        icon: Sliders,
        allowedRoles: [UserRole.ADMIN, UserRole.MANAGER, UserRole.ANALYST],
      },
      { id: 'field', label: 'Field & Facilities', icon: MapPin },
      { id: 'projects', label: 'Project Portfolio', icon: FolderGit2 },
      { id: 'research', label: 'Research (RMS)', icon: FlaskConical },
    ],
  },
  {
    groupLabel: 'Intelligence & Decision',
    items: [
      { id: 'data-intelligence', label: 'Data Intelligence', icon: Database },
      { id: 'fabric', label: 'Data & Knowledge Fabric', icon: Network },
      { id: 'governance', label: 'Governance & Risk', icon: ShieldCheck },
      { id: 'decisions', label: 'Decision Centre', icon: Scale, badgeKey: 'decisionsCount' },
      { id: 'investigations', label: 'AI Investigations', icon: Brain },
      { id: 'builder', label: 'Dashboard Builder', icon: LayoutGrid },
      { id: 'outcomes', label: 'Outcome Monitor', icon: CheckCircle2 },
    ],
  },
  {
    groupLabel: 'Platform & Audit',
    items: [
      { id: 'activity', label: 'Enterprise Activity', icon: History },
      { id: 'service-health', label: 'Service Health', icon: Activity, badgeKey: 'healthAlert' },
    ],
  },
  {
    groupLabel: 'Resilience & Assurance',
    items: [
      {
        id: 'resilience-operations' as CommandCentreTab,
        label: 'Resilience Overview',
        icon: ShieldAlert,
      },
      {
        id: 'incidents' as CommandCentreTab,
        label: 'Incident Lifecycle',
        icon: AlertOctagon,
      },
      {
        id: 'dr-drills' as CommandCentreTab,
        label: 'Recovery Drills',
        icon: Flame,
      },
      {
        id: 'backups' as CommandCentreTab,
        label: 'Backup Assurance',
        icon: RotateCcw,
      },
      {
        id: 'integrity' as CommandCentreTab,
        label: 'Data Integrity',
        icon: FileCheck2,
      },
      {
        id: 'runbooks' as CommandCentreTab,
        label: 'Recovery Runbooks',
        icon: BookOpen,
      },
      {
        id: 'evidence' as CommandCentreTab,
        label: 'Resilience Evidence',
        icon: ShieldCheck,
      },
    ],
  },
  {
    groupLabel: 'Institutional Intelligence',
    items: [
      { id: 'institutional-condition' as CommandCentreTab, label: 'Institutional Condition', icon: Activity },
      { id: 'knowledge-graph' as CommandCentreTab, label: 'Knowledge Graph', icon: Network },
      { id: 'objectives-kpi' as CommandCentreTab, label: 'Objectives & KPI', icon: Target },
      { id: 'risk-intelligence' as CommandCentreTab, label: 'Risk Intelligence', icon: ShieldAlert },
      { id: 'ai-recommendations' as CommandCentreTab, label: 'AI Recommendations', icon: Brain },
      { id: 'data-lineage' as CommandCentreTab, label: 'Data Lineage', icon: GitBranch },
    ],
  },
  {
    groupLabel: 'Platform Administration',
    items: [
      {
        id: 'platform-health' as CommandCentreTab,
        label: 'Platform Health',
        icon: Server,
        allowedRoles: [UserRole.ADMIN],
      },
      {
        id: 'security' as CommandCentreTab,
        label: 'Security & Audit',
        icon: Lock,
        allowedRoles: [UserRole.ADMIN],
      },
    ],
  },
];

interface CommandCentreNavProps {
  user: User;
  activeTab: CommandCentreTab;
  onSelectTab: (tab: CommandCentreTab) => void;
  badges?: {
    actionCount?: number;
    tasksCount?: number;
    decisionsCount?: number;
    healthAlert?: boolean;
  };
}

export const CommandCentreNav: React.FC<CommandCentreNavProps> = ({
  user,
  activeTab,
  onSelectTab,
  badges = {},
}) => {
  const { logout } = useAuth();
  const [confirmSignOut, setConfirmSignOut] = useState(false);

  const handleSignOut = () => {
    setConfirmSignOut(false);
    logout();
  };

  return (
    <>
      <aside className="w-64 flex-shrink-0 bg-white border-r border-gray-200 min-h-[calc(100vh-4rem)] flex flex-col justify-between p-4 select-none">
        <div className="space-y-6">
          {/* User Identity Header */}
          <div className="pb-3 border-b border-gray-100 flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-blue-100 text-blue-700 font-bold flex items-center justify-center text-sm shadow-sm">
              {user.name ? user.name.slice(0, 2).toUpperCase() : 'SG'}
            </div>
            <div className="min-w-0 flex-1">
              <div className="text-sm font-semibold text-gray-900 truncate">{user.name}</div>
              <div className="flex items-center gap-1.5">
                <span className="inline-block w-2 h-2 rounded-full bg-green-500" />
                <span className="text-xs uppercase font-medium tracking-wider text-gray-500">
                  {user.role}
                </span>
              </div>
            </div>
          </div>

          {/* Navigation Groups */}
          {NAV_GROUPS.map((group) => {
            const filteredItems = group.items.filter((item) => {
              if (!item.allowedRoles) return true;
              return item.allowedRoles.includes(user.role);
            });

            if (filteredItems.length === 0) return null;

            return (
              <div key={group.groupLabel} className="space-y-1">
                <div className="px-3 text-[11px] font-semibold uppercase tracking-wider text-gray-400">
                  {group.groupLabel}
                </div>
                {filteredItems.map((item) => {
                  const Icon = item.icon;
                  const isActive = activeTab === item.id;

                  let badgeNode = null;
                  if (item.badgeKey === 'actionCount' && (badges.actionCount ?? 0) > 0) {
                    badgeNode = (
                      <span className="ml-auto px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-red-100 text-red-700">
                        {badges.actionCount}
                      </span>
                    );
                  } else if (item.badgeKey === 'tasksCount' && (badges.tasksCount ?? 0) > 0) {
                    badgeNode = (
                      <span className="ml-auto px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-blue-100 text-blue-700">
                        {badges.tasksCount}
                      </span>
                    );
                  } else if (item.badgeKey === 'decisionsCount' && (badges.decisionsCount ?? 0) > 0) {
                    badgeNode = (
                      <span className="ml-auto px-1.5 py-0.5 text-[10px] font-bold rounded-full bg-amber-100 text-amber-800">
                        {badges.decisionsCount}
                      </span>
                    );
                  } else if (item.badgeKey === 'healthAlert' && badges.healthAlert) {
                    badgeNode = (
                      <span className="ml-auto w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
                    );
                  }

                  return (
                    <button
                      key={item.id}
                      onClick={() => onSelectTab(item.id)}
                      className={`w-full flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-lg transition-all ${
                        isActive
                          ? 'bg-blue-50 text-blue-700 font-semibold shadow-sm'
                          : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                      }`}
                    >
                      <Icon className={`w-4 h-4 ${isActive ? 'text-blue-700' : 'text-gray-400'}`} />
                      <span className="truncate">{item.label}</span>
                      {badgeNode}
                    </button>
                  );
                })}
              </div>
            );
          })}
        </div>

        {/* Bottom Actions: Sign Out & Institutional Footer */}
        <div className="pt-4 border-t border-gray-100 space-y-3">
          <button
            onClick={() => setConfirmSignOut(true)}
            className="w-full flex items-center gap-3 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg transition-colors"
          >
            <LogOut className="w-4 h-4 text-red-500" />
            <span>Sign Out</span>
          </button>

          <div className="text-[11px] text-gray-400 text-center">
            <span>StatGate Phase XI</span>
            <div className="text-[10px] text-gray-400">Institutional Resilience & Assurance</div>
          </div>
        </div>
      </aside>

      {/* Confirmation Modal */}
      {confirmSignOut && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full p-6 space-y-4 animate-in fade-in zoom-in-95">
            <div className="flex items-center gap-3 text-red-600">
              <div className="p-3 bg-red-50 rounded-xl">
                <LogOut className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-lg font-bold text-gray-900">Sign Out Confirmation</h3>
                <p className="text-xs text-gray-500">StatGate Enterprise Session</p>
              </div>
            </div>
            <p className="text-sm text-gray-600">
              Are you sure you want to sign out? Your active sessions and real-time feeds will be disconnected.
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setConfirmSignOut(false)}
                className="btn btn-secondary text-sm"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSignOut}
                className="btn btn-danger text-sm"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

