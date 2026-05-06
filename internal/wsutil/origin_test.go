package wsutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOriginCheck(t *testing.T) {
	tests := []struct {
		name      string
		extra     []string
		hostHdr   string
		originHdr string
		want      bool
	}{
		{
			name: "empty origin allowed (non-browser client)",
			want: true,
		},
		{
			name:      "exact same-origin allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://opendray.local:8770",
			want:      true,
		},
		{
			name:      "different origin rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "https://evil.example.com",
			want:      false,
		},
		{
			name:      "loopback allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://127.0.0.1:5173",
			want:      true,
		},
		{
			name:      "ipv6 loopback allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://[::1]:5173",
			want:      true,
		},
		{
			name:      "localhost name allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://localhost:5173",
			want:      true,
		},
		{
			name:      "RFC1918 192.168/16 allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://192.168.3.21:8770",
			want:      true,
		},
		{
			name:      "RFC1918 10/8 allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://10.0.0.5:8770",
			want:      true,
		},
		{
			name:      "RFC1918 172.16/12 allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://172.20.5.5:8770",
			want:      true,
		},
		{
			name:      "172.32 (outside 172.16-31) rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://172.32.5.5:8770",
			want:      false,
		},
		{
			name:      "public IP rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://8.8.8.8",
			want:      false,
		},
		{
			name:      "explicit allowlist match",
			extra:     []string{"my-tunnel.example.com"},
			hostHdr:   "opendray.local:8770",
			originHdr: "https://my-tunnel.example.com",
			want:      true,
		},
		{
			name:      "explicit allowlist case-insensitive",
			extra:     []string{"My-Tunnel.Example.Com"},
			hostHdr:   "opendray.local:8770",
			originHdr: "https://my-tunnel.example.com",
			want:      true,
		},
		{
			name:      "malformed origin rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "://not a url",
			want:      false,
		},
		{
			name:      "host-only origin (no scheme) rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "evil.example.com",
			want:      false,
		},
		{
			name:      "scheme-only origin (empty authority) rejected",
			hostHdr:   "opendray.local:8770",
			originHdr: "http://",
			want:      false,
		},
		{
			name:      "https origin against same host allowed",
			hostHdr:   "opendray.local:8770",
			originHdr: "https://opendray.local:8770",
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := SameOriginCheck(tc.extra...)
			r := httptest.NewRequest(http.MethodGet, "http://x/ws", nil)
			if tc.hostHdr != "" {
				r.Host = tc.hostHdr
			}
			if tc.originHdr != "" {
				r.Header.Set("Origin", tc.originHdr)
			}
			if got := check(r); got != tc.want {
				t.Errorf("got %v, want %v (origin=%q host=%q)",
					got, tc.want, tc.originHdr, tc.hostHdr)
			}
		})
	}
}

func TestAllowAnyOrigin(t *testing.T) {
	check := AllowAnyOrigin()
	r := httptest.NewRequest(http.MethodGet, "http://x/ws", nil)
	r.Header.Set("Origin", "https://evil.example.com")
	if !check(r) {
		t.Error("AllowAnyOrigin should always return true")
	}
}
