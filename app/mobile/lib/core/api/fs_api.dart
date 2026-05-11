import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/fs/* — server-side filesystem browser used by the
// directory picker. Server runs on the gateway host (LAN /
// Cloudflare tunnel / etc.); the phone has no direct access to
// that filesystem, so the browse loop has to round-trip through
// the gateway.
class FsEntry {
  FsEntry({required this.name, required this.path, required this.isDir});

  factory FsEntry.fromJson(Map<String, dynamic> json) => FsEntry(
        name: json['name'] as String? ?? '',
        path: json['path'] as String? ?? '',
        isDir: json['is_dir'] as bool? ?? false,
      );

  final String name;
  final String path;
  final bool isDir;
}

class FsListResponse {
  FsListResponse({
    required this.path,
    required this.parent,
    required this.entries,
  });

  factory FsListResponse.fromJson(Map<String, dynamic> json) {
    final raw = json['entries'];
    final entries = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(FsEntry.fromJson)
            .toList()
        : <FsEntry>[];
    return FsListResponse(
      path: json['path'] as String? ?? '',
      parent: json['parent'] as String? ?? '',
      entries: entries,
    );
  }

  final String path;
  final String parent;
  final List<FsEntry> entries;
}

class FsApi {
  FsApi(this._dio);
  final Dio _dio;

  Future<FsListResponse> list({String? path}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/fs/list',
        queryParameters: {if (path != null && path.isNotEmpty) 'path': path},
      );
      return FsListResponse.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<String> home() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/fs/home');
      return res.data?['path'] as String? ?? '/';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /api/v1/fs/read — pulls a file's bytes from the gateway
  // host. Server caps the response at 256 KiB (see
  // internal/fs/handler.go); returns the path as the X-OpenDray-Path
  // header. Mobile uses this for the inspector "View" action to
  // peek at a file's text without having to spawn a session.
  Future<List<int>> read(String path) async {
    try {
      final res = await _dio.get<List<int>>(
        '/api/v1/fs/read',
        queryParameters: {'path': path},
        options: Options(
          responseType: ResponseType.bytes,
          headers: {'Accept': 'application/octet-stream'},
        ),
      );
      return res.data ?? <int>[];
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<String> mkdir({required String parent, required String name}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/fs/mkdir',
        data: {'parent': parent, 'name': name},
      );
      return res.data?['path'] as String? ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final fsApiProvider = Provider<FsApi>((ref) {
  return FsApi(ref.watch(dioProvider));
});
