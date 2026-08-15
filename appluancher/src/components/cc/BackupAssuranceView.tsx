import React, { useEffect, useState, useCallback } from 'react';
import {
  Database,
  CheckCircle2,
  RefreshCw,
  Key,
  RotateCcw,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface BackupRecord {
  id: string;
  service_id: string;
  service_name: string;
  backup_type: string;
  data_scope: string;
  storage_location: string;
  encryption_status: string;
  retention_policy: string;
  size_bytes: number;
  checksum_sha256: string;
  status: string;
  last_restore_test?: string;
  restore_verified: boolean;
  certified_by?: string;
  created_at: string;
}

export const BackupAssuranceView: React.FC = () => {
  const [backups, setBackups] = useState<BackupRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchBackups = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/backups`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setBackups(data.backups ?? []);
      }
    } catch (e) {
      console.error('Failed to fetch backups:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchBackups();
  }, [fetchBackups]);

  const handleVerify = async (backupId: string) => {
    setActionLoading(backupId);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      await fetch(`${ENTERPRISE_CORE}/api/backups/${backupId}/verify`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      await fetchBackups();
    } catch (e) {
      console.error('Verify failed:', e);
    } finally {
      setActionLoading(null);
    }
  };

  const handleRestoreTest = async (backupId: string) => {
    setActionLoading(backupId);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      await fetch(`${ENTERPRISE_CORE}/api/backups/${backupId}/restore-test`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      await fetchBackups();
    } catch (e) {
      console.error('Restore test failed:', e);
    } finally {
      setActionLoading(null);
    }
  };

  function formatBytes(bytes: number) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-bold bg-purple-100 text-purple-800 rounded-full">
              Non-Repudiation
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <Database className="w-6 h-6 text-purple-600" />
              Backup & Restore Assurance Registry
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Proof of recoverability: &ldquo;A backup that has never been restored and verified is not considered compliant.&rdquo;
          </p>
        </div>
        <button
          onClick={fetchBackups}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Verification Workflow Banner */}
      <div className="bg-purple-950 text-white rounded-xl p-5 shadow-lg border border-purple-900">
        <div className="text-xs font-bold uppercase tracking-widest text-purple-300 mb-3">
          Institutional 6-Stage Recovery Certification Workflow
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 text-center text-xs">
          {[
            { step: '1', title: 'BACKUP CREATED', color: 'bg-white/10 text-white' },
            { step: '2', title: 'CHECKSUM VERIFIED', color: 'bg-white/10 text-white' },
            { step: '3', title: 'RESTORE EXECUTED', color: 'bg-white/10 text-white' },
            { step: '4', title: 'INTEGRITY CHECK', color: 'bg-white/10 text-white' },
            { step: '5', title: 'SERVICE VALIDATED', color: 'bg-white/10 text-white' },
            { step: '6', title: 'RECOVERY CERTIFIED', color: 'bg-emerald-500 text-white font-bold' },
          ].map((s) => (
            <div key={s.step} className={`p-2.5 rounded-lg ${s.color} border border-white/10`}>
              <div className="text-[10px] opacity-70">Step {s.step}</div>
              <div className="font-bold text-[11px] mt-0.5">{s.title}</div>
            </div>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          Loading backup assurance registry…
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-5 py-4 border-b border-gray-100 font-bold text-sm text-gray-800">
            Governed Backup Inventory ({backups.length})
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-gray-50 border-b border-gray-100 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                <tr>
                  <th className="px-5 py-3">Service & Scope</th>
                  <th className="px-5 py-3">Type & Size</th>
                  <th className="px-5 py-3">Encryption & Storage</th>
                  <th className="px-5 py-3">Checksum SHA-256</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {backups.map((b) => (
                  <tr key={b.id} className="hover:bg-gray-50/80 transition-colors">
                    <td className="px-5 py-4">
                      <div className="font-bold text-gray-900">{b.service_name || b.service_id}</div>
                      <div className="text-xs text-gray-500 mt-0.5">{b.data_scope}</div>
                      <div className="text-[10px] font-mono text-gray-400 mt-1">ID: {b.id}</div>
                    </td>
                    <td className="px-5 py-4 text-xs font-mono">
                      <div className="font-bold text-purple-700">{b.backup_type}</div>
                      <div className="text-gray-500 mt-0.5">{formatBytes(b.size_bytes)}</div>
                    </td>
                    <td className="px-5 py-4 text-xs">
                      <div className="flex items-center gap-1 font-semibold text-gray-700">
                        <Key className="w-3 h-3 text-emerald-600" />
                        {b.encryption_status}
                      </div>
                      <div className="text-[11px] text-gray-400 font-mono mt-0.5 truncate max-w-[200px]">
                        {b.storage_location}
                      </div>
                    </td>
                    <td className="px-5 py-4 font-mono text-[11px] text-gray-500 truncate max-w-[150px]">
                      {b.checksum_sha256}
                    </td>
                    <td className="px-5 py-4">
                      <span
                        className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-bold border ${
                          b.status === 'RESTORE_TESTED'
                            ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                            : b.status === 'VERIFIED'
                            ? 'bg-purple-50 text-purple-700 border-purple-200'
                            : 'bg-gray-50 text-gray-700 border-gray-200'
                        }`}
                      >
                        <CheckCircle2 className="w-3 h-3" />
                        {b.status}
                      </span>
                      {b.certified_by && (
                        <div className="text-[10px] text-gray-400 mt-1">By: {b.certified_by}</div>
                      )}
                    </td>
                    <td className="px-5 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          disabled={actionLoading === b.id}
                          onClick={() => handleVerify(b.id)}
                          className="px-2.5 py-1 text-xs font-semibold bg-gray-100 text-gray-700 hover:bg-gray-200 rounded-lg transition-colors disabled:opacity-40"
                        >
                          Verify
                        </button>
                        <button
                          disabled={actionLoading === b.id}
                          onClick={() => handleRestoreTest(b.id)}
                          className="px-2.5 py-1 text-xs font-semibold bg-purple-600 text-white hover:bg-purple-700 rounded-lg shadow-sm transition-colors disabled:opacity-40 flex items-center gap-1"
                        >
                          <RotateCcw className="w-3 h-3" />
                          Restore Test
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};
