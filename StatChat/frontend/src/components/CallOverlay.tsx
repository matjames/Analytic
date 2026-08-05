import { useEffect, useRef, useState } from 'react';
import { CallParticipant, CallSession, CallRecording } from '../types';
import { fetchCallRecordings, uploadCallRecording } from '../api/client';
import RecordingPlayer from './RecordingPlayer';

interface Props {
  session: CallSession;
  participants: CallParticipant[];
  localStream: MediaStream | null;
  remoteStreams: Record<string, MediaStream>;
  micMuted: boolean;
  cameraOff: boolean;
  connecting: boolean;
  currentUserName?: string;
  onToggleMute: () => void;
  onToggleCamera: () => void;
  onHangUp: () => void;
  onEndCall?: () => void;
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

export default function CallOverlay({
  session,
  participants,
  localStream,
  remoteStreams,
  micMuted,
  cameraOff,
  connecting,
  currentUserName,
  onToggleMute,
  onToggleCamera,
  onHangUp,
  onEndCall,
}: Props) {
  const localVideoRef = useRef<HTMLVideoElement | null>(null);
  const remoteVideoRefs = useRef<Record<string, HTMLVideoElement | null>>({});
  const [timer, setTimer] = useState(0);
  const [isRecording, setIsRecording] = useState(false);
  const [mediaRecorder, setMediaRecorder] = useState<MediaRecorder | null>(null);
  const [showRecordings, setShowRecordings] = useState(false);
  const [recordings, setRecordings] = useState<CallRecording[]>([]);
  const [playbackUrl, setPlaybackUrl] = useState<string | null>(null);
  const recordedChunksRef = useRef<Blob[]>([]);

  const isHost = participants.some(
    (p) => p.role === 'host' && p.userId === participants[0]?.userId
  );

  // Timer
  useEffect(() => {
    const interval = setInterval(() => setTimer((t) => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  // Attach local stream
  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream;
    }
  }, [localStream]);

  // Attach remote streams
  useEffect(() => {
    Object.entries(remoteStreams).forEach(([userId, stream]) => {
      const videoEl = remoteVideoRefs.current[userId];
      if (videoEl && videoEl.srcObject !== stream) {
        videoEl.srcObject = stream;
      }
    });
  }, [remoteStreams]);

  const startRecording = async () => {
    if (!localStream) return;
    const streamToRecord = new MediaStream();
    // Add local audio to recording
    localStream.getAudioTracks().forEach((track) => streamToRecord.addTrack(track));
    // Add all remote audio/video
    Object.values(remoteStreams).forEach((rs) => {
      rs.getTracks().forEach((track) => streamToRecord.addTrack(track));
    });
    try {
      const recorder = new MediaRecorder(streamToRecord);
      recordedChunksRef.current = [];
      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) recordedChunksRef.current.push(e.data);
      };
      recorder.onstop = async () => {
        const blob = new Blob(recordedChunksRef.current, { type: 'video/webm' });
        const formData = new FormData();
        formData.append('file', blob, `recording-${session.id}-${Date.now()}.webm`);
        formData.append('title', `Meeting recording ${new Date().toLocaleString()}`);
        try {
          await uploadCallRecording(session.id, formData);
        } catch {
          // ignore
        }
      };
      recorder.start();
      setMediaRecorder(recorder);
      setIsRecording(true);
    } catch {
      // recording not supported
    }
  };

  const stopRecording = () => {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.stop();
    }
    setIsRecording(false);
    setMediaRecorder(null);
  };

  const loadRecordings = async () => {
    try {
      const recs = await fetchCallRecordings(session.id);
      setRecordings(recs);
    } catch {
      // ignore
    }
  };

  const remoteUserIds = Object.keys(remoteStreams);
  const hasRemoteVideo = remoteUserIds.length > 0;
  const participantNames = participants.map((p) => p.userName).join(', ');

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      zIndex: 100,
      background: '#0a0f1a',
      display: 'flex',
      flexDirection: 'column',
      color: '#fff',
    }}>
      {/* Status header */}
      <div style={{
        padding: '12px 24px',
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexShrink: 0,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontWeight: 700, fontSize: 18 }}>{session.roomName}</span>
          <span style={{
            padding: '3px 10px',
            borderRadius: 999,
            fontSize: 12,
            fontWeight: 600,
            background: '#22c55e',
            color: '#000',
          }}>
            {session.kind === 'video' ? '📹 Video' : '🎤 Voice'}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span style={{ fontVariantNumeric: 'tabular-nums', fontSize: 16 }}>
            {formatDuration(timer)}
          </span>
          <span style={{ fontSize: 13, opacity: 0.7 }}>
            {participants.length} participant{participants.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      {connecting && (
        <div style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 18,
          opacity: 0.7,
        }}>
          Connecting to call...
        </div>
      )}

      {!connecting && (
        <div style={{
          flex: 1,
          display: 'flex',
          flexDirection: hasRemoteVideo ? 'row' : 'column',
          gap: 12,
          padding: 16,
          overflow: 'auto',
          justifyContent: 'center',
          alignItems: 'center',
        }}>
          {/* Local video (self) */}
          <div style={{
            width: hasRemoteVideo ? 240 : 360,
            maxHeight: hasRemoteVideo ? 180 : 320,
            borderRadius: 12,
            overflow: 'hidden',
            background: '#1a1f2e',
            position: 'relative',
            border: '2px solid rgba(255,255,255,0.1)',
            flexShrink: 0,
          }}>
            <video
              ref={localVideoRef}
              autoPlay
              playsInline
              muted
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                transform: 'scaleX(-1)',
                display: cameraOff || !localStream ? 'none' : 'block',
              }}
            />
            {(!localStream || cameraOff) && (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                height: 120,
                fontSize: 40,
                opacity: 0.5,
              }}>
                🚀
              </div>
            )}
            <div style={{
              position: 'absolute',
              bottom: 6,
              left: 8,
              fontSize: 11,
              background: 'rgba(0,0,0,0.6)',
              padding: '2px 8px',
              borderRadius: 6,
            }}>
              You {micMuted ? '🔇' : ''}
            </div>
          </div>

          {/* Remote videos */}
          {hasRemoteVideo ? (
            remoteUserIds.map((userId) => (
              <div key={userId} style={{
                width: 240,
                maxHeight: 180,
                borderRadius: 12,
                overflow: 'hidden',
                background: '#1a1f2e',
                position: 'relative',
                border: '2px solid rgba(255,255,255,0.1)',
                flexShrink: 0,
              }}>
                <video
                  ref={(el) => { remoteVideoRefs.current[userId] = el; }}
                  autoPlay
                  playsInline
                  style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
                <div style={{
                  position: 'absolute',
                  bottom: 6,
                  left: 8,
                  fontSize: 11,
                  background: 'rgba(0,0,0,0.6)',
                  padding: '2px 8px',
                  borderRadius: 6,
                }}>
                  {participants.find((p) => p.userId === userId)?.userName ?? userId}
                </div>
              </div>
            ))
          ) : (
            <div style={{
              fontSize: 24,
              opacity: 0.5,
              textAlign: 'center',
              padding: '20px 0',
            }}>
              {participantNames || 'No other participants'}
              {session.kind === 'voice' && (
                <div style={{ fontSize: 14, marginTop: 8, opacity: 0.7 }}>
                  Voice call in progress
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Controls */}
      <div style={{
        padding: '16px 24px',
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        gap: 16,
        flexShrink: 0,
      }}>
        <button
          type="button"
          onClick={onToggleMute}
          style={{
            width: 56,
            height: 56,
            borderRadius: '50%',
            border: 'none',
            background: micMuted ? '#ef4444' : '#ffffff22',
            color: '#fff',
            fontSize: 24,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            transition: 'background 0.15s',
          }}
          title={micMuted ? 'Unmute' : 'Mute'}
        >
          {micMuted ? '🔇' : '🎤'}
        </button>

        {session.kind === 'video' && (
          <button
            type="button"
            onClick={onToggleCamera}
            style={{
              width: 56,
              height: 56,
              borderRadius: '50%',
              border: 'none',
              background: cameraOff ? '#ef4444' : '#ffffff22',
              color: '#fff',
              fontSize: 24,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
            title={cameraOff ? 'Turn on camera' : 'Turn off camera'}
          >
            {cameraOff ? '📷' : '🎥'}
          </button>
        )}

        <button
          type="button"
          onClick={isRecording ? stopRecording : async () => {
            if (!isRecording) {
              await startRecording();
            }
          }}
          style={{
            width: 56,
            height: 56,
            borderRadius: '50%',
            border: 'none',
            background: isRecording ? '#ef4444' : '#ffffff22',
            color: '#fff',
            fontSize: 24,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title={isRecording ? 'Stop recording' : 'Record'}
        >
          {isRecording ? '⏹️' : '⏺️'}
        </button>

        <button
          type="button"
          onClick={onHangUp}
          style={{
            width: 64,
            height: 64,
            borderRadius: '50%',
            border: 'none',
            background: '#ef4444',
            color: '#fff',
            fontSize: 28,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title="Leave call"
        >
          📞
        </button>

        <button
          type="button"
          onClick={() => {
            setShowRecordings((prev) => !prev);
            if (!showRecordings) loadRecordings();
          }}
          style={{
            width: 56,
            height: 56,
            borderRadius: '50%',
            border: 'none',
            background: showRecordings ? '#165c92' : '#ffffff22',
            color: '#fff',
            fontSize: 24,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title="Recordings"
        >
          🎬
        </button>

        {onEndCall && (
          <button
            type="button"
            onClick={onEndCall}
            style={{
              width: 56,
              height: 56,
              borderRadius: '50%',
              border: 'none',
              background: '#dc2626',
              color: '#fff',
              fontSize: 16,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontWeight: 700,
            }}
            title="End call for all"
          >
            ⛔
          </button>
        )}
      </div>

      {/* Recordings panel */}
      {showRecordings && recordings.length > 0 && (
        <div style={{
          position: 'absolute',
          right: 24,
          top: 80,
          width: 300,
          maxHeight: 400,
          overflowY: 'auto',
          background: '#1a1f2e',
          borderRadius: 16,
          padding: 12,
          border: '1px solid rgba(255,255,255,0.1)',
        }}>
          <strong style={{ display: 'block', marginBottom: 8, fontSize: 14 }}>
            Recordings
          </strong>
          {recordings.map((rec) => (
            <div
              key={rec.id}
              style={{
                padding: '8px 10px',
                borderRadius: 10,
                background: 'rgba(255,255,255,0.05)',
                marginBottom: 6,
                cursor: 'pointer',
                fontSize: 13,
              }}
              onClick={() => setPlaybackUrl(rec.url)}
            >
              <div style={{ fontWeight: 600 }}>{rec.title}</div>
              <div style={{ opacity: 0.6, fontSize: 11, marginTop: 2 }}>
                {rec.duration ? `${rec.duration} · ` : ''}
                {new Date(rec.createdAt).toLocaleString()}
              </div>
            </div>
          ))}
        </div>
      )}

      {playbackUrl && (
        <RecordingPlayer url={playbackUrl} onClose={() => setPlaybackUrl(null)} />
      )}
    </div>
  );
}
