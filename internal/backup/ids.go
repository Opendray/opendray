package backup

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

// idPrefixBackup is the textual prefix for backup row IDs.
const (
	idPrefixBackup   = "bk"
	idPrefixExport   = "exp"
	idPrefixSchedule = "sch"
	idPrefixTarget   = "tgt"
	idPrefixImport   = "imp"
)

// idEntropyBytes is the random byte count behind every generated ID.
// 14 bytes → ~22 base32 chars → ~110 bits of entropy. Safe against
// birthday collision in any plausible deployment.
const idEntropyBytes = 14

// newID returns "<prefix>_<22 lowercase base32 chars>".
func newID(prefix string) string {
	var b [idEntropyBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand on linux/darwin reads from getrandom(2)/getentropy(2);
		// failure means the kernel CSPRNG is unavailable, which is fatal.
		panic("backup: rand.Read: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return prefix + "_" + strings.ToLower(enc)
}

// NewBackupID returns a new "bk_..." identifier.
func NewBackupID() string { return newID(idPrefixBackup) }

// NewExportID returns a new "exp_..." identifier.
func NewExportID() string { return newID(idPrefixExport) }

// NewScheduleID returns a new "sch_..." identifier.
func NewScheduleID() string { return newID(idPrefixSchedule) }

// NewTargetID returns a new "tgt_..." identifier (used when the
// operator doesn't supply a human-readable id).
func NewTargetID() string { return newID(idPrefixTarget) }

// NewImportID returns a new "imp_..." identifier.
func NewImportID() string { return newID(idPrefixImport) }

// NewDownloadToken is the per-export download credential. 22 base32
// chars (~110 bits) is overkill but keeps the surface uniform.
func NewDownloadToken() string {
	var b [idEntropyBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("backup: rand.Read: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return strings.ToLower(enc)
}
