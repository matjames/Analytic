package api

import (
	"math"
	"strings"
	"testing"
)

func TestValidateLocationRequest(t *testing.T) {
	tests := []struct {
		name string
		req  sendLocationRequest
		ok   bool
	}{
		{name: "valid", req: sendLocationRequest{ConversationID: "group-1", Latitude: -1.2921, Longitude: 36.8219, AccuracyMeters: 12.5, Label: "Nairobi office"}, ok: true},
		{name: "valid boundaries", req: sendLocationRequest{ConversationID: "group-1", Latitude: 90, Longitude: -180}, ok: true},
		{name: "missing conversation", req: sendLocationRequest{Latitude: 1, Longitude: 2}},
		{name: "latitude out of range", req: sendLocationRequest{ConversationID: "group-1", Latitude: 91}},
		{name: "longitude out of range", req: sendLocationRequest{ConversationID: "group-1", Longitude: -181}},
		{name: "non finite latitude", req: sendLocationRequest{ConversationID: "group-1", Latitude: math.NaN()}},
		{name: "non finite longitude", req: sendLocationRequest{ConversationID: "group-1", Longitude: math.Inf(1)}},
		{name: "negative accuracy", req: sendLocationRequest{ConversationID: "group-1", AccuracyMeters: -1}},
		{name: "excessive accuracy", req: sendLocationRequest{ConversationID: "group-1", AccuracyMeters: 100001}},
		{name: "long label", req: sendLocationRequest{ConversationID: "group-1", Label: strings.Repeat("a", 121)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validateLocationRequest(test.req); (got == nil) != test.ok {
				t.Fatalf("validateLocationRequest() error = %v, want success %v", got, test.ok)
			}
		})
	}
}
