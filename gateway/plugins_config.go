package gateway

// Plugin user-config endpoints.
//
//   GET  /api/plugins/{name}/config → { schema, values }
//   PUT  /api/plugins/{name}/config → writes values, kills the sidecar
//
// Values flow into two backing stores based on each field's type:
//
//   type != "secret"  → plugin_kv, key "__config.<field-key>"
//   type == "secret"  → plugin_secret (AES-GCM), key "__config.<field-key>"
//
// Secrets are NEVER returned by GET — each response uses the sentinel
// "__set__" for fields that have a stored value and "" for fields that
// have never been set. The client uses this to render a "change or
// leave unchanged" password input. PUT accepts either a real new
// value (overwrite) or "" / missing key (leave unchanged).
//
// After a successful PUT the handler calls hostSupervisor.Kill so the
// next invoke respawns the sidecar with the new environment. For
// non-host plugins this is a no-op.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
)

// configKeyPrefix reserves a slice of the plugin's kv / secret
// namespace for platform-managed user config, so the plugin's own
// runtime storage doesn't collide with the Configure form's keys.
const configKeyPrefix = "__config."

// secretSentinel replaces the real value of a secret field in GET
// responses — the client interprets it as "a value is set; leave
// blank to keep it".
const secretSentinel = "__set__"

// configStore is the narrow subset of store.DB the config handlers
// need. Local interface so tests can stub without booting embedded PG.
type configStore interface {
	KVGet(ctx context.Context, plugin, key string) (json.RawMessage, bool, error)
	KVSet(ctx context.Context, plugin, key string, value json.RawMessage) error
	KVDelete(ctx context.Context, plugin, key string) error
}

// platformSecrets is the platform-side surface used to read / write
// encrypted values. *bridge.SecretAPI satisfies this directly via its
// PlatformSet / PlatformGet / PlatformDelete helpers.
type platformSecrets interface {
	PlatformSet(ctx context.Context, plugin, key, value string) error
	PlatformGet(ctx context.Context, plugin, key string) (string, bool, error)
	PlatformDelete(ctx context.Context, plugin, key string) error
}

// sidecarKiller is the subset of *host.Supervisor the config PUT
// handler needs to restart a plugin after its config changes.
type sidecarKiller interface {
	Kill(pluginName, reason string) error
}

// ─── Server resolvers ───────────────────────────────────────────────────────

func (s *Server) configKVStore() configStore {
	if s.configKVTestOverride != nil {
		return s.configKVTestOverride
	}
	if s.hub == nil {
		return nil
	}
	return s.hub.DB()
}

func (s *Server) configSecrets() platformSecrets {
	if s.configSecretsTestOverride != nil {
		return s.configSecretsTestOverride
	}
	if s.secretAPI == nil {
		return nil
	}
	return s.secretAPI
}

func (s *Server) configSupervisor() sidecarKiller {
	if s.configKillerTestOverride != nil {
		return s.configKillerTestOverride
	}
	if s.hostSupervisor == nil {
		return nil
	}
	return s.hostSupervisor
}

// resolveConfigSchema loads the manifest's ConfigSchema for the named
// plugin. Returns (nil, false) when the plugin isn't registered — the
// caller returns 404. An empty schema is returned as (schema, true)
// so the caller can still return {schema:[], values:{}}.
func (s *Server) resolveConfigSchema(name string) ([]plugin.ConfigField, bool) {
	if s.bridgePluginsOverride != nil {
		p, ok := s.bridgePluginsOverride(name)
		if !ok {
			return nil, false
		}
		return p.ConfigSchema, true
	}
	if s.plugins == nil {
		return nil, false
	}
	p, ok := s.plugins.Get(name)
	if !ok {
		return nil, false
	}
	return p.ConfigSchema, true
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// pluginsConfigGet handles GET /api/plugins/{name}/config.
//
// Response shape:
//
//	{
//	  "schema": [ConfigField, ...],
//	  "values": { "<key>": "string-value", ... }
//	}
//
// Secret fields render as "__set__" when stored, "" otherwise.
func (s *Server) pluginsConfigGet(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	schema, ok := s.resolveConfigSchema(name)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ENOENT", "plugin not found: "+name)
		return
	}

	values := make(map[string]string, len(schema))
	kvStore := s.configKVStore()
	secrets := s.configSecrets()

	for _, f := range schema {
		storeKey := configKeyPrefix + f.Key
		if f.Type == "secret" {
			if secrets == nil {
				values[f.Key] = ""
				continue
			}
			_, found, err := secrets.PlatformGet(r.Context(), name, storeKey)
			if err != nil {
				s.logger.Error("config: get secret", "plugin", name, "key", f.Key, "err", err)
				values[f.Key] = ""
				continue
			}
			if found {
				values[f.Key] = secretSentinel
			} else {
				values[f.Key] = ""
			}
			continue
		}
		// Non-secret field via plugin_kv.
		if kvStore == nil {
			values[f.Key] = ""
			continue
		}
		raw, found, err := kvStore.KVGet(r.Context(), name, storeKey)
		if err != nil {
			s.logger.Error("config: get kv", "plugin", name, "key", f.Key, "err", err)
			values[f.Key] = ""
			continue
		}
		if !found {
			values[f.Key] = ""
			continue
		}
		// Values are stored as JSON strings so the sidecar can parse
		// them uniformly. Fall back to the raw bytes on decode failure
		// — better to show a corrupted value than to 500 the page.
		var s string
		if uerr := json.Unmarshal(raw, &s); uerr != nil {
			values[f.Key] = string(raw)
			continue
		}
		values[f.Key] = s
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schema": schema,
		"values": values,
	})
}

// pluginsConfigPut handles PUT /api/plugins/{name}/config.
//
// Request body:
//
//	{ "values": { "<key>": "string" | bool | number, ... } }
//
// Each key is looked up in the plugin's ConfigSchema; unknown keys
// are rejected as EINVAL. Secret fields skip write when the submitted
// value is "" or equals the sentinel ("__set__" = user didn't retype
// it). Non-secret fields always overwrite. After all writes the
// plugin's sidecar is killed so next invoke respawns fresh.
func (s *Server) pluginsConfigPut(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	schema, ok := s.resolveConfigSchema(name)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ENOENT", "plugin not found: "+name)
		return
	}
	if len(schema) == 0 {
		writeJSONError(w, http.StatusBadRequest, "ENOSCHEMA",
			"plugin has no configSchema — nothing to write")
		return
	}

	var req struct {
		Values map[string]any `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "EINVAL", "invalid JSON body")
		return
	}
	if req.Values == nil {
		req.Values = map[string]any{}
	}

	// Build a schema map for O(1) lookups + unknown-key rejection.
	byKey := make(map[string]plugin.ConfigField, len(schema))
	for _, f := range schema {
		byKey[f.Key] = f
	}
	for k := range req.Values {
		if _, ok := byKey[k]; !ok {
			writeJSONError(w, http.StatusBadRequest, "EINVAL",
				fmt.Sprintf("unknown config key %q", k))
			return
		}
	}

	kvStore := s.configKVStore()
	secrets := s.configSecrets()

	for _, f := range schema {
		raw, present := req.Values[f.Key]
		storeKey := configKeyPrefix + f.Key

		if f.Type == "secret" {
			// Secret write rules: only overwrite when the client sent
			// a real new value. "" and the "__set__" sentinel mean
			// "leave the existing value alone" — that's how the UI
			// renders "password unchanged" on re-save.
			if !present {
				continue
			}
			str, ok := raw.(string)
			if !ok {
				writeJSONError(w, http.StatusBadRequest, "EINVAL",
					fmt.Sprintf("%s: secret value must be a string", f.Key))
				return
			}
			if str == "" || str == secretSentinel {
				continue
			}
			if secrets == nil {
				writeJSONError(w, http.StatusServiceUnavailable, "ECONFIG",
					"secret store not configured")
				return
			}
			if err := secrets.PlatformSet(r.Context(), name, storeKey, str); err != nil {
				s.logger.Error("config: put secret", "plugin", name, "key", f.Key, "err", err)
				writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
					"failed to persist secret")
				return
			}
			continue
		}

		// Non-secret: always write (or clear on explicit null).
		if kvStore == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "ECONFIG",
				"kv store not configured")
			return
		}
		if !present || raw == nil {
			if err := kvStore.KVDelete(r.Context(), name, storeKey); err != nil {
				s.logger.Error("config: delete kv", "plugin", name, "key", f.Key, "err", err)
				writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
					"failed to clear value")
				return
			}
			continue
		}
		// Everything is coerced to a JSON string for uniform sidecar
		// reads. Numbers → "123", booleans → "true"/"false", strings
		// pass through. The sidecar's config helper handles type
		// interpretation per field.
		var strVal string
		switch v := raw.(type) {
		case string:
			strVal = v
		case bool:
			if v {
				strVal = "true"
			} else {
				strVal = "false"
			}
		case float64: // json.Decoder number type
			strVal = fmt.Sprintf("%v", v)
		default:
			writeJSONError(w, http.StatusBadRequest, "EINVAL",
				fmt.Sprintf("%s: unsupported value type %T", f.Key, raw))
			return
		}
		jsonVal, err := json.Marshal(strVal)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
				fmt.Sprintf("%s: marshal: %v", f.Key, err))
			return
		}
		if err := kvStore.KVSet(r.Context(), name, storeKey, jsonVal); err != nil {
			if errors.Is(err, store.ErrValueTooLarge) || errors.Is(err, store.ErrPluginQuotaExceeded) {
				writeJSONError(w, http.StatusRequestEntityTooLarge, "EQUOTA", err.Error())
				return
			}
			s.logger.Error("config: put kv", "plugin", name, "key", f.Key, "err", err)
			writeJSONError(w, http.StatusInternalServerError, "EINTERNAL",
				"failed to persist value")
			return
		}
	}

	// Restart the sidecar so the new config takes effect on the next
	// invoke. Kill is idempotent for plugins that aren't currently
	// running, and a no-op for non-host plugins that never spawn one.
	if sv := s.configSupervisor(); sv != nil {
		_ = sv.Kill(name, "config-change")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"updated": true,
		"name":    name,
	})
}
