import type { NextApiRequest, NextApiResponse } from 'next';
import { Application } from '@typings/index';
import { MOCK_APPLICATIONS } from '@utils/mockData';

type ResponseData = {
  success: boolean;
  data?: Application[];
  error?: string;
  message?: string;
};

/**
 * GET /api/applications
 * Fetch all available applications
 * 
 * Query Parameters:
 *   - category: Filter by category (analytics, database, reporting, admin, tools, integration)
 *   - role: Filter by required role
 */
export default function handler(
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
    let applications = [...MOCK_APPLICATIONS];

    // Filter by category if provided
    if (req.query.category) {
      applications = applications.filter(
        (app) => app.category === req.query.category
      );
    }

    // Filter by role if provided
    if (req.query.role) {
      applications = applications.filter((app) =>
        app.requiredRoles.includes(req.query.role as any)
      );
    }

    res.status(200).json({
      success: true,
      data: applications,
      message: `Retrieved ${applications.length} applications`,
    });
  } catch (error) {
    console.error('Error fetching applications:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to fetch applications',
    });
  }
}
