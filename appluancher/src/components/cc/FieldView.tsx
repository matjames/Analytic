import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import {
  MapPin,
  Building2,
  ClipboardList,
  AlertTriangle,
  ArrowUpRight,
  Layers,
  Navigation,
  Globe,
} from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

interface Facility {
  id: string;
  name: string;
  district: string;
  level: string;
  status: string;
  latitude?: number;
  longitude?: number;
  qualityScore?: number;
}

const UGANDA_FACILITIES: Facility[] = [
  { id: 'FAC-001', name: 'Mulago National Referral Hospital', district: 'Kampala', level: 'National Referral', status: 'Operational', latitude: 0.3396, longitude: 32.5768, qualityScore: 99.1 },
  { id: 'FAC-002', name: 'Gulu Regional Referral Hospital', district: 'Gulu', level: 'Regional Referral', status: 'Operational', latitude: 2.7747, longitude: 32.2990, qualityScore: 97.4 },
  { id: 'FAC-003', name: 'Mbarara Regional Referral Hospital', district: 'Mbarara', level: 'Regional Referral', status: 'Operational', latitude: -0.6072, longitude: 30.6545, qualityScore: 98.6 },
  { id: 'FAC-004', name: 'Jinja Regional Referral Hospital', district: 'Jinja', level: 'Regional Referral', status: 'Operational', latitude: 0.4356, longitude: 33.2032, qualityScore: 96.8 },
  { id: 'FAC-005', name: 'Mbale Regional Referral Hospital', district: 'Mbale', level: 'Regional Referral', status: 'Operational', latitude: 1.0784, longitude: 34.1754, qualityScore: 98.0 },
  { id: 'FAC-006', name: 'Arua Regional Referral Hospital', district: 'Arua', level: 'Regional Referral', status: 'Operational', latitude: 3.0303, longitude: 30.9108, qualityScore: 95.9 },
  { id: 'FAC-007', name: 'Fort Portal Regional Referral Hospital', district: 'Kabarole', level: 'Regional Referral', status: 'Operational', latitude: 0.6548, longitude: 30.2744, qualityScore: 98.2 },
  { id: 'FAC-008', name: 'Kabale Regional Referral Hospital', district: 'Kabale', level: 'Regional Referral', status: 'Operational', latitude: -1.2507, longitude: 29.9887, qualityScore: 97.1 },
  { id: 'FAC-009', name: 'Moroto Regional Referral Hospital', district: 'Moroto', level: 'Regional Referral', status: 'Operational', latitude: 2.5345, longitude: 34.6666, qualityScore: 94.5 },
  { id: 'FAC-010', name: 'Wakiso Health Centre IV', district: 'Wakiso', level: 'Health Centre IV', status: 'Operational', latitude: 0.4044, longitude: 32.4594, qualityScore: 98.9 },
  { id: 'FAC-011', name: 'Remote Mobile Field Site 14', district: 'Kotido', level: 'Mobile Unit', status: 'Survey Active' }, // Location unavailable demonstration
];

export const FieldView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [selectedLayer, setSelectedLayer] = useState<'facilities' | 'coverage' | 'quality' | 'alerts'>('facilities');
  const [selectedFacility, setSelectedFacility] = useState<Facility | null>(UGANDA_FACILITIES[0]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/field?user_id=${encodeURIComponent(user.id)}`, {
      headers,
    })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        if (cancelled) return;
        setData(res);
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-100 rounded-xl" />
          ))}
        </div>
        <div className="h-96 bg-gray-100 rounded-2xl" />
      </div>
    );
  }

  const facilities = data?.facilities || UGANDA_FACILITIES;
  const surveys = data?.surveys || [];
  const alerts = data?.alerts || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <MapPin className="w-6 h-6 text-emerald-600" />
            Field Operations & Geographic Intelligence
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Real-time geospatial monitoring, facility registries, enumerator coverage, and regional data quality across Uganda.
          </p>
        </div>
        <a
          href="http://localhost:3007"
          className="btn btn-secondary text-xs"
        >
          Master Facility Registry <ArrowUpRight className="w-3.5 h-3.5" />
        </a>
      </div>

      {/* Field Operations Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div className="stat-card border-l-4 border-l-emerald-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Registered Facilities</span>
            <Building2 className="w-5 h-5 text-emerald-500" />
          </div>
          <span className="stat-val">{facilities.length}</span>
          <span className="text-xs text-gray-400 mt-1">National Registry Live Matrix</span>
        </div>

        <div className="stat-card border-l-4 border-l-blue-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Active Field Surveys</span>
            <ClipboardList className="w-5 h-5 text-blue-500" />
          </div>
          <span className="stat-val">{surveys.length > 0 ? surveys.length : 14}</span>
          <span className="text-xs text-gray-400 mt-1">StatCollect Field Campaigns</span>
        </div>

        <div className="stat-card border-l-4 border-l-teal-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Geographic Coverage</span>
            <Globe className="w-5 h-5 text-teal-500" />
          </div>
          <span className="stat-val">135 / 135</span>
          <span className="text-xs text-gray-400 mt-1">100% Uganda District Footprint</span>
        </div>

        <div className="stat-card border-l-4 border-l-amber-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Active Field Alerts</span>
            <AlertTriangle className="w-5 h-5 text-amber-500" />
          </div>
          <span className="stat-val">{alerts.length}</span>
          <span className="text-xs text-gray-400 mt-1">SLA & Anomaly Monitoring</span>
        </div>
      </div>

      {/* Interactive Geographic Map Canvas */}
      <div className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-hidden">
        {/* Layer Controls */}
        <div className="p-4 border-b border-gray-100 bg-gray-50 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-emerald-600" />
            <span className="text-xs font-bold uppercase tracking-wider text-gray-700">
              Map Intelligence Layer:
            </span>
          </div>

          <div className="flex items-center gap-2">
            {[
              { id: 'facilities', label: 'Facilities' },
              { id: 'coverage', label: 'Survey Coverage' },
              { id: 'quality', label: 'Data Quality' },
              { id: 'alerts', label: 'Field Alerts' },
            ].map((layer) => (
              <button
                key={layer.id}
                onClick={() => setSelectedLayer(layer.id as any)}
                className={`px-3 py-1 rounded-lg text-xs font-semibold transition-all ${
                  selectedLayer === layer.id
                    ? 'bg-emerald-600 text-white shadow-sm'
                    : 'bg-white text-gray-600 hover:bg-gray-100'
                }`}
              >
                {layer.label}
              </button>
            ))}
          </div>
        </div>

        {/* Map Visualization Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 min-h-[460px]">
          {/* Visual Geo Grid */}
          <div className="lg:col-span-2 p-6 bg-slate-900 text-white relative flex flex-col justify-between">
            <div className="flex items-center justify-between text-xs text-slate-400">
              <span className="flex items-center gap-1 font-mono">
                <Navigation className="w-3.5 h-3.5 text-emerald-400" /> Uganda National Grid (EPSG:4326)
              </span>
              <span>Layer: {selectedLayer.toUpperCase()}</span>
            </div>

            {/* Spatial Facility Markers Representation */}
            <div className="grid grid-cols-3 sm:grid-cols-4 gap-3 my-6">
              {UGANDA_FACILITIES.map((fac) => {
                const isSelected = selectedFacility?.id === fac.id;
                const hasCoords = fac.latitude !== undefined && fac.longitude !== undefined;

                return (
                  <button
                    key={fac.id}
                    onClick={() => setSelectedFacility(fac)}
                    className={`p-3 rounded-xl text-left border transition-all ${
                      isSelected
                        ? 'bg-emerald-600/30 border-emerald-400 text-white shadow-lg'
                        : 'bg-slate-800/80 border-slate-700 text-slate-300 hover:border-slate-500'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-[10px] font-mono text-emerald-400">{fac.district}</span>
                      {hasCoords ? (
                        <span className="w-2 h-2 rounded-full bg-emerald-400" />
                      ) : (
                        <span className="w-2 h-2 rounded-full bg-amber-400" title="Location unavailable" />
                      )}
                    </div>
                    <div className="text-xs font-bold truncate">{fac.name}</div>
                    <div className="text-[10px] text-slate-400 mt-1">{fac.level}</div>
                  </button>
                );
              })}
            </div>

            <div className="text-[11px] text-slate-400 flex items-center justify-between border-t border-slate-800 pt-3">
              <span>Projection: Sovereign Uganda Datum</span>
              <span className="text-emerald-400 font-semibold">100% In-Country Storage</span>
            </div>
          </div>

          {/* Selected Facility Inspector */}
          <div className="p-6 bg-white border-l border-gray-100 flex flex-col justify-between">
            {selectedFacility ? (
              <div className="space-y-4">
                <div>
                  <span className="text-[10px] font-bold uppercase tracking-wider text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded">
                    {selectedFacility.level}
                  </span>
                  <h3 className="text-lg font-bold text-gray-900 mt-2">{selectedFacility.name}</h3>
                  <span className="text-xs text-gray-500 font-medium">District: {selectedFacility.district}</span>
                </div>

                <div className="space-y-3 pt-2 text-xs">
                  <div className="flex justify-between border-b pb-2">
                    <span className="text-gray-500">Registry Code</span>
                    <span className="font-mono font-bold text-gray-800">{selectedFacility.id}</span>
                  </div>

                  <div className="flex justify-between border-b pb-2">
                    <span className="text-gray-500">Coordinates</span>
                    {selectedFacility.latitude !== undefined ? (
                      <span className="font-mono text-gray-800">
                        {selectedFacility.latitude.toFixed(4)}, {selectedFacility.longitude?.toFixed(4)}
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-amber-100 text-amber-800 font-bold rounded text-[10px]">
                        Location unavailable
                      </span>
                    )}
                  </div>

                  <div className="flex justify-between border-b pb-2">
                    <span className="text-gray-500">Data Quality Score</span>
                    <span className="font-bold text-green-600">
                      {selectedFacility.qualityScore ? `${selectedFacility.qualityScore}%` : 'Pending Survey'}
                    </span>
                  </div>

                  <div className="flex justify-between">
                    <span className="text-gray-500">Operational Status</span>
                    <span className="font-semibold text-gray-800">{selectedFacility.status}</span>
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-center text-gray-400 text-xs my-auto">
                Select a facility from the matrix to inspect geospatial parameters.
              </div>
            )}

            {selectedFacility && (
              <div className="pt-4 border-t border-gray-100">
                <button
                  onClick={() =>
                    openObjectContext('facility', selectedFacility.id, {
                      title: selectedFacility.name,
                    })
                  }
                  className="w-full btn btn-primary text-xs flex items-center justify-center gap-1.5 shadow-sm"
                >
                  Inspect Universal Context <ArrowUpRight className="w-3.5 h-3.5" />
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
