package handlers

import (
	"path/filepath"
	"testing"
)

func TestNormalizeDomainDocumentRootJailsToUserHome(t *testing.T) {
	homeBase := "/srv/zenspanel/home"
	username := "alice"

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid relative docroot",
			input:   "public_html/example.com",
			want:    filepath.Clean("/srv/zenspanel/home/alice/public_html/example.com"),
			wantErr: false,
		},
		{
			name:    "relative traversal outside home",
			input:   "../../etc",
			wantErr: true,
		},
		{
			name:    "absolute host path",
			input:   "/etc",
			wantErr: true,
		},
		{
			name:    "shell metacharacters remain data inside home",
			input:   "public_html/$(touch pwned)",
			want:    filepath.Clean("/srv/zenspanel/home/alice/public_html/$(touch pwned)"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDomainDocumentRoot(homeBase, username, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeDomainDocumentRoot(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDomainDocumentRoot(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeDomainDocumentRoot(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
