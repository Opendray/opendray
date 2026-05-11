import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/git-hosts — registered git host credentials (GitHub,
// GitLab, Bitbucket, custom). Sessions consume these via /git/prs +
// /git/remote when listing PRs or detecting the host of the active
// repo. Tokens are write-only over the API; the list response carries
// a token_mask preview only.

class GitHost {
  GitHost({
    required this.id,
    required this.kind,
    required this.host,
    required this.name,
    required this.tokenMask,
    required this.enabled,
    required this.createdAt,
    required this.updatedAt,
  });

  factory GitHost.fromJson(Map<String, dynamic> json) => GitHost(
        id: json['id'] as String? ?? '',
        // github | gitlab | bitbucket | gitea | custom
        kind: json['kind'] as String? ?? '',
        host: json['host'] as String? ?? '',
        name: json['name'] as String? ?? '',
        tokenMask: json['token_mask'] as String? ?? '',
        enabled: json['enabled'] as bool? ?? false,
        createdAt:
            DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
        updatedAt:
            DateTime.tryParse(json['updated_at'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
      );

  final String id;
  final String kind;
  // Full host URL (e.g. "api.github.com" or self-hosted GitLab base).
  final String host;
  final String name;
  // Server-redacted preview ("ghp_…abcd"); the raw token is never
  // returned by the API.
  final String tokenMask;
  final bool enabled;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class GitHostsApi {
  GitHostsApi(this._dio);
  final Dio _dio;

  Future<List<GitHost>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/git-hosts');
      final raw = res.data?['hosts'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(GitHost.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<GitHost> create({
    required String kind,
    required String host,
    required String name,
    required String token,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/git-hosts',
        data: {
          'kind': kind,
          'host': host,
          'name': name,
          'token': token,
        },
      );
      return GitHost.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /git-hosts/{id} — pointer-style patch. Fields not provided
  // stay untouched. Token is write-only; sending null leaves the
  // existing one intact.
  Future<GitHost> update(
    String id, {
    String? kind,
    String? host,
    String? name,
    String? token,
    bool? enabled,
  }) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/git-hosts/$id',
        data: {
          if (kind != null) 'kind': kind,
          if (host != null) 'host': host,
          if (name != null) 'name': name,
          if (token != null) 'token': token,
          if (enabled != null) 'enabled': enabled,
        },
      );
      return GitHost.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/git-hosts/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final gitHostsApiProvider = Provider<GitHostsApi>((ref) {
  return GitHostsApi(ref.watch(dioProvider));
});
