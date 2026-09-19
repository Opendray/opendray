package session

import (
	"strings"
	"testing"
)

// WithSessionEnv injects gateway-level env (e.g. the Jev key) into every
// session. The spawn path layers it as base < sessionEnv < extraEnv, so
// the tool key reaches sessions but a provider can still override a key
// deliberately. This locks that layering + the option.
func TestSessionEnvLayering(t *testing.T) {
	m := &Manager{}
	WithSessionEnv(map[string]string{"TYPESAFE_API_KEY": "jevkey", "FOO": "1"})(m)
	if m.sessionEnv["TYPESAFE_API_KEY"] != "jevkey" {
		t.Fatalf("WithSessionEnv did not set the field: %v", m.sessionEnv)
	}

	base := []string{"PATH=/bin"}
	// exact composition used in spawn()
	got := mergeEnv(mergeEnv(base, m.sessionEnv), map[string]string{"TYPESAFE_API_KEY": "provider-wins"})
	find := func(k string) string {
		for _, kv := range got {
			if strings.HasPrefix(kv, k+"=") {
				return kv[len(k)+1:]
			}
		}
		return ""
	}
	if find("FOO") != "1" {
		t.Errorf("session env FOO missing from spawn env: %v", got)
	}
	if find("TYPESAFE_API_KEY") != "provider-wins" {
		t.Errorf("provider extraEnv should override session env, got %q", find("TYPESAFE_API_KEY"))
	}
	if find("PATH") != "/bin" {
		t.Errorf("base env PATH lost: %v", got)
	}
}
