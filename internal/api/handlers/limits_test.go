package handlers

import (
	"strings"
	"testing"
)

func TestEnforceLimitRejectsQuotaAbuse(t *testing.T) {
	tests := []struct {
		name    string
		current int
		max     int
		kind    string
		wantErr bool
	}{
		{
			name:    "valid legitimate create under package limit",
			current: 1,
			max:     3,
			kind:    "database",
			wantErr: false,
		},
		{
			name:    "malicious repeated database creation at limit",
			current: 2,
			max:     2,
			kind:    "database",
			wantErr: true,
		},
		{
			name:    "malicious repeated domain creation above limit",
			current: 4,
			max:     2,
			kind:    "domain",
			wantErr: true,
		},
		{
			name:    "unlimited package keeps administrative workflow working",
			current: 100,
			max:     0,
			kind:    "domain",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforceLimit(tt.current, tt.max, tt.kind)
			if tt.wantErr {
				if err == nil {
					t.Fatal("enforceLimit() = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.kind+" quota reached") {
					t.Fatalf("enforceLimit() error = %q, want quota error for %q", err.Error(), tt.kind)
				}
				return
			}
			if err != nil {
				t.Fatalf("enforceLimit() error = %v", err)
			}
		})
	}
}
