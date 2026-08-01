import React, { useState } from 'react';
import { Application, AppStatus } from '@typings/index';
import {
  Zap,
  AlertTriangle,
  Wrench,
  BarChart3,
  Lock,
  Clock,
  ExternalLink,
} from 'lucide-react';
import { getStatusColor, getTimeSinceUpdate } from '@utils/helpers';

interface AppCardProps {
  app: Application;
  onLaunch?: (app: Application) => void;
  canAccess?: boolean;
}

export const AppCard: React.FC<AppCardProps> = ({
  app,
  onLaunch,
  canAccess = true,
}) => {
  const [hovering, setHovering] = useState(false);

  const statusIcon = {
    [AppStatus.OPERATIONAL]: <Zap className="w-3 h-3" />,
    [AppStatus.DEGRADED]: <AlertTriangle className="w-3 h-3" />,
    [AppStatus.MAINTENANCE]: <Wrench className="w-3 h-3" />,
    [AppStatus.OFFLINE]: <Lock className="w-3 h-3" />,
  };

  const statusLabel = {
    [AppStatus.OPERATIONAL]: 'Operational',
    [AppStatus.DEGRADED]: 'Degraded',
    [AppStatus.MAINTENANCE]: 'Maintenance',
    [AppStatus.OFFLINE]: 'Offline',
  };

  const handleLaunch = () => {
    if (canAccess && onLaunch) {
      onLaunch(app);
    }
  };

  return (
    <div
      className={`relative bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-all duration-200 overflow-hidden group ${
        !canAccess ? 'opacity-60' : ''
      } ${hovering ? 'border-blue-300' : ''}`}
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      {/* Status Badge */}
      <div
        className="absolute top-2 right-2 px-1.5 py-0.5 rounded text-[10px] font-medium text-white flex items-center space-x-1"
        style={{ backgroundColor: getStatusColor(app.status) }}
      >
        {statusIcon[app.status]}
        <span>{statusLabel[app.status]}</span>
      </div>

      {/* Locked Overlay */}
      {!canAccess && (
        <div className="absolute inset-0 bg-black bg-opacity-30 flex items-center justify-center z-10">
          <Lock className="w-6 h-6 text-white" />
        </div>
      )}

      {/* Card Content */}
      <div className="p-3">
        {/* Icon & Title */}
        <div className="flex items-start space-x-2 mb-2">
          <div className="text-2xl flex-shrink-0">{app.icon}</div>
          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-semibold text-gray-900 line-clamp-2">
              {app.name}
            </h3>
            <p className="text-[10px] font-medium text-gray-500 uppercase tracking-wide mt-0.5">
              {app.category}
            </p>
          </div>
        </div>

        {/* Description */}
        <p className="text-xs text-gray-600 line-clamp-2 mb-2">
          {app.description}
        </p>

        {/* Metadata */}
        <div className="space-y-1 mb-2 pb-2 border-b border-gray-100">
          {app.version && (
            <div className="flex items-center text-[10px] text-gray-500">
              <BarChart3 className="w-2.5 h-2.5 mr-1.5" />
              <span>v{app.version}</span>
            </div>
          )}
          {app.lastUpdated && (
            <div className="flex items-center text-[10px] text-gray-500">
              <Clock className="w-2.5 h-2.5 mr-1.5" />
              <span>Updated {getTimeSinceUpdate(app.lastUpdated)}</span>
            </div>
          )}
        </div>

        {/* Tags */}
        {app.tags && app.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mb-2">
            {app.tags.slice(0, 2).map((tag) => (
              <span
                key={tag}
                className="inline-block px-1.5 py-0.5 bg-blue-50 text-blue-700 text-[10px] rounded"
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Launch Button */}
        <button
          onClick={handleLaunch}
          disabled={!canAccess}
          className={`w-full py-1.5 px-2 rounded-md font-medium text-xs transition-all duration-200 flex items-center justify-center space-x-1.5 ${
            canAccess
              ? 'text-white hover:opacity-90 cursor-pointer'
              : 'bg-gray-100 text-gray-400 cursor-not-allowed'
          }`}
          style={canAccess ? { background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)' } : {}}
        >
          <span>Launch</span>
          <ExternalLink className="w-3 h-3" />
        </button>

        {/* Documentation Link */}
        {app.documentation && canAccess && (
          <a
            href={app.documentation}
            target="_blank"
            rel="noopener noreferrer"
            className="block mt-2 text-center text-[10px] text-blue-600 hover:text-blue-800 transition-colors"
          >
            View Documentation →
          </a>
        )}
      </div>
    </div>
  );
};
