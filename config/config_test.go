package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateAction(t *testing.T) {
	tests := []struct {
		name    string
		action  ScaleAction
		wantErr bool
	}{
		{name: "ScaleUp is valid", action: ScaleUp, wantErr: false},
		{name: "ScaleDown is valid", action: ScaleDown, wantErr: false},
		{name: "empty is invalid", action: "", wantErr: true},
		{name: "unknown is invalid", action: "Sideways", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Config{Action: tc.action}.validateAction()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseBoolEnv(t *testing.T) {
	const key = "TEST_PARSE_BOOL_ENV"

	tests := []struct {
		name     string
		set      bool
		value    string
		def      bool
		expected bool
	}{
		{name: "unset returns default true", set: false, def: true, expected: true},
		{name: "unset returns default false", set: false, def: false, expected: false},
		{name: "true overrides default false", set: true, value: "true", def: false, expected: true},
		{name: "false overrides default true", set: true, value: "false", def: true, expected: false},
		{name: "1 parses as true", set: true, value: "1", def: false, expected: true},
		{name: "0 parses as false", set: true, value: "0", def: true, expected: false},
		{name: "unparseable falls back to default true", set: true, value: "yes", def: true, expected: true},
		{name: "unparseable falls back to default false", set: true, value: "nope", def: false, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(key, tc.value)
			}

			assert.Equal(t, tc.expected, parseBoolEnv(key, tc.def))
		})
	}
}

func TestParseDurationEnv(t *testing.T) {
	const key = "TEST_PARSE_DURATION_ENV"

	tests := []struct {
		name     string
		set      bool
		value    string
		def      time.Duration
		expected time.Duration
	}{
		{name: "unset returns default", set: false, def: 10 * time.Minute, expected: 10 * time.Minute},
		{name: "valid duration overrides default", set: true, value: "30s", def: 10 * time.Minute, expected: 30 * time.Second},
		{name: "zero is honoured", set: true, value: "0s", def: 10 * time.Minute, expected: 0},
		{name: "unparseable falls back to default", set: true, value: "soon", def: 5 * time.Minute, expected: 5 * time.Minute},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(key, tc.value)
			}

			assert.Equal(t, tc.expected, parseDurationEnv(key, tc.def))
		})
	}
}
