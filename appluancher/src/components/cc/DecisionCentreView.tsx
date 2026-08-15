import React, { useEffect, useState, useCallback } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { Scale, Clock, Sparkles, Plus, ArrowRight } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const DecisionCentreView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [decisionText, setDecisionText] = useState('');
  const [actionRequired, setActionRequired] = useState('');
  const [deadline, setDeadline] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchData = useCallback(() => {
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/decisions?user_id=${encodeURIComponent(user.id)}`, {
      headers,
    })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        setData(res);
        setLoading(false);
      });
  }, [user.id, user.role]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreateDecision = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!decisionText.trim()) return;

    setCreating(true);
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-User-ID': user.id,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    try {
      const res = await fetch(`${API_BASE}/api/decisions`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          decision: decisionText,
          action_required: actionRequired,
          responsible_person: user.id,
          decision_maker: user.name || user.id,
          deadline: deadline || new Date(Date.now() + 7 * 86400000).toISOString(),
          status: 'open',
        }),
      });
      if (res.ok) {
        setShowModal(false);
        setDecisionText('');
        setActionRequired('');
        setDeadline('');
        fetchData();
      }
    } finally {
      setCreating(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-28 bg-gray-100 rounded-xl" />
        <div className="h-96 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  const awaiting = data?.awaiting_me || [];
  const created = data?.created_by_me || [];
  const recent = data?.recent || [];
  const recommendations = data?.ai_recommendations || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <Scale className="w-6 h-6 text-amber-600" />
            Decision Centre & Closed-Loop Intelligence
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Governed decision records: Data → Anomaly → Investigation → Evidence → Recommendation → Human Decision → Action → Outcome.
          </p>
        </div>
        <button
          onClick={() => setShowModal(true)}
          className="btn btn-primary text-xs flex items-center gap-1.5 shadow-sm"
        >
          <Plus className="w-4 h-4" />
          Record New Decision
        </button>
      </div>

      {/* Decision Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <div className="stat-card border-l-4 border-l-amber-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Decisions Awaiting Action</span>
            <Clock className="w-5 h-5 text-amber-500" />
          </div>
          <span className="stat-val">{awaiting.length}</span>
          <span className="text-xs text-gray-400 mt-1">Open decisions assigned to you</span>
        </div>

        <div className="stat-card border-l-4 border-l-blue-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Recorded Decisions</span>
            <Scale className="w-5 h-5 text-blue-500" />
          </div>
          <span className="stat-val">{created.length}</span>
          <span className="text-xs text-gray-400 mt-1">Created by your account</span>
        </div>

        <div className="stat-card border-l-4 border-l-purple-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">AI Recommendations</span>
            <Sparkles className="w-5 h-5 text-purple-500" />
          </div>
          <span className="stat-val">{recommendations.length}</span>
          <span className="text-xs text-purple-700 font-medium mt-1">Grounded suggestions</span>
        </div>
      </div>

      {/* Grid: Awaiting Me & Recent Decisions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Decisions Awaiting Action */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Decisions Awaiting Action</h3>
            <span className="text-xs text-gray-400">{awaiting.length}</span>
          </div>

          {awaiting.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">No decisions currently awaiting your action.</p>
          ) : (
            <div className="divide-y divide-gray-100">
              {awaiting.map((d: any) => (
                <div
                  key={d.id}
                  onClick={() => openObjectContext('decision', d.id, { title: d.decision })}
                  className="py-3 flex items-start justify-between gap-4 p-2 hover:bg-amber-50/50 rounded-lg cursor-pointer transition-colors group"
                >
                  <div>
                    <div className="text-sm font-semibold text-gray-900 group-hover:text-amber-800 transition-colors">
                      {d.decision}
                    </div>
                    <div className="text-xs text-gray-600 mt-1">
                      Action required: {d.action_required || 'Review and execute'}
                    </div>
                    {d.deadline && (
                      <div className="text-[11px] text-gray-400 mt-1">Deadline: {d.deadline}</div>
                    )}
                  </div>
                  <span className="badge badge-high text-[10px]">{d.status || 'Open'}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* AI Recommendations */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-purple-600" />
              Governed AI Recommendations
            </h3>
            <span className="text-xs text-purple-700 font-semibold bg-purple-50 px-2 py-0.5 rounded border border-purple-200">
              Human Review Required
            </span>
          </div>

          {recommendations.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">No pending AI recommendations.</p>
          ) : (
            <div className="space-y-3">
              {recommendations.map((rec: any) => (
                <div
                  key={rec.id}
                  onClick={() => openObjectContext('recommendation', rec.id, { title: rec.title })}
                  className="p-4 rounded-xl border border-purple-200 bg-purple-50/40 hover:bg-purple-50 cursor-pointer transition-colors space-y-2"
                >
                  <div className="text-sm font-semibold text-purple-950">{rec.title}</div>
                  <p className="text-xs text-purple-900 leading-relaxed">{rec.recommendation || rec.description}</p>
                  <div className="flex items-center justify-between pt-2 border-t border-purple-100 text-[11px] text-purple-700">
                    <span>Evidence: Grounded (EV-{rec.id?.slice(0, 5)})</span>
                    <span className="font-bold flex items-center gap-1">
                      Review & Decide <ArrowRight className="w-3 h-3" />
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Recent Institutional Decisions */}
      {recent.length > 0 && (
        <div className="glass-panel">
          <h3 className="text-base font-semibold text-gray-900 mb-3">
            Recent Institutional Decisions ({recent.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {recent.slice(0, 4).map((d: any) => (
              <div
                key={d.id}
                onClick={() => openObjectContext('decision', d.id, { title: d.decision })}
                className="p-3 rounded-lg border border-gray-200 bg-gray-50/50 hover:bg-blue-50/50 cursor-pointer transition-colors"
              >
                <div className="text-xs font-semibold text-gray-900">{d.decision}</div>
                <div className="text-xs text-gray-500 mt-1">
                  Maker: {d.decision_maker || 'Executive Lead'} • Status: {d.status || 'Recorded'}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Decision Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <form
            onSubmit={handleCreateDecision}
            className="bg-white rounded-2xl shadow-2xl max-w-lg w-full p-6 space-y-4"
          >
            <h3 className="text-lg font-bold text-gray-900">Record Institutional Decision</h3>
            <p className="text-xs text-gray-500">
              Preserve institutional memory. Every decision record is auditable and linked to evidence.
            </p>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Decision Statement</label>
              <textarea
                value={decisionText}
                onChange={(e) => setDecisionText(e.target.value)}
                placeholder="E.g. Approved deployment of mobile field teams to Gulu district."
                rows={3}
                className="w-full border border-gray-300 rounded-lg p-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Action Required</label>
              <input
                type="text"
                value={actionRequired}
                onChange={(e) => setActionRequired(e.target.value)}
                placeholder="E.g. Allocate logistics budget and notify regional field manager."
                className="w-full border border-gray-300 rounded-lg p-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="btn btn-secondary text-xs"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="btn btn-primary text-xs"
              >
                {creating ? 'Recording...' : 'Save Decision Record'}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
