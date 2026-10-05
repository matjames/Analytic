import React, { useEffect, useState } from 'react';

const EMPTY_EVENT = { eventName: '', startDate: '', endDate: '', location: '', registrationUrl: '', status: 'Planned' };
const EMPTY_ATTENDEE = { userId: '', name: '', email: '', attendanceType: 'Delegate' };

export default function ConferenceEventsPanel({ apiBase, researchId }) {
  const [events, setEvents] = useState([]);
  const [eventForm, setEventForm] = useState(EMPTY_EVENT);
  const [eventEditing, setEventEditing] = useState(null);
  const [eventOpen, setEventOpen] = useState(false);
  const [selectedEvent, setSelectedEvent] = useState(null);
  const [attendees, setAttendees] = useState([]);
  const [attendeeForm, setAttendeeForm] = useState(EMPTY_ATTENDEE);

  const loadEvents = () => {
    if (!researchId) return setEvents([]);
    fetch(`${apiBase}/api/research/${researchId}/conference-events`).then(r => r.json()).then(d => setEvents(Array.isArray(d) ? d : [])).catch(() => setEvents([]));
  };
  useEffect(loadEvents, [researchId]);

  const loadAttendance = (eventId) => {
    fetch(`${apiBase}/api/conference-events/${eventId}/attendance`).then(r => r.json()).then(d => setAttendees(Array.isArray(d) ? d : [])).catch(() => setAttendees([]));
  };
  const toggleAttendance = (eventId) => {
    if (selectedEvent === eventId) { setSelectedEvent(null); setAttendees([]); return; }
    setSelectedEvent(eventId);
    loadAttendance(eventId);
  };
  const saveEvent = (e) => {
    e.preventDefault();
    const method = eventEditing ? 'PUT' : 'POST';
    const url = eventEditing ? `${apiBase}/api/conference-events/${eventEditing}` : `${apiBase}/api/conference-events`;
    fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...eventForm, researchId }) }).then(r => { if (!r.ok) throw new Error('save failed'); setEventForm(EMPTY_EVENT); setEventEditing(null); setEventOpen(false); loadEvents(); }).catch(() => {});
  };
  const editEvent = (event) => { setEventEditing(event.id); setEventForm({ eventName: event.eventName || '', startDate: event.startDate || '', endDate: event.endDate || '', location: event.location || '', registrationUrl: event.registrationUrl || '', status: event.status || 'Planned' }); setEventOpen(true); };
  const removeEvent = (eventId) => { if (window.confirm('Delete this conference event and its attendance records?')) fetch(`${apiBase}/api/conference-events/${eventId}`, { method: 'DELETE' }).then(loadEvents); };
  const registerAttendee = (e, eventId) => {
    e.preventDefault();
    fetch(`${apiBase}/api/conference-events/${eventId}/attendance`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(attendeeForm) }).then(r => { if (!r.ok) throw new Error('registration failed'); setAttendeeForm(EMPTY_ATTENDEE); loadAttendance(eventId); }).catch(() => {});
  };
  const updateAttendee = (attendee, status) => fetch(`${apiBase}/api/conference-attendance/${attendee.id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...attendee, status }) }).then(r => { if (r.ok) loadAttendance(attendee.eventId); });
  const removeAttendee = (attendee) => { if (window.confirm(`Remove ${attendee.name} from the event?`)) fetch(`${apiBase}/api/conference-attendance/${attendee.id}`, { method: 'DELETE' }).then(r => { if (r.ok) loadAttendance(attendee.eventId); }); };

  return <div style={{ marginTop: '2rem' }}>
    <div className="section-heading"><div><h2>Conference Calendar and Attendance</h2><p>Plan conference dates, track registration, and record who attended or presented.</p></div><button className="btn btn-primary" onClick={() => { setEventEditing(null); setEventForm(EMPTY_EVENT); setEventOpen(v => !v); }}>{eventOpen ? 'Close' : 'New Event'}</button></div>
    {eventOpen && <form onSubmit={saveEvent} className="inline-form">
      {['eventName', 'startDate', 'endDate', 'location', 'registrationUrl'].map(key => <label key={key}>{key.replace(/[A-Z]/g, m => ` ${m}`).replace(/^./, m => m.toUpperCase())}<input type={key.endsWith('Date') ? 'date' : 'text'} value={eventForm[key]} onChange={e => setEventForm({ ...eventForm, [key]: e.target.value })} required={key === 'eventName' || key === 'startDate' || key === 'endDate'} /></label>)}
      <label>Status<select value={eventForm.status} onChange={e => setEventForm({ ...eventForm, status: e.target.value })}><option>Planned</option><option>Registration Open</option><option>Completed</option><option>Cancelled</option></select></label>
      <div><button className="btn btn-primary" type="submit">{eventEditing ? 'Update Event' : 'Save Event'}</button></div>
    </form>}
    <div className="data-list">{events.length ? events.map(event => <article className="data-card" key={event.id}>
      <div><strong>{event.eventName}</strong><p>{event.startDate} to {event.endDate} | {event.location || 'Location not set'} | {event.status}</p>{event.registrationUrl && <small><a href={event.registrationUrl} target="_blank" rel="noreferrer">Registration link</a></small>}</div>
      <div><button className="btn btn-secondary" onClick={() => editEvent(event)}>Edit</button> <button className="btn btn-secondary" onClick={() => toggleAttendance(event.id)}>{selectedEvent === event.id ? 'Close attendance' : 'Attendance'}</button> <button className="btn btn-secondary" onClick={() => removeEvent(event.id)}>Delete</button></div>
      {selectedEvent === event.id && <div style={{ gridColumn: '1 / -1', borderTop: '1px solid #e2e8f0', paddingTop: '1rem' }}><strong>Registrations and attendance</strong><div className="data-list" style={{ marginTop: '0.75rem' }}>{attendees.length ? attendees.map(attendee => <article className="data-card" key={attendee.id}><div><strong>{attendee.name}</strong><p>{attendee.email} | {attendee.attendanceType} | {attendee.status}</p></div><div><select value={attendee.status} onChange={e => updateAttendee(attendee, e.target.value)}><option>Registered</option><option>Attended</option><option>Cancelled</option></select> <button className="btn btn-secondary" onClick={() => removeAttendee(attendee)}>Remove</button></div></article>) : <small>No attendees registered yet.</small>}</div><form onSubmit={e => registerAttendee(e, event.id)} className="inline-form"><label>Name<input required value={attendeeForm.name} onChange={e => setAttendeeForm({ ...attendeeForm, name: e.target.value })} /></label><label>Email<input type="email" required value={attendeeForm.email} onChange={e => setAttendeeForm({ ...attendeeForm, email: e.target.value })} /></label><label>User ID<input value={attendeeForm.userId} onChange={e => setAttendeeForm({ ...attendeeForm, userId: e.target.value })} /></label><label>Attendance type<select value={attendeeForm.attendanceType} onChange={e => setAttendeeForm({ ...attendeeForm, attendanceType: e.target.value })}><option>Delegate</option><option>Presenter</option><option>Chair</option><option>Organiser</option></select></label><div><button className="btn btn-primary" type="submit">Register attendee</button></div></form></div>}
    </article>) : <div className="empty-state">No conference events recorded for this study.</div>}</div>
  </div>;
}
