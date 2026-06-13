import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Wraps /api/v1/cortex/* — the unified module governing the
// Memory → Notes → Knowledge flywheel. Cross-layer endpoints only:
// flywheel status, the quarantine review queue, the doc blueprint
// (sections + AI proposer), and curation conversations. The per-layer
// CRUD clients keep their legacy routes (dual-mounted server-side).
class CortexApi {
  CortexApi(this._dio);
  final Dio _dio;

  // ── flywheel status ────────────────────────────────────────────

  Future<CortexStatus> status() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/cortex/status');
      return CortexStatus.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── quarantine review queue ────────────────────────────────────

  Future<(List<Memory>, int)> listQuarantined({int limit = 200}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/cortex/memory/quarantine',
        queryParameters: {'n': limit},
      );
      final raw = res.data?['memories'];
      final rows = raw is List
          ? raw.whereType<Map<String, dynamic>>().map(Memory.fromJson).toList()
          : <Memory>[];
      final count = (res.data?['count'] as num?)?.toInt() ?? rows.length;
      return (rows, count);
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> promoteQuarantined(String id) async {
    try {
      await _dio.post<void>('/api/v1/cortex/memory/quarantine/$id/promote');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> discardQuarantined(String id) async {
    try {
      await _dio.post<void>('/api/v1/cortex/memory/quarantine/$id/discard');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── doc blueprint (sections live on /project-docs/blueprint) ──

  Future<List<BlueprintSection>> listSections(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/project-docs/blueprint',
        queryParameters: {'cwd': cwd},
      );
      final raw = res.data?['sections'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(BlueprintSection.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Asks the AI to classify the project and propose a tailored section
  /// set. Nothing is persisted — apply via [applyBlueprint] on accept.
  Future<BlueprintProposal> proposeBlueprint(String cwd) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/cortex/blueprint/propose',
        queryParameters: {'cwd': cwd},
      );
      return BlueprintProposal.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Replaces the project's blueprint with the given section set
  /// (sections absent from the list are removed; overview is reserved).
  Future<List<BlueprintSection>> applyBlueprint(
    String cwd,
    List<BlueprintSection> sections,
  ) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/cortex/blueprint',
        data: {
          'cwd': cwd,
          'sections': [for (final s in sections) s.toJson()],
        },
      );
      final raw = res.data?['sections'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(BlueprintSection.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── curation conversations ─────────────────────────────────────

  Future<CortexConversation> createConversation({
    required String targetKind,
    required String targetCwd,
    required String targetSlug,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/cortex/conversations',
        data: {
          'target_kind': targetKind,
          'target_cwd': targetCwd,
          'target_slug': targetSlug,
        },
      );
      return CortexConversation.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<CortexConversation>> listConversations({
    String? cwd,
    String? slug,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/cortex/conversations',
        queryParameters: {
          if (cwd != null && cwd.isNotEmpty) 'cwd': cwd,
          if (slug != null && slug.isNotEmpty) 'slug': slug,
        },
      );
      final raw = res.data?['conversations'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CortexConversation.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<(CortexConversation, List<ConversationMessage>)> getConversation(
    String id,
  ) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/cortex/conversations/$id',
      );
      final conv = CortexConversation.fromJson(
        (res.data?['conversation'] as Map?)?.cast<String, dynamic>() ??
            const {},
      );
      final raw = res.data?['messages'];
      final msgs = raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(ConversationMessage.fromJson)
              .toList()
          : <ConversationMessage>[];
      return (conv, msgs);
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Sends an operator message. The AI reply lands asynchronously —
  /// re-poll [getConversation] until the last message isn't ours.
  Future<ConversationMessage> sendMessage(String id, String content) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/cortex/conversations/$id/messages',
        data: {'content': content},
      );
      return ConversationMessage.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Escalates the conversation into a full agent session (grounded in
  /// the codebase). Returns the updated conversation with the session id.
  Future<CortexConversation> escalate(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/cortex/conversations/$id/escalate',
      );
      return CortexConversation.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> closeConversation(String id) async {
    try {
      await _dio.post<void>('/api/v1/cortex/conversations/$id/close');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

// ── models ───────────────────────────────────────────────────────

class CortexStatus {
  CortexStatus({
    required this.notesProjects,
    required this.notesActiveProjects,
    required this.notesPendingProposals,
    required this.memoryEnabled,
    required this.quarantineCount,
    required this.knowledgeEnabled,
    required this.knowledgePendingProposals,
  });

  factory CortexStatus.fromJson(Map<String, dynamic> j) {
    final notes = (j['notes'] as Map?)?.cast<String, dynamic>() ?? const {};
    final mem = (j['memory'] as Map?)?.cast<String, dynamic>() ?? const {};
    final know = (j['knowledge'] as Map?)?.cast<String, dynamic>() ?? const {};
    return CortexStatus(
      notesProjects: (notes['projects'] as num?)?.toInt() ?? 0,
      notesActiveProjects: (notes['active_projects'] as num?)?.toInt() ?? 0,
      notesPendingProposals:
          (notes['pending_proposals'] as num?)?.toInt() ?? 0,
      memoryEnabled: mem['enabled'] == true,
      quarantineCount: (mem['quarantine_count'] as num?)?.toInt() ?? 0,
      knowledgeEnabled: know['enabled'] == true,
      knowledgePendingProposals:
          (know['pending_proposals'] as num?)?.toInt() ?? 0,
    );
  }

  final int notesProjects;
  final int notesActiveProjects;
  final int notesPendingProposals;
  final bool memoryEnabled;
  final int quarantineCount;
  final bool knowledgeEnabled;
  final int knowledgePendingProposals;
}

/// One section of a project's doc blueprint — its slug IS the doc kind.
class BlueprintSection {
  BlueprintSection({
    required this.cwd,
    required this.slug,
    required this.title,
    required this.description,
    required this.position,
    required this.maintainerMode,
    required this.promptHint,
    required this.pinned,
    required this.inject,
  });

  factory BlueprintSection.fromJson(Map<String, dynamic> j) =>
      BlueprintSection(
        cwd: j['cwd']?.toString() ?? '',
        slug: j['slug']?.toString() ?? '',
        title: j['title']?.toString() ?? '',
        description: j['description']?.toString() ?? '',
        position: (j['position'] as num?)?.toInt() ?? 0,
        maintainerMode: j['maintainer_mode']?.toString() ?? 'ai',
        promptHint: j['prompt_hint']?.toString() ?? '',
        pinned: j['pinned'] == true,
        inject: j['inject'] == true,
      );

  Map<String, dynamic> toJson() => {
        'cwd': cwd,
        'slug': slug,
        'title': title,
        'description': description,
        'position': position,
        'maintainer_mode': maintainerMode,
        'prompt_hint': promptHint,
        'pinned': pinned,
        'inject': inject,
      };

  final String cwd;
  final String slug;
  final String title;
  final String description;
  final int position;
  final String maintainerMode; // ai | human | scanner
  final String promptHint;
  final bool pinned;
  final bool inject;
}

class BlueprintProposal {
  BlueprintProposal({
    required this.projectType,
    required this.reason,
    required this.sections,
  });

  factory BlueprintProposal.fromJson(Map<String, dynamic> j) {
    final raw = j['sections'];
    return BlueprintProposal(
      projectType: j['project_type']?.toString() ?? '',
      reason: j['reason']?.toString() ?? '',
      sections: raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(BlueprintSection.fromJson)
              .toList()
          : const [],
    );
  }

  final String projectType;
  final String reason;
  final List<BlueprintSection> sections;
}

class CortexConversation {
  CortexConversation({
    required this.id,
    required this.targetKind,
    required this.targetCwd,
    required this.targetSlug,
    required this.status,
    required this.escalatedSessionId,
  });

  factory CortexConversation.fromJson(Map<String, dynamic> j) =>
      CortexConversation(
        id: j['id']?.toString() ?? '',
        targetKind: j['target_kind']?.toString() ?? '',
        targetCwd: j['target_cwd']?.toString() ?? '',
        targetSlug: j['target_slug']?.toString() ?? '',
        status: j['status']?.toString() ?? 'open',
        escalatedSessionId: j['escalated_session_id']?.toString() ?? '',
      );

  final String id;
  final String targetKind; // doc_section | kb_page | blueprint
  final String targetCwd;
  final String targetSlug;
  final String status; // open | closed | escalated
  final String escalatedSessionId;
}

class ConversationMessage {
  ConversationMessage({
    required this.id,
    required this.role,
    required this.content,
    required this.revisionAction,
    required this.revisionRef,
    required this.createdAt,
  });

  factory ConversationMessage.fromJson(Map<String, dynamic> j) =>
      ConversationMessage(
        id: j['id']?.toString() ?? '',
        role: j['role']?.toString() ?? '',
        content: j['content']?.toString() ?? '',
        revisionAction: j['revision_action']?.toString() ?? '',
        revisionRef: j['revision_ref']?.toString() ?? '',
        createdAt:
            DateTime.tryParse(j['created_at']?.toString() ?? '') ??
                DateTime.now(),
      );

  final String id;
  final String role; // operator | ai | system
  final String content;
  final String revisionAction; // '' | applied | proposed
  final String revisionRef;
  final DateTime createdAt;
}

final cortexApiProvider = Provider<CortexApi>((ref) {
  return CortexApi(ref.watch(dioProvider));
});
