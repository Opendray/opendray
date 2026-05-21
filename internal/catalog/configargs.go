package catalog

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// applyConfigSchema walks the provider's ConfigSchema and translates the
// per-provider user config into spawn-time CLI args and env vars. This
// is the single place that turns manifest-declared cliFlag / cliValue /
// envVar metadata into actual process inputs.
//
// Field handling:
//
//   - boolean: when value is true and cliFlag is set, append cliFlag.
//   - string / select / secret: when the stringified value is non-empty,
//     either set env[envVar] (if envVar declared) or append cliFlag with
//     the value when cliValue=true (or just the flag when cliValue=false).
//   - number: same as the scalar branch above; non-empty stringification.
//   - args: append the slice as-is (used for free-form extraArgs).
//
// Fields with neither cliFlag nor envVar are ignored. The "command"
// override and the skills_* booleans are handled by Resolve directly.
//
// dependsOn / dependsVal: a field is skipped when cfg[dependsOn] is
// missing or unequal to dependsVal — lets fields like apiKey only apply
// when authType=="custom".
//
// modelArgs renders the per-provider default model into CLI args when
// the operator has set one (config["model"]) and the manifest declares a
// model flag. A per-session --model in the spawn args still wins — the
// session manager drops provider-derived flags the user re-specifies.
func modelArgs(m Manifest, cfg map[string]any) []string {
	if m.ModelFlag == "" {
		return nil
	}
	model, _ := cfg["model"].(string)
	if model == "" {
		return nil
	}
	return []string{m.ModelFlag, model}
}

func applyConfigSchema(schema []ConfigField, cfg map[string]any) ([]string, map[string]string) {
	if len(schema) == 0 || len(cfg) == 0 {
		return nil, nil
	}
	var args []string
	var env map[string]string
	setEnv := func(k, v string) {
		if env == nil {
			env = map[string]string{}
		}
		env[k] = v
	}

	for _, f := range schema {
		if f.DependsOn != "" {
			got, ok := cfg[f.DependsOn]
			if !ok || !equalAny(got, f.DependsVal) {
				continue
			}
		}
		raw, ok := cfg[f.Key]
		if !ok {
			continue
		}

		switch f.Type {
		case "boolean":
			if f.CliFlag == "" {
				continue
			}
			if b, _ := raw.(bool); b {
				args = append(args, f.CliFlag)
			}
		case "args":
			args = append(args, stringifyArgs(raw)...)
		case "string", "select", "secret", "number":
			s := stringifyScalar(raw)
			if s == "" {
				continue
			}
			switch {
			case f.EnvVar != "":
				setEnv(f.EnvVar, s)
			case f.CliFlag != "" && f.CliValue:
				args = append(args, f.CliFlag, s)
			case f.CliFlag != "":
				args = append(args, f.CliFlag)
			}
		}
	}
	return args, env
}

// equalAny compares two JSON-decoded values loosely. JSON numbers come
// back as float64 and dependsVal is decoded the same way, so direct
// equality works for primitive types; we fall through to a stringified
// compare for the long tail (e.g. an int dependsVal authored in Go-only
// tests).
func equalAny(a, b any) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// stringifyScalar renders a JSON-decoded primitive for CLI use.
// Returns "" when the value is empty/nil/zero or not a primitive.
func stringifyScalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case float32:
		return stringifyScalar(float64(t))
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

// stringifyArgs flattens JSON-decoded args into []string. Accepts both
// []any (JSONB shape) and []string (test/Go-native callers). Empty
// strings are dropped so an unsaved input field doesn't spawn a stray
// "" arg.
func stringifyArgs(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s := stringifyScalar(x); s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}
