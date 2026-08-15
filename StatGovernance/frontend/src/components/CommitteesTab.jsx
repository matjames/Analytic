import React, { useState, useEffect } from 'react';

export default function CommitteesTab({ apiBase, token, onRefreshDashboard }) {
  const [committees, setCommittees] = useState([]);
  const [meetings, setMeetings] = useState([]);
  const [selectedCommittee, setSelectedCommittee] = useState(null);
  const [showCommitteeModal, setShowCommitteeModal] = useState(false);
  const [showMeetingModal, setShowMeetingModal] = useState(false);

  const [committeeForm, setCommitteeForm] = useState({
    name: '',
    code: '',
    mandate: '',
    chairperson: '',
    secretary: '',
    meeting_frequency: 'Monthly'
  });

  const [meetingForm, setMeetingForm] = useState({
    committee_id: '',
    title: '',
    meeting_date: '',
    location: 'StatGate Boardroom / StatChat Video',
    agenda: '',
    minutes: ''
  });

  const fetchCommitteesData = async () => {
    try {
      const [comRes, mtgRes] = await Promise.all([
        fetch(`${apiBase}/api/committees`, { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch(`${apiBase}/api/meetings`, { headers: { 'Authorization': `Bearer ${token}` } })
      ]);
      if (comRes.ok) {
        const coms = await comRes.json();
        setCommittees(coms);
        if (coms.length > 0 && !selectedCommittee) {
          setSelectedCommittee(coms[0]);
        }
      }
      if (mtgRes.ok) setMeetings(await mtgRes.json());
    } catch (err) {
      console.error('Error loading committees:', err);
    }
  };

  useEffect(() => {
    fetchCommitteesData();
  }, [apiBase, token]);

  const handleCreateCommittee = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/committees`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(committeeForm)
      });
      if (res.ok) {
        setShowCommitteeModal(false);
        fetchCommitteesData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating committee:', err);
    }
  };

  const handleScheduleMeeting = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/meetings`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          ...meetingForm,
          committee_id: meetingForm.committee_id || selectedCommittee?.id
        })
      });
      if (res.ok) {
        setShowMeetingModal(false);
        fetchCommitteesData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error scheduling meeting:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>🏛️</span> Institutional Governance Committees & Meetings
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Oversee executive boards, ethics committees, risk bodies, and scheduled sessions synced with Enterprise Calendar & StatChat.
          </p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn btn-secondary" onClick={() => setShowMeetingModal(true)}>
            + Schedule Session
          </button>
          <button className="btn btn-primary" onClick={() => setShowCommitteeModal(true)}>
            + Charter Committee
          </button>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: 24 }}>
        {/* Committee Roster */}
        <div className="glass-panel" style={{ padding: 22 }}>
          <div className="section-header">
            <span className="section-title" style={{ fontSize: 15 }}>Chartered Committees</span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {committees.map(c => (
              <div
                key={c.id}
                onClick={() => setSelectedCommittee(c)}
                style={{
                  padding: 16,
                  borderRadius: 8,
                  border: '1px solid var(--border-light)',
                  cursor: 'pointer',
                  background: selectedCommittee?.id === c.id ? '#f1f5f9' : '#ffffff',
                  transition: 'all 0.2s'
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                  <span style={{ fontWeight: 700, fontSize: 14 }}>{c.name}</span>
                  <span className="badge badge-draft">{c.code}</span>
                </div>
                <div style={{ fontSize: 12.5, color: 'var(--text-secondary)', marginBottom: 8 }}>
                  {c.mandate}
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11.5, color: 'var(--text-muted)' }}>
                  <span>Chair: <strong>{c.chairperson}</strong></span>
                  <span>Frequency: <strong>{c.meeting_frequency}</strong></span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Scheduled Sessions & Minutes */}
        <div className="glass-panel" style={{ padding: 22 }}>
          <div className="section-header">
            <span className="section-title" style={{ fontSize: 15 }}>
              <span>📅</span> Sessions for {selectedCommittee?.name || 'Selected Committee'}
            </span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {meetings.filter(m => !selectedCommittee || m.committee_id === selectedCommittee.id).length === 0 ? (
              <div style={{ color: 'var(--text-muted)', textAlign: 'center', padding: 30 }}>
                No meetings scheduled for this committee.
              </div>
            ) : (
              meetings.filter(m => !selectedCommittee || m.committee_id === selectedCommittee.id).map(m => (
                <div key={m.id} style={{ background: '#f8fafc', padding: 14, borderRadius: 8, border: '1px solid var(--border-light)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <span style={{ fontWeight: 700, fontSize: 13.5 }}>{m.title}</span>
                    <span className="badge badge-scheduled">{m.status}</span>
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 6 }}>
                    📍 {m.location} • 🕒 {new Date(m.meeting_date).toLocaleString()}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-dark)', background: '#ffffff', padding: 8, borderRadius: 6, border: '1px solid var(--border-light)' }}>
                    <strong>Agenda:</strong> {m.agenda || 'Regular monthly governance agenda.'}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Charter Committee Modal */}
      {showCommitteeModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Charter Governance Committee</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowCommitteeModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateCommittee}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Committee Name *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Research Ethics & Data Safety Board"
                      value={committeeForm.name}
                      onChange={e => setCommitteeForm({ ...committeeForm, name: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Committee Code *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. REDSB"
                      value={committeeForm.code}
                      onChange={e => setCommitteeForm({ ...committeeForm, code: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>Mandate & Terms of Reference *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify the regulatory mandate, supervisory authority, and scope..."
                    value={committeeForm.mandate}
                    onChange={e => setCommitteeForm({ ...committeeForm, mandate: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Chairperson *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Prof. Jane Nakimera"
                      value={committeeForm.chairperson}
                      onChange={e => setCommitteeForm({ ...committeeForm, chairperson: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Secretary *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Dr. Robert Sserunjogi"
                      value={committeeForm.secretary}
                      onChange={e => setCommitteeForm({ ...committeeForm, secretary: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowCommitteeModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Charter Committee</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Schedule Meeting Modal */}
      {showMeetingModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Schedule Committee Session</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowMeetingModal(false)}>✕</button>
            </div>
            <form onSubmit={handleScheduleMeeting}>
              <div className="modal-body">
                <div className="form-group">
                  <label>Select Committee *</label>
                  <select
                    className="form-control"
                    required
                    value={meetingForm.committee_id || selectedCommittee?.id}
                    onChange={e => setMeetingForm({ ...meetingForm, committee_id: e.target.value })}
                  >
                    {committees.map(c => (
                      <option key={c.id} value={c.id}>{c.name} ({c.code})</option>
                    ))}
                  </select>
                </div>

                <div className="form-group">
                  <label>Session Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. 2026 Q3 Statutory Risk & Data Privacy Review"
                    value={meetingForm.title}
                    onChange={e => setMeetingForm({ ...meetingForm, title: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Date & Time *</label>
                    <input
                      type="datetime-local"
                      className="form-control"
                      required
                      value={meetingForm.meeting_date}
                      onChange={e => setMeetingForm({ ...meetingForm, meeting_date: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Location / StatChat Link</label>
                    <input
                      className="form-control"
                      value={meetingForm.location}
                      onChange={e => setMeetingForm({ ...meetingForm, location: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>Meeting Agenda & Discussion Points</label>
                  <textarea
                    className="form-control"
                    placeholder="Enumerate discussion points, policy reviews, or risk items..."
                    value={meetingForm.agenda}
                    onChange={e => setMeetingForm({ ...meetingForm, agenda: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowMeetingModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Schedule Session</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
