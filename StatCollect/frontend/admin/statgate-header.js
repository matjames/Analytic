/**
 * StatGate Universal Header & Navigation Component
 * Standardized across all StatGate applications to match the Stage Register theme
 */

(function () {
  window.StatGate = window.StatGate || {};

  const APPS = [
    { name: 'Analytics Hub', icon: '📊', url: 'http://localhost:5000', desc: 'Decision Support & ML' },
    { name: 'StatCollect', icon: '📋', url: 'http://localhost:8084/admin/index.html', desc: 'Enterprise Field CAPI' },
    { name: 'StatFederation', icon: '🌐', url: 'http://localhost:3017', desc: 'Global Data Mesh' },
    { name: 'Registry & MFL', icon: '🪪', url: 'http://localhost:9090/api/search', desc: 'Master Facility & DID' },
    { name: 'Knowledge Portal', icon: '🧠', url: 'http://localhost:8099/health', desc: 'CKAN & SDMX Registry' },
    { name: 'AI Autonomy', icon: '🤖', url: 'http://localhost:8101/health', desc: 'Autonomous Agent Fleet' },
    { name: 'Learning CRM', icon: '📚', url: 'http://localhost:8102/health', desc: 'Capacity & Training' },
    { name: 'GeoIntel GIS', icon: '🗺️', url: 'http://localhost:8103/health', desc: 'Spatial Disaggregation' },
    { name: 'BPM Workflows', icon: '⚙️', url: 'http://localhost:8104/health', desc: 'Process Orchestration' },
    { name: 'StatIoT Telemetry', icon: '📡', url: 'http://localhost:8106/health', desc: 'Sensor Anomaly Bridge' },
    { name: 'StatData Lake', icon: '🗄️', url: 'http://localhost:8107/health', desc: 'Sovereign Lakehouse' },
    { name: 'Executive Suite', icon: '👑', url: 'http://localhost:5000/executive', desc: 'National KPI Cockpit' }
  ];

  window.toggleAppLauncher = function (e) {
    if (e) e.stopPropagation();
    const menu = document.getElementById('statgateLauncherMenu') || document.getElementById('appLauncherMenu');
    if (!menu) return;
    const isShown = menu.style.display === 'block';
    menu.style.display = isShown ? 'none' : 'block';
  };

  document.addEventListener('click', function (e) {
    const menu = document.getElementById('statgateLauncherMenu') || document.getElementById('appLauncherMenu');
    const toggle = document.querySelector('.launcher-toggle');
    if (menu && !menu.contains(e.target) && (!toggle || !toggle.contains(e.target))) {
      menu.style.display = 'none';
    }
  });

  window.StatGate.renderLauncherHTML = function () {
    return `
      <div class="app-launcher-wrapper">
        <button type="button" class="launcher-toggle" onclick="window.toggleAppLauncher(event)" aria-label="Open StatGate App Launcher" title="StatGate Unified App Launcher">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
            <path d="M4 4h4v4H4V4zm6 0h4v4h-4V4zm6 0h4v4h-4V4zM4 10h4v4H4v-4zm6 0h4v4h-4v-4zm6 0h4v4h-4v-4zM4 16h4v4H4v-4zm6 0h4v4h-4v-4zm6 0h4v4h-4v-4z"/>
          </svg>
        </button>
        <div id="statgateLauncherMenu" class="app-launcher-menu" style="display:none;">
          <div class="launcher-title">StatGate Sovereign Ecosystem</div>
          <div class="launcher-grid">
            ${APPS.map(app => `
              <a href="${app.url}" class="launcher-card" title="${app.desc}">
                <span class="launcher-icon">${app.icon}</span>
                <span>${app.name}</span>
              </a>
            `).join('')}
          </div>
        </div>
      </div>
    `;
  };
})();
