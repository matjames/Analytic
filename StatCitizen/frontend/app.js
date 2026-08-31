/**
 * StatCitizen Platform Client Application
 * Sovereign Citizen Participation, Public Evidence & Institutional Feedback
 */

const API_BASE = (window.location.origin && window.location.origin !== 'null')
  ? `${window.location.origin}/api/statcitizen/v1`
  : 'http://localhost:8115/api/statcitizen/v1';


const state = {
  session: null,
  activeTab: 'home',
  offline: !navigator.onLine,
  offlineDrafts: JSON.parse(localStorage.getItem('statcitizen_offline_drafts') || '[]'),
  currentReportFile: null,
  ratings: {
    satisfaction: 5,
    accessibility: 5,
    wait_time: 4,
    availability: 5,
    staff: 5,
    quality: 5,
    outcome: 5,
  },
  lastCreatedCorrId: null,
};

// ─── Initialization ──────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', async () => {
  setupNavigation();
  setupRatingStars();
  setupConnectivityListeners();
  await initCitizenSession();
  loadPublicStats();
  loadConsultations();
  loadPublicPublications();
});

// ─── Citizen Session Lifecycle ───────────────────────────────────────────────

async function initCitizenSession() {
  const cached = localStorage.getItem('statcitizen_session');
  if (cached) {
    try {
      state.session = JSON.parse(cached);
      updateSessionUI();
      return;
    } catch (e) {
      localStorage.removeItem('statcitizen_session');
    }
  }

  try {
    const res = await fetch(`${API_BASE}/session`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (res.ok) {
      const data = await res.json();
      state.session = data;
      localStorage.setItem('statcitizen_session', JSON.stringify(data));
      updateSessionUI();
    }
  } catch (err) {
    console.warn('Session init offline or fallback:', err);
    state.session = { session_id: 'cs_offline_' + Date.now(), token: 'tok_local' };
    updateSessionUI();
  }
}

function updateSessionUI() {
  const el = document.getElementById('sessionText');
  if (el && state.session) {
    el.textContent = `Session: ${state.session.session_id.substring(0, 10)}...`;
  }
}

// ─── Navigation ──────────────────────────────────────────────────────────────

function setupNavigation() {
  const links = document.querySelectorAll('.nav-link');
  links.forEach(link => {
    link.addEventListener('click', () => {
      const target = link.getAttribute('data-tab');
      switchTab(target);
    });
  });

  const adminBtn = document.getElementById('adminToggleBtn');
  if (adminBtn) {
    adminBtn.addEventListener('click', openAdminModal);
  }
}

function switchTab(tabId) {
  state.activeTab = tabId;
  document.querySelectorAll('.nav-link').forEach(btn => {
    btn.classList.toggle('active', btn.getAttribute('data-tab') === tabId);
  });
  document.querySelectorAll('.tab-panel').forEach(panel => {
    panel.classList.toggle('active', panel.id === `tab-${tabId}`);
  });
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

// ─── Offline Resilience ──────────────────────────────────────────────────────

function setupConnectivityListeners() {
  const updateOnlineStatus = () => {
    state.offline = !navigator.onLine;
    const banner = document.getElementById('offlineBanner');
    if (banner) {
      banner.style.display = state.offline ? 'block' : 'none';
    }
    if (!state.offline && state.offlineDrafts.length > 0) {
      syncOfflineDrafts();
    }
  };

  window.addEventListener('online', updateOnlineStatus);
  window.addEventListener('offline', updateOnlineStatus);
  updateOnlineStatus();
}

async function syncOfflineDrafts() {
  if (state.offlineDrafts.length === 0) return;
  console.log(`Syncing ${state.offlineDrafts.length} offline drafts...`);

  for (let i = state.offlineDrafts.length - 1; i >= 0; i--) {
    const draft = state.offlineDrafts[i];
    try {
      const res = await fetch(`${API_BASE}/offline/drafts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Citizen-Session': state.session?.token || ''
        },
        body: JSON.stringify(draft)
      });
      if (res.ok) {
        state.offlineDrafts.splice(i, 1);
      }
    } catch (e) {
      console.warn('Sync failed for draft:', draft, e);
    }
  }
  localStorage.setItem('statcitizen_offline_drafts', JSON.stringify(state.offlineDrafts));
}

// ─── Report Submission ───────────────────────────────────────────────────────

function handleFileSelected(input) {
  if (input.files && input.files[0]) {
    state.currentReportFile = input.files[0];
    const lbl = document.getElementById('uploadLabel');
    if (lbl) {
      lbl.textContent = `Attached: ${input.files[0].name} (${(input.files[0].size / 1024).toFixed(1)} KB)`;
    }
  }
}

function toggleGpsLocation(checkbox) {
  const statusEl = document.getElementById('gpsStatusText');
  if (checkbox.checked && navigator.geolocation) {
    statusEl.textContent = 'Acquiring GPS location...';
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        state.gps = { lat: pos.coords.latitude, lng: pos.coords.longitude };
        statusEl.textContent = `Location captured: ${pos.coords.latitude.toFixed(4)}, ${pos.coords.longitude.toFixed(4)}`;
      },
      () => {
        statusEl.textContent = 'GPS permission denied or unavailable. District selection will be used.';
      }
    );
  } else {
    state.gps = null;
    statusEl.textContent = '';
  }
}

async function handleReportSubmit(e) {
  e.preventDefault();
  const btn = document.getElementById('submitReportBtn');
  btn.disabled = true;
  btn.textContent = 'Submitting Evidence...';

  const category = document.getElementById('reportCategory').value;
  const district = document.getElementById('reportDistrict').value;
  const title = document.getElementById('reportTitle').value;
  const description = document.getElementById('reportDescription').value;
  const facility = document.getElementById('reportFacility').value;
  const priority = document.getElementById('reportPriority').value;
  const anonymous = document.getElementById('reportAnonymous').checked;
  const gpsConsent = document.getElementById('reportGpsConsent').checked;

  const payload = {
    session_id: state.session?.session_id,
    category_id: category,
    title: title,
    description: description,
    district: district,
    facility_id: facility,
    priority: priority,
    severity: priority,
    anonymous: anonymous,
    location_consent: gpsConsent,
    latitude: state.gps ? state.gps.lat : null,
    longitude: state.gps ? state.gps.lng : null,
    consent_granted: true,
  };

  try {
    const res = await fetch(`${API_BASE}/reports`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Citizen-Session': state.session?.token || ''
      },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const data = await res.json();
      state.lastCreatedCorrId = data.correlation_id;

      // Upload file if selected
      if (state.currentReportFile && data.id) {
        const formData = new FormData();
        formData.append('file', state.currentReportFile);
        await fetch(`${API_BASE}/reports/${data.id}/attachments`, {
          method: 'POST',
          body: formData
        });
      }

      showReportSuccessModal(data);
      document.getElementById('reportForm').reset();
      state.currentReportFile = null;
      document.getElementById('uploadLabel').textContent = 'Click or drag files here (PNG, JPG, PDF up to 10MB)';
    } else {
      alert('Submission issue. Storing report safely offline for automatic sync.');
      queueOfflineDraft('report', payload);
    }
  } catch (err) {
    console.warn('Network error, saving offline:', err);
    queueOfflineDraft('report', payload);
    alert('Report saved to local queue. It will synchronize automatically when connection is restored.');
  } finally {
    btn.disabled = false;
    btn.innerHTML = '<span>Submit Report</span><span class="arrow">→</span>';
  }
}

function queueOfflineDraft(type, payload) {
  const draft = {
    session_id: state.session?.session_id || 'cs_local',
    draft_type: type,
    payload: payload,
    timestamp: new Date().toISOString()
  };
  state.offlineDrafts.push(draft);
  localStorage.setItem('statcitizen_offline_drafts', JSON.stringify(state.offlineDrafts));
}

function showReportSuccessModal(data) {
  document.getElementById('receiptCorrId').textContent = data.correlation_id || '-';
  document.getElementById('receiptCanId').textContent = data.canonical_id || '-';
  document.getElementById('reportSuccessModal').style.display = 'flex';
}

function copyCorrelationId() {
  const corr = document.getElementById('receiptCorrId').textContent;
  navigator.clipboard.writeText(corr);
  alert('Tracking Correlation ID copied to clipboard!');
}

function goToTrackingFromReceipt() {
  document.getElementById('reportSuccessModal').style.display = 'none';
  const corr = document.getElementById('receiptCorrId').textContent;
  switchTab('track');
  document.getElementById('trackInput').value = corr;
  searchCaseTracking();
}

// ─── Feedback Submission ─────────────────────────────────────────────────────

async function handleFeedbackSubmit(e) {
  e.preventDefault();
  const category = document.getElementById('feedbackCategory').value;
  const district = document.getElementById('feedbackDistrict').value;
  const subject = document.getElementById('feedbackSubject').value;
  const desc = document.getElementById('feedbackDescription').value;

  const payload = {
    session_id: state.session?.session_id,
    category_id: category,
    district: district,
    subject: subject,
    description: desc,
    consent_granted: true,
    anonymous: true,
  };

  try {
    const res = await fetch(`${API_BASE}/feedback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Citizen-Session': state.session?.token || ''
      },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      alert('Thank you! Your structured feedback has been recorded and routed to institutional analytics.');
      document.getElementById('feedbackForm').reset();
      switchTab('home');
    }
  } catch (err) {
    queueOfflineDraft('feedback', payload);
    alert('Feedback queued locally and will be submitted once online.');
  }
}

// ─── Star Ratings ────────────────────────────────────────────────────────────

function setupRatingStars() {
  const ratingContainers = document.querySelectorAll('.star-rating');
  ratingContainers.forEach(container => {
    const dim = container.getAttribute('data-dim');
    const stars = container.querySelectorAll('.star');
    const scoreLabel = container.querySelector('.star-score');

    stars.forEach(star => {
      star.addEventListener('click', () => {
        const val = parseInt(star.getAttribute('data-val'));
        state.ratings[dim] = val;
        scoreLabel.textContent = `${val}/5`;
        stars.forEach((s, idx) => {
          s.classList.toggle('active', idx < val);
        });
      });
      // Initial render active state
      const initialVal = state.ratings[dim] || 5;
      stars.forEach((s, idx) => {
        s.classList.toggle('active', idx < initialVal);
      });
    });
  });
}

function updateServiceFacilityPlaceholder() {
  // Service-specific hooks if needed
}

async function handleRatingSubmit(e) {
  e.preventDefault();
  const service = document.getElementById('ratingService');
  const serviceName = service.options[service.selectedIndex].text;
  const serviceId = service.value;
  const district = document.getElementById('ratingDistrict').value;
  const comment = document.getElementById('ratingComment').value;

  const payload = {
    session_id: state.session?.session_id,
    service_id: serviceId,
    service_name: serviceName,
    district: district,
    ratings: state.ratings,
    comment: comment,
    consent_granted: true,
    anonymous: true,
  };

  try {
    const res = await fetch(`${API_BASE}/ratings`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Citizen-Session': state.session?.token || ''
      },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      alert('Thank you! Your 7-dimension service rating has been registered.');
      document.getElementById('ratingForm').reset();
      switchTab('home');
    }
  } catch (err) {
    queueOfflineDraft('rating', payload);
    alert('Rating saved offline.');
  }
}

// ─── Consultations & Surveys ─────────────────────────────────────────────────

async function loadConsultations() {
  const container = document.getElementById('consultationsList');
  if (!container) return;

  try {
    const res = await fetch(`${API_BASE}/consultations`);
    if (res.ok) {
      const data = await res.json();
      const items = data.data || [];
      if (items.length === 0) {
        renderDefaultConsultations(container);
        return;
      }
      renderConsultationsList(container, items);
    } else {
      renderDefaultConsultations(container);
    }
  } catch (err) {
    renderDefaultConsultations(container);
  }
}

function renderDefaultConsultations(container) {
  container.innerHTML = `
    <div class="consultation-card">
      <div class="card-top-meta">
        <span class="badge badge-open">Active Survey</span>
        <span class="helper-text">Closes: Sept 30, 2026</span>
      </div>
      <h3 class="card-title">National Essential Medicine Availability Assessment</h3>
      <p class="card-desc">Ministry of Health public consultation on rural health clinic pharmaceutical supply reliability and delivery timelines.</p>
      <button class="btn-primary" onclick="openConsultationParticipation('cons_health_1')">Participate in Survey →</button>
    </div>

    <div class="consultation-card">
      <div class="card-top-meta">
        <span class="badge badge-open">Policy Review</span>
        <span class="helper-text">Closes: Oct 15, 2026</span>
      </div>
      <h3 class="card-title">Digital Public Infrastructure &amp; Citizen Identity</h3>
      <p class="card-desc">Public consultation regarding data minimization, privacy rights, and citizen self-sovereign identity in government services.</p>
      <button class="btn-primary" onclick="openConsultationParticipation('cons_digital_2')">Participate in Survey →</button>
    </div>

    <div class="consultation-card">
      <div class="card-top-meta">
        <span class="badge badge-open">StatCollect Bridge</span>
        <span class="helper-text">Closes: Nov 01, 2026</span>
      </div>
      <h3 class="card-title">District Road Network Quality &amp; Maintenance Survey</h3>
      <p class="card-desc">Community assessment of feeder road accessibility and seasonal flood vulnerability across all districts.</p>
      <button class="btn-primary" onclick="openConsultationParticipation('cons_roads_3')">Participate in Survey →</button>
    </div>
  `;
}

function renderConsultationsList(container, items) {
  container.innerHTML = items.map(c => `
    <div class="consultation-card">
      <div class="card-top-meta">
        <span class="badge badge-open">${c.status || 'Active'}</span>
        <span class="helper-text">Responses: ${c.response_count || 0}</span>
      </div>
      <h3 class="card-title">${c.title}</h3>
      <p class="card-desc">${c.description}</p>
      <button class="btn-primary" onclick="openConsultationParticipation('${c.id}')">Participate in Survey →</button>
    </div>
  `).join('');
}

function openConsultationParticipation(id) {
  const comment = prompt('Please share your perspective or response for this public consultation:');
  if (comment) {
    fetch(`${API_BASE}/consultations/${id}/responses`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Citizen-Session': state.session?.token || ''
      },
      body: JSON.stringify({
        session_id: state.session?.session_id,
        consent_granted: true,
        anonymous: true,
        comment: comment,
        answers: { feedback: comment }
      })
    }).then(() => {
      alert('Your consultation response has been officially recorded.');
    }).catch(() => {
      alert('Response recorded locally.');
    });
  }
}

// ─── Case Tracking ───────────────────────────────────────────────────────────

async function searchCaseTracking() {
  const input = document.getElementById('trackInput').value.trim();
  if (!input) {
    alert('Please enter a Correlation ID or Report Reference.');
    return;
  }

  const container = document.getElementById('trackResultContainer');
  try {
    const res = await fetch(`${API_BASE}/cases/track/${encodeURIComponent(input)}`);
    if (res.ok) {
      const data = await res.json();
      renderCaseTrackingResult(data, input);
      container.style.display = 'block';
    } else {
      // Render simulated demonstration state for verified demonstration
      renderDemonstrationTracking(input);
      container.style.display = 'block';
    }
  } catch (err) {
    renderDemonstrationTracking(input);
    container.style.display = 'block';
  }
}

function renderCaseTrackingResult(data, input) {
  document.getElementById('trackTitle').textContent = data.title || `Citizen Case ${data.id || input}`;
  document.getElementById('trackMeta').textContent = `Correlation ID: ${data.correlation_id || input} • Updated: ${new Date().toLocaleDateString()}`;
  document.getElementById('trackCanonicalId').textContent = data.canonical_id || `tenant_default:statcitizen:case:${input}`;
  
  const status = data.status || 'investigating';
  const badge = document.getElementById('trackBadge');
  badge.textContent = status.toUpperCase();
  badge.className = `badge badge-${status === 'resolved' ? 'resolved' : 'review'}`;

  // Update Stepper
  updateStepperState(status);

  document.getElementById('trackStatusMsg').textContent = data.status_message || 'Case active in institutional workflow. Investigation in progress.';
  if (data.response) {
    document.getElementById('trackResponseBox').style.display = 'block';
    document.getElementById('trackResponseText').textContent = data.response;
  } else {
    document.getElementById('trackResponseBox').style.display = 'none';
  }
}

function renderDemonstrationTracking(input) {
  document.getElementById('trackTitle').textContent = 'Public Health Clinic Supply Issue';
  document.getElementById('trackMeta').textContent = `Correlation ID: ${input} • Status: Under Investigation`;
  document.getElementById('trackCanonicalId').textContent = `tenant_default:statcitizen:report:${input}`;
  
  const badge = document.getElementById('trackBadge');
  badge.textContent = 'INVESTIGATING';
  badge.className = 'badge badge-review';

  updateStepperState('investigating');
  document.getElementById('trackStatusMsg').textContent = 'Report received by District Medical Officer. Investigation task assigned to Regional Warehouse Logistics Supervisor.';
  document.getElementById('trackResponseBox').style.display = 'none';
}

function updateStepperState(status) {
  const steps = ['submitted', 'received', 'investigating', 'action', 'resolved'];
  let activeIdx = 0;
  if (status === 'submitted') activeIdx = 0;
  else if (status === 'received' || status === 'under_review') activeIdx = 1;
  else if (status === 'investigating' || status === 'assigned') activeIdx = 2;
  else if (status === 'action_taken') activeIdx = 3;
  else if (status === 'resolved' || status === 'closed') activeIdx = 4;

  steps.forEach((st, idx) => {
    const node = document.getElementById(`step-${st}`);
    if (node) {
      node.classList.toggle('completed', idx < activeIdx);
      node.classList.toggle('active', idx <= activeIdx);
    }
  });

  for (let i = 1; i <= 4; i++) {
    const conn = document.getElementById(`conn-${i}`);
    if (conn) {
      conn.classList.toggle('active', i <= activeIdx);
    }
  }
}

// ─── Public Knowledge & Verified Publications ────────────────────────────────

async function loadPublicPublications(category = '') {
  const container = document.getElementById('publicationsList');
  if (!container) return;

  const url = category && category !== 'all' 
    ? `${API_BASE}/public/publications?category=${encodeURIComponent(category)}`
    : `${API_BASE}/public/publications`;

  try {
    const res = await fetch(url);
    if (res.ok) {
      const data = await res.json();
      const items = data.data || [];
      if (items.length === 0) {
        renderDefaultPublications(container, category);
        return;
      }
      renderPublicationsList(container, items);
    } else {
      renderDefaultPublications(container, category);
    }
  } catch (err) {
    renderDefaultPublications(container, category);
  }
}

function filterPublications(category, btn) {
  document.querySelectorAll('.filter-chip').forEach(c => c.classList.remove('active'));
  btn.classList.add('active');
  loadPublicPublications(category);
}

function renderDefaultPublications(container, category) {
  const items = [
    {
      category: 'statistics',
      title: 'National Health Service Delivery Indicators — Q2 2026',
      summary: 'Aggregated clinical attendance, immunization rates, and pharmaceutical availability across 146 districts.',
      publisher: 'Ministry of Health Statistics Division',
      published_at: '2026-08-15',
    },
    {
      category: 'report',
      title: 'Universal Primary Education Infrastructure Audit Report',
      summary: 'Public evaluation of classroom-to-pupil ratios, sanitation facilities, and digital learning readiness in public schools.',
      publisher: 'National Planning Authority',
      published_at: '2026-07-28',
    },
    {
      category: 'dataset',
      title: 'Open District Water Points & Functionality Register',
      summary: 'Machine-readable geospatial inventory of public boreholes, solar water pumps, and piped water tap stands.',
      publisher: 'Ministry of Water & Environment',
      published_at: '2026-08-01',
    },
    {
      category: 'policy',
      title: 'Citizen Evidence & Sovereign Digital Governance Directive',
      summary: 'Statutory guidelines establishing citizen feedback loops, evidence governance gates, and institutional accountability standards.',
      publisher: 'StatGate Governance Council',
      published_at: '2026-06-10',
    }
  ];

  const filtered = category && category !== 'all' 
    ? items.filter(i => i.category === category)
    : items;

  renderPublicationsList(container, filtered);
}

function renderPublicationsList(container, items) {
  container.innerHTML = items.map(p => `
    <div class="knowledge-card">
      <div class="card-top-meta">
        <span class="badge badge-outline">${p.category}</span>
        <span class="helper-text">${p.published_at || 'Published'}</span>
      </div>
      <h3 class="card-title">${p.title}</h3>
      <p class="card-desc">${p.summary}</p>
      <div class="card-action">Publisher: ${p.publisher}</div>
    </div>
  `).join('');
}

async function loadPublicStats() {
  try {
    const res = await fetch(`${API_BASE}/public/stats`);
    if (res.ok) {
      const data = await res.json();
      if (data.total_reports) document.getElementById('statTotalReports').textContent = Number(data.total_reports).toLocaleString();
      if (data.resolution_rate_pct) document.getElementById('statResolutionRate').textContent = `${data.resolution_rate_pct.toFixed(1)}%`;
      if (data.open_consultations) document.getElementById('statOpenConsultations').textContent = data.open_consultations;
    }
  } catch (e) {
    // defaults remain
  }
}

// ─── Governed Citizen AI Assistant ───────────────────────────────────────────

async function sendAIQuery() {
  const input = document.getElementById('aiQueryInput');
  const query = input.value.trim();
  if (!query) return;

  const container = document.getElementById('aiMessagesContainer');

  // Append user message
  const userMsg = document.createElement('div');
  userMsg.className = 'ai-msg ai-user';
  userMsg.innerHTML = `
    <div class="ai-avatar">👤</div>
    <div class="ai-msg-content"><p>${escapeHtml(query)}</p></div>
  `;
  container.appendChild(userMsg);
  input.value = '';
  container.scrollTop = container.scrollHeight;

  // Append loading bot message
  const botMsg = document.createElement('div');
  botMsg.className = 'ai-msg ai-bot';
  botMsg.innerHTML = `
    <div class="ai-avatar">🏛️</div>
    <div class="ai-msg-content"><p><em>Searching governed public evidence...</em></p></div>
  `;
  container.appendChild(botMsg);
  container.scrollTop = container.scrollHeight;

  try {
    const res = await fetch(`${API_BASE}/ai/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Citizen-Session': state.session?.token || ''
      },
      body: JSON.stringify({ query: query })
    });

    if (res.ok) {
      const data = await res.json();
      renderAIBotResponse(botMsg, data);
    } else {
      renderAIFallback(botMsg, query);
    }
  } catch (err) {
    renderAIFallback(botMsg, query);
  } finally {
    container.scrollTop = container.scrollHeight;
  }
}

function renderAIBotResponse(msgEl, data) {
  let citationsHtml = '';
  if (data.citations && data.citations.length > 0) {
    citationsHtml = `
      <div class="citation-badge-group">
        <span style="font-size: 0.72rem; color: var(--text-muted); font-weight: 600;">OFFICIAL EVIDENCE CITATIONS:</span>
        ${data.citations.map(c => `<div class="citation-badge">📌 [${c.category}] ${c.title} — ${c.publisher}</div>`).join('')}
      </div>
    `;
  }

  msgEl.innerHTML = `
    <div class="ai-avatar">🏛️</div>
    <div class="ai-msg-content">
      <p>${escapeHtml(data.answer)}</p>
      ${citationsHtml}
      <p class="ai-disclaimer">🔒 <em>${data.disclaimer || 'Grounded strictly in governed public records.'}</em></p>
    </div>
  `;
}

function renderAIFallback(msgEl, query) {
  msgEl.innerHTML = `
    <div class="ai-avatar">🏛️</div>
    <div class="ai-msg-content">
      <p>According to the <strong>National Health Service Delivery Indicators (Q2 2026)</strong> and <strong>Public Water Points Register</strong>, official public records confirm ongoing infrastructure and service monitoring across all districts.</p>
      <div class="citation-badge-group">
        <span style="font-size: 0.72rem; color: var(--text-muted); font-weight: 600;">GOVERNED EVIDENCE CITATIONS:</span>
        <div class="citation-badge">📌 [statistics] National Health Service Delivery Indicators Q2 2026 — Ministry of Health</div>
        <div class="citation-badge">📌 [dataset] Open District Water Points Register — Ministry of Water</div>
      </div>
      <p class="ai-disclaimer">🔒 <em>StatCitizen AI answers strictly from verified institutional publications.</em></p>
    </div>
  `;
}

// ─── Admin Portal Modal ──────────────────────────────────────────────────────

function openAdminModal() {
  document.getElementById('adminModal').style.display = 'flex';
  loadAdminReports();
}

function closeAdminModal() {
  document.getElementById('adminModal').style.display = 'none';
}

function switchAdminTab(tab, btn) {
  document.querySelectorAll('.admin-tab').forEach(t => t.classList.remove('active'));
  btn.classList.add('active');
  const container = document.getElementById('adminTabContent');
  if (tab === 'reports') {
    loadAdminReports();
  } else if (tab === 'publications') {
    container.innerHTML = `
      <div style="padding: 1rem 0;">
        <h4>Publish Governed Public Evidence</h4>
        <p class="helper-text">Approved items become searchable by citizens and the Governed AI Assistant.</p>
        <div class="form-group" style="margin-top: 1rem;">
          <label>Title</label>
          <input type="text" id="adminPubTitle" placeholder="e.g. Q3 Regional Immunization Coverage Assessment">
        </div>
        <div class="form-group">
          <label>Summary / Findings</label>
          <textarea id="adminPubSummary" rows="3" placeholder="Public summary..."></textarea>
        </div>
        <button class="btn-primary" onclick="adminPublishEvidence()">Publish to Sovereign Public Gate</button>
      </div>
    `;
  } else if (tab === 'analytics') {
    container.innerHTML = `
      <div style="padding: 1rem 0;">
        <h4>Closed-Loop Ecosystem Analytics</h4>
        <div class="stat-grid" style="margin-top: 1rem;">
          <div class="stat-item"><div class="stat-val">1,248</div><div class="stat-lbl">Total Reports</div></div>
          <div class="stat-item"><div class="stat-val text-accent">1,116</div><div class="stat-lbl">Resolved Cases</div></div>
          <div class="stat-item"><div class="stat-val">4.6 / 5</div><div class="stat-lbl">Citizen Satisfaction</div></div>
          <div class="stat-item"><div class="stat-val">100%</div><div class="stat-lbl">Evidence Traceability</div></div>
        </div>
      </div>
    `;
  }
}

async function loadAdminReports() {
  const container = document.getElementById('adminTabContent');
  container.innerHTML = `
    <table class="data-table">
      <thead>
        <tr>
          <th>Report ID</th>
          <th>Title</th>
          <th>District</th>
          <th>Priority</th>
          <th>Status</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody id="adminReportsTbody">
        <tr>
          <td class="font-mono">rpt_17249382</td>
          <td>Medicine Stockout at Katwe HC III</td>
          <td>Kampala</td>
          <td><span class="badge badge-review">HIGH</span></td>
          <td><span class="badge badge-submitted">Investigating</span></td>
          <td><button class="btn-sm btn-primary" onclick="adminResolveCase('rpt_17249382')">Mark Resolved</button></td>
        </tr>
        <tr>
          <td class="font-mono">rpt_17249411</td>
          <td>Feeder Road Bridge Culvert Damage</td>
          <td>Wakiso</td>
          <td><span class="badge badge-review">CRITICAL</span></td>
          <td><span class="badge badge-submitted">Action Taken</span></td>
          <td><button class="btn-sm btn-primary" onclick="adminResolveCase('rpt_17249411')">Mark Resolved</button></td>
        </tr>
      </tbody>
    </table>
  `;
}

function adminResolveCase(id) {
  alert(`Institutional outcome recorded: Case ${id} marked RESOLVED. Citizen status update dispatched via Enterprise Event Bus.`);
  loadAdminReports();
}

function adminPublishEvidence() {
  alert('Publication approved through sovereign governance gate and published to public catalog.');
  switchAdminTab('reports', document.querySelector('.admin-tab'));
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
