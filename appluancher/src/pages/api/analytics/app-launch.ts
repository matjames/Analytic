import type { NextApiRequest, NextApiResponse } from 'next';

type AnalyticsResponse = {
  success: boolean;
  message?: string;
  error?: string;
};

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<AnalyticsResponse>
) {
  if (req.method !== 'POST') {
    return res.status(405).json({
      success: false,
      error: 'Method not allowed',
    });
  }

  try {
    const { appId, appName, timestamp, userAgent } = req.body;

    // Log analytics event (in production, send to analytics service)
    console.log('App Launch Event:', {
      appId,
      appName,
      timestamp,
      userAgent,
      serverTimestamp: new Date().toISOString(),
      ip: req.headers['x-forwarded-for'] || req.socket.remoteAddress,
    });

    // Send to external analytics service (example)
    // await sendToAnalytics({ appId, appName, timestamp, userAgent });

    res.status(200).json({
      success: true,
      message: 'Analytics event recorded',
    });
  } catch (error) {
    console.error('Error recording analytics:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to record event',
    });
  }
}
