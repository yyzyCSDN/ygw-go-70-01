package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	rt := BuildRuntime()
	server := NewServer(rt)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", rec.Code)
	}
}

func TestConsolePageServed(t *testing.T) {
	rt := BuildRuntime()
	server := NewServer(rt)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "光伏电站逆变器群控台") {
		t.Fatal("index page does not contain the console title")
	}
}

func TestInvertersEndpoint(t *testing.T) {
	rt := BuildRuntime()
	server := NewServer(rt)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inverters", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("inverters status = %d, want 200", rec.Code)
	}
	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode inverters: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("inverters count = %d, want 3", len(rows))
	}
}

func TestForecastEndpoint(t *testing.T) {
	rt := BuildRuntime()
	rt.Forecast.Add(700)
	rt.Forecast.Add(500)
	server := NewServer(rt)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forecast", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forecast status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode forecast: %v", err)
	}
	if body["average"] != float64(600) {
		t.Fatalf("forecast average = %v, want 600", body["average"])
	}
}
