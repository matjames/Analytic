import { useEffect, useState } from 'react';
import type { User } from '../types';
import type { CalendarSubView } from '../App';
import {
  fetchMeetings,
  fetchMeetingRooms,
  fetchMeetingRecordings,
  type Meeting,
  type MeetingRoom,
  type MeetingRecording,
} from '../api/client';
import styles from './CalendarPanel.module.css';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
  activeCalendarView: CalendarSubView;
  nativeMode: boolean;
}

type NativeMeeting = Meeting;
type MeetingRoomItem = MeetingRoom;

const seedMeetings: NativeMeeting[] = [];
const seedRooms: MeetingRoomItem[] = [];
const seedRecordings: MeetingRecording[] = [];

export default function CalendarPanel({ user, theme, isMobile, activeCalendarView, nativeMode }: Props) {
  const [meetings, setMeetings] = useState<NativeMeeting[]>([]);
  const [rooms, setRooms] = useState<MeetingRoomItem[]>([]);
  const [recordings, setRecordings] = useState<MeetingRecording[]>([]);
  const isDark = theme === 'dark';
  const bg = isDark ? '#0a2b45' : '#ffffff';
  const textColor = isDark ? '#e8eef4' : '#1a1a1a';
  const borderColor = isDark ? '#6b7280' : '#e5e7eb';

  useEffect(() => {
    fetchMeetings().then(setMeetings).catch(() => {});
    fetchMeetingRooms().then(setRooms).catch(() => {});
    fetchMeetingRecordings().then(setRecordings).catch(() => {});
  }, []);

  const renderOverview = () => (
    <div>
      <div className={styles.statsRow}>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{meetings.filter((m) => m.status === 'upcoming').length}</div>
          <div className={styles.statLabel}>Upcoming Meetings</div>
        </div>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{meetings.filter((m) => m.status === 'live').length}</div>
          <div className={styles.statLabel}>Live Now</div>
        </div>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{rooms.filter((r) => r.status === 'available').length}</div>
          <div className={styles.statLabel}>Available Rooms</div>
        </div>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{recordings.length}</div>
          <div className={styles.statLabel}>Recent Recordings</div>
        </div>
      </div>

      <h3 className={styles.sectionHeader} style={{ fontSize: 16 }}>Today's Schedule</h3>
      {meetings.filter((m) => m.status === 'live').map((meeting) => (
        <div key={meeting.id} className={styles.meetingCard} style={{ background: bg, borderColor }}>
          <div style={{ flex: 1 }}>
            <div className={styles.liveBadge}>🔴 LIVE NOW</div>
            <div className={styles.meetingTitle}>{meeting.title}</div>
            <div className={styles.meetingMeta}>
              <span>👥 {meeting.participants} participants</span>
              <span>🏠 {meeting.room}</span>
              <span>Host: {meeting.host}</span>
            </div>
          </div>
          <button type="button" className={styles.joinBtn}>Join Now</button>
        </div>
      ))}

      {meetings.filter((m) => m.status === 'upcoming').slice(0, 3).map((meeting) => (
        <div key={meeting.id} className={styles.meetingCard} style={{ background: bg, borderColor }}>
          <div style={{ flex: 1 }}>
            <div className={styles.meetingTitle}>{meeting.title}</div>
            <div className={styles.meetingMeta}>
              <span>📅 {meeting.date} · 🕐 {meeting.time}</span>
              <span>⏱ {meeting.duration}</span>
              <span>👥 {meeting.participants}</span>
            </div>
          </div>
          <button type="button" className={styles.scheduleBtn}>View</button>
        </div>
      ))}
    </div>
  );

  const renderSchedule = () => (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 className={styles.sectionHeader} style={{ margin: 0 }}>Upcoming Meetings</h2>
        <button type="button" className={styles.primaryBtn}>+ Schedule Meeting</button>
      </div>
      <p className={styles.subheader}>Native StatChat meetings — no external accounts needed</p>
      {meetings.map((meeting) => (
        <div key={meeting.id} className={styles.meetingCard} style={{ background: bg, borderColor }}>
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
              <span className={meeting.status === 'live' ? styles.liveBadge : styles.upcomingBadge}>
                {meeting.status === 'live' ? '🔴 LIVE' : 'Upcoming'}
              </span>
              <span style={{ fontSize: 12, opacity: 0.6 }}>{meeting.room}</span>
            </div>
            <div className={styles.meetingTitle}>{meeting.title}</div>
            <div className={styles.meetingMeta}>
              <span>📅 {meeting.date}</span>
              <span>🕐 {meeting.time}</span>
              <span>⏱ {meeting.duration}</span>
              <span>👥 {meeting.participants} participants</span>
              <span>Host: {meeting.host}</span>
            </div>
          </div>
          <button type="button" className={styles.joinBtn}>{meeting.status === 'live' ? 'Join' : 'Join'}</button>
        </div>
      ))}
    </div>
  );

  const renderRooms = () => (
    <div>
      <h2 className={styles.sectionHeader}>Meeting Rooms</h2>
      <p className={styles.subheader}>Persistent StatChat meeting rooms with secure access codes</p>
      <div className={styles.roomGrid}>
        {rooms.map((room) => (
          <div key={room.id} className={styles.roomCard} style={{ background: bg, borderColor }}>
            <div className={styles.roomIcon}>{room.status === 'available' ? '🟢' : '🔴'}</div>
            <div className={styles.roomName}>{room.name}</div>
            <div className={styles.roomCapacity}>Capacity: {room.capacity}</div>
            <div style={{ fontSize: 12, opacity: 0.7, fontFamily: 'monospace' }}>{room.url}</div>
            <div style={{ fontSize: 12, opacity: 0.7 }}>Code: {room.password}</div>
            <button
              type="button"
              className={styles.joinBtn}
              disabled={room.status === 'in-use'}
              style={{ opacity: room.status === 'in-use' ? 0.5 : 1 }}
            >
              {room.status === 'in-use' ? 'In Use' : 'Enter Room'}
            </button>
          </div>
        ))}
      </div>
    </div>
  );

  const renderRecordings = () => (
    <div>
      <h2 className={styles.sectionHeader}>Recordings</h2>
      <p className={styles.subheader}>Access your past StatChat meeting recordings</p>
      {recordings.map((rec) => (
        <div key={rec.id} className={styles.meetingCard} style={{ background: bg, borderColor }}>
          <div style={{ fontSize: 28, marginRight: 16 }}>🎬</div>
          <div style={{ flex: 1 }}>
            <div className={styles.meetingTitle}>{rec.title}</div>
            <div className={styles.meetingMeta}>
              <span>📅 {rec.date}</span>
              <span>⏱ {rec.duration}</span>
              <span>💾 {rec.size}</span>
            </div>
          </div>
          <button type="button" className={styles.scheduleBtn}>Play</button>
        </div>
      ))}
    </div>
  );

  const renderContent = () => {
    switch (activeCalendarView) {
      case 'overview': return renderOverview();
      case 'schedule': return renderSchedule();
      case 'rooms': return renderRooms();
      case 'recordings': return renderRecordings();
      default: return renderOverview();
    }
  };

  return (
    <section className={styles.calendarShell} style={{ color: textColor }}>
      <div className={styles.calendarScroll}>
        {renderContent()}
      </div>
    </section>
  );
}