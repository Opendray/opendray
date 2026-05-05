package backup

import (
	"context"
	"fmt"
)

// RunRetention enforces the "keep most recent N succeeded backups
// per target" policy. Older rows have their blob deleted from the
// target (best-effort) and the row flipped to status='deleted'.
//
// Called by the scheduler after each successful run; safe to
// invoke out-of-band whenever retention shrinks (e.g. operator
// lowers the schedule's retention via UI).
func (s *Service) RunRetention(ctx context.Context, targetID string, keep int) error {
	if keep < 0 {
		return fmt.Errorf("retention: keep must be >= 0 (got %d)", keep)
	}
	rows, err := s.store.ListSucceededByTargetOldestFirst(ctx, targetID)
	if err != nil {
		return fmt.Errorf("retention: list: %w", err)
	}
	if len(rows) <= keep {
		return nil
	}
	toDelete := rows[:len(rows)-keep] // oldest first
	target := s.targets.get(targetID)
	for _, b := range toDelete {
		if target != nil && b.TargetPath != "" {
			ref := TargetRef{Target: targetID, Path: b.TargetPath}
			if err := target.Delete(ctx, ref); err != nil {
				s.log.Warn("retention: target.Delete failed; flipping row anyway",
					"backup_id", b.ID, "err", err)
			}
		}
		if err := s.store.MarkBackupDeleted(ctx, b.ID); err != nil {
			return fmt.Errorf("retention: mark deleted %s: %w", b.ID, err)
		}
	}
	s.log.Info("retention applied",
		"target_id", targetID, "kept", keep, "removed", len(toDelete))
	return nil
}
