import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/plugin_bridge_channel.dart';
import 'package:opendray/features/workbench/webview_host.dart';

/// Pairs a client-side stream/sink with a `peer` sink/stream so tests
/// can script server frames and inspect what the channel sent.
class _FakeEndpoint {
  _FakeEndpoint() {
    _clientOut = StreamController<dynamic>.broadcast();
    _serverOut = StreamController<dynamic>.broadcast();
  }

  late final StreamController<dynamic> _clientOut;
  late final StreamController<dynamic> _serverOut;

  /// Stream of envelopes the client has sent (server-side view).
  Stream<dynamic> get sentByClient => _clientOut.stream;

  /// Incoming stream the channel reads (as if from the server).
  Stream<dynamic> get incoming => _serverOut.stream;

  /// Outgoing sink the channel writes to (as if to the server).
  StreamSink<dynamic> get outgoing => _clientOut.sink;

  /// Scripts a server → client envelope.
  void serverSend(Map<String, dynamic> envelope) {
    _serverOut.add(jsonEncode(envelope));
  }

  Future<void> close() async {
    if (!_clientOut.isClosed) await _clientOut.close();
    if (!_serverOut.isClosed) await _serverOut.close();
  }
}

Map<String, dynamic> _parseFrame(dynamic frame) {
  final s = frame is String ? frame : utf8.decode(frame as List<int>);
  return (jsonDecode(s) as Map).cast<String, dynamic>();
}

PluginBridgeChannel _makeChannel(_FakeEndpoint endpoint) {
  return PluginBridgeChannel.withStreams(
    incoming: endpoint.incoming,
    outgoing: endpoint.outgoing,
  );
}

void main() {
  group('PluginBridgeChannel.call', () {
    test('sends an envelope with auto-incrementing id', () async {
      final endpoint = _FakeEndpoint();
      final ch = _makeChannel(endpoint);
      addTearDown(() async {
        await ch.dispose();
        await endpoint.close();
      });

      final frames = <Map<String, dynamic>>[];
      final sub = endpoint.sentByClient.listen((f) => frames.add(_parseFrame(f)));
      addTearDown(sub.cancel);

      // Fire-and-forget: we care about the outgoing frames, not the
      // responses — attach a .catchError so the pending rejections at
      // teardown don't surface as uncaught async errors.
      ch.call('workbench', 'showMessage', <dynamic>['hi']).catchError((_) => null);
      ch.call('storage', 'get', <dynamic>['cards']).catchError((_) => null);

      await Future<void>.delayed(Duration.zero);
      expect(frames, hasLength(2));
      expect(frames[0]['v'], 1);
      expect(frames[0]['ns'], 'workbench');
      expect(frames[0]['method'], 'showMessage');
      expect(frames[0]['args'], <dynamic>['hi']);
      expect(frames[0]['id'], '1');
      expect(frames[1]['id'], '2');
      expect(frames[1]['ns'], 'storage');
    });

    test('resolves on a matching response envelope', () async {
      final endpoint = _FakeEndpoint();
      final ch = _makeChannel(endpoint);
      addTearDown(() async {
        await ch.dispose();
        await endpoint.close();
      });

      final future = ch.call('storage', 'get', <dynamic>['k']);
      await Future<void>.delayed(Duration.zero);

      endpoint.serverSend(<String, dynamic>{
        'v': 1,
        'id': '1',
        'result': <String, dynamic>{'x': 42},
      });

      final result = await future;
      expect(result, <String, dynamic>{'x': 42});
    });

    test('rejects with a BridgeException on an error envelope', () async {
      final endpoint = _FakeEndpoint();
      final ch = _makeChannel(endpoint);
      addTearDown(() async {
        await ch.dispose();
        await endpoint.close();
      });

      final future = ch.call('storage', 'set', <dynamic>[]);
      await Future<void>.delayed(Duration.zero);

      endpoint.serverSend(<String, dynamic>{
        'v': 1,
        'id': '1',
        'error': <String, dynamic>{
          'code': 'EPERM',
          'message': 'storage not granted',
        },
      });

      await expectLater(
        future,
        throwsA(
          isA<BridgeException>()
              .having((e) => e.code, 'code', 'EPERM')
              .having((e) => e.message, 'message', 'storage not granted'),
        ),
      );
    });
  });

  group('PluginBridgeChannel.subscribe', () {
    test('emits chunks until stream:end', () async {
      final endpoint = _FakeEndpoint();
      final ch = _makeChannel(endpoint);
      addTearDown(() async {
        await ch.dispose();
        await endpoint.close();
      });

      final stream = ch.subscribe('events', 'subscribe', <dynamic>['session.*']);
      await Future<void>.delayed(Duration.zero);

      endpoint.serverSend(<String, dynamic>{
        'v': 1,
        'id': '1',
        'stream': 'chunk',
        'data': <String, dynamic>{'type': 'session.idle'},
      });
      endpoint.serverSend(<String, dynamic>{
        'v': 1,
        'id': '1',
        'stream': 'chunk',
        'data': <String, dynamic>{'type': 'session.resume'},
      });
      endpoint.serverSend(<String, dynamic>{
        'v': 1,
        'id': '1',
        'stream': 'end',
      });

      final collected = await stream.toList();
      expect(collected, hasLength(2));
      expect((collected[0] as Map)['type'], 'session.idle');
      expect((collected[1] as Map)['type'], 'session.resume');
    });
  });

  group('PluginBridgeChannel.dispose', () {
    test('rejects all pending calls', () async {
      final endpoint = _FakeEndpoint();
      final ch = _makeChannel(endpoint);
      addTearDown(endpoint.close);

      final a = ch.call('workbench', 'showMessage', <dynamic>['a']);
      final b = ch.call('workbench', 'showMessage', <dynamic>['b']);

      // Attach the failure matchers BEFORE dispose runs so the
      // completer rejections have a subscribed handler when they fire.
      final aFail = expectLater(a, throwsA(isA<StateError>()));
      final bFail = expectLater(b, throwsA(isA<StateError>()));

      await Future<void>.delayed(Duration.zero);
      await ch.dispose();

      await aFail;
      await bFail;
    });
  });

  group('URL helpers', () {
    test('buildPluginAssetUrl includes token query param', () {
      final u = buildPluginAssetUrl(
        'https://host:8080',
        'kanban',
        'index.html',
        'tok-123',
      );
      expect(u.scheme, 'https');
      expect(u.host, 'host');
      expect(u.port, 8080);
      expect(u.path, '/api/plugins/kanban/assets/index.html');
      expect(u.queryParameters['token'], 'tok-123');
    });

    test('buildPluginBridgeUrl swaps http→ws and https→wss', () {
      final plain = buildPluginBridgeUrl('http://127.0.0.1:8640', 'kanban');
      expect(plain.scheme, 'ws');
      expect(plain.path, '/api/plugins/kanban/bridge/ws');

      final tls = buildPluginBridgeUrl('https://api.example.com', 'kanban');
      expect(tls.scheme, 'wss');
      expect(tls.path, '/api/plugins/kanban/bridge/ws');
    });
  });

  group('PluginWebView widget', () {
    testWidgets('builds without crashing and shows a loading indicator',
        (tester) async {
      final endpoint = _FakeEndpoint();
      final bridge = _makeChannel(endpoint);
      addTearDown(() async {
        await bridge.dispose();
        await endpoint.close();
      });

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: PluginWebView(
              pluginName: 'kanban',
              viewId: 'board',
              entryPath: 'index.html',
              baseUrl: 'http://127.0.0.1:8640',
              bearerToken: 'tok-abc',
              bridgeFactory: (_, _) => bridge,
              skipControllerForTests: true,
            ),
          ),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('disposes the bridge channel on unmount', (tester) async {
      final endpoint = _FakeEndpoint();
      final bridge = _TrackingBridge(
        incoming: endpoint.incoming,
        outgoing: endpoint.outgoing,
      );
      addTearDown(endpoint.close);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: PluginWebView(
              pluginName: 'kanban',
              viewId: 'board',
              entryPath: 'index.html',
              baseUrl: 'http://127.0.0.1:8640',
              bearerToken: 'tok-abc',
              bridgeFactory: (_, _) => bridge,
              skipControllerForTests: true,
            ),
          ),
        ),
      );

      expect(bridge.disposed, isFalse);
      await tester.pumpWidget(const MaterialApp(home: SizedBox.shrink()));
      await tester.pumpAndSettle();
      expect(bridge.disposed, isTrue);
    });
  });

  group('pluginPreloadShim', () {
    test('exposes workbench/storage/events namespaces verbatim', () {
      expect(pluginPreloadShim, contains('window.opendray'));
      expect(pluginPreloadShim, contains('OpenDrayBridge.postMessage'));
      expect(pluginPreloadShim, contains('__opendray_onMessage'));
      expect(
        pluginPreloadShim,
        contains('"showMessage","openView","updateStatusBar"'),
      );
      expect(pluginPreloadShim, contains('"get","set","delete","list"'));
      // Line budget: the plan caps the shim at 40 lines — assert so we
      // notice if someone inflates it on a future edit.
      final nonEmpty = pluginPreloadShim
          .split('\n')
          .where((l) => l.trim().isNotEmpty)
          .length;
      expect(nonEmpty, lessThanOrEqualTo(40));
    });
  });
}

/// Tracks [dispose] so widget tests can assert unmount cleanup.
class _TrackingBridge extends PluginBridgeChannel {
  _TrackingBridge({
    required super.incoming,
    required super.outgoing,
  }) : super.withStreams();

  bool disposed = false;

  @override
  Future<void> dispose() async {
    disposed = true;
    await super.dispose();
  }
}
