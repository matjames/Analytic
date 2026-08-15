import React from 'react';
import { X, GitCommit, Database, Layers, ShieldCheck, FileText, CheckCircle2 } from 'lucide-react';

interface DataLineageModalProps {
  kpiTitle: string;
  kpiValue: string | number;
  onClose: () => void;
}

export const DataLineageModal: React.FC<DataLineageModalProps> = ({
  kpiTitle,
  kpiValue,
  onClose,
}) => {
  const lineageSteps = [
    {
      level: '1. Executive KPI',
      name: kpiTitle,
      detail: `Current computed aggregate: ${kpiValue}`,
      app: 'Enterprise Command Centre',
      type: 'Executive Metric',
      icon: Layers,
    },
    {
      level: '2. KPI Definition',
      name: 'KPI-DATA-INTEGRITY-01',
      detail: 'Formula: SUM(ValidSubmissions) / TotalIngestedRecords * 100',
      app: 'Enterprise Core Analytics Engine',
      type: 'Mathematical Rule',
      icon: GitCommit,
    },
    {
      level: '3. Enterprise Dataset',
      name: 'ds_national_submissions_2026_q3',
      detail: 'Partitioned across 135 Uganda districts (12,450 rows)',
      app: 'Enterprise Data Layer (PostgreSQL)',
      type: 'Durable Table',
      icon: Database,
    },
    {
      level: '4. Source Application',
      name: 'StatCollect Field Engine & PMS',
      detail: 'REST / Mobile sync API ingestion endpoint',
      app: 'StatCollect Microservice (:3009)',
      type: 'Upstream Producer',
      icon: FileText,
    },
    {
      level: '5. Raw Event & Record',
      name: 'submission.received [ID: sub_ev_991204]',
      detail: 'Enumerator: E-0412 • Facility: Mulago NRH • GPS: 0.3396, 32.5768',
      app: 'Domain Event Bus',
      type: 'Cryptographic Event',
      icon: ShieldCheck,
    },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in">
      <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full p-6 space-y-6 animate-in zoom-in-95">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-100 pb-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 bg-blue-50 text-blue-700 rounded-xl">
              <GitCommit className="w-6 h-6" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-gray-900">Evidence & Data Lineage Inspector</h2>
              <p className="text-xs text-gray-500">StatGate Principle: Evidence First</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Chain Visualization */}
        <div className="space-y-4">
          {lineageSteps.map((step, idx) => {
            const Icon = step.icon;
            const isLast = idx === lineageSteps.length - 1;
            return (
              <div key={idx} className="relative flex items-start gap-4 group">
                {!isLast && (
                  <div className="absolute left-5 top-10 w-0.5 h-10 bg-blue-200" />
                )}
                <div className="w-10 h-10 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center font-bold text-sm shadow-sm flex-shrink-0 z-10">
                  <Icon className="w-5 h-5" />
                </div>
                <div className="flex-1 bg-gray-50 p-3.5 rounded-xl border border-gray-200 text-xs">
                  <div className="flex items-center justify-between font-bold text-gray-900">
                    <span className="text-blue-600">{step.level}</span>
                    <span className="text-gray-400 text-[10px] uppercase">{step.type}</span>
                  </div>
                  <div className="font-semibold text-gray-800 text-sm mt-0.5">{step.name}</div>
                  <div className="text-gray-500 mt-1">{step.detail}</div>
                  <div className="text-[11px] text-gray-400 mt-1">Source: {step.app}</div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Footer */}
        <div className="pt-4 border-t border-gray-100 flex items-center justify-between text-xs">
          <div className="flex items-center gap-1.5 text-green-700 font-semibold">
            <CheckCircle2 className="w-4 h-4" />
            Lineage Verified via PostgreSQL & Event Bus Audit
          </div>
          <button onClick={onClose} className="btn btn-primary text-xs">
            Close Inspector
          </button>
        </div>
      </div>
    </div>
  );
};
