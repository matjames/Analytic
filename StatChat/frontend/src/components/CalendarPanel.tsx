import { useEffect, useState } from 'react';
import type { User, CallSession, CallRecording } from '../types';
import type { CalendarSubView } from '../App';
import {
  fetchMeetings,
  fetchMeetingRooms,
  fetchMeetingRecordings,
  fetchMeetingCallSession,
  fetchCallRecordings,
  createCallSession,
  type Meeting,
  type MeetingRoom,
  type MeetingRecording,
} from '../api/client';
import { useCall } from '../hooks/useCall';
import CallOverlay from './CallOverlay';
import RecordingPlayer from './RecordingPlayer';
import styles from './CalendarPanel.module.css';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
  activeCalendarView: CalendarSubView;
  nativeMode: boolean;
  onJoinMeeting?: (meetingId: string) => Promise<void>;
}

export default function CalendarPanel({ user, theme, isMobile, activeCalendarView, nativeMode, onJoinMeeting }: Props) {
  const [meetings, setMeetings] = useState<Meeting[]>([]);
  const [rooms, setRooms] = useState<MeetingRoom[]>([]);
  const [recordings, setRecordings] = useState<MeetingRecording[]>([]);
  const [callActive, setCallActive] = useState(false);
  const [activeSession, setActiveSession] = useState<CallSession | null>(null);
  const [remoteStreams, setRemoteStreams] = useState<Record<string, MediaStream>>({});
  const [sessionRecordings, setSessionRecordings] = useState<CallRecording[]>([]);
  const [playbackUrl, setPlaybackUrl] = useState<string | null>(null);
  const [joined, setJoined] = useState(false);
  const isDark = theme === 'dark';
  const bg = isDark ? '#0a2b45' : '#ffffff';
  const textColor = isDark ? '#e8eef4' : '#1a1a1a';
  const borderColor = isDark ? '#6b7280' : '#e5e7eb';

  const {
    session,
    participants,
    localStream,
    micMuted,
    cameraOff,
    connecting,
    joinCall,
    toggleMute,
    toggleCamera,
    hangUp,
    endCall,
  } = useCall({
    user,
    onIncomingRemoteStream: (stream, userId) => {
      setRemoteStreams((prev) => ({ ...prev, [userId]: stream }));
    },
    onRemoteLeave: (userId) => {
      setRemoteStreams((prev) => {
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    },
    onCallEnded: () => {
      setCallActive(false);
      setRemoteStreams({});
      setActiveSession(null);
      setJoined(false);
    },
  });

  useEffect(() => {
    fetchMeetings().then(setMeetings).catch(() => {});
    fetchMeetingRooms().then(setRooms).catch(() => {});
    fetchMeetingRecordings().then(setRecordings).catch(() => {});
  }, []);

  useEffect(() => {
    if (session && joined && session.id) {
      fetchCallRecordings(session.id)
        .then((recs) => setSessionRecordings(recs))
        .catch(() => {});
    }
  }, [session, joined]);

  const handleJoinMeeting = async (meeting: Meeting) => {
    try {
      setPlaybackUrl(null);
      if (onJoinMeeting) {
        await onJoinMeeting(meeting.id);
        return;
      }
      const callSession = await fetchMeetingCallSession(meeting.id);
      const joinedSession = await joinCall(callSession.id);
      setActiveSession(joinedSession);
      setJoined(true);
      setCallActive(true);
    } catch {
      setCallActive(false);
      setJoined(false);
    }
  };

  const handleJoinRoom = async (room: MeetingRoom) => {
    try {
      setPlaybackUrl(null);
      const callSession = await createCallSession({
        kind: 'video',
        roomName: room.name,
      });
      const joinedSession = await joinCall(callSession.id);
      setActiveSession(joinedSession);
      setJoined(true);
      setCallActive(true);
    } catch {
      setCallActive(false);
      setJoined(false);
    }
  };

  const handlePlayRecording = (url?: string) => {
    if (url) {
      setPlaybackUrl(url);
    } else if (sessionRecordings.length > 0) {
      setPlaybackUrl(sessionRecordings[0].url);
    }
  };

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
          <button
            type="button"
            className={styles.joinBtn}
            onClick={() => handleJoinMeeting(meeting)}
            disabled={connecting}
          >
            {connecting ? 'Joining…' : 'Join'}
          </button>
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
          <button type="button" className={styles.scheduleBtn} onClick={() => handleJoinMeeting(meeting)} disabled={connecting}>
            {connecting ? 'Joining…' : 'Join Now'}
          </button>
        </div>
      ))}
    </div>
  );

  const renderSchedule = () => (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 className={styles.sectionHeader} style={{ margin: 0 }}>Upcoming Meetings</h2>
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
          <button
            type="button"
            className={styles.joinBtn}
            onClick={() => handleJoinMeeting(meeting)}
            disabled={connecting}
          >
            {connecting ? 'Joining…' : meeting.status === 'live' ? 'Join Live' : 'Join Now'}
          </button>
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
              onClick={() => handleJoinRoom(room)}
              disabled={connecting}
            >
              {connecting ? 'Joining…' : 'Join Room'}
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
      {playbackUrl && (
        <RecordingPlayer url={playbackUrl} onClose={() => setPlaybackUrl(null)} />
      )}
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
          <button type="button" className={styles.scheduleBtn} onClick={() => handlePlayRecording(rec.url)}>
            Play
          </button>
        </div>
      ))}
      {sessionRecordings.length > 0 && (
        <h3 className={styles.sectionHeader} style={{ marginTop: 24 }}>This Call's Recordings</h3>
      )}
      {sessionRecordings.map((rec) => (
        <div key={rec.id} className={styles.meetingCard} style={{ background: bg, borderColor }}>
          <div style={{ fontSize: 28, marginRight: 16 }}>🎬</div>
          <div style={{ flex: 1 }}>
            <div className={styles.meetingTitle}>{rec.title}</div>
            <div className={styles.meetingMeta}>
              <span>💾 {rec.size} bytes</span>
            </div>
          </div>
          <button type="button" className={styles.scheduleBtn} onClick={() => setPlaybackUrl(rec.url)}>
            Play
          </button>
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

  // Force-refresh session recordings while in a live call
  useEffect(() => {
    if (!callActive || !session?.id) return;
    const interval = setInterval(() => {
      fetchCallRecordings(session.id)
        .then((recs) => setSessionRecordings(recs))
        .catch(() => {});
    }, 15000);
    return () => clearInterval(interval);
  }, [callActive, session?.id]);

  return (
    <section className={styles.calendarShell} style={{ color: textColor }}>
      <div className={styles.calendarScroll}>
        {renderContent()}
      </div>

      {callActive && session && (
        <CallOverlay
          session={session}
          participants={participants}
          localStream={localStream}
          remoteStreams={remoteStreams}
          micMuted={micMuted}
          cameraOff={cameraOff}
          connecting={connecting}
          currentUserName={user?.name}
          onToggleMute={toggleMute}
          onToggleCamera={toggleCamera}
          onHangUp={() => {
            setCallActive(false);
            setJoined(false);
            setRemoteStreams({});
            hangUp();
          }}
          onEndCall={() => {
            setCallActive(false);
            setJoined(false);
            setRemoteStreams({});
            endCall();
          }}
        />
      )}
    </section>
  );
}