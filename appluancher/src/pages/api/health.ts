import type { NextApiRequest, NextApiResponse } from 'next';

type HealthResponse = {
  status: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  uptime: number;
  message?: string;
};

const startTime = Date.now();

export default function handler(
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
    const uptime = Date.now() - startTime;

    res.status(200).json({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      uptime,
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
