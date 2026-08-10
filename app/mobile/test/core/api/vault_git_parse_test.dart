import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/vault_git_api.dart';

// Field-name typos in a fromJson are silent: every value falls back to
// its default and the screen renders a plausible-looking "clean repo,
// auto-sync off". These pin the wire shape against internal/vaultgit.
void main() {
  group('VaultStatus.fromJson', () {
    test('reads a dirty repo that is ahead of its upstream', () {
      final s = VaultStatus.fromJson({
        'is_repo': true,
        'branch': 'main',
        'upstream': 'origin/main',
        'ahead': 3,
        'behind': 1,
        'root': '/Users/x/.opendray/vault',
        'files': [
          {'xy': ' M', 'path': 'daily/2026-08-10.md'},
          {'xy': '??', 'path': 'notes/new.md'},
        ],
      });
      expect(s.isRepo, isTrue);
      expect(s.branch, 'main');
      expect(s.ahead, 3);
      expect(s.behind, 1);
      expect(s.files, hasLength(2));
      expect(s.files.first.xy, ' M');
      expect(s.hasRemote, isTrue);
      expect(s.isClean, isFalse);
    });

    test('an absent upstream means there is nowhere to push', () {
      final s = VaultStatus.fromJson({'is_repo': true, 'branch': 'main'});
      expect(s.hasRemote, isFalse);
      expect(s.isClean, isTrue);
      expect(s.state, isNull);
    });

    test('surfaces a repo stuck mid-rebase', () {
      final s = VaultStatus.fromJson({
        'is_repo': true,
        'state': {
          'rebase_in_progress': true,
          'conflicted_files': ['a.md'],
        },
      });
      expect(s.state!.isMidOperation, isTrue);
      expect(s.state!.conflictedFiles, ['a.md']);
    });

    test('a merge or cherry-pick counts as mid-operation too', () {
      expect(
        VaultStatus.fromJson({
          'state': {'merge_in_progress': true},
        }).state!.isMidOperation,
        isTrue,
      );
      expect(
        VaultStatus.fromJson({
          'state': {'cherry_pick_in_progress': true},
        }).state!.isMidOperation,
        isTrue,
      );
    });

    test('an empty payload does not throw', () {
      final s = VaultStatus.fromJson({});
      expect(s.isRepo, isFalse);
      expect(s.files, isEmpty);
    });
  });

  group('VaultSyncConfig.fromJson', () {
    test('reads intervals and timestamps', () {
      final c = VaultSyncConfig.fromJson({
        'enabled': true,
        'commit_interval': '10m0s',
        'pull_interval': '1h0m0s',
        'push_enabled': true,
        'pull_enabled': false,
        'commit_message': 'Auto-sync: {date}',
        'last_commit_at': '2026-08-10T00:00:00Z',
        'last_commit_hash': 'abc1234',
      });
      expect(c.enabled, isTrue);
      expect(c.commitInterval, '10m0s');
      expect(c.pullInterval, '1h0m0s');
      expect(c.pushEnabled, isTrue);
      expect(c.pullEnabled, isFalse);
      expect(c.commitMessage, 'Auto-sync: {date}');
      expect(c.lastCommitAt, isNotNull);
      expect(c.lastCommitHash, 'abc1234');
    });

    test('absent timestamps stay null rather than becoming the epoch', () {
      // Rendering 1970 as "last push" would read as a real sync.
      final c = VaultSyncConfig.fromJson({'enabled': false});
      expect(c.lastCommitAt, isNull);
      expect(c.lastPushAt, isNull);
      expect(c.lastPullAt, isNull);
      expect(c.lastErrorAt, isNull);
    });

    test('an empty timestamp string is treated as absent', () {
      final c = VaultSyncConfig.fromJson({'last_push_at': ''});
      expect(c.lastPushAt, isNull);
    });
  });
}
