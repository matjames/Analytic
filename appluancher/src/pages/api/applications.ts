import type { NextApiRequest, NextApiResponse } from 'next';
import { Application, AppCategory, AppStatus, UserRole } from '@typings/index';
import { MOCK_APPLICATIONS } from '@utils/mockData';

type ResponseData = {
  success: boolean;
  data?: Application[];
  error?: string;
  message?: string;
};

// The Analytics Flask service exposes /api/services and /api/services/health.
// When it's reachable, we dynamically fetch the live service list; otherwise
// we fall back to MOCK_APPLICATIONS so the launcher still works offline.
const FLASK_ANALYTICS_URL =
  process.env.FLASK_ANALYTICS_URL ?? 'http://localhost:5000';

interface LiveService {
  id: string;
  name: string;
  ui?: string | null;
}

/**
 * GET /api/applications
 * Fetch all available applications dynamically from StatGate Analytics
 * (which maintains the authoritative service registry), falling back to
 * bundled mock data when the service is unreachable.
 *
 * Query Parameters:
 *   - category: Filter by category (analytics, database, reporting, admin, tools, integration)
 *   - role: Filter by required role
 */
export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ResponseData>
) {
  if (req.method !== 'GET') {
    return res.status(405).json({
      success: false,
      error: 'Method not allowed',
    });
  }

  try {
    // ── Dynamic service discovery ─────────────────────────────
    let applications: Application[] = [];
    let source: 'live' | 'mock' = 'mock';

    try {
      const serviceRes = await fetch(`${FLASK_ANALYTICS_URL}/api/services`, {
        signal: AbortSignal.timeout(4000),
      });
      if (serviceRes.ok) {
        const { services }: { services: LiveService[] } =
          await serviceRes.json();
        const knownApps = [...MOCK_APPLICATIONS];
        applications = (services ?? [])
          .filter((s) => s.ui)
          .map((svc) => {
            // Preserve icon/category/roles from mock defaults where possible,
            // but always use the live service name, URL, and ID.
            const template = knownApps.find(
              (a) => a.url === svc.ui || a.id === svc.id
            );
            const uiUrl = svc.ui ?? '';
            return {
              ...(template ?? {
                id: svc.id,
                name: svc.name,
                description: svc.name,
                icon: '🧩',
                category: AppCategory.TOOLS,
                status: AppStatus.OPERATIONAL,
                requiredRoles: [] as UserRole[],
                version: '1.0.0',
                lastUpdated: new Date(),
              }),
              id: svc.id,
              name: svc.name,
              url: uiUrl,
              documentation: uiUrl,
              healthEndpoint: uiUrl,
            };
          });
        if (applications.length > 0) {
          source = 'live';
        }
      }
    } catch {
      // Analytics unreachable — fall through to mock data.
    }

    if (source === 'mock') {
      applications = [...MOCK_APPLICATIONS];
    }

    // ── Filters ───────────────────────────────────────────────
    if (req.query.category) {
      applications = applications.filter(
        (app) => app.category === req.query.category
      );
    }

    if (req.query.role) {
      applications = applications.filter((app) =>
        app.requiredRoles.includes(req.query.role as any)
      );
    }

    res.status(200).json({
      success: true,
      data: applications,
      message: `Retrieved ${applications.length} applications (source: ${source})`,
    });
  } catch (error) {
    console.error('Error fetching applications:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to fetch applications',
    });
  }
}