import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// /api/v1/providers — list of CLI providers configured on the
// gateway (Claude Code / Codex / Gemini / etc.). Used by the spawn-
// session form to populate the provider picker, and by the
// More → Providers screen for enable toggles + config edits.

// ConfigField mirrors internal/catalog/manifest.go's ConfigField —
// the schema entries the mobile config editor renders dynamically
// instead of carrying per-provider hardcoded forms.
class ConfigField {
  ConfigField({
    required this.key,
    required this.label,
    required this.type,
    this.defaultValue,
    this.options,
    this.placeholder,
    this.description,
    this.group,
    this.envVar,
    this.dependsOn,
    this.dependsVal,
  });

  factory ConfigField.fromJson(Map<String, dynamic> json) {
    final opts = json['options'];
    return ConfigField(
      key: json['key'] as String? ?? '',
      label: (json['label'] as String?)?.isNotEmpty ?? false
          ? json['label'] as String
          : (json['key'] as String? ?? ''),
      type: json['type'] as String? ?? 'string',
      defaultValue: json['default'],
      options: opts is List ? opts.whereType<String>().toList() : null,
      placeholder: json['placeholder'] as String?,
      description: json['description'] as String?,
      group: json['group'] as String?,
      envVar: json['envVar'] as String?,
      dependsOn: json['dependsOn'] as String?,
      dependsVal: json['dependsVal'],
    );
  }

  final String key;
  final String label;
  // string | number | boolean | select | secret | args
  final String type;
  final Object? defaultValue;
  final List<String>? options;
  final String? placeholder;
  final String? description;
  final String? group;
  final String? envVar;
  // For conditional fields: shown only when [dependsOn] field's value
  // equals [dependsVal]. Matches the web admin's behavior.
  final String? dependsOn;
  final Object? dependsVal;
}

// ProviderDetail is the full /providers/{id} response: manifest +
// configSchema + persisted config + enabled flag.
class ProviderDetail {
  ProviderDetail({
    required this.id,
    required this.displayName,
    required this.description,
    required this.version,
    required this.kind,
    required this.config,
    required this.configSchema,
    required this.enabled,
  });

  factory ProviderDetail.fromJson(Map<String, dynamic> json) {
    final manifest = json['manifest'] as Map<String, dynamic>? ?? {};
    final cfg = json['config'];
    final schema = manifest['configSchema'];
    return ProviderDetail(
      id: manifest['id'] as String? ?? '',
      displayName: manifest['displayName'] as String? ??
          manifest['displayName_zh'] as String? ??
          manifest['id'] as String? ??
          '',
      description: manifest['description'] as String? ??
          manifest['description_zh'] as String? ??
          '',
      version: manifest['version'] as String? ?? '',
      kind: manifest['kind'] as String? ?? '',
      config: cfg is Map ? Map<String, dynamic>.from(cfg) : <String, dynamic>{},
      configSchema: schema is List
          ? schema
              .whereType<Map<String, dynamic>>()
              .map(ConfigField.fromJson)
              .toList()
          : <ConfigField>[],
      enabled: json['enabled'] as bool? ?? false,
    );
  }

  final String id;
  final String displayName;
  final String description;
  final String version;
  final String kind;
  final Map<String, dynamic> config;
  final List<ConfigField> configSchema;
  final bool enabled;
}

class ProvidersApi {
  ProvidersApi(this._dio);
  final Dio _dio;

  Future<List<ProviderSummary>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/providers');
      final raw = res.data?['providers'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ProviderSummary.fromGatewayJson)
          .where((p) => p.id.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /providers/{id} — full record including manifest + schema.
  Future<ProviderDetail> get(String id) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/providers/$id');
      return ProviderDetail.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /providers/{id}/toggle. Server fires `provider.toggle` audit
  // event and refuses to disable the only enabled provider — handler
  // returns the patched record on success.
  Future<void> setEnabled(String id, {required bool enabled}) async {
    try {
      await _dio.patch<Map<String, dynamic>>(
        '/api/v1/providers/$id/toggle',
        data: {'enabled': enabled},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /providers/{id}/config replaces the whole config map.
  // Caller is responsible for merging — send the complete desired
  // config, not just the deltas.
  Future<void> updateConfig(String id, Map<String, dynamic> config) async {
    try {
      await _dio.patch<Map<String, dynamic>>(
        '/api/v1/providers/$id/config',
        data: config,
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final providersApiProvider = Provider<ProvidersApi>((ref) {
  return ProvidersApi(ref.watch(dioProvider));
});

final providersListProvider =
    FutureProvider.autoDispose<List<ProviderSummary>>((ref) {
  return ref.watch(providersApiProvider).list();
});
