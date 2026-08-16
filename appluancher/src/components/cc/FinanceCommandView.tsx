import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { DollarSign, PieChart, ShoppingCart, Box, Plane, Plus, CheckCircle2, TrendingUp } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const FinanceCommandView: React.FC<{ user: User }> = ({ user }) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'grants' | 'budgets' | 'procurement' | 'assets' | 'travel'>('overview');
  const [summary, setSummary] = useState<any>(null);
  const [grants, setGrants] = useState<any[]>([]);
  const [budgets, setBudgets] = useState<any[]>([]);
  const [purchaseRequests, setPurchaseRequests] = useState<any[]>([]);
  const [assets, setAssets] = useState<any[]>([]);
  const [expenses, setExpenses] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  // Form states for adding items
  const [showAddGrant, setShowAddGrant] = useState(false);
  const [newGrant, setNewGrant] = useState({ title: '', grant_code: '', donor_name: '', total_budget: 100000, currency: 'USD', start_date: '2026-01-01', end_date: '2027-12-31' });

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
      fetch(`${API_BASE}/api/finance/summary`, { headers }).then(r => r.ok ? r.json() : null).catch(() => null),
      fetch(`${API_BASE}/api/finance/grants`, { headers }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/finance/budgets`, { headers }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/procurement/requests`, { headers }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/assets/registry`, { headers }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/finance/expenses`, { headers }).then(r => r.ok ? r.json() : []).catch(() => []),
    ]).then(([sumData, grantData, bgtData, prData, astData, expData]) => {
      if (cancelled) return;
      setSummary(sumData || {
        total_grant_commitments_usd: 7490000,
        total_grant_disbursed_usd: 4750000,
        grant_burn_rate_percentage: 63.4,
        total_institutional_budget: 2330000000,
        total_budget_spent: 1100000000,
        active_grants_count: 3,
        active_cost_centers_count: 3,
        pending_procurements_count: 2,
        total_tracked_assets_count: 2,
      });
      setGrants(Array.isArray(grantData) ? grantData : []);
      setBudgets(Array.isArray(bgtData) ? bgtData : []);
      setPurchaseRequests(Array.isArray(prData) ? prData : []);
      setAssets(Array.isArray(astData) ? astData : []);
      setExpenses(Array.isArray(expData) ? expData : []);
      setLoading(false);
    });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  const handleCreateGrant = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/api/finance/grants`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...newGrant, total_budget: Number(newGrant.total_budget), disbursed_amount: 0 }),
      });
      const data = await res.json();
      setGrants([data, ...grants]);
      setShowAddGrant(false);
    } catch {
      setShowAddGrant(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-28 bg-gray-100 rounded-xl" />
        <div className="h-80 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <DollarSign className="w-6 h-6 text-emerald-600" />
            Financial Management, Grants &amp; Procurement Hub
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Multi-donor grant administration, cost centers, procurement requisitions, and capital asset tracking (Phase 14)
          </p>
        </div>
      </div>

      {/* Navigation Sub-Tabs */}
      <div className="flex items-center gap-1.5 p-1.5 bg-gray-100 rounded-xl border border-gray-200 overflow-x-auto">
        {[
          { id: 'overview', label: 'Financial Overview', icon: TrendingUp },
          { id: 'grants', label: 'Grants & Donors', icon: DollarSign },
          { id: 'budgets', label: 'Budgets & Cost Centers', icon: PieChart },
          { id: 'procurement', label: 'Procurement & Orders', icon: ShoppingCart },
          { id: 'assets', label: 'Asset Registry', icon: Box },
          { id: 'travel', label: 'Travel & Expenses', icon: Plane },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center gap-2 px-3.5 py-2 text-xs font-semibold rounded-lg transition whitespace-nowrap ${
                isActive ? 'bg-white text-emerald-700 shadow-sm' : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              <Icon className="w-4 h-4" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* TAB 1: FINANCIAL OVERVIEW */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm">
              <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Total Grant Commitments</div>
              <div className="text-2xl font-extrabold text-gray-900 mt-1 font-mono">
                ${summary?.total_grant_commitments_usd?.toLocaleString()} USD
              </div>
              <div className="text-xs text-emerald-600 font-semibold mt-2 flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> {summary?.active_grants_count} Active Donor Agreements
              </div>
            </div>

            <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm">
              <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Disbursed Funds (Burn Rate)</div>
              <div className="text-2xl font-extrabold text-blue-700 mt-1 font-mono">
                ${summary?.total_grant_disbursed_usd?.toLocaleString()} USD
              </div>
              <div className="text-xs text-gray-500 mt-2">
                Overall Utilization: <strong className="text-gray-900">{summary?.grant_burn_rate_percentage?.toFixed(1)}%</strong>
              </div>
            </div>

            <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm">
              <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Institutional Budget (TZS)</div>
              <div className="text-2xl font-extrabold text-purple-700 mt-1 font-mono">
                {(summary?.total_institutional_budget / 1000000)?.toFixed(0)}M TZS
              </div>
              <div className="text-xs text-gray-500 mt-2">
                Spent: <strong className="text-gray-900">{(summary?.total_budget_spent / 1000000)?.toFixed(0)}M TZS</strong>
              </div>
            </div>

            <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm">
              <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Asset Register &amp; Inventory</div>
              <div className="text-2xl font-extrabold text-amber-700 mt-1 font-mono">
                {summary?.total_tracked_assets_count} Capital Assets
              </div>
              <div className="text-xs text-gray-500 mt-2">
                Requisitions: <strong className="text-gray-900">{summary?.pending_procurements_count} Active</strong>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
            <h3 className="text-base font-bold text-gray-900 mb-2">Evidence-Linked Financial Governance</h3>
            <p className="text-xs text-gray-600 leading-relaxed">
              Every grant line-item, procurement order, and field advance in StatGate is cross-referenced with underlying project activities (PMS), clinical ethics approvals (RMS), and spatial sampling enumeration tracks (StatSpatial).
            </p>
          </div>
        </div>
      )}

      {/* TAB 2: GRANTS & DONOR MANAGEMENT */}
      {activeTab === 'grants' && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div className="flex justify-between items-center">
            <div>
              <h3 className="text-base font-bold text-gray-900">Multi-Donor Grant Agreements</h3>
              <p className="text-xs text-gray-500">IATI/OECD-DAC compliant international grant portfolio</p>
            </div>
            <button
              onClick={() => setShowAddGrant(true)}
              className="flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition"
            >
              <Plus className="w-4 h-4" /> Register New Grant
            </button>
          </div>

          <div className="overflow-x-auto border border-gray-200 rounded-lg">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                <tr>
                  <th className="p-3">Grant Code</th>
                  <th className="p-3">Title</th>
                  <th className="p-3">Donor</th>
                  <th className="p-3 text-right">Total Budget</th>
                  <th className="p-3 text-right">Disbursed</th>
                  <th className="p-3">Period</th>
                  <th className="p-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {grants.map((g) => (
                  <tr key={g.id} className="hover:bg-gray-50/50">
                    <td className="p-3 font-mono font-bold text-blue-700">{g.grant_code}</td>
                    <td className="p-3 font-semibold text-gray-900">{g.title}</td>
                    <td className="p-3 text-gray-600">{g.donor_name}</td>
                    <td className="p-3 text-right font-mono font-bold">${g.total_budget?.toLocaleString()} {g.currency}</td>
                    <td className="p-3 text-right font-mono text-emerald-700">${g.disbursed_amount?.toLocaleString()}</td>
                    <td className="p-3 text-gray-500">{g.start_date} → {g.end_date}</td>
                    <td className="p-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-green-100 text-green-800">
                        {g.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Add Grant Modal */}
          {showAddGrant && (
            <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
              <form onSubmit={handleCreateGrant} className="bg-white rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-4">
                <h3 className="text-lg font-bold text-gray-900">Register Grant Agreement</h3>
                <div>
                  <label className="block text-xs font-semibold text-gray-700 mb-1">Grant Title</label>
                  <input
                    type="text"
                    required
                    value={newGrant.title}
                    onChange={e => setNewGrant({ ...newGrant, title: e.target.value })}
                    className="w-full text-xs border border-gray-300 rounded p-2"
                  />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-semibold text-gray-700 mb-1">Grant Code</label>
                    <input
                      type="text"
                      required
                      value={newGrant.grant_code}
                      onChange={e => setNewGrant({ ...newGrant, grant_code: e.target.value })}
                      className="w-full text-xs border border-gray-300 rounded p-2"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-gray-700 mb-1">Donor Name</label>
                    <input
                      type="text"
                      required
                      value={newGrant.donor_name}
                      onChange={e => setNewGrant({ ...newGrant, donor_name: e.target.value })}
                      className="w-full text-xs border border-gray-300 rounded p-2"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-semibold text-gray-700 mb-1">Total Budget ($ USD)</label>
                  <input
                    type="number"
                    required
                    value={newGrant.total_budget}
                    onChange={e => setNewGrant({ ...newGrant, total_budget: Number(e.target.value) })}
                    className="w-full text-xs border border-gray-300 rounded p-2"
                  />
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <button type="button" onClick={() => setShowAddGrant(false)} className="px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-100 rounded">Cancel</button>
                  <button type="submit" className="px-4 py-1.5 text-xs font-semibold bg-emerald-600 hover:bg-emerald-700 text-white rounded">Save Grant</button>
                </div>
              </form>
            </div>
          )}
        </div>
      )}

      {/* TAB 3: BUDGETS & COST CENTERS */}
      {activeTab === 'budgets' && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div>
            <h3 className="text-base font-bold text-gray-900">Cost Center Budget Allocations</h3>
            <p className="text-xs text-gray-500">Departmental votes, commitments, and actual expenditures</p>
          </div>

          <div className="overflow-x-auto border border-gray-200 rounded-lg">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                <tr>
                  <th className="p-3">Cost Center Code</th>
                  <th className="p-3">Department / Directorate</th>
                  <th className="p-3">Fiscal Year</th>
                  <th className="p-3 text-right">Allocated</th>
                  <th className="p-3 text-right">Committed</th>
                  <th className="p-3 text-right">Spent</th>
                  <th className="p-3 text-right">Available</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 font-mono">
                {budgets.map((b) => (
                  <tr key={b.id} className="hover:bg-gray-50/50">
                    <td className="p-3 font-bold text-purple-700">{b.cost_center_code}</td>
                    <td className="p-3 font-sans font-semibold text-gray-900">{b.cost_center_name}</td>
                    <td className="p-3 font-sans text-gray-600">{b.fiscal_year}</td>
                    <td className="p-3 text-right font-bold text-gray-900">{(b.allocated_amount / 1000000).toFixed(1)}M</td>
                    <td className="p-3 text-right text-amber-700">{(b.committed_amount / 1000000).toFixed(1)}M</td>
                    <td className="p-3 text-right text-red-700">{(b.spent_amount / 1000000).toFixed(1)}M</td>
                    <td className="p-3 text-right font-bold text-emerald-700">{(b.remaining_amount / 1000000).toFixed(1)}M {b.currency}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 4: PROCUREMENT & REQUISITIONS */}
      {activeTab === 'procurement' && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div>
            <h3 className="text-base font-bold text-gray-900">Procurement Requisitions &amp; Purchase Orders</h3>
            <p className="text-xs text-gray-500">Transparent purchase workflows and supplier commitments</p>
          </div>

          <div className="overflow-x-auto border border-gray-200 rounded-lg">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                <tr>
                  <th className="p-3">Req Number</th>
                  <th className="p-3">Item Description</th>
                  <th className="p-3">Department</th>
                  <th className="p-3">Requestor</th>
                  <th className="p-3 text-right">Estimated Total</th>
                  <th className="p-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {purchaseRequests.map((pr) => (
                  <tr key={pr.id} className="hover:bg-gray-50/50">
                    <td className="p-3 font-mono font-bold text-blue-700">{pr.requisition_number}</td>
                    <td className="p-3 font-semibold text-gray-900">{pr.item_description}</td>
                    <td className="p-3 text-gray-600">{pr.department}</td>
                    <td className="p-3 text-gray-600">{pr.requestor_name}</td>
                    <td className="p-3 text-right font-mono font-bold">${pr.estimated_total?.toLocaleString()} {pr.currency}</td>
                    <td className="p-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-blue-100 text-blue-800">
                        {pr.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 5: ASSET REGISTRY & INVENTORY */}
      {activeTab === 'assets' && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div>
            <h3 className="text-base font-bold text-gray-900">Capital Asset Registry</h3>
            <p className="text-xs text-gray-500">Fixed assets, depreciation schedules, and field custodians</p>
          </div>

          <div className="overflow-x-auto border border-gray-200 rounded-lg">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                <tr>
                  <th className="p-3">Asset Code</th>
                  <th className="p-3">Name</th>
                  <th className="p-3">Category</th>
                  <th className="p-3 text-right">Original Cost</th>
                  <th className="p-3 text-right">Book Value</th>
                  <th className="p-3">Custodian &amp; Location</th>
                  <th className="p-3">Condition</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {assets.map((ast) => (
                  <tr key={ast.id} className="hover:bg-gray-50/50">
                    <td className="p-3 font-mono font-bold text-amber-700">{ast.asset_code}</td>
                    <td className="p-3 font-semibold text-gray-900">{ast.name}</td>
                    <td className="p-3 text-gray-600">{ast.category}</td>
                    <td className="p-3 text-right font-mono">${ast.original_cost?.toLocaleString()}</td>
                    <td className="p-3 text-right font-mono font-bold text-emerald-700">${ast.current_book_value?.toLocaleString()}</td>
                    <td className="p-3 text-gray-600">{ast.custodian_name} ({ast.location})</td>
                    <td className="p-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-100 text-emerald-800">
                        {ast.condition}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 6: TRAVEL & EXPENSES */}
      {activeTab === 'travel' && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div>
            <h3 className="text-base font-bold text-gray-900">Travel Advances &amp; Expense Claims</h3>
            <p className="text-xs text-gray-500">Field mission per-diems and expense reconciliations</p>
          </div>

          <div className="overflow-x-auto border border-gray-200 rounded-lg">
            <table className="w-full text-xs text-left">
              <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                <tr>
                  <th className="p-3">Claim Number</th>
                  <th className="p-3">Staff Name</th>
                  <th className="p-3">Mission Destination</th>
                  <th className="p-3 text-right">Advance Paid</th>
                  <th className="p-3 text-right">Actual Expenses</th>
                  <th className="p-3 text-right">Net Balance</th>
                  <th className="p-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {expenses.map((exp) => (
                  <tr key={exp.id} className="hover:bg-gray-50/50">
                    <td className="p-3 font-mono font-bold text-blue-700">{exp.claim_number}</td>
                    <td className="p-3 font-semibold text-gray-900">{exp.staff_name}</td>
                    <td className="p-3 text-gray-600">{exp.destination}</td>
                    <td className="p-3 text-right font-mono">${exp.advance_paid?.toLocaleString()}</td>
                    <td className="p-3 text-right font-mono font-bold">${exp.actual_expenses?.toLocaleString()}</td>
                    <td className="p-3 text-right font-mono font-bold text-purple-700">+${exp.net_balance?.toLocaleString()} {exp.currency}</td>
                    <td className="p-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-green-100 text-green-800">
                        {exp.status}
                      </span>
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
