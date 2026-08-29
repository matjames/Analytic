package api

import (
	"bytes"
	"testing"

	"statchat/pkg/model"
)

func TestCanEndCallRequiresSessionHost(t *testing.T) {
	session := model.CallSession{HostID: "host-1"}
	if !canEndCall(session, "host-1") {
		t.Fatal("expected session host to be allowed to end the call")
	}
	if canEndCall(session, "participant-2") {
		t.Fatal("expected non-host participant to be denied")
	}
	if canEndCall(session, "") {
		t.Fatal("expected empty identity to be denied")
	}
}

func TestCallSessionTenantBoundary(t *testing.T) {
	session := model.CallSession{TenantID: "tenant-a"}
	allowed, err := canAccessCallSession(session, "member-a", "tenant-a")
	if err != nil || !allowed {
		t.Fatalf("expected same-tenant room call access, allowed=%v err=%v", allowed, err)
	}
	allowed, err = canAccessCallSession(session, "member-b", "tenant-b")
	if err != nil || allowed {
		t.Fatalf("expected foreign tenant denial, allowed=%v err=%v", allowed, err)
	}
}

func TestAllowedCallSignalTypes(t *testing.T) {
	for _, signalType := range []string{"offer", "answer", "ice-candidate", "screen-share-started", "screen-share-stopped"} {
		if !isAllowedCallSignalType(signalType) {
			t.Fatalf("expected %q to be allowed", signalType)
		}
	}
	for _, signalType := range []string{"", "join", "leave", "screen-share-admin", "message"} {
		if isAllowedCallSignalType(signalType) {
			t.Fatalf("expected %q to be rejected", signalType)
		}
	}
}

func TestCallQualityClassificationAndValidation(t *testing.T) {
	tests := []struct {
		request callQualityRequest
		quality string
	}{
		{request: callQualityRequest{RTTMs: 40, JitterMs: 5, PacketLossPct: .2, BitrateKbps: 1200}, quality: "excellent"},
		{request: callQualityRequest{RTTMs: 200, JitterMs: 35, PacketLossPct: 1.5, BitrateKbps: 800}, quality: "good"},
		{request: callQualityRequest{RTTMs: 450, JitterMs: 70, PacketLossPct: 5, BitrateKbps: 400}, quality: "fair"},
		{request: callQualityRequest{RTTMs: 900, JitterMs: 140, PacketLossPct: 12, BitrateKbps: 100}, quality: "poor"},
	}
	for _, test := range tests {
		if !validCallQualityMetrics(test.request) {
			t.Fatalf("expected valid metrics: %#v", test.request)
		}
		if got := classifyCallQuality(test.request); got != test.quality {
			t.Fatalf("quality=%q, want %q for %#v", got, test.quality, test.request)
		}
	}
	for _, invalid := range []callQualityRequest{
		{RTTMs: -1},
		{PacketLossPct: 101},
		{JitterMs: 10001},
		{BitrateKbps: 1000001},
	} {
		if validCallQualityMetrics(invalid) {
			t.Fatalf("expected invalid metrics: %#v", invalid)
		}
	}
}

func TestHasWebMSignature(t *testing.T) {
	valid := bytes.NewReader([]byte{0x1a, 0x45, 0xdf, 0xa3, 0x01})
	if !hasWebMSignature(valid) {
		t.Fatal("expected EBML signature to be accepted")
	}
	position, err := valid.Seek(0, 1)
	if err != nil || position != 0 {
		t.Fatalf("expected file position to be reset, position=%d err=%v", position, err)
	}
	if hasWebMSignature(bytes.NewReader([]byte("not-webm"))) {
		t.Fatal("expected non-WebM content to be rejected")
	}
	if hasWebMSignature(bytes.NewReader([]byte{0x1a, 0x45})) {
		t.Fatal("expected truncated signature to be rejected")
	}
}

func TestIsAllowedCallRecordingRequiresWebM(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		allowed     bool
	}{
		{name: "video WebM", filename: "meeting.webm", contentType: "video/webm", allowed: true},
		{name: "audio WebM with codec", filename: "meeting.WEBM", contentType: "audio/webm; codecs=opus", allowed: true},
		{name: "wrong extension", filename: "meeting.png", contentType: "video/webm"},
		{name: "wrong content type", filename: "meeting.webm", contentType: "application/octet-stream"},
		{name: "missing content type", filename: "meeting.webm"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAllowedCallRecording(test.filename, test.contentType); got != test.allowed {
				t.Fatalf("isAllowedCallRecording(%q, %q) = %v, want %v", test.filename, test.contentType, got, test.allowed)
			}
		})
	}
}
