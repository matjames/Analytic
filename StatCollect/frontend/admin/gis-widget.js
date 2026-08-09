/**
 * StatGate GIS Widget — Geographic Intelligence Layer
 * Phase X: Leaflet.js-powered submission maps for StatCollect.
 *
 * Requires Leaflet from CDN:
 * <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9/dist/leaflet.css"/>
 * <script src="https://unpkg.com/leaflet@1.9/dist/leaflet.js"></script>
 *
 * Usage:
 *   GISWidget.renderSubmissionMap(el, geoPoints, options);
 *   GISWidget.renderCoverageMap(el, regionData, options);
 *   GISWidget.update(map, geoPoints);
 */

const GISWidget = (() => {
  // Status → colour mapping
  const STATUS_COLORS = {
    approved:  '#10b981',
    rejected:  '#ef4444',
    flagged:   '#f59e0b',
    received:  '#2563eb',
    in_review: '#8b5cf6',
    default:   '#64748b',
  };

  // Active Leaflet map registry (el → Leaflet map instance)
  const _maps = new WeakMap();

  function _statusColor(status) {
    return STATUS_COLORS[status] || STATUS_COLORS.default;
  }

  function _destroyMap(el) {
    if (_maps.has(el)) {
      _maps.get(el).remove();
      _maps.delete(el);
    }
  }

  /**
   * Renders a clustered submission point map with colour-coded status dots.
   * Falls back to a placeholder if no GPS data or Leaflet isn't loaded.
   *
   * @param {HTMLElement} el - Container element (must have a defined height)
   * @param {GeoPoint[]} geoPoints - Array of { lat, lng, instance_id, form_id, status }
   * @param {object} [options]
   * @param {number[]} [options.center=[1.3733, 32.2903]] - Default center (Uganda)
   * @param {number} [options.zoom=7]
   * @param {boolean} [options.clustering=true]
   * @returns {L.Map|null}
   */
  function renderSubmissionMap(el, geoPoints = [], options = {}) {
    if (typeof L === 'undefined') {
      el.innerHTML = `
        <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;height:100%;
          background:linear-gradient(135deg,#e0f2fe,#bae6fd);color:#0369a1;font-family:Inter,sans-serif">
          <div style="font-size:36px;margin-bottom:8px">🗺️</div>
          <div style="font-weight:700;font-size:14px">Leaflet.js required</div>
          <div style="font-size:12px;color:#0369a1;opacity:0.7;margin-top:4px">Add Leaflet CDN to enable maps</div>
        </div>`;
      return null;
    }

    _destroyMap(el);
    el.innerHTML = '';

    const center = options.center || [1.3733, 32.2903]; // Uganda default
    const zoom   = options.zoom   || 7;

    const map = L.map(el, { zoomControl: true }).setView(center, zoom);
    _maps.set(el, map);

    // OpenStreetMap tile layer
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    }).addTo(map);

    // Add points
    const validPoints = (geoPoints || []).filter(p => p.lat && p.lng && p.lat !== 0);
    if (validPoints.length === 0) {
      _addEmptyOverlay(map, el);
      return map;
    }

    const bounds = [];
    validPoints.forEach(pt => {
      const color = _statusColor(pt.status);
      const marker = L.circleMarker([pt.lat, pt.lng], {
        radius: 7,
        fillColor: color,
        color: '#fff',
        weight: 2,
        opacity: 1,
        fillOpacity: 0.85,
      });
      marker.bindPopup(`
        <div style="font-family:Inter,sans-serif;min-width:160px">
          <div style="font-weight:700;font-size:13px;margin-bottom:4px">Submission</div>
          <div style="font-size:11px;color:#475569"><b>ID:</b> <code>${pt.instance_id || '—'}</code></div>
          <div style="font-size:11px;color:#475569"><b>Form:</b> ${pt.form_id || '—'}</div>
          <div style="font-size:11px;margin-top:4px">
            <span style="display:inline-flex;align-items:center;gap:4px;background:${color}22;color:${color};font-weight:700;padding:2px 8px;border-radius:10px;font-size:10px">${(pt.status||'unknown').toUpperCase()}</span>
          </div>
          <div style="font-size:10px;color:#94a3b8;margin-top:4px">${pt.lat.toFixed(5)}, ${pt.lng.toFixed(5)}</div>
        </div>
      `);
      marker.addTo(map);
      bounds.push([pt.lat, pt.lng]);
    });

    if (bounds.length > 1) {
      map.fitBounds(bounds, { padding: [20, 20], maxZoom: 14 });
    }

    // Legend
    _addLegend(map, Object.keys(STATUS_COLORS).filter(k => k !== 'default'));

    return map;
  }

  /**
   * Renders a choropleth-style coverage bar map showing region rankings.
   * Since browser-based choropleths require boundary GeoJSON (not always available),
   * this renders a ranked coverage table with optional dot-map overlay.
   *
   * @param {HTMLElement} el
   * @param {RegionStat[]} regionData - Array of { name, submissions }
   * @param {object} [options]
   */
  function renderCoverageMap(el, regionData = [], options = {}) {
    if (!regionData || regionData.length === 0) {
      el.innerHTML = `<div style="display:flex;align-items:center;justify-content:center;height:100%;color:#94a3b8;font-family:Inter,sans-serif;font-size:13px">
        No regional coverage data available
      </div>`;
      return;
    }

    const max = Math.max(...regionData.map(r => r.submissions), 1);
    const sorted = [...regionData].sort((a,b) => b.submissions - a.submissions);

    el.innerHTML = `
      <div style="overflow-y:auto;height:100%;font-family:Inter,sans-serif">
        <div style="padding:12px;font-size:12px;font-weight:700;color:#475569;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #e2e8f0">
          Regional Coverage (${regionData.length} regions reporting)
        </div>
        ${sorted.map((r, i) => {
          const pct = Math.round((r.submissions / max) * 100);
          const color = i < 3 ? DashboardFramework.PALETTE.blue : (i < 7 ? DashboardFramework.PALETTE.teal : '#94a3b8');
          return `
            <div style="padding:10px 14px;border-bottom:1px solid #f1f5f9">
              <div style="display:flex;justify-content:space-between;margin-bottom:4px">
                <span style="font-size:13px;font-weight:600;color:#334155">${_esc(r.name)}</span>
                <span style="font-size:12px;color:#64748b;font-weight:700">${r.submissions.toLocaleString()}</span>
              </div>
              <div style="background:#f1f5f9;border-radius:6px;height:6px;overflow:hidden">
                <div style="background:${color};width:${pct}%;height:100%;border-radius:6px;transition:width 0.6s ease"></div>
              </div>
            </div>
          `;
        }).join('')}
      </div>
    `;
  }

  /**
   * Updates an existing Leaflet map with new geo points.
   * Clears old markers and adds the new set.
   * @param {L.Map} map
   * @param {GeoPoint[]} geoPoints
   */
  function update(map, geoPoints) {
    if (!map) return;
    map.eachLayer(layer => {
      if (layer instanceof L.CircleMarker) map.removeLayer(layer);
    });
    (geoPoints || []).forEach(pt => {
      if (!pt.lat || !pt.lng) return;
      L.circleMarker([pt.lat, pt.lng], {
        radius: 6,
        fillColor: _statusColor(pt.status),
        color: '#fff',
        weight: 2,
        fillOpacity: 0.8,
      }).addTo(map);
    });
  }

  function _addEmptyOverlay(map, el) {
    const div = L.DomUtil.create('div', '');
    div.style.cssText = 'position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);text-align:center;z-index:9999;pointer-events:none;font-family:Inter,sans-serif';
    div.innerHTML = `<div style="font-size:28px">📍</div><div style="font-size:12px;color:#64748b;margin-top:4px">No GPS data available</div>`;
    el.appendChild(div);
  }

  function _addLegend(map, statuses) {
    const legend = L.control({ position: 'bottomright' });
    legend.onAdd = () => {
      const div = L.DomUtil.create('div', '');
      div.style.cssText = 'background:white;padding:8px 12px;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.15);font-family:Inter,sans-serif;font-size:11px;line-height:1.6';
      div.innerHTML = statuses
        .filter(s => s !== 'default')
        .map(s => `<div style="display:flex;align-items:center;gap:6px">
          <div style="width:10px;height:10px;border-radius:50%;background:${_statusColor(s)}"></div>
          <span style="color:#475569;text-transform:capitalize">${s}</span>
        </div>`).join('');
      return div;
    };
    legend.addTo(map);
  }

  function _esc(s) {
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  }

  return {
    renderSubmissionMap,
    renderCoverageMap,
    update,
  };
})();

window.GISWidget = GISWidget;
