// Dart mirror of internal/memory/ranking.go (and the web TS
// equivalent in app/shared/src/lib/memoryRanking.ts). Kept in
// lockstep manually — the formula is tiny, the cost of a port is
// zero, and the value of letting mobile inspectors show *why* a
// row ranks where it does (without a backend roundtrip) is high.

import 'package:opendray/core/api/models.dart';

const double ageDecayDays = 180;
const double ageFloor = 0.5;
const double hitsPerBoostUnit = 0.02;
const double hitBoostCap = 0.5;
const double confidenceFloor = 0.3;

/// One row of math for an effective-score computation. The mobile
/// inspector tile uses [effectiveScore] for the rank badge and the
/// other multipliers for the explanation dialog.
class RankingBreakdown {
  RankingBreakdown({
    required this.similarity,
    required this.ageMultiplier,
    required this.hitMultiplier,
    required this.confidenceMultiplier,
    required this.effectiveScore,
    required this.ageDays,
  });

  final double similarity;
  final double ageMultiplier;
  final double hitMultiplier;
  final double confidenceMultiplier;
  final double effectiveScore;
  final int ageDays;
}

/// Compute the same effective score the Go backend uses, plus the
/// intermediate multipliers so the UI can show a tooltip
/// explaining the math. [similarity] defaults to 1.0 so the
/// inspector (which lists memories without a query) can show "if
/// we matched perfectly, how would this rank" as a baseline.
RankingBreakdown rankingBreakdown(
  Memory mem, {
  double similarity = 1.0,
  DateTime? now,
}) {
  if (similarity <= 0) {
    return RankingBreakdown(
      similarity: 0,
      ageMultiplier: 0,
      hitMultiplier: 0,
      confidenceMultiplier: 0,
      effectiveScore: 0,
      ageDays: 0,
    );
  }
  final tsNow = now ?? DateTime.now();
  final ageDays = tsNow.difference(mem.createdAt).inDays.clamp(0, 1 << 30);
  final ageMultiplier = (1 - ageDays / ageDecayDays).clamp(ageFloor, 1.0);
  final hitBoost = (mem.hitCount * hitsPerBoostUnit).clamp(0.0, hitBoostCap);
  final hitMultiplier = 1 + hitBoost;
  final conf = mem.confidence == null
      ? 1.0
      : mem.confidence!.clamp(confidenceFloor, 1.0);
  return RankingBreakdown(
    similarity: similarity,
    ageMultiplier: ageMultiplier,
    hitMultiplier: hitMultiplier,
    confidenceMultiplier: conf,
    effectiveScore: similarity * ageMultiplier * hitMultiplier * conf,
    ageDays: ageDays,
  );
}
