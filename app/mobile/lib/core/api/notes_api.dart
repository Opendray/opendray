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

  // PUT /api/v1/notes/project-mapping — pin a cwd to a vault-relative
  // path. Empty `path` clears the override (revert to default).
  Future<void> setProjectMapping({
    required String cwd,
    required String path,
  }) async {
    try {
      await _dio.put<void>(
        '/api/v1/notes/project-mapping',
        data: {'cwd': cwd, 'path': path},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /api/v1/notes/write — overwrite a note's full body. Used by
  // the personal scratchpad's auto-save and by "New doc" creation.
  // DELETE /api/v1/notes/delete?path=… — irreversible. Server responds
  // 204 on success, 404 if the path didn't exist.
  Future<void> delete(String path) async {
    try {
      await _dio.delete<void>(
        '/api/v1/notes/delete',
        queryParameters: {'path': path},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<NoteSummary> write({required String path, required String body}) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/notes/write',
        data: {'path': path, 'body': body},
      );
      return NoteSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<NoteTemplate>> templates() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/notes/templates');
      final raw = res.data?['templates'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(NoteTemplate.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Create a note from a template. Separate from [write] because it
  /// refuses to overwrite, and because the placeholders are rendered
  /// server-side — so a doc started here and one started on the web
  /// come out identical instead of drifting.
  Future<NoteSummary> newFromTemplate({
    required String path,
    required String template,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/notes/new',
        data: {'path': path, 'template': template},
      );
      return NoteSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Move or rename a note, repointing the [[wiki-links]] that pointed
  /// at it. Without the rewrite a rename silently strands every
  /// reference, which is why reorganising was previously unsafe.
  Future<MoveResult> move({required String from, required String to}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/notes/move',
        data: {'from': from, 'to': to},
      );
      return MoveResult.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

/// One starting shape for a new note. [source] is "builtin" or "vault"
/// — the latter is a template the operator authored under `_templates/`.
class NoteTemplate {
  const NoteTemplate({
    required this.id,
    required this.name,
    required this.source,
  });

  factory NoteTemplate.fromJson(Map<String, dynamic> json) => NoteTemplate(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        source: json['source'] as String? ?? 'builtin',
      );

  final String id;
  final String name;
  final String source;
}

/// Filenames treated as a folder's index page, in preference order.
/// Mirrors INDEX_NAMES in app/shared/src/lib/notes.ts — resolved on the
/// client because the listing is already here.
const kIndexNames = ['README.md', 'index.md', '_index.md'];

/// Outcome of a move. [warning] is set when the file moved but the link
/// rewrite didn't finish — the move still happened, so the UI must not
/// report it as a failure.
class MoveResult {
  const MoveResult({
    required this.to,
    required this.linksRewritten,
    required this.notesRewritten,
    this.warning,
  });

  factory MoveResult.fromJson(Map<String, dynamic> json) {
    // The handler nests the result under `moved` when it also has a
    // warning to report.
    final moved = json['moved'];
    final body = moved is Map<String, dynamic> ? moved : json;
    final rewritten = body['rewritten_in'];
    return MoveResult(
      to: (body['to'] ?? '') as String,
      linksRewritten: (body['links_rewritten'] ?? 0) as int,
      notesRewritten: rewritten is List ? rewritten.length : 0,
      warning: json['warning'] as String?,
    );
  }

  final String to;
  final int linksRewritten;
  final int notesRewritten;
  final String? warning;
}

final notesApiProvider = Provider<NotesApi>((ref) {
  return NotesApi(ref.watch(dioProvider));
});
