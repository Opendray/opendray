package plugin

// manifest_strict.go — M5 E1: strict unknown-field validation.
//
// Post-v1-freeze, any manifest that carries a top-level or contributes.*
// key the host doesn't recognise is rejected at install time. This is the
// forward-compat guard described in docs/plugin-platform/12-roadmap.md:
// future manifest additions must bump the schema version so older hosts
// error out loud instead of silently ignoring new fields.
//
// The struct-based [ValidateV1] doesn't catch unknowns on its own —
// json.Unmarshal discards fields with no matching struct tag. This file
// re-parses the raw bytes into a field-name set and compares against the
// v1 whitelist.

import (
	"encoding/json"
	"fmt"
)

// v1TopLevelFields lists every key the v1 manifest schema permits at the
// top level. Update in lockstep with Provider struct changes.
// $schema is allowed because JSON Schema tooling writes it automatically.
var v1TopLevelFields = map[string]bool{
	"$schema":         true,
	"name":            true,
	"displayName":     true,
	"displayName_zh":  true, // i18n overlay; see Provider.DisplayNameZh
	"description":     true,
	"description_zh":  true, // i18n overlay; see Provider.DescriptionZh
	"icon":            true,
	"version":         true,
	"type":            true, // legacy field; still accepted under v1 for back-compat
	"category":        true,
	"cli":             true,
	"capabilities":    true,
	"configSchema":    true,
	"required":        true,
	"publisher":       true,
	"engines":         true,
	"form":            true,
	"activation":      true,
	"contributes":     true,
	"permissions":     true,
	"host":            true,
	"v2Reserved":      true, // explicit escape hatch for forward compat
}

// v1ContributesFields lists every key permitted under `contributes`.
// Keep in lockstep with ContributesV1 struct.
var v1ContributesFields = map[string]bool{
	"commands":       true,
	"statusBar":      true,
	"keybindings":    true,
	"menus":          true,
	"activityBar":    true,
	"views":          true,
	"panels":         true,
	"editorActions":  true,
	"sessionActions": true,
}

// ValidateV1Strict verifies a v1 manifest against both the rule-based
// checks in ValidateV1 AND the unknown-field whitelist. raw must be the
// exact bytes parsed into p — pass them through from LoadManifestWithRaw.
//
// Legacy manifests (IsV1()==false) skip strict-field validation; they
// flow through the compat layer (see 07-lifecycle.md) and may legitimately
// carry fields that have no v1 counterpart.
func ValidateV1Strict(p Provider, raw []byte) []ValidationError {
	errs := ValidateV1(p)
	if !p.IsV1() {
		return errs
	}
	return append(errs, validateNoUnknownFields(raw)...)
}

// validateNoUnknownFields is the whitelist check. Splits top-level and
// contributes.* namespaces so the error paths point at the right place.
func validateNoUnknownFields(raw []byte) []ValidationError {
	var errs []ValidationError
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		// ValidateV1 (struct-based) already surfaces this path as a parse
		// error from earlier; nothing to add here.
		return nil
	}
	for k := range obj {
		if !v1TopLevelFields[k] {
			errs = append(errs, ValidationError{
				Path: k,
				Msg:  fmt.Sprintf("unknown top-level field %q — manifest v1 schema is locked at M5 (use v2Reserved for forward-compat)", k),
			})
		}
	}
	if contribRaw, ok := obj["contributes"]; ok && len(contribRaw) > 0 && string(contribRaw) != "null" {
		var c map[string]json.RawMessage
		if err := json.Unmarshal(contribRaw, &c); err == nil {
			for k := range c {
				if !v1ContributesFields[k] {
					errs = append(errs, ValidationError{
						Path: "contributes." + k,
						Msg:  fmt.Sprintf("unknown contributes field %q — v1 schema frozen", k),
					})
				}
			}
		}
	}
	return errs
}
