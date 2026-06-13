import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/skills — agent skills (Tier-1 SKILL.md docs the
// gateway injects into provider system prompts at session spawn).
//
// Two sources: builtin (embedded in the gateway binary, read-only)
// and vault (operator-edited markdown files). Editing a builtin
// produces a vault override with the same id; deleting a vault
// override that has a builtin sibling resets back to the builtin.

class SkillSummary {
  SkillSummary({
    required this.id,
    required this.name,
    required this.description,
    required this.source,
    required this.overridesBuiltin,
    required this.hasBuiltin,
  });

  factory SkillSummary.fromJson(Map<String, dynamic> json) => SkillSummary(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        // "vault" | "builtin"
        source: json['source'] as String? ?? '',
        overridesBuiltin: json['overrides_builtin'] as bool? ?? false,
        hasBuiltin: json['has_builtin'] as bool? ?? false,
      );

  final String id;
  final String name;
  final String description;
  final String source;
  // True for vault rows whose id matches a builtin — UI labels this
  // as "overrides built-in" and offers a "Reset to built-in" delete.
  final bool overridesBuiltin;
  // True for builtin rows OR vault rows that override a builtin.
  // Lets the UI offer a "Customize" affordance on builtins (clone
  // the body into a vault entry).
  final bool hasBuiltin;

  bool get isBuiltin => source == 'builtin';
  bool get isVault => source == 'vault';
}

class Skill {
  Skill({
    required this.summary,
    required this.body,
  });

  factory Skill.fromJson(Map<String, dynamic> json) => Skill(
        summary: SkillSummary.fromJson(json),
        body: json['body'] as String? ?? '',
      );

  final SkillSummary summary;
  final String body;
}

class SkillsApi {
  SkillsApi(this._dio);
  final Dio _dio;

  Future<List<SkillSummary>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/skills');
      final raw = res.data?['skills'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(SkillSummary.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<Skill> get(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/skills/$id');
      return Skill.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /skills with `{id, body}`. Server rejects when id already
  // exists in the vault, but accepts ids that exist as a builtin
  // (the create produces a vault override).
  Future<Skill> create({required String id, required String body}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/skills',
        data: {'id': id, 'body': body},
      );
      return Skill.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /skills/upload installs a SKILL.md as a vault skill, deriving
  // the id from the file's frontmatter `name:` (slugified) — no id
  // prompt. The mobile counterpart of the web's drag-and-drop install.
  // Server rejects a blank `name:`, an id collision (409 — delete first),
  // an empty file, or a body over 4 MB.
  Future<Skill> upload({
    required String filename,
    required List<int> bytes,
  }) async {
    try {
      final form = FormData.fromMap({
        'file': MultipartFile.fromBytes(bytes, filename: filename),
      });
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/skills/upload',
        data: form,
      );
      return Skill.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /skills/{id} writes the body to the vault — works whether or
  // not a vault entry already exists. The "customize a builtin" flow
  // uses this with the cloned-from-builtin body.
  Future<Skill> update({required String id, required String body}) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/skills/$id',
        data: {'id': id, 'body': body},
      );
      return Skill.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // DELETE /skills/{id} removes the vault entry. Builtins remain
  // because they're embedded in the binary, so a delete on a vault
  // row that overrides a builtin effectively "resets to builtin".
  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/skills/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final skillsApiProvider = Provider<SkillsApi>((ref) {
  return SkillsApi(ref.watch(dioProvider));
});
