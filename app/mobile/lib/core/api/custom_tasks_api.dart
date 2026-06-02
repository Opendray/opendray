// Wraps /api/v1/custom-tasks — user-defined slash commands that
// the Inspector's task picker exposes inside a session.
//
// "cwd" disambiguates global vs project-scoped tasks: empty cwd =
// visible from any session; absolute path = only visible when the
// session's cwd matches. The mobile management page always passes
// ?all=1 to list every defined task regardless of any session's
// cwd — operators on the mobile UI are editing the catalogue, not
// invoking from a session.

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

class CustomTask {
  CustomTask({
    required this.id,
    required this.name,
    required this.command,
    required this.description,
    required this.cwd,
    required this.createdAt,
    required this.updatedAt,
  });

  factory CustomTask.fromJson(Map<String, dynamic> json) => CustomTask(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        command: json['command'] as String? ?? '',
        description: json['description'] as String? ?? '',
        cwd: json['cwd'] as String? ?? '',
        createdAt:
            DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
        updatedAt:
            DateTime.tryParse(json['updated_at'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
      );

  final String id;
  final String name;
  final String command;
  final String description;
  // Empty = global; absolute path = project-scoped.
  final String cwd;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class CustomTasksApi {
  CustomTasksApi(this._dio);
  final Dio _dio;

  // GET /custom-tasks?all=1 — list every task regardless of cwd.
  Future<List<CustomTask>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/custom-tasks',
        queryParameters: {'all': '1'},
      );
      final raw = res.data?['tasks'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CustomTask.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /custom-tasks?cwd=<path> — globals plus tasks scoped to this
  // cwd. This is what the session inspector's task picker uses: an
  // operator running tasks from inside a session wants the global
  // catalogue and anything pinned to the project they're sitting in,
  // not every scoped task across all projects (that's what list() is
  // for, on the management page).
  Future<List<CustomTask>> listForCwd(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/custom-tasks',
        queryParameters: {if (cwd.isNotEmpty) 'cwd': cwd},
      );
      final raw = res.data?['tasks'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CustomTask.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<CustomTask> create({
    required String name,
    required String command,
    String description = '',
    String cwd = '',
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/custom-tasks',
        data: {
          'name': name,
          'command': command,
          'description': description,
          'cwd': cwd,
        },
      );
      return CustomTask.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH-style — fields omitted are left untouched. We send only
  // the keys the caller actually provided so the server can apply
  // partial updates without us shipping unchanged values.
  Future<CustomTask> update(
    String id, {
    String? name,
    String? command,
    String? description,
    String? cwd,
  }) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/custom-tasks/$id',
        data: {
          if (name != null) 'name': name,
          if (command != null) 'command': command,
          if (description != null) 'description': description,
          if (cwd != null) 'cwd': cwd,
        },
      );
      return CustomTask.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/custom-tasks/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final customTasksApiProvider = Provider<CustomTasksApi>((ref) {
  return CustomTasksApi(ref.watch(dioProvider));
});
