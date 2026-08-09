package settings

import (
	"errors"
	"log/slog"
)

// RecordVaultLayout writes the vault layout into config.toml.
//
// This exists because the layout is a DECISION, not an observation, and
// every path in the doc library depends on it. The gateway records it
// at first start; the flatten migration records it again the moment it
// changes the shape on disk.
//
// That second call is not redundant, and leaving it out was a real bug:
// startup detection only runs when the setting is EMPTY, so a vault
// recorded as nested stayed nested after being flattened. Projects kept
// resolving to `projects/<name>` and personal notes kept landing in
// `personal/` against a vault that no longer had either directory.
//
// Reads the file first so unrelated settings survive: Update writes the
// whole document, and a patch built from nothing would blank everything
// the operator has configured.
func RecordVaultLayout(configPath, layout string, log *slog.Logger) error {
	if configPath == "" {
		return errors.New("no config file to record the vault layout in")
	}
	if log == nil {
		log = slog.Default()
	}
	svc := NewService(configPath, log)
	cur, err := svc.Get()
	if err != nil {
		return err
	}
	if cur.Vault.Layout == layout {
		return nil
	}
	cur.Vault.Layout = layout
	return svc.Update(cur)
}
