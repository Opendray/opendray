// Release "what's new" feed for the mobile More → Resources → Updates
// sheet. Dart mirror of app/shared/src/lib/releases.ts — keep the two
// in sync (highlight parsing, changelog fallback, read-state semantics).
//
// Source of truth order:
//   1. GitHub Releases (latest) — freshest publish date + notes URL
//   2. CHANGELOG.md on main — when the release body is a stub that
//      only points at the changelog (common for this project)
//
// Read state is local-first (SharedPreferences), mirroring the web
// `opendray:lastReadRelease` localStorage key. A later change can sync
// it per-user via the backend.
//
// IMPORTANT: this talks to github.com, NOT the gateway, so it uses a
// bare Dio() with no interceptors — never the shared `dioProvider`,
// which would attach the operator's gateway bearer token to GitHub and
// turn a GitHub 401/403 (rate limit) into a spurious forced sign-out.

import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _repo = 'Opendray/opendray';
const _releasesLatest = 'https://api.github.com/repos/$_repo/releases/latest';
const _changelogRaw =
    'https://raw.githubusercontent.com/$_repo/main/CHANGELOG.md';
const _changelogHtml = 'https://github.com/$_repo/blob/main/CHANGELOG.md';

/// SharedPreferences key for the last release the operator marked read.
const lastReadReleaseKey = 'opendray.lastReadRelease';

enum ReleaseSource { githubRelease, changelog }

class ReleaseInfo {
  const ReleaseInfo({
    required this.tag,
    required this.version,
    required this.name,
    required this.htmlUrl,
    required this.highlights,
    required this.source,
    this.publishedAt,
  });

  /// Tag as published, e.g. "v2.11.2".
  final String tag;

  /// Normalised without leading "v", e.g. "2.11.2".
  final String version;
  final String name;

  /// ISO publish date when known.
  final String? publishedAt;

  /// GitHub release page (or changelog section fallback).
  final String htmlUrl;

  /// 3–5 short highlight lines for the sheet.
  final List<String> highlights;

  /// Where the highlights were parsed from.
  final ReleaseSource source;
}

String normalizeReleaseVersion(String? v) {
  if (v == null) return '';
  var s = v.trim();
  if (s.toLowerCase().startsWith('v')) s = s.substring(1);
  final plus = s.indexOf('+');
  if (plus >= 0) s = s.substring(0, plus);
  return s;
}

String formatReleaseTag(String version) {
  final n = normalizeReleaseVersion(version);
  return n.isNotEmpty ? 'v$n' : '';
}

/// True when [latestVersion] differs from the last one marked read.
bool isReleaseUnread(String? latestVersion, String? lastRead) {
  final latest = normalizeReleaseVersion(latestVersion);
  if (latest.isEmpty) return false;
  final read = normalizeReleaseVersion(lastRead);
  if (read.isEmpty) return true;
  return read != latest;
}

/// Pull short highlight lines from markdown (release body or CHANGELOG
/// section). Prefers the bold title in `- **Title.** rest` bullets
/// (CHANGELOG style); falls back to the first line of a plain list
/// item. Continuation lines under a bullet are ignored.
List<String> extractHighlights(String markdown, {int limit = 5}) {
  if (markdown.trim().isEmpty) return const [];
  // Drop "Announce on X" / tweet blocks that ship in many release bodies.
  final cleaned = markdown
      .replaceAll(
        RegExp(r'##\s*Announce on X[\s\S]*$', caseSensitive: false),
        '',
      )
      .replaceAll(RegExp(r'```[\s\S]*?```'), '');

  final lines = cleaned.split(RegExp(r'\r?\n'));
  final out = <String>[];
  for (final line in lines) {
    // Only real list starters (not wrapped continuation indented deeper).
    final m = RegExp(r'^\s*[-*+]\s+(.+)$').firstMatch(line);
    if (m == null) continue;
    final item = m.group(1)!;
    final bold = RegExp(r'^\*\*(.+?)\*\*').firstMatch(item);
    var text = bold != null ? bold.group(1)!.trim() : item.trim();
    text = text
        .replaceAll('**', '')
        .replaceAllMapped(RegExp('`([^`]+)`'), (mm) => mm.group(1)!)
        .replaceAllMapped(
          RegExp(r'\[([^\]]+)\]\([^)]+\)'),
          (mm) => mm.group(1)!,
        )
        .replaceAll(RegExp(r'\s+'), ' ')
        .replaceAll(RegExp(r'[.:]+$'), '')
        .trim();
    if (text.length < 4) continue;
    // Cap length so the sheet stays scannable.
    if (text.length > 120) text = '${text.substring(0, 117).trimRight()}…';
    out.add(text);
    if (out.length >= limit) break;
  }
  return out;
}

/// Extract the section under `## [vX.Y.Z]` from Keep-a-Changelog markdown.
String extractChangelogSection(String changelog, String version) {
  final v = normalizeReleaseVersion(version);
  if (v.isEmpty || changelog.isEmpty) return '';
  final esc = RegExp.escape(v);
  // Match ## [v2.11.2], ## [2.11.2], ## v2.11.2, with optional date suffix.
  final re = RegExp(
    '^##\\s*\\[?v?$esc\\]?\\s*(?:—|-|–).*\$',
    multiLine: true,
    caseSensitive: false,
  );
  final reAlt = RegExp(
    '^##\\s*\\[?v?$esc\\]?\\s*\$',
    multiLine: true,
    caseSensitive: false,
  );
  final startMatch = re.firstMatch(changelog) ?? reAlt.firstMatch(changelog);
  if (startMatch == null) return '';
  final from = startMatch.start + startMatch.group(0)!.length;
  final rest = changelog.substring(from);
  final next = RegExp(r'^##\s+', multiLine: true).firstMatch(rest);
  return next != null ? rest.substring(0, next.start) : rest;
}

bool _isStubBody(String body) {
  final t = body.trim();
  if (t.isEmpty) return true;
  // Typical stub: "See CHANGELOG.md for the full release notes." + announce.
  final withoutAnnounce = t
      .replaceAll(
        RegExp(r'##\s*Announce on X[\s\S]*$', caseSensitive: false),
        '',
      )
      .trim();
  if (withoutAnnounce.length < 80) return true;
  if (RegExp(r'see\s+\[?changelog', caseSensitive: false)
          .hasMatch(withoutAnnounce) &&
      extractHighlights(withoutAnnounce).isEmpty) {
    return true;
  }
  return false;
}

class ReleasesApi {
  ReleasesApi([Dio? dio])
      : _dio = dio ??
            Dio(
              BaseOptions(
                connectTimeout: const Duration(seconds: 8),
                receiveTimeout: const Duration(seconds: 15),
                // We parse the body ourselves (JSON for releases, plain
                // text for the changelog) so keep Dio out of it.
                responseType: ResponseType.plain,
                validateStatus: (_) => true,
              ),
            );

  final Dio _dio;

  /// Load the latest release highlights. Prefer GitHub Releases; fall
  /// back to CHANGELOG.md for bullet text.
  Future<ReleaseInfo> fetchLatestReleaseInfo() async {
    final res = await _dio.get<String>(
      _releasesLatest,
      options: Options(headers: {'Accept': 'application/vnd.github+json'}),
    );
    final status = res.statusCode ?? 0;
    if (status < 200 || status >= 300) {
      // No releases API (rate limit / offline) — try changelog head only.
      return _fetchFromChangelogOnly();
    }
    final rel = jsonDecode(res.data ?? '{}') as Map<String, dynamic>;
    final tag = (rel['tag_name'] as String?)?.trim() ?? '';
    final version = normalizeReleaseVersion(tag);
    if (version.isEmpty) throw Exception('github release missing tag_name');

    final body = (rel['body'] as String?) ?? '';
    var highlights = extractHighlights(body);
    var source = ReleaseSource.githubRelease;

    if (_isStubBody(body) || highlights.isEmpty) {
      try {
        final changelog = await _fetchText(_changelogRaw);
        final section = extractChangelogSection(changelog, version);
        final fromLog = extractHighlights(section);
        if (fromLog.isNotEmpty) {
          highlights = fromLog;
          source = ReleaseSource.changelog;
        }
      } on Object {
        // Keep whatever we got from the release body.
      }
    }

    final name = (rel['name'] as String?)?.trim();
    final htmlUrl = (rel['html_url'] as String?)?.trim();
    return ReleaseInfo(
      tag: tag.startsWith('v') ? tag : 'v$version',
      version: version,
      name: (name != null && name.isNotEmpty)
          ? name
          : (tag.isNotEmpty ? tag : 'v$version'),
      publishedAt: rel['published_at'] as String?,
      htmlUrl: (htmlUrl != null && htmlUrl.isNotEmpty)
          ? htmlUrl
          : 'https://github.com/$_repo/releases/tag/v$version',
      highlights: highlights,
      source: source,
    );
  }

  Future<ReleaseInfo> _fetchFromChangelogOnly() async {
    final changelog = await _fetchText(_changelogRaw);
    // First version heading after Unreleased.
    final m = RegExp(
      r'^##\s*\[?(v?\d+\.\d+\.\d+)\]?\s*(?:—|-|–)\s*(\d{4}-\d{2}-\d{2})?',
      multiLine: true,
    ).firstMatch(changelog);
    if (m == null) throw Exception('changelog: no version heading found');
    final version = normalizeReleaseVersion(m.group(1));
    final section = extractChangelogSection(changelog, version);
    final highlights = extractHighlights(section);
    final date = m.group(2);
    return ReleaseInfo(
      tag: formatReleaseTag(version),
      version: version,
      name: formatReleaseTag(version),
      publishedAt: date != null ? '${date}T00:00:00Z' : null,
      htmlUrl: '$_changelogHtml#v${version.replaceAll('.', '')}',
      highlights: highlights,
      source: ReleaseSource.changelog,
    );
  }

  Future<String> _fetchText(String url) async {
    final res = await _dio.get<String>(
      url,
      options: Options(headers: {'Accept': 'text/plain, text/markdown, */*'}),
    );
    final status = res.statusCode ?? 0;
    if (status < 200 || status >= 300) {
      throw Exception('fetch $url: $status');
    }
    return res.data ?? '';
  }
}

final releasesApiProvider = Provider<ReleasesApi>((ref) => ReleasesApi());

final latestReleaseProvider = FutureProvider.autoDispose<ReleaseInfo>((ref) {
  return ref.watch(releasesApiProvider).fetchLatestReleaseInfo();
});

/// Last release the operator marked as read (normalised, e.g. "2.11.2";
/// null when never marked). Persisted to SharedPreferences like the web
/// localStorage key so the unread badge survives restarts.
class LastReadReleaseController extends StateNotifier<String?> {
  LastReadReleaseController() : super(null) {
    _restore();
  }

  Future<void> _restore() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(lastReadReleaseKey);
      final n = normalizeReleaseVersion(raw);
      if (n.isNotEmpty) state = n;
    } on Object {
      // Best-effort; a null state simply badges the newest release.
    }
  }

  Future<void> markRead(String version) async {
    final n = normalizeReleaseVersion(version);
    if (n.isEmpty) return;
    state = n;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(lastReadReleaseKey, formatReleaseTag(n));
    } on Object {
      // Best-effort persistence; the badge still clears this run.
    }
  }
}

final lastReadReleaseProvider =
    StateNotifierProvider<LastReadReleaseController, String?>(
  (ref) => LastReadReleaseController(),
);
