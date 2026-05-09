import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/channels — notification destinations (Slack/Feishu/
// DingTalk/WeCom/bridge). Mobile UI is read-only with three actions:
// test-send, toggle-enabled, toggle-mute. Create/edit/delete stays on
// the web admin because per-kind config schemas vary too much to
// build a usable mobile form.

class ChannelView {
  ChannelView({
    required this.id,
    required this.kind,
    required this.config,
    required this.enabled,
    required this.running,
    required this.muted,
    required this.capabilities,
  });

  factory ChannelView.fromJson(Map<String, dynamic> json) {
    final cfg = json['config'];
    final config = cfg is Map ? Map<String, dynamic>.from(cfg) : <String, dynamic>{};
    final caps = json['capabilities'];
    final capabilities = caps is List
        ? caps.whereType<String>().toList()
        : <String>[];
    return ChannelView(
      id: json['id'] as String? ?? '',
      kind: json['kind'] as String? ?? '',
      config: config,
      enabled: json['enabled'] as bool? ?? false,
      running: json['running'] as bool? ?? false,
      muted: json['muted'] as bool? ?? false,
      capabilities: capabilities,
    );
  }

  final String id;
  final String kind;
  // Raw kind-specific config blob — opaque on mobile; just shown as
  // formatted JSON in the detail action sheet.
  final Map<String, dynamic> config;
  final bool enabled;
  // running implies enabled+started — server reports both; only enabled
  // is operator-controlled.
  final bool running;
  final bool muted;
  final List<String> capabilities;
}

class ChannelsApi {
  ChannelsApi(this._dio);
  final Dio _dio;

  Future<List<ChannelView>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/channels');
      final raw = res.data?['channels'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ChannelView.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ChannelView> get(String id) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/channels/$id');
      return ChannelView.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /channels/{id} with `enabled` only — leaves the config blob
  // untouched on the server side.
  Future<ChannelView> setEnabled(String id, {required bool enabled}) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/channels/$id',
        data: {'enabled': enabled},
      );
      return ChannelView.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Mute lives inside the config JSON — there's no top-level field.
  // Server's PATCH config does a full replace, so we read first, merge
  // `muted: <new>`, then write the merged blob back.
  Future<ChannelView> setMuted(String id, {required bool muted}) async {
    try {
      final current = await get(id);
      final merged = Map<String, dynamic>.from(current.config)
        ..['muted'] = muted;
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/channels/$id',
        data: {'config': merged},
      );
      return ChannelView.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /channels/{id}/test — server picks the kind-specific test
  // payload (a "hello from opendray" line for text channels, etc.).
  Future<void> test(String id) async {
    try {
      await _dio.post<void>('/api/v1/channels/$id/test');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final channelsApiProvider = Provider<ChannelsApi>((ref) {
  return ChannelsApi(ref.watch(dioProvider));
});
