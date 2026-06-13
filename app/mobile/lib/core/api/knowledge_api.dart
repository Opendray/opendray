import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/knowledge/* — the M-KG structured knowledge graph.
// Mirrors the Go shapes in internal/knowledge. Dual-auth (admin OR
// integration); the mobile app uses the operator's bearer token.

class KnowledgeNode {
  KnowledgeNode({
    required this.id,
    required this.kind,
    required this.title,
    required this.body,
    required this.scope,
    required this.scopeKey,
    required this.maturity,
    this.entityType = '',
    this.enabled = true,
    this.useCount = 0,
    this.successCount = 0,
    this.failureCount = 0,
    this.lastUsedAt,
    this.provenance = const {},
  });

  factory KnowledgeNode.fromJson(Map<String, dynamic> json) => KnowledgeNode(
    id: json['id'] as String? ?? '',
    kind: json['kind'] as String? ?? '',
    title: json['title'] as String? ?? '',
    body: json['body'] as String? ?? '',
    scope: json['scope'] as String? ?? '',
    scopeKey: json['scope_key'] as String? ?? '',
    maturity: json['maturity'] as String? ?? '',
    entityType: json['entity_type'] as String? ?? '',
    enabled: json['enabled'] != false,
    useCount: (json['use_count'] as num?)?.toInt() ?? 0,
    successCount: (json['success_count'] as num?)?.toInt() ?? 0,
    failureCount: (json['failure_count'] as num?)?.toInt() ?? 0,
    lastUsedAt: (json['last_used_at'] is String)
        ? DateTime.tryParse(json['last_used_at'] as String)
        : null,
    provenance: (json['provenance'] is Map)
        ? Map<String, dynamic>.from(json['provenance'] as Map)
        : const {},
  );

  final String id;
  final String kind;
  final String title;
  final String body;
  final String scope;
  final String scopeKey;
  final String maturity;
  final String entityType;
  final bool enabled;
  final int useCount;
  final int successCount;
  final int failureCount;
  final DateTime? lastUsedAt;
  final Map<String, dynamic> provenance;
}

class KnowledgeNeighbor {
  KnowledgeNeighbor({
    required this.node,
    required this.edgeType,
    required this.direction,
  });

  factory KnowledgeNeighbor.fromJson(Map<String, dynamic> json) =>
      KnowledgeNeighbor(
        node: KnowledgeNode.fromJson(
          (json['node'] as Map?)?.cast<String, dynamic>() ?? const {},
        ),
        edgeType: json['edge_type'] as String? ?? '',
        direction: json['direction'] as String? ?? '',
      );

  final KnowledgeNode node;
  final String edgeType;
  final String direction;
}

class KnowledgeHit {
  KnowledgeHit({required this.node, required this.similarity});

  factory KnowledgeHit.fromJson(Map<String, dynamic> json) => KnowledgeHit(
    node: KnowledgeNode.fromJson(
      (json['node'] as Map?)?.cast<String, dynamic>() ?? const {},
    ),
    similarity: (json['similarity'] as num?)?.toDouble() ?? 0,
  );

  final KnowledgeNode node;
  final double similarity;
}

class KnowledgeApi {
  KnowledgeApi(this._dio);
  final Dio _dio;

  Future<List<KnowledgeNode>> list({
    String? kind,
    String? scope,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/knowledge/nodes',
        queryParameters: {
          if (kind != null && kind.isNotEmpty) 'kind': kind,
          if (scope != null && scope.isNotEmpty) 'scope': scope,
        },
      );
      final raw = res.data?['nodes'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(KnowledgeNode.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<KnowledgeHit>> search({
    required String query,
    String cwd = '',
    int topK = 20,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/knowledge/search',
        queryParameters: {
          'q': query,
          if (cwd.isNotEmpty) 'cwd': cwd,
          'top_k': topK,
        },
      );
      final raw = res.data?['hits'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(KnowledgeHit.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<KnowledgeNeighbor>> graph(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/knowledge/nodes/$id/graph',
      );
      final raw = res.data?['neighbors'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(KnowledgeNeighbor.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> promote(
    String id, {
    String scope = 'global',
    String scopeKey = '',
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/knowledge/nodes/$id/promote',
        data: {'scope': scope, 'scope_key': scopeKey},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<KnowledgeNode> skillify(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/knowledge/nodes/$id/skillify',
      );
      return KnowledgeNode.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Flips a skill's enabled flag — writes/removes the vault SKILL.md so
  // sessions only load enabled skills.
  Future<KnowledgeNode> setEnabled(String id, {required bool enabled}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/knowledge/nodes/$id/enable',
        data: {'enabled': enabled},
      );
      return KnowledgeNode.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // draftKb triggers a background regeneration of all curated KB pages.
  Future<void> draftKb() async {
    try {
      await _dio.post<void>('/api/v1/knowledge/kb/draft');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // delete removes a node — used to undo an accidental promote / skillify.
  // Skills stay deleted (SKILL.md removed server-side); auto-derived
  // facts/entities may re-appear on the next anchor sweep.
  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/knowledge/nodes/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final knowledgeApiProvider = Provider<KnowledgeApi>((ref) {
  return KnowledgeApi(ref.watch(dioProvider));
});
