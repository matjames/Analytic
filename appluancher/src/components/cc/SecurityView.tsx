import React, { useEffect, useState, useCallback } from 'react';
import {
  Lock,
  Search,
  Filter,
  RefreshCw,
  User,
  Clock,
  ShieldAlert,
  CheckCircle,
  XCircle,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface AuditEntry {
  actor: string;
  action: string;
  resource: string;
  resource_id: string;
  tenant_id: string;
  correlation_id: string;
  request_id: string;
  outcome: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

interface AuditResponse {
  total: number;
  count: number;
  offset: number;
  entries: AuditEntry[];
  source: string;
}

function ActionBadge({ action }: { action: string }) {
  const category = action.split('.')[0];
  const colors: Record<string, string> = {
    events:      'bg-blue-50 text-blue-700',
    decisions:   'bg-purple-50 text-purple-700',
    workflows:   'bg-indigo-50 text-indigo-700',
    auth:        'bg-amber-50 text-amber-700',
    permissions: 'bg-red-50 text-red-700',
    knowledge:   'bg-emerald-50 text-emerald-700',
    ai:          'bg-pink-50 text-pink-700',
  };
  const cls = colors[category] ?? 'bg-gray-100 text-gray-700';
  return (
    <span className={`inline-flex px-2 py-0.5 rounded-md text-[11px] font-semibold font-mono ${cls}`}>
      {action}
    </span>
  );
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  if (outcome === 'success') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-emerald-600 font-medium">
        <CheckCircle className="w-3.5 h-3.5" /> success
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs text-red-600 font-medium">
      <XCircle className="w-3.5 h-3.5" /> {outcome}
    </span>
  );
}

function AuditRow({ entry }: { entry: AuditEntry }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = entry.metadata && Object.keys(entry.metadata).length > 0;

  return (
    <>
      <tr
        className={`border-b border-gray-50 hover:bg-gray-50 transition-colors ${hasDetails ? 'cursor-pointer' : ''}`}
        onClick={() => hasDetails && setExpanded(!expanded)}
      >
        <td className="px-4 py-3 text-xs font-mono text-gray-500 whitespace-nowrap">
          {new Date(entry.created_at).toLocaleString()}
        </td>
        <td className="px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-700 font-bold flex items-center justify-center text-[11px] flex-shrink-0">
              {entry.actor ? entry.actor.slice(0, 2).toUpperCase() : '??'}
            </div>
            <span className="text-xs text-gray-700 font-medium truncate max-w-[120px]">
              {entry.actor || '—'}
            </span>
          </div>
        </td>
        <td className="px-4 py-3">
          <ActionBadge action={entry.action} />
        </td>
        <td className="px-4 py-3 text-xs text-gray-600 font-mono truncate max-w-[140px]">
          {entry.resource}/{entry.resource_id || '—'}
        </td>
        <td className="px-4 py-3">
          <OutcomeBadge outcome={entry.outcome} />
        </td>
        <td className="px-4 py-3 text-center">
          {hasDetails && (
            expanded
              ? <ChevronDown className="w-3.5 h-3.5 text-gray-400 mx-auto" />
              : <ChevronRight className="w-3.5 h-3.5 text-gray-400 mx-auto" />
          )}
        </td>
      </tr>
      {expanded && hasDetails && (
        <tr className="bg-gray-50">
          <td colSpan={6} className="px-6 py-3">
            <div className="text-xs font-mono text-gray-600 bg-white border border-gray-100 rounded-lg p-3 overflow-x-auto">
              <div className="grid grid-cols-2 gap-2">
                {entry.correlation_id && (
                  <div><span className="text-gray-400">correlation: </span>{entry.correlation_id}</div>
                )}
                {entry.tenant_id && (
                  <div><span className="text-gray-400">tenant: </span>{entry.tenant_id}</div>
                )}
              </div>
              {entry.metadata && (
                <pre className="mt-2 text-[11px] text-gray-500 whitespace-pre-wrap">
                  {JSON.stringify(entry.metadata, null, 2)}
                </pre>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export const SecurityView: React.FC = () => {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [source, setSource] = useState('');
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const [filterActor, setFilterActor] = useState('');
  const [filterAction, setFilterAction] = useState('');
  const [filterResource, setFilterResource] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const limit = 50;

  const fetchAuditLog = useCallback(async (reset = false) => {
    const off = reset ? 0 : offset;
    if (reset) setOffset(0);
    setRefreshing(true);
    try {
      const params = new URLSearchParams({
        limit: String(limit),
        offset: String(off),
        ...(filterActor ? { actor: filterActor } : {}),
        ...(filterAction ? { action: filterAction } : {}),
        ...(filterResource ? { resource: filterResource } : {}),
      });
      const r = await fetch(`${ENTERPRISE_CORE}/api/apis/audit?${params}`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token') ?? ''}` },
      });
      if (r.ok) {
        const data: AuditResponse = await r.json();
        setEntries(data.entries ?? []);
        setTotal(data.total ?? 0);
        setSource(data.source ?? '');
      }
    } catch { /* non-fatal */ } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [offset, filterActor, filterAction, filterResource]);

  useEffect(() => { fetchAuditLog(true); }, [fetchAuditLog]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    fetchAuditLog(true);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
            <Lock className="w-6 h-6 text-red-600" />
            Security & Audit Log
          </h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Phase X — Immutable institutional audit trail
            {source === 'postgresql' && (
              <span className="ml-2 text-xs text-emerald-600 font-medium">● PostgreSQL (authoritative)</span>
            )}
            {source === 'unavailable' && (
              <span className="ml-2 text-xs text-amber-600 font-medium">⚠ Database not configured</span>
            )}
          </p>
        </div>
        <button
          id="security-view-refresh"
          onClick={() => fetchAuditLog(true)}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-600 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Security Status Cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <div className="flex items-center gap-2 mb-1">
            <ShieldAlert className="w-4 h-4 text-blue-600" />
            <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Total Audit Events</span>
          </div>
          <div className="text-2xl font-bold text-gray-900">{total.toLocaleString()}</div>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <div className="flex items-center gap-2 mb-1">
            <User className="w-4 h-4 text-emerald-600" />
            <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Audit Store</span>
          </div>
          <div className="text-sm font-semibold text-gray-700">
            {source === 'postgresql' ? (
              <span className="text-emerald-600">PostgreSQL (durable)</span>
            ) : source === 'unavailable' ? (
              <span className="text-amber-600">Not configured</span>
            ) : (
              <span className="text-gray-400">—</span>
            )}
          </div>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <div className="flex items-center gap-2 mb-1">
            <Clock className="w-4 h-4 text-purple-600" />
            <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">Showing</span>
          </div>
          <div className="text-2xl font-bold text-gray-900">{entries.length}</div>
          <div className="text-xs text-gray-400">of {total}</div>
        </div>
      </div>

      {/* Filters */}
      <form onSubmit={handleSearch} className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
        <div className="flex items-center gap-2 mb-3">
          <Filter className="w-4 h-4 text-gray-400" />
          <span className="text-sm font-semibold text-gray-700">Filter Audit Log</span>
        </div>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="text-xs font-medium text-gray-500 mb-1 block">Actor</label>
            <input
              id="audit-filter-actor"
              type="text"
              value={filterActor}
              onChange={e => setFilterActor(e.target.value)}
              placeholder="user ID or name"
              className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-500 mb-1 block">Action</label>
            <input
              id="audit-filter-action"
              type="text"
              value={filterAction}
              onChange={e => setFilterAction(e.target.value)}
              placeholder="e.g. events.replay"
              className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="text-xs font-medium text-gray-500 mb-1 block">Resource</label>
            <input
              id="audit-filter-resource"
              type="text"
              value={filterResource}
              onChange={e => setFilterResource(e.target.value)}
              placeholder="e.g. platform_events"
              className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>
        <div className="flex justify-end mt-3">
          <button
            id="audit-filter-search"
            type="submit"
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Search className="w-3.5 h-3.5" />
            Search
          </button>
        </div>
      </form>

      {/* Audit Table */}
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        {source === 'unavailable' ? (
          <div className="px-5 py-12 text-center">
            <AlertTriangle className="w-10 h-10 text-amber-400 mx-auto mb-3" />
            <div className="text-sm font-semibold text-gray-700">Audit Database Not Configured</div>
            <div className="text-xs text-gray-500 mt-1">
              Set <code className="bg-gray-100 px-1 rounded">STATGATE_DB_URL</code> in your environment to enable persistent audit logging.
            </div>
          </div>
        ) : loading ? (
          <div className="flex items-center justify-center h-48 text-gray-400">
            <RefreshCw className="w-5 h-5 animate-spin mr-2" /> Loading audit log…
          </div>
        ) : entries.length === 0 ? (
          <div className="px-5 py-12 text-center text-sm text-gray-400">
            No audit events match the current filters.
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-100">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Time</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Actor</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Action</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Resource</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Outcome</th>
                    <th className="px-4 py-3 w-8" />
                  </tr>
                </thead>
                <tbody>
                  {entries.map((entry, i) => (
                    <AuditRow key={`${entry.correlation_id}-${i}`} entry={entry} />
                  ))}
                </tbody>
              </table>
            </div>
            {/* Pagination */}
            {total > limit && (
              <div className="px-4 py-3 border-t border-gray-100 flex items-center justify-between">
                <span className="text-xs text-gray-400">
                  Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}
                </span>
                <div className="flex gap-2">
                  <button
                    id="audit-prev-page"
                    disabled={offset === 0}
                    onClick={() => { setOffset(Math.max(0, offset - limit)); fetchAuditLog(); }}
                    className="px-3 py-1 text-xs font-medium border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50"
                  >
                    ← Previous
                  </button>
                  <button
                    id="audit-next-page"
                    disabled={offset + limit >= total}
                    onClick={() => { setOffset(offset + limit); fetchAuditLog(); }}
                    className="px-3 py-1 text-xs font-medium border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50"
                  >
                    Next →
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
