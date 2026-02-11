package cmd

import "testing"

func TestParseTunnelSpec(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantLocal  int
		wantHost   string
		wantRemote int
		wantErr    bool
	}{
		{
			name:       "simple two-port",
			spec:       "8080:80",
			wantLocal:  8080,
			wantHost:   "localhost",
			wantRemote: 80,
		},
		{
			name:       "same port",
			spec:       "3000:3000",
			wantLocal:  3000,
			wantHost:   "localhost",
			wantRemote: 3000,
		},
		{
			name:       "three-part with host",
			spec:       "9090:10.0.0.2:80",
			wantLocal:  9090,
			wantHost:   "10.0.0.2",
			wantRemote: 80,
		},
		{
			name:       "three-part with hostname",
			spec:       "5432:db.internal:5432",
			wantLocal:  5432,
			wantHost:   "db.internal",
			wantRemote: 5432,
		},
		{
			name:       "local port zero (auto)",
			spec:       "0:8080",
			wantLocal:  0,
			wantHost:   "localhost",
			wantRemote: 8080,
		},
		{
			name:    "invalid local port",
			spec:    "abc:80",
			wantErr: true,
		},
		{
			name:    "invalid remote port",
			spec:    "8080:abc",
			wantErr: true,
		},
		{
			name:    "remote port zero",
			spec:    "8080:0",
			wantErr: true,
		},
		{
			name:    "port out of range",
			spec:    "70000:80",
			wantErr: true,
		},
		{
			name:    "single value",
			spec:    "8080",
			wantErr: true,
		},
		{
			name:    "too many colons",
			spec:    "8080:host:80:extra",
			wantErr: true,
		},
		{
			name:    "empty string",
			spec:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			local, host, remote, err := parseTunnelSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for spec %q, got none", tt.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for spec %q: %v", tt.spec, err)
			}
			if local != tt.wantLocal {
				t.Errorf("local port: got %d, want %d", local, tt.wantLocal)
			}
			if host != tt.wantHost {
				t.Errorf("remote host: got %q, want %q", host, tt.wantHost)
			}
			if remote != tt.wantRemote {
				t.Errorf("remote port: got %d, want %d", remote, tt.wantRemote)
			}
		})
	}
}
