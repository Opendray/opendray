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
    this.personalPath = '',
  });

  factory ProjectMapping.fromJson(Map<String, dynamic> json) => ProjectMapping(
        cwd: json['cwd'] as String? ?? '',
        path: json['path'] as String? ?? '',
        defaultPath: json['default_path'] as String? ?? '',
        custom: json['custom'] as bool? ?? false,
        personalPath: json['personal_path'] as String? ?? '',
      );

  // Absolute filesystem path resolved as the vault folder for the
  // given session cwd.
  final String cwd;
  final String path;
  final String defaultPath;
  final bool custom;

  /// Where this cwd's personal scratchpad belongs, resolved by the
  /// gateway. It depends on the vault layout — flat keeps it inside the
  /// project directory, so a project override moves it too — which is
  /// why the phone does not work it out itself. Empty when talking to a
  /// gateway that predates the field.
  final String personalPath;
}

class NotesInfo {
  NotesInfo({
    required this.root,
    required this.personalPrefix,
    required this.projectsPrefix,
    this.layout = '',
    this.flattenable = false,
  });

  factory NotesInfo.fromJson(Map<String, dynamic> json) => NotesInfo(
        root: json['root'] as String? ?? '',
        personalPrefix: json['personal_prefix'] as String? ?? '',
        projectsPrefix: json['projects_prefix'] as String? ?? '',
        layout: json['layout'] as String? ?? '',
        flattenable: json['flattenable'] as bool? ?? false,
      );

  // Absolute filesystem path of the vault root.
  final String root;
  final String personalPrefix;
  final String projectsPrefix;

  /// "flat" files each project at the vault root under its own name;
  /// "nested" uses the prefixes above. Empty on an older gateway.
  final String layout;

  /// True when this vault still nests projects AND has something the
  /// conversion would move — the phone offers it rather than leaving
  /// the migration to whoever happens to read the release notes.
  final bool flattenable;
}

/// One document the flatten migration would move, or did.
class FlattenMove {
  FlattenMove({required this.from, required this.to, this.linksRewritten = 0});

  factory FlattenMove.fromJson(Map<String, dynamic> json) => FlattenMove(
        from: json['from'] as String? ?? '',
        to: json['to'] as String? ?? '',
        linksRewritten: json['links_rewritten'] as int? ?? 0,
      );

  final String from;
  final String to;
  final int linksRewritten;
}

/// A document the migration refused to touch, with the reason stated
/// in the operator's terms. Surface it — do not summarise it away.
class FlattenSkip {
  FlattenSkip({required this.path, required this.reason});

  factory FlattenSkip.fromJson(Map<String, dynamic> json) => FlattenSkip(
        path: json['path'] as String? ?? '',
        reason: json['reason'] as String? ?? '',
      );

  final String path;
  final String reason;
}

class FlattenResult {
  FlattenResult({
    required this.moves,
    required this.skips,
    required this.dryRun,
    this.mappingsRewritten = 0,
  });

  factory FlattenResult.fromJson(Map<String, dynamic> json) => FlattenResult(
        moves: (json['moves'] as List<dynamic>? ?? [])
            .map((e) => FlattenMove.fromJson(e as Map<String, dynamic>))
            .toList(),
        skips: (json['skips'] as List<dynamic>? ?? [])
            .map((e) => FlattenSkip.fromJson(e as Map<String, dynamic>))
            .toList(),
        dryRun: json['dry_run'] as bool? ?? true,
        mappingsRewritten: json['mappings_rewritten'] as int? ?? 0,
      );

  final List<FlattenMove> moves;
  final List<FlattenSkip> skips;
  final bool dryRun;
  final int mappingsRewritten;
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

  // POST /api/v1/notes/flatten — preview or perform the nested → flat
  // conversion. `apply` defaults to false and must be passed
  // explicitly: this renames every project document in the vault, so
  // looking and rewriting are not one field apart.
  Future<FlattenResult> flatten({bool apply = false}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/notes/flatten',
        data: {'apply': apply},
      );
      return FlattenResult.fromJson(res.data ?? {});
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

  /// GET /api/v1/notes/backlinks?path=… — every note whose body links
  /// to [path]. The scan runs server-side over the whole vault.
  Future<List<Backlink>> backlinks(String path) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/notes/backlinks',
        queryParameters: {'path': path},
      );
      final raw = res.data?['links'];
      if (raw is! List) return const [];
      return raw.whereType<Map<String, dynamic>>().map(Backlink.fromJson).toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// GET /api/v1/notes/tags — every tag in the vault with its note
  /// count. [prefix] restricts the scan to one subtree.
  Future<List<TagCount>> tags({String? prefix}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/notes/tags',
        queryParameters: {
          if (prefix != null && prefix.isNotEmpty) 'prefix': prefix,
        },
      );
      final raw = res.data?['tags'];
      if (raw is! List) return const [];
      return raw.whereType<Map<String, dynamic>>().map(TagCount.fromJson).toList();
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

/// One note that links to the document being viewed, with a couple of
/// matching lines for context. Mirrors notes.Backlink in
/// internal/notes/links.go.
class Backlink {
  const Backlink({
    required this.path,
    required this.title,
    required this.lines,
  });

  factory Backlink.fromJson(Map<String, dynamic> json) => Backlink(
        path: json['path'] as String? ?? '',
        title: json['title'] as String? ?? '',
        lines: (json['lines'] as List<dynamic>? ?? const [])
            .whereType<String>()
            .toList(),
      );

  final String path;
  final String title;

  /// Snippets of the lines carrying the link.
  final List<String> lines;
}

/// One tag and how many notes mention it. Mirrors notes.TagCount in
/// internal/notes/links.go. [notes] is what makes filtering by tag
/// possible without a second round-trip per tag.
class TagCount {
  const TagCount({
    required this.tag,
    required this.count,
    this.notes = const [],
  });

  factory TagCount.fromJson(Map<String, dynamic> json) => TagCount(
        tag: json['tag'] as String? ?? '',
        count: (json['count'] as num?)?.toInt() ?? 0,
        notes: (json['notes'] as List<dynamic>? ?? const [])
            .whereType<String>()
            .toList(),
      );

  final String tag;
  final int count;

  /// Vault paths mentioning the tag. Omitted by the gateway when empty.
  final List<String> notes;
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
