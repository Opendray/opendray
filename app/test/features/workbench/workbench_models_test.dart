import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/workbench_models.dart';

void main() {
  group('FlatContributions.fromJson', () {
    test('empty payload parses to empty slots', () {
      final c = FlatContributions.fromJson({});
      expect(c.commands, isEmpty);
      expect(c.statusBar, isEmpty);
      expect(c.keybindings, isEmpty);
      expect(c.menus, isEmpty);
    });

    test('missing fields default to empty lists / maps', () {
      final c = FlatContributions.fromJson({
        'commands': null,
        'statusBar': 'not a list',
        'keybindings': [],
        'menus': null,
      });
      expect(c.commands, isEmpty);
      expect(c.statusBar, isEmpty);
      expect(c.keybindings, isEmpty);
      expect(c.menus, isEmpty);
    });

    test('parses time-ninja shape end to end', () {
      final json = {
        'commands': [
          {
            'pluginName': 'time-ninja',
            'id': 'time.start',
            'title': 'Start Pomodoro',
            'category': 'Time Ninja',
          },
        ],
        'statusBar': [
          {
            'pluginName': 'time-ninja',
            'id': 'time.bar',
            'text': '🍅 25:00',
            'tooltip': 'Start a pomodoro',
            'command': 'time.start',
            'alignment': 'right',
            'priority': 50,
          },
        ],
        'keybindings': [
          {
            'pluginName': 'time-ninja',
            'command': 'time.start',
            'key': 'ctrl+alt+p',
            'mac': 'cmd+alt+p',
          },
        ],
        'menus': {
          'appBar/right': [
            {
              'pluginName': 'time-ninja',
              'command': 'time.start',
              'group': 'timer@1',
            },
          ],
        },
      };
      final c = FlatContributions.fromJson(json);
      expect(c.commands.single.id, 'time.start');
      expect(c.commands.single.pluginName, 'time-ninja');
      expect(c.statusBar.single.text, '🍅 25:00');
      expect(c.statusBar.single.priority, 50);
      expect(c.keybindings.single.key, 'ctrl+alt+p');
      expect(c.keybindings.single.mac, 'cmd+alt+p');
      expect(c.menus['appBar/right']!.single.command, 'time.start');
    });

    test('ignores non-map list entries without crashing', () {
      final c = FlatContributions.fromJson({
        'commands': ['stringy', 42, null],
        'statusBar': [
          {'id': 'a', 'text': 'A', 'pluginName': 'p'},
        ],
      });
      expect(c.commands, isEmpty);
      expect(c.statusBar.single.id, 'a');
    });
  });

  group('InvokeResult.fromJson', () {
    test('notify kind', () {
      final r = InvokeResult.fromJson({
        'kind': 'notify',
        'message': 'hi',
      });
      expect(r.kind, 'notify');
      expect(r.message, 'hi');
    });

    test('exec kind with exit and output', () {
      final r = InvokeResult.fromJson({
        'kind': 'exec',
        'output': 'hello\n',
        'exit': 0,
      });
      expect(r.kind, 'exec');
      expect(r.exit, 0);
      expect(r.output, 'hello\n');
    });

    test('openView kind surfaces viewId', () {
      final r = InvokeResult.fromJson({
        'kind': 'openView',
        'viewId': 'kanban.board',
      });
      expect(r.kind, 'openView');
      expect(r.viewId, 'kanban.board');
    });

    test('defaults fill missing fields', () {
      final r = InvokeResult.fromJson({});
      expect(r.kind, '');
      expect(r.exit, 0);
    });
  });

  group('Typed exceptions', () {
    test('PluginPermissionDeniedException renders useful message', () {
      final e = PluginPermissionDeniedException('p', 'cmd', 'nope');
      expect(e.toString(), contains('Permission denied'));
      expect(e.toString(), contains('p.cmd'));
      expect(e.toString(), contains('nope'));
    });

    test('PluginCommandUnavailableException distinguishes deferred flag', () {
      final notFound = PluginCommandUnavailableException('p', 'c', 'gone');
      expect(notFound.deferred, isFalse);
      final deferred = PluginCommandUnavailableException('p', 'c', 'm2',
          deferred: true);
      expect(deferred.deferred, isTrue);
      expect(deferred.toString(), contains('p.c'));
    });
  });
}
