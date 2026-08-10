import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/notes/vault_sync_screen.dart';

// The server hands back Go's full duration form ("10m0s", "1h0m0s")
// while the interval dropdown is keyed on the short form ("10m", "1h").
// If the two ever disagree the dropdown shows a value nobody chose and
// saving sends it straight back to the API, so this conversion is worth
// pinning down.
void main() {
  group('normaliseGoDuration', () {
    test('trims the zero seconds Go always prints', () {
      expect(normaliseGoDuration('10m0s'), '10m');
      expect(normaliseGoDuration('30m0s'), '30m');
    });

    test('trims zero minutes and seconds from whole hours', () {
      expect(normaliseGoDuration('1h0m0s'), '1h');
      expect(normaliseGoDuration('24h0m0s'), '24h');
    });

    test('leaves values that are already short alone', () {
      expect(normaliseGoDuration('30s'), '30s');
      expect(normaliseGoDuration('15m'), '15m');
      expect(normaliseGoDuration('6h'), '6h');
    });

    test('leaves a genuinely compound duration intact', () {
      // Someone can set 1h30m from the web's Custom field; mangling it
      // into "1h" would quietly change their setting on save.
      expect(normaliseGoDuration('1h30m0s'), '1h30m0s');
    });

    test('never emits a literal capture-group reference', () {
      // Dart's replaceAll does NOT expand $1 — it substitutes the two
      // characters. The first cut of this function used it and turned
      // every interval into the string r'$1'.
      for (final input in ['10m0s', '1h0m0s', '30s', '1h30m0s', '']) {
        expect(normaliseGoDuration(input), isNot(contains(r'$')),
            reason: 'input: $input');
      }
    });

    test('handles an empty value without throwing', () {
      expect(normaliseGoDuration(''), '');
    });
  });
}
