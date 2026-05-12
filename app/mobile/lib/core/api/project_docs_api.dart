import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/project-docs/*, /project-doc-proposals/*, and
// /session-logs/*. Backs the Project screen on the More menu.
class ProjectDocsApi {
  ProjectDocsApi(this._dio);
  final Dio _dio;

  // ── docs (goal / plan) ─────────────────────────────────────────

  Future<List<ProjectDoc>> listDocs(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/project-docs',
        queryParameters: {'cwd': cwd},
      );
      final raw = res.data?['docs'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ProjectDoc.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ProjectDoc> getDoc(String cwd, String kind) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/project-docs/$kind',
        queryParameters: {'cwd': cwd},
      );
      return ProjectDoc.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ProjectDoc> putDoc({
    required String cwd,
    required String kind,
    required String content,
  }) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/project-docs/$kind',
        data: {
          'cwd': cwd,
          'content': content,
          'updated_by': 'operator',
        },
      );
      return ProjectDoc.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── proposals ──────────────────────────────────────────────────

  Future<List<DocProposal>> listPendingProposals({String? cwd}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/project-doc-proposals/pending',
        queryParameters: {
          if (cwd != null && cwd.isNotEmpty) 'cwd': cwd,
        },
      );
      final raw = res.data?['proposals'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(DocProposal.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ProjectDoc> approveProposal(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/project-doc-proposals/$id/approve',
      );
      return ProjectDoc.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> rejectProposal(String id) async {
    try {
      await _dio.post<void>('/api/v1/project-doc-proposals/$id/reject');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── session logs (journal) ─────────────────────────────────────

  Future<List<SessionLogEntry>> listLogs(String cwd, {int limit = 50}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/session-logs',
        queryParameters: {'cwd': cwd, 'n': limit},
      );
      final raw = res.data?['logs'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(SessionLogEntry.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<SessionLogEntry> appendLog({
    required String cwd,
    required String content,
    String? title,
    String kind = 'manual',
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/session-logs',
        data: {
          'cwd': cwd,
          'kind': kind,
          if (title != null && title.isNotEmpty) 'title': title,
          'content': content,
          'updated_by': 'operator',
        },
      );
      return SessionLogEntry.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deleteLog(String id) async {
    try {
      await _dio.delete<void>('/api/v1/session-logs/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

// Server returns Doc{} for non-existent rows (so we always have a
// shape to render); detect "empty" via id.isEmpty.
class ProjectDoc {
  ProjectDoc({
    required this.id,
    required this.cwd,
    required this.kind,
    required this.content,
    required this.updatedBy,
  });

  factory ProjectDoc.fromJson(Map<String, dynamic> j) => ProjectDoc(
        id: j['id']?.toString() ?? '',
        cwd: j['cwd']?.toString() ?? '',
        kind: j['kind']?.toString() ?? '',
        content: j['content']?.toString() ?? '',
        updatedBy: j['updated_by']?.toString() ?? '',
      );

  final String id;
  final String cwd;
  final String kind;
  final String content;
  final String updatedBy;

  bool get isPersisted => id.isNotEmpty;
}

class DocProposal {
  DocProposal({
    required this.id,
    required this.cwd,
    required this.kind,
    required this.proposedContent,
    required this.reason,
    required this.proposedBySession,
    required this.createdAt,
  });

  factory DocProposal.fromJson(Map<String, dynamic> j) => DocProposal(
        id: j['id']?.toString() ?? '',
        cwd: j['cwd']?.toString() ?? '',
        kind: j['kind']?.toString() ?? '',
        proposedContent: j['proposed_content']?.toString() ?? '',
        reason: j['reason']?.toString() ?? '',
        proposedBySession: j['proposed_by_session']?.toString() ?? '',
        createdAt: DateTime.tryParse(j['created_at']?.toString() ?? '') ??
            DateTime.now(),
      );

  final String id;
  final String cwd;
  final String kind;
  final String proposedContent;
  final String reason;
  final String proposedBySession;
  final DateTime createdAt;
}

class SessionLogEntry {
  SessionLogEntry({
    required this.id,
    required this.cwd,
    required this.sessionId,
    required this.kind,
    required this.title,
    required this.content,
    required this.updatedBy,
    required this.createdAt,
  });

  factory SessionLogEntry.fromJson(Map<String, dynamic> j) => SessionLogEntry(
        id: j['id']?.toString() ?? '',
        cwd: j['cwd']?.toString() ?? '',
        sessionId: j['session_id']?.toString() ?? '',
        kind: j['kind']?.toString() ?? '',
        title: j['title']?.toString() ?? '',
        content: j['content']?.toString() ?? '',
        updatedBy: j['updated_by']?.toString() ?? '',
        createdAt: DateTime.tryParse(j['created_at']?.toString() ?? '') ??
            DateTime.now(),
      );

  final String id;
  final String cwd;
  final String sessionId;
  final String kind;
  final String title;
  final String content;
  final String updatedBy;
  final DateTime createdAt;
}

final projectDocsApiProvider = Provider<ProjectDocsApi>((ref) {
  return ProjectDocsApi(ref.watch(dioProvider));
});
