package main

import "time"

type ServiceTarget struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	BaseURL     string `json:"base_url"`
	Criticality string `json:"criticality"`
	Enabled     bool   `json:"enabled"`
}

type ProbeResult struct {
	ServiceID  string    `json:"service_id"`
	Endpoint   string    `json:"endpoint"`
	Status     string    `json:"status"`
	HTTPStatus int       `json:"http_status"`
	LatencyMS  int64     `json:"latency_ms"`
	Detail     string    `json:"detail,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

type Alert struct {
	ID          string            `json:"id"`
	Fingerprint string            `json:"fingerprint"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Source      string            `json:"source"`
	Summary     string            `json:"summary"`
	Labels      map[string]string `json:"labels"`
	StartsAt    time.Time         `json:"starts_at"`
}

type TelemetryEnvelope struct {
	ServiceID  string         `json:"service_id" binding:"required"`
	Timestamp  time.Time      `json:"timestamp"`
	Attributes map[string]any `json:"attributes"`
	Body       map[string]any `json:"body"`
}
