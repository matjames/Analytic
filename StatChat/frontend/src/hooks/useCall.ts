import { useCallback, useEffect, useRef, useState } from 'react';
import { CallParticipant, CallSession, CallSignal } from '../types';
import { WS_URL, createCallSession, joinCallSession, leaveCallSession, endCallSession, fetchCallSession } from '../api/client';

export interface CallUser {
  id: string;
  name: string;
}

interface UseCallOptions {
  user?: Pick<CallUser, 'id' | 'name'> | null;
  onIncomingRemoteStream?: (stream: MediaStream, userId: string, userName: string) => void;
  onRemoteLeave?: (userId: string) => void;
  onCallEnded?: () => void;
}

// Only include TURN when a real server is configured. The old default
// (turn.example.com:3478) is a placeholder that silently breaks calls.
const turnUrl = (import.meta.env.VITE_TURN_URL ?? '').trim().replace(/^turn:/, '');
const turnUsername = (import.meta.env.VITE_TURN_USERNAME ?? '').trim();
const turnCredential = (import.meta.env.VITE_TURN_CREDENTIAL ?? '').trim();

const iceServers: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
];

if (turnUrl && turnUrl !== 'turn.example.com:3478' && turnUsername && turnCredential) {
  iceServers.push({ urls: `turn:${turnUrl}`, username: turnUsername, credential: turnCredential });
}

const RTC_CONFIG: RTCConfiguration = { iceServers };

export function useCall({ user, onIncomingRemoteStream, onRemoteLeave, onCallEnded }: UseCallOptions) {
  const [session, setSession] = useState<CallSession | null>(null);
  const [participants, setParticipants] = useState<CallParticipant[]>([]);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [micMuted, setMicMuted] = useState(false);
  const [cameraOff, setCameraOff] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const peerConnectionsRef = useRef<Record<string, RTCPeerConnection>>({});
  const remoteStreamsRef = useRef<Record<string, MediaStream>>({});
  const pendingCandidatesRef = useRef<Record<string, RTCIceCandidateInit[]>>({});
  const localStreamRef = useRef<MediaStream | null>(null);
  const sessionRef = useRef<CallSession | null>(null);
  const userRef = useRef(user);
  userRef.current = user;

  const getUserId = useCallback(() => userRef.current?.id ?? 'user-001', []);
  const getUserName = useCallback(() => userRef.current?.name ?? 'StatChat User', []);

  const sendSignal = useCallback((signal: Omit<CallSignal, 'from' | 'fromName'>) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
      action: 'signal',
      from: getUserId(),
      fromName: getUserName(),
      sessionId: signal.sessionId,
      type: signal.type,
      to: signal.to ?? '',
      payload: signal.payload ?? '',
    }));
  }, [getUserId, getUserName]);

  const createPeerConnection = useCallback((remoteUserId: string) => {
    if (peerConnectionsRef.current[remoteUserId]) {
      return peerConnectionsRef.current[remoteUserId];
    }
    const pc = new RTCPeerConnection(RTC_CONFIG);
    peerConnectionsRef.current[remoteUserId] = pc;

    const localStream = localStreamRef.current;
    if (localStream) {
      localStream.getTracks().forEach((track) => {
        try {
          pc.addTrack(track, localStream);
        } catch {
          // track already added
        }
      });
    }

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        const signalPayload = JSON.stringify(event.candidate);
        sendSignal({
          type: 'ice-candidate',
          sessionId: sessionRef.current?.id ?? '',
          to: remoteUserId,
          payload: signalPayload,
        });
      }
    };

    pc.ontrack = (event) => {
      const [stream] = event.streams;
      if (stream) {
        remoteStreamsRef.current[remoteUserId] = stream;
        const remoteUser = participants.find((p) => p.userId === remoteUserId);
        onIncomingRemoteStream?.(stream, remoteUserId, remoteUser?.userName ?? remoteUserId);
      }
    };

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed') {
        try {
          pc.close();
        } catch {
          // ignore
        }
        delete peerConnectionsRef.current[remoteUserId];
        delete remoteStreamsRef.current[remoteUserId];
        onRemoteLeave?.(remoteUserId);
      }
    };

    return pc;
  }, [sendSignal, participants, onIncomingRemoteStream, onRemoteLeave]);

  const handleSignal = useCallback(async (signal: CallSignal) => {
    if (signal.from === getUserId()) return;
    const remoteUserId = signal.from;
    const pc = createPeerConnection(remoteUserId);

    if (signal.type === 'offer') {
      await pc.setRemoteDescription(JSON.parse(signal.payload ?? '{}'));
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      sendSignal({
        type: 'answer',
        sessionId: signal.sessionId,
        to: remoteUserId,
        payload: JSON.stringify(answer),
      });
      // flush pending candidates
      const pending = pendingCandidatesRef.current[remoteUserId] ?? [];
      for (const candidate of pending) {
        try {
          await pc.addIceCandidate(candidate);
        } catch {
          // ignore
        }
      }
      delete pendingCandidatesRef.current[remoteUserId];
    } else if (signal.type === 'answer') {
      await pc.setRemoteDescription(JSON.parse(signal.payload ?? '{}'));
    } else if (signal.type === 'ice-candidate') {
      const candidate = JSON.parse(signal.payload ?? '{}');
      if (pc.remoteDescription) {
        try {
          await pc.addIceCandidate(candidate);
        } catch {
          // ignore
        }
      } else {
        pendingCandidatesRef.current[remoteUserId] = [
          ...(pendingCandidatesRef.current[remoteUserId] ?? []),
          candidate,
        ];
      }
    }
  }, [createPeerConnection, getUserId, sendSignal]);

  const connectWs = useCallback(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;
    ws.onopen = () => {
      const current = sessionRef.current;
      if (current) {
        ws.send(JSON.stringify({
          action: 'join-call',
          sessionId: current.id,
          userId: getUserId(),
          userName: getUserName(),
        }));
      }
    };
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.event === 'call-signal' && data.payload) {
          handleSignal(data.payload);
        } else if (data.event === 'call-participant-joined' && data.payload && data.payload.userId !== getUserId()) {
          setParticipants((prev) => {
            if (prev.some((p) => p.userId === data.payload.userId)) return prev;
            return [...prev, data.payload];
          });
        } else if (data.event === 'call-participant-left' && data.payload) {
          setParticipants((prev) => prev.filter((p) => p.userId !== data.payload.userId));
          onRemoteLeave?.(data.payload.userId);
        } else if (data.event === 'call-ended') {
          onCallEnded?.();
        }
      } catch {
        // ignore parse errors
      }
    };
    ws.onclose = () => {
      wsRef.current = null;
    };
    return ws;
  }, [getUserId, getUserName, handleSignal, onRemoteLeave, onCallEnded]);

  const startLocalMedia = useCallback(async (kind: 'voice' | 'video') => {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: kind === 'video',
      audio: true,
    });
    localStreamRef.current = stream;
    setLocalStream(stream);
    return stream;
  }, []);

  const startCall = useCallback(async (kind: 'voice' | 'video', roomName?: string, conversationId?: string) => {
    setConnecting(true);
    setError(null);
    try {
      const currentUser = userRef.current;
      const createdSession = await createCallSession({
        kind,
        roomName,
        conversationId,
        hostId: currentUser?.id,
        hostName: currentUser?.name,
      });
      const stream = await startLocalMedia(kind);
      sessionRef.current = createdSession;
      setSession(createdSession);
      setParticipants([{
        id: `${createdSession.id}-host`,
        sessionId: createdSession.id,
        userId: getUserId(),
        userName: getUserName(),
        role: 'host',
        joinedAt: new Date().toISOString(),
      }]);
      const ws = connectWs();
      ws.onopen = () => {
        ws.send(JSON.stringify({
          action: 'join-call',
          sessionId: createdSession.id,
          userId: getUserId(),
          userName: getUserName(),
        }));
      };
      void stream;
      return createdSession;
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to start call';
      setError(message);
      throw e;
    } finally {
      setConnecting(false);
    }
  }, [startLocalMedia, connectWs, getUserId, getUserName]);

  const initiateCall = useCallback(async (targetUserId: string) => {
    const current = sessionRef.current;
    if (!current) return;
    const pc = createPeerConnection(targetUserId);
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    sendSignal({
      type: 'offer',
      sessionId: current.id,
      to: targetUserId,
      payload: JSON.stringify(offer),
    });
  }, [createPeerConnection, sendSignal]);

  const joinCall = useCallback(async (sessionId: string) => {
    setConnecting(true);
    setError(null);
    try {
      const currentUser = userRef.current;
      await joinCallSession(sessionId, currentUser?.id, currentUser?.name);

      // Load the session details and any participants who are already in the call.
      let existingSession: CallSession = { ...(sessionRef.current ?? { id: sessionId } as CallSession), id: sessionId };
      let existingParticipants: CallParticipant[] = [];
      try {
        const data = await fetchCallSession(sessionId);
        existingSession = { ...existingSession, ...data.session };
        existingParticipants = data.participants ?? [];
      } catch {
        // Non-fatal: fall back to whatever we already know.
      }
      sessionRef.current = existingSession;
      setSession(existingSession);
      setParticipants(existingParticipants);

      // Start local media so this peer has tracks to offer.
      const kind = existingSession.kind === 'voice' ? 'voice' : 'video';
      const stream = await startLocalMedia(kind);

      const ws = connectWs();
      const currentId = getUserId();
      const currentName = getUserName();
      ws.onopen = () => {
        ws.send(JSON.stringify({
          action: 'join-call',
          sessionId,
          userId: currentId,
          userName: currentName,
        }));
        // After joining, initiate WebRTC offers to everyone already present —
        // otherwise no peer connection is ever created (audit 2.19).
        for (const participant of existingParticipants) {
          if (participant.userId !== currentId) {
            initiateCall(participant.userId).catch(() => {});
          }
        }
      };
      return existingSession;
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to join call';
      setError(message);
      throw e;
    } finally {
      setConnecting(false);
    }
  }, [connectWs, getUserId, getUserName, startLocalMedia, initiateCall]);

  const toggleMute = useCallback(() => {
    setMicMuted((prev) => {
      const next = !prev;
      localStreamRef.current?.getAudioTracks().forEach((track) => {
        track.enabled = !next;
      });
      return next;
    });
  }, []);

  const toggleCamera = useCallback(() => {
    setCameraOff((prev) => {
      const next = !prev;
      localStreamRef.current?.getVideoTracks().forEach((track) => {
        track.enabled = !next;
      });
      return next;
    });
  }, []);

  const hangUp = useCallback(async () => {
    const current = sessionRef.current;
    if (current) {
      try {
        await leaveCallSession(current.id, getUserId());
      } catch {
        // ignore
      }
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          action: 'leave-call',
          sessionId: current.id,
          userId: getUserId(),
        }));
      }
    }
    // Close all peer connections
    Object.values(peerConnectionsRef.current).forEach((pc) => {
      try {
        pc.close();
      } catch {
        // ignore
      }
    });
    peerConnectionsRef.current = {};
    remoteStreamsRef.current = {};
    localStreamRef.current?.getTracks().forEach((track) => track.stop());
    localStreamRef.current = null;
    setLocalStream(null);
    setSession(null);
    setParticipants([]);
    setMicMuted(false);
    setCameraOff(false);
    wsRef.current?.close();
    wsRef.current = null;
  }, [getUserId]);

  const endCall = useCallback(async () => {
    const current = sessionRef.current;
    if (current) {
      try {
        await endCallSession(current.id);
      } catch {
        // ignore
      }
    }
    await hangUp();
  }, [hangUp]);

  useEffect(() => {
    return () => {
      Object.values(peerConnectionsRef.current).forEach((pc) => {
        try {
          pc.close();
        } catch {
          // ignore
        }
      });
      localStreamRef.current?.getTracks().forEach((track) => track.stop());
      wsRef.current?.close();
    };
  }, []);

  return {
    session,
    participants,
    localStream,
    micMuted,
    cameraOff,
    connecting,
    error,
    startCall,
    joinCall,
    initiateCall,
    toggleMute,
    toggleCamera,
    hangUp,
    endCall,
  };
}
