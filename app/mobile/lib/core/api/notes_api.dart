import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/notes/* — read-only access to the operator's notes
// vault. The inspector's Notes tab uses this to surface the per-
// project subset (resolved via /notes/project-mapping?cwd=…).

class NoteSummary {
  NoteSummary({
    required this.path,
    required this.title,
    required this.modified,
    required this.size,
  });

  factory NoteSummary.fromJson(Map<String, dynamic> json) => NoteSummary(
        path: json['path'] as String? ?? '',
        title: json['title'] as String? ?? '',
        modified:
            DateTime.tryParse(json['modified'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
        size: (json['size'] as num?)?.toInt() ?? 0,
      );

  final String path; // vault-relative, e.g. "projects/foo.md"
  final String title;
  final DateTime modified;
  final int size;
}

class FullNote {
  FullNote({
    required this.path,
    required this.title,
    required this.modified,
    required this.size,
    required this.body,
  });

  factory FullNote.fromJson(Map<String, dynamic> json) => FullNote(
        path: json['path'] as String? ?? '',
        title: json['title'] as String? ?? '',
        modified:
            DateTime.tryParse(json['modified'] as String? ?? '')?.toUtc() ??
                DateTime.fromMillisecondsSinceEpoch(0),
        size: (json['size'] as num?)?.toInt() ?? 0,
        body: json['body'] as String? ?? '',
      );

  final String path;
  final String title;
  final DateTime modified;
  final int size;
  final String body;
}

class ProjectMapping {
  ProjectMapping({
    required this.cwd,
    required this.path,
    required this.defaultPath,
    required this.custom,
  });

  factory ProjectMapping.fromJson(Map<String, dynamic> json) => ProjectMapping(
        cwd: json['cwd'] as String? ?? '',
        path: json['path'] as String? ?? '',
        defaultPath: json['default_path'] as String? ?? '',
        custom: json['custom'] as bool? ?? false,
      );

  // Absolute filesystem path resolved as the vault folder for the
  // given session cwd.
  final String cwd;
  final String path;
  final String defaultPath;
  final bool custom;
}

class NotesInfo {
  NotesInfo({
    required this.root,
    required this.personalPrefix,
    required this.projectsPrefix,
  });

  factory NotesInfo.fromJson(Map<String, dynamic> json) => NotesInfo(
        root: json['root'] as String? ?? '',
        personalPrefix: json['personal_prefix'] as String? ?? '',
        projectsPrefix: json['projects_prefix'] as String? ?? '',
      );

  // Absolute filesystem path of the vault root.
  final String root;
  final String personalPrefix;
  final String projectsPrefix;
}

class NotesApi {
  NotesApi(this._dio);
  final Dio _dio;

  Future<NotesInfo> info() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/notes/info');
      return NotesInfo.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<NoteSummary>> list({String? prefix}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/notes/list',
        queryParameters: {if (prefix != null && prefix.isNotEmpty) 'prefix': prefix},
      );
      final raw = res.data?['notes'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(NoteSummary.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<FullNote> read(String path) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/notes/read',
        queryParameters: {'path': path},
      );
      return FullNote.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ProjectMapping> projectMapping(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/notes/project-mapping',
        queryParameters: {'cwd': cwd},
      );
      return ProjectMapping.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final notesApiProvider = Provider<NotesApi>((ref) {
  return NotesApi(ref.watch(dioProvider));
});
