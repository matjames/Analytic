import React, { useState, useEffect } from 'react';
import { User } from '@typings/index';
import { CommandCentreNav, CommandCentreTab } from '@components/CommandCentreNav';
import { HomeView } from '@components/cc/HomeView';
import { ExecutiveView } from '@components/cc/ExecutiveView';
import { OperationsView } from '@components/cc/OperationsView';
import { FieldView } from '@components/cc/FieldView';
import { ProjectView } from '@components/cc/ProjectView';
import { ResearchView } from '@components/cc/ResearchView';
import { DataIntelligenceView } from '@components/cc/DataIntelligenceView';
import { GovernanceView } from '@components/cc/GovernanceView';
import { DecisionCentreView } from '@components/cc/DecisionCentreView';
import { InvestigationsView } from '@components/cc/InvestigationsView';
import { DashboardBuilderView } from '@components/cc/DashboardBuilderView';
import { KnowledgeFabricView } from '@components/cc/KnowledgeFabricView';
import { ActivityView } from '@components/cc/ActivityView';
import { ServiceHealthView } from '@components/cc/ServiceHealthView';
import { OutcomeMonitorView } from '@components/cc/OutcomeMonitorView';
import { ActionCentreView } from '@components/cc/ActionCentreView';
import { MyWorkView } from '@components/cc/MyWorkView';
import { MyBriefingView } from '@components/cc/MyBriefingView';
import { PlatformHealthView } from '@components/cc/PlatformHealthView';
import { SecurityView } from '@components/cc/SecurityView';
import { ResilienceOperationsView } from '@components/cc/ResilienceOperationsView';
import { IncidentManagementView } from '@components/cc/IncidentManagementView';
import { RecoveryDrillView } from '@components/cc/RecoveryDrillView';
import { BackupAssuranceView } from '@components/cc/BackupAssuranceView';
import { DataIntegrityView } from '@components/cc/DataIntegrityView';
import { RunbookView } from '@components/cc/RunbookView';
import { ResilienceEvidenceView } from '@components/cc/ResilienceEvidenceView';
import { InstitutionalConditionView } from '@components/cc/InstitutionalConditionView';
import { KnowledgeGraphView } from '@components/cc/KnowledgeGraphView';
import { ObjectivesKPIView } from '@components/cc/ObjectivesKPIView';
import { RiskIntelligenceView } from '@components/cc/RiskIntelligenceView';
import { AIRecommendationsView } from '@components/cc/AIRecommendationsView';
import { DataLineageView } from '@components/cc/DataLineageView';
import { ObjectContextProvider } from '@context/ObjectContext';
import { ObjectContextModal } from '@components/ObjectContextModal';
import { Radio } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const CommandCentre: React.FC<{ user: User }> = ({ user }) => {
  const [activeTab, setActiveTab] = useState<CommandCentreTab>('home');
  const [liveConnected, setLiveConnected] = useState(false);
  const [badges, setBadges] = useState<{
    actionCount?: number;
    tasksCount?: number;
    decisionsCount?: number;
    healthAlert?: boolean;
  }>({});

  // Fetch summary badge counts from Enterprise Core
  useEffect(() => {
    let cancelled = false;
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/summary?user_id=${encodeURIComponent(user.id)}&role=${encodeURIComponent(user.role)}`, {
      headers,
    })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((data) => {
        if (cancelled || !data) return;
        setBadges({
          actionCount: data.attention?.total_items || 0,
          tasksCount: data.tasks?.length || 0,
          decisionsCount: data.decisions?.count || 0,
          healthAlert: data.service_health?.some((s: any) => s.status !== 'healthy'),
        });
      });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role, activeTab]);

  // Real-Time SSE Stream integration
  useEffect(() => {
    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource(`${API_BASE}/api/analytics/stream`);
      eventSource.onopen = () => {
        setLiveConnected(true);
      };
      eventSource.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed && (parsed.event_type || parsed.type)) {
            setBadges((prev) => ({
              ...prev,
              actionCount: (prev.actionCount || 0) + 1,
            }));
          }
        } catch {
          // Heartbeat
        }
      };
      eventSource.onerror = () => {
        setLiveConnected(false);
      };
    } catch {
      setLiveConnected(false);
    }

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, []);

  const renderActiveView = () => {
    switch (activeTab) {
      case 'home':
        return <HomeView user={user} />;
      case 'briefing':
        return <MyBriefingView user={user} />;
      case 'my-work':
        return <MyWorkView user={user} />;
      case 'action-centre':
        return <ActionCentreView user={user} />;
      case 'executive':
        return <ExecutiveView user={user} />;
      case 'operations':
        return <OperationsView user={user} />;
      case 'field':
        return <FieldView user={user} />;
      case 'projects':
        return <ProjectView user={user} />;
      case 'research':
        return <ResearchView user={user} />;
      case 'data-intelligence':
        return <DataIntelligenceView user={user} />;
      case 'governance':
        return <GovernanceView user={user} />;
      case 'decisions':
        return <DecisionCentreView user={user} />;
      case 'investigations':
        return <InvestigationsView user={user} />;
      case 'fabric':
        return <KnowledgeFabricView user={user} />;
      case 'builder':
        return <DashboardBuilderView user={user} />;
      case 'activity':
        return <ActivityView user={user} />;
      case 'service-health':
        return <ServiceHealthView user={user} />;
      case 'outcomes':
        return <OutcomeMonitorView user={user} />;
      case 'platform-health':
        return <PlatformHealthView />;
      case 'security':
        return <SecurityView />;
      case 'resilience-operations':
        return <ResilienceOperationsView onNavigate={setActiveTab} />;
      case 'incidents':
        return <IncidentManagementView />;
      case 'dr-drills':
        return <RecoveryDrillView />;
      case 'backups':
        return <BackupAssuranceView />;
      case 'integrity':
        return <DataIntegrityView />;
      case 'runbooks':
        return <RunbookView />;
      case 'evidence':
        return <ResilienceEvidenceView />;
      case 'institutional-condition':
        return <InstitutionalConditionView />;
      case 'knowledge-graph':
        return <KnowledgeGraphView />;
      case 'objectives-kpi':
        return <ObjectivesKPIView />;
      case 'risk-intelligence':
        return <RiskIntelligenceView />;
      case 'ai-recommendations':
        return <AIRecommendationsView />;
      case 'data-lineage':
        return <DataLineageView />;
      default:
        return <HomeView user={user} />;
    }
  };

  return (
    <ObjectContextProvider>
      <div className="flex min-h-[calc(100vh-4rem)] bg-gray-50/50">
        {/* Sidebar Navigation */}
        <CommandCentreNav
          user={user}
          activeTab={activeTab}
          onSelectTab={setActiveTab}
          badges={badges}
        />

        {/* Main Content Area */}
        <main className="flex-1 p-6 md:p-8 max-w-7xl mx-auto w-full min-w-0 overflow-y-auto">
          {/* Live Event Stream Indicator */}
          <div className="flex items-center justify-between pb-6 mb-6 border-b border-gray-200">
            <div className="flex items-center gap-2">
              <span
                className={`w-2.5 h-2.5 rounded-full ${
                  liveConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-300'
                }`}
              />
              <span className="text-xs font-semibold uppercase tracking-wider text-gray-500 flex items-center gap-1">
                <Radio className="w-3.5 h-3.5 text-gray-400" />
                {liveConnected ? 'Enterprise Real-Time Event Bus Connected' : 'Event Polling Mode'}
              </span>
            </div>

            <div className="text-xs text-gray-400 font-mono">
              {new Date().toLocaleDateString(undefined, {
                weekday: 'short',
                year: 'numeric',
                month: 'short',
                day: 'numeric',
              })}
            </div>
          </div>

          {/* View Component */}
          <div className="animate-fade-in">{renderActiveView()}</div>
        </main>

        {/* Universal Object Context Modal */}
        <ObjectContextModal user={user} />
      </div>
    </ObjectContextProvider>
  );
};

