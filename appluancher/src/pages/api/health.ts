import type { NextApiRequest, NextApiResponse } from 'next';

type HealthResponse = {
  status: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  uptime: number;
  message?: string;
  services?: Array<{ id: string; name: string; status: string; detail?: string | null }>;
  summary?: { total: number; healthy: number; degraded: number; down: number };
  source?: 'launcher' | 'platform';
};

const startTime = Date.now();

// Analytics Flask aggregates the authoritative platform service registry at
// /api/services/health (single source = frontend/config/services.json).
const FLASK_ANALYTICS_URL =
  process.env.FLASK_ANALYTICS_URL ?? 'http://localhost:5000';

function platformStatus(services: HealthResponse['services'] = []): HealthResponse {
  let healthy = 0;
  let degraded = 0;
  let down = 0;
  for (const s of services) {
    const st = String(s?.status ?? '');
    if (st === 'ok' || st === 'healthy') healthy += 1;
    else if (st === 'down' || st === 'unreachable') down += 1;
    else degraded += 1;
  }
  const status = down > 0 ? 'unhealthy' : degraded > 0 ? 'degraded' : 'healthy';
  return {
    status,
    timestamp: new Date().toISOString(),
    uptime: Date.now() - startTime,
    summary: { total: services.length, healthy, degraded, down },
    source: 'platform',
  };
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<HealthResponse>
) {
  if (req.method !== 'GET') {
    return res.status(405).json({
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      uptime: Date.now() - startTime,
      message: 'Method not allowed',
    });
  }

  try {
    // Attempt to surface the aggregate platform health from Analytics.
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 4000);
    try {
      const r = await fetch(`${FLASK_ANALYTICS_URL}/api/services/health`, {
        signal: ctrl.signal,
      });
      clearTimeout(timer);
      if (r.ok) {
        const data = await r.json();
        // Accepted response shapes: {services:[{id,name,status,detail}]} or [{id,name,status}]
        const raw = Array.isArray(data) ? data : data?.services;
        if (Array.isArray(raw)) {
          const services = raw.map((s: any) => ({
            id: s?.id,
            name: s?.name ?? s?.id,
            status: s?.status ?? 'unknown',
            detail: s?.detail ?? null,
          }));
          const resp = platformStatus(services);
          resp.services = services;
          resp.source = 'platform';
          return res.status(200).json(resp);
        }
      }
    } catch {
      // Analytics unreachable — fall through to launcher-level status.
    } finally {
      clearTimeout(timer);
    }

    // Degraded live check with full detail.
    const uptime = Date.now() - startTime;
    return res.status(200).json({
      status: 'degraded',
      timestamp: new Date().toISOString(),
      uptime,
      message: 'Launcher healthy; platform health source (Analytics) unreachable',
      source: 'launcher',
    });
  } catch (error) {
    res.status(503).json({
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      uptime: Date.now() - startTime,
      message: 'Service unavailable',
    });
  }
}
