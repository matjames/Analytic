import { useCallback, useEffect, useRef, useState } from 'react';
import { CallParticipant, CallSession, CallSignal, type CallConnectionState, type CallQuality, type CallQualitySample } from '../types';
import { getWebSocketURL, getWebSocketProtocols, createCallSession, joinCallSession, leaveCallSession, endCallSession, fetchCallSession, reportCallQuality } from '../api/client';

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
const turnUrls = (import.meta.env.VITE_TURN_URL ?? '').split(',').map((value: string) => value.trim()).filter(Boolean);
const turnUsername = (import.meta.env.VITE_TURN_USERNAME ?? '').trim();
const turnCredential = (import.meta.env.VITE_TURN_CREDENTIAL ?? '').trim();

const iceServers: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
];

if (turnUrls.length > 0 && turnUsername && turnCredential) {
	iceServers.push({ urls: turnUrls.map((url: string) => /^(turn|turns):/.test(url) ? url : `turn:${url}`), username: turnUsername, credential: turnCredential });
}

const RTC_CONFIG: RTCConfiguration = { iceServers };

export function useCall({ user, onIncomingRemoteStream, onRemoteLeave, onCallEnded }: UseCallOptions) {
  const [session, setSession] = useState<CallSession | null>(null);
  const [participants, setParticipants] = useState<CallParticipant[]>([]);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [micMuted, setMicMuted] = useState(false);
  const [cameraOff, setCameraOff] = useState(false);
  const [screenStream, setScreenStream] = useState<MediaStream | null>(null);
  const [isScreenSharing, setIsScreenSharing] = useState(false);
  const [remoteScreenSharers, setRemoteScreenSharers] = useState<Record<string, string>>({});
  const [connecting, setConnecting] = useState(false);
  const [connectionState, setConnectionState] = useState<CallConnectionState>('connecting');
  const [quality, setQuality] = useState<CallQuality>('unknown');
  const [qualityMetrics, setQualityMetrics] = useState<Pick<CallQualitySample, 'rttMs' | 'jitterMs' | 'packetLossPct' | 'bitrateKbps'> | null>(null);
  const [pendingMuteRequest, setPendingMuteRequest] = useState<{ fromId: string; fromName: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const peerConnectionsRef = useRef<Record<string, RTCPeerConnection>>({});
  const remoteStreamsRef = useRef<Record<string, MediaStream>>({});
  const pendingCandidatesRef = useRef<Record<string, RTCIceCandidateInit[]>>({});
  const localStreamRef = useRef<MediaStream | null>(null);
  const screenStreamRef = useRef<MediaStream | null>(null);
  const screenSendersRef = useRef<Record<string, RTCRtpSender>>({});
  const isScreenSharingRef = useRef(false);
  const sessionRef = useRef<CallSession | null>(null);
  const participantsRef = useRef<CallParticipant[]>([]);
  const connectWsRef = useRef<() => WebSocket>();
  const initiateCallRef = useRef<(targetUserId: string, iceRestart?: boolean) => Promise<void>>();
  const hangUpRef = useRef<() => Promise<void>>();
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const intentionalCloseRef = useRef(false);
  const peerRecoveryTimersRef = useRef<Record<string, number>>({});
  const previousInboundStatsRef = useRef<Record<string, { bytes: number; timestamp: number }>>({});
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
    const activeScreenStream = screenStreamRef.current;
    if (localStream) {
      localStream.getTracks().forEach((track) => {
        if (track.kind === 'video' && activeScreenStream) return;
        try {
          pc.addTrack(track, localStream);
        } catch {
          // track already added
        }
      });
    }
    if (activeScreenStream) {
      const screenTrack = activeScreenStream.getVideoTracks()[0];
      if (screenTrack) {
        screenSendersRef.current[remoteUserId] = pc.addTrack(screenTrack, activeScreenStream);
      }
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
      const clearRecoveryTimer = () => {
        const timer = peerRecoveryTimersRef.current[remoteUserId];
        if (timer !== undefined) window.clearTimeout(timer);
        delete peerRecoveryTimersRef.current[remoteUserId];
      };
      if (pc.connectionState === 'connected') {
        clearRecoveryTimer();
        setConnectionState('connected');
        return;
      }
      if (pc.connectionState !== 'disconnected' && pc.connectionState !== 'failed') return;
      setConnectionState(navigator.onLine ? 'reconnecting' : 'offline');
      if (peerRecoveryTimersRef.current[remoteUserId] !== undefined) return;
      const delay = pc.connectionState === 'failed' ? 0 : 4000;
      peerRecoveryTimersRef.current[remoteUserId] = window.setTimeout(async () => {
        delete peerRecoveryTimersRef.current[remoteUserId];
        if (pc.connectionState === 'connected' || pc.connectionState === 'closed') return;
        try {
          pc.restartIce();
          await initiateCallRef.current?.(remoteUserId, true);
        } catch {
          // A WebSocket reconnect will retry negotiation when signaling returns.
        }
        peerRecoveryTimersRef.current[remoteUserId] = window.setTimeout(() => {
          delete peerRecoveryTimersRef.current[remoteUserId];
          if (pc.connectionState === 'connected') return;
          try { pc.close(); } catch { /* already closed */ }
          delete peerConnectionsRef.current[remoteUserId];
          delete remoteStreamsRef.current[remoteUserId];
          delete screenSendersRef.current[remoteUserId];
          onRemoteLeave?.(remoteUserId);
        }, 12000);
      }, delay);
    };

    return pc;
  }, [sendSignal, participants, onIncomingRemoteStream, onRemoteLeave]);

  const handleSignal = useCallback(async (signal: CallSignal) => {
    if (signal.from === getUserId()) return;
    const remoteUserId = signal.from;
    if (signal.type === 'mute-requested') {
      setPendingMuteRequest({ fromId: remoteUserId, fromName: signal.fromName ?? remoteUserId });
      return;
    }
    if (signal.type === 'mute-accepted' || signal.type === 'mute-declined') return;
    if (signal.type === 'screen-share-started') {
      setRemoteScreenSharers((current) => ({ ...current, [remoteUserId]: signal.fromName ?? remoteUserId }));
      return;
    }
    if (signal.type === 'screen-share-stopped') {
      setRemoteScreenSharers((current) => {
        const next = { ...current };
        delete next[remoteUserId];
        return next;
      });
      return;
    }
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
    if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
      return wsRef.current;
    }
    const ws = new WebSocket(getWebSocketURL(), getWebSocketProtocols());
    wsRef.current = ws;
    ws.onopen = () => {
      reconnectAttemptsRef.current = 0;
      setConnectionState('connected');
      const current = sessionRef.current;
      if (current) {
        ws.send(JSON.stringify({
          action: 'join-call',
          sessionId: current.id,
          userId: getUserId(),
          userName: getUserName(),
        }));
        for (const participant of participantsRef.current) {
          if (participant.userId !== getUserId()) {
            void initiateCallRef.current?.(participant.userId, true).catch(() => undefined);
          }
        }
      }
    };
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.event === 'call-signal' && data.payload) {
          void handleSignal(data.payload).catch(() => setError('Call media negotiation failed.'));
        } else if (data.event === 'call-participant-joined' && data.payload && data.payload.userId !== getUserId()) {
          setParticipants((prev) => {
            if (prev.some((p) => p.userId === data.payload.userId)) return prev;
            return [...prev, data.payload];
          });
          if (isScreenSharingRef.current && sessionRef.current) {
            sendSignal({ type: 'screen-share-started', sessionId: sessionRef.current.id, to: data.payload.userId });
          }
        } else if (data.event === 'call-participant-left' && data.payload) {
          setParticipants((prev) => prev.filter((p) => p.userId !== data.payload.userId));
          onRemoteLeave?.(data.payload.userId);
          setRemoteScreenSharers((current) => {
            const next = { ...current };
            delete next[data.payload.userId];
            return next;
          });
        } else if (data.event === 'call-participant-role-changed' && data.payload) {
          setParticipants((prev) => prev.map((participant) => participant.userId === data.payload.userId
            ? { ...participant, role: data.payload.role }
            : participant));
        } else if (data.event === 'call-ended') {
          void hangUpRef.current?.();
          onCallEnded?.();
        }
      } catch {
        // ignore parse errors
      }
    };
    ws.onclose = () => {
      if (wsRef.current === ws) wsRef.current = null;
      if (intentionalCloseRef.current || !sessionRef.current) return;
      setConnectionState(navigator.onLine ? 'reconnecting' : 'offline');
      const attempt = reconnectAttemptsRef.current++;
      const delay = Math.min(1000 * (2 ** attempt), 15000);
      if (reconnectTimerRef.current !== null) window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = window.setTimeout(() => {
        reconnectTimerRef.current = null;
        if (!intentionalCloseRef.current && sessionRef.current && navigator.onLine) connectWsRef.current?.();
      }, delay);
    };
    ws.onerror = () => setConnectionState(navigator.onLine ? 'reconnecting' : 'offline');
    return ws;
  }, [getUserId, getUserName, handleSignal, onRemoteLeave, onCallEnded, sendSignal]);
  connectWsRef.current = connectWs;

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
    setConnectionState('connecting');
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
      intentionalCloseRef.current = false;
      sessionRef.current = createdSession;
      setSession(createdSession);
      const hostParticipants = [{
        id: `${createdSession.id}-host`,
        sessionId: createdSession.id,
        userId: getUserId(),
        userName: getUserName(),
        role: 'host',
        joinedAt: new Date().toISOString(),
      }];
      participantsRef.current = hostParticipants;
      setParticipants(hostParticipants);
      connectWs();
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

  const initiateCall = useCallback(async (targetUserId: string, iceRestart = false) => {
    const current = sessionRef.current;
    if (!current) return;
    const pc = createPeerConnection(targetUserId);
    if (pc.signalingState !== 'stable') return;
    const offer = await pc.createOffer({ iceRestart });
    await pc.setLocalDescription(offer);
    sendSignal({
      type: 'offer',
      sessionId: current.id,
      to: targetUserId,
      payload: JSON.stringify(offer),
    });
  }, [createPeerConnection, sendSignal]);
  initiateCallRef.current = initiateCall;

  const stopScreenShare = useCallback(async () => {
    const activeScreenStream = screenStreamRef.current;
    if (!activeScreenStream) return;
    const screenTrack = activeScreenStream.getVideoTracks()[0];
    if (screenTrack) screenTrack.onended = null;
    const cameraTrack = localStreamRef.current?.getVideoTracks()[0] ?? null;
    const renegotiate: string[] = [];
    for (const [remoteUserId, sender] of Object.entries(screenSendersRef.current)) {
      const pc = peerConnectionsRef.current[remoteUserId];
      if (!pc) continue;
      if (cameraTrack) {
        await sender.replaceTrack(cameraTrack);
      } else {
        try {
          pc.removeTrack(sender);
          renegotiate.push(remoteUserId);
        } catch {
          // Peer may already have disconnected.
        }
      }
    }
    activeScreenStream.getTracks().forEach((track) => track.stop());
    screenSendersRef.current = {};
    screenStreamRef.current = null;
    isScreenSharingRef.current = false;
    setScreenStream(null);
    setIsScreenSharing(false);
    const current = sessionRef.current;
    if (current) sendSignal({ type: 'screen-share-stopped', sessionId: current.id });
    await Promise.all(renegotiate.map((remoteUserId) => initiateCall(remoteUserId).catch(() => undefined)));
  }, [initiateCall, sendSignal]);

  const startScreenShare = useCallback(async () => {
    const current = sessionRef.current;
    if (!current || !navigator.mediaDevices?.getDisplayMedia) {
      setError('Screen sharing is not supported by this browser.');
      return;
    }
    setError(null);
    try {
      const displayStream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: false });
      const displayTrack = displayStream.getVideoTracks()[0];
      if (!displayTrack) {
        displayStream.getTracks().forEach((track) => track.stop());
        return;
      }
      displayTrack.contentHint = 'detail';
      screenStreamRef.current = displayStream;
      isScreenSharingRef.current = true;
      setScreenStream(displayStream);
      setIsScreenSharing(true);
      const renegotiate: string[] = [];
      for (const [remoteUserId, pc] of Object.entries(peerConnectionsRef.current)) {
        const existingVideoSender = pc.getSenders().find((sender) => sender.track?.kind === 'video');
        if (existingVideoSender) {
          screenSendersRef.current[remoteUserId] = existingVideoSender;
          await existingVideoSender.replaceTrack(displayTrack);
        } else {
          screenSendersRef.current[remoteUserId] = pc.addTrack(displayTrack, displayStream);
          renegotiate.push(remoteUserId);
        }
      }
      displayTrack.onended = () => { void stopScreenShare(); };
      sendSignal({ type: 'screen-share-started', sessionId: current.id });
      await Promise.all(renegotiate.map((remoteUserId) => initiateCall(remoteUserId).catch(() => undefined)));
    } catch (shareError) {
      if (shareError instanceof DOMException && shareError.name === 'NotAllowedError') {
        setError('Screen sharing was cancelled or not permitted.');
      } else {
        setError('Could not start screen sharing.');
      }
    }
  }, [initiateCall, sendSignal, stopScreenShare]);

  const toggleScreenShare = useCallback(() => {
    return isScreenSharingRef.current ? stopScreenShare() : startScreenShare();
  }, [startScreenShare, stopScreenShare]);

  const joinCall = useCallback(async (sessionId: string) => {
    setConnecting(true);
    setConnectionState('connecting');
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
      intentionalCloseRef.current = false;
      setSession(existingSession);
      participantsRef.current = existingParticipants;
      setParticipants(existingParticipants);

      // Start local media so this peer has tracks to offer.
      const kind = existingSession.kind === 'voice' ? 'voice' : 'video';
      const stream = await startLocalMedia(kind);

      connectWs();
      return existingSession;
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to join call';
      setError(message);
      throw e;
    } finally {
      setConnecting(false);
    }
  }, [connectWs, startLocalMedia]);

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
    intentionalCloseRef.current = true;
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    Object.values(peerRecoveryTimersRef.current).forEach((timer) => window.clearTimeout(timer));
    peerRecoveryTimersRef.current = {};
    const current = sessionRef.current;
    if (current) {
      try {
        await leaveCallSession(current.id);
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
    await stopScreenShare();
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
    sessionRef.current = null;
    participantsRef.current = [];
    previousInboundStatsRef.current = {};
    setLocalStream(null);
    setSession(null);
    setParticipants([]);
    setMicMuted(false);
    setCameraOff(false);
    setRemoteScreenSharers({});
    setConnectionState('connecting');
    setQuality('unknown');
    setQualityMetrics(null);
    wsRef.current?.close();
    wsRef.current = null;
  }, [getUserId, stopScreenShare]);
  hangUpRef.current = hangUp;

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
    participantsRef.current = participants;
  }, [participants]);

  useEffect(() => {
    if (!session) return;
    let cancelled = false;
    const collectQuality = async () => {
      const peerEntries = Object.entries(peerConnectionsRef.current);
      if (peerEntries.length === 0) {
        setQuality(navigator.onLine ? 'unknown' : 'offline');
        return;
      }
      let rttMs = 0;
      let jitterMs = 0;
      let packetsLost = 0;
      let packetsReceived = 0;
      let bitrateKbps = 0;
      for (const [peerID, pc] of peerEntries) {
        const reports = await pc.getStats();
        reports.forEach((raw) => {
          const report = raw as RTCStats & Record<string, number | string | boolean>;
          if (report.type === 'candidate-pair' && report.state === 'succeeded' && (report.nominated || report.selected)) {
            rttMs = Math.max(rttMs, Number(report.currentRoundTripTime ?? 0) * 1000);
            bitrateKbps = Math.max(bitrateKbps, Number(report.availableIncomingBitrate ?? 0) / 1000);
          }
          if (report.type !== 'inbound-rtp' || report.isRemote) return;
          jitterMs = Math.max(jitterMs, Number(report.jitter ?? 0) * 1000);
          packetsLost += Math.max(0, Number(report.packetsLost ?? 0));
          packetsReceived += Math.max(0, Number(report.packetsReceived ?? 0));
          const bytes = Number(report.bytesReceived ?? 0);
          const timestamp = Number(report.timestamp ?? 0);
          const key = `${peerID}:${report.id}`;
          const previous = previousInboundStatsRef.current[key];
          if (previous && timestamp > previous.timestamp && bytes >= previous.bytes) {
            bitrateKbps = Math.max(bitrateKbps, ((bytes - previous.bytes) * 8) / (timestamp - previous.timestamp));
          }
          previousInboundStatsRef.current[key] = { bytes, timestamp };
        });
      }
      const totalPackets = packetsLost + packetsReceived;
      const metrics = {
        rttMs: Math.round(rttMs * 10) / 10,
        jitterMs: Math.round(jitterMs * 10) / 10,
        packetLossPct: totalPackets > 0 ? Math.round((packetsLost / totalPackets) * 1000) / 10 : 0,
        bitrateKbps: Math.round(bitrateKbps * 10) / 10,
      };
      if (cancelled) return;
      setQualityMetrics(metrics);
      try {
        const sample = await reportCallQuality(session.id, metrics);
        if (!cancelled) setQuality(sample.quality);
      } catch {
        // Quality reporting must never interrupt media.
      }
    };
    const initial = window.setTimeout(() => { void collectQuality(); }, 3000);
    const interval = window.setInterval(() => { void collectQuality(); }, 10000);
    return () => {
      cancelled = true;
      window.clearTimeout(initial);
      window.clearInterval(interval);
    };
  }, [session]);

  useEffect(() => {
    const handleOffline = () => {
      setConnectionState('offline');
      setQuality('offline');
    };
    const handleOnline = () => {
      if (!sessionRef.current || intentionalCloseRef.current) return;
      setConnectionState('reconnecting');
      connectWsRef.current?.();
      for (const remoteUserID of Object.keys(peerConnectionsRef.current)) {
        void initiateCallRef.current?.(remoteUserID, true).catch(() => undefined);
      }
    };
    window.addEventListener('offline', handleOffline);
    window.addEventListener('online', handleOnline);
    return () => {
      window.removeEventListener('offline', handleOffline);
      window.removeEventListener('online', handleOnline);
    };
  }, []);

  const requestMute = useCallback((userId: string) => {
    const current = sessionRef.current;
    if (!current || !userId || userId === getUserId()) return;
    sendSignal({ type: 'mute-requested', sessionId: current.id, to: userId });
  }, [getUserId, sendSignal]);

  const respondToMuteRequest = useCallback((accept: boolean) => {
    const request = pendingMuteRequest;
    const current = sessionRef.current;
    if (!request || !current) return;
    if (accept && !micMuted) toggleMute();
    sendSignal({ type: accept ? 'mute-accepted' : 'mute-declined', sessionId: current.id, to: request.fromId });
    setPendingMuteRequest(null);
  }, [micMuted, pendingMuteRequest, sendSignal, toggleMute]);

  useEffect(() => {
    return () => {
      intentionalCloseRef.current = true;
      if (reconnectTimerRef.current !== null) window.clearTimeout(reconnectTimerRef.current);
      Object.values(peerRecoveryTimersRef.current).forEach((timer) => window.clearTimeout(timer));
      Object.values(peerConnectionsRef.current).forEach((pc) => {
        try {
          pc.close();
        } catch {
          // ignore
        }
      });
      localStreamRef.current?.getTracks().forEach((track) => track.stop());
      screenStreamRef.current?.getTracks().forEach((track) => track.stop());
      wsRef.current?.close();
    };
  }, []);

  return {
    session,
    participants,
    localStream,
    screenStream,
    isScreenSharing,
    remoteScreenSharers,
    screenShareSupported: typeof navigator !== 'undefined' && Boolean(navigator.mediaDevices?.getDisplayMedia),
    micMuted,
    cameraOff,
    connecting,
    connectionState,
    quality,
    qualityMetrics,
    pendingMuteRequest,
    error,
    startCall,
    joinCall,
    initiateCall,
    toggleMute,
    requestMute,
    respondToMuteRequest,
    toggleCamera,
    toggleScreenShare,
    hangUp,
    endCall,
  };
}
