import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// /api/v1/version — gateway version + update-availability, mobile mirror
// of app/shared/src/lib/version.ts. Read-only here: the mobile About
// screen displays the running gateway version and whether an upgrade is
// available, but does NOT trigger the in-app self-update (that restarts
// the gateway binary and is left to the web dashboard / host shell).
//
// The endpoint soft-fails: when the upstream release check is
// unreachable it still returns current/commit + capability flags and
// sets checkError, so the call always resolves with useful data.

class VersionInfo {
  VersionInfo({
    required this.current,
    required this.updateAvailable,
    required this.selfUpdate,
    required this.pending,
    this.commit,
    this.latest,
    this.notesUrl,
    this.checkError,
  });

  factory VersionInfo.fromJson(Map<String, dynamic> j) {
    String? str(String key) {
      final v = j[key];
      return v is String && v.isNotEmpty ? v : null;
    }

    return VersionInfo(
      current: (j['current'] as String?) ?? '',
      updateAvailable: j['updateAvailable'] as bool? ?? false,
      selfUpdate: j['selfUpdate'] as bool? ?? false,
      pending: j['pending'] as bool? ?? false,
      commit: str('commit'),
      latest: str('latest'),
      notesUrl: str('notesUrl'),
      checkError: str('checkError'),
    );
  }

  final String current;
  final bool updateAvailable;
  final bool selfUpdate; // gateway is capable of in-app self-update
  final bool pending; // an upgrade is already in progress
  final String? commit;
  final String? latest;
  final String? notesUrl;
  final String? checkError; // set when the release check couldn't run
}

class VersionApi {
  VersionApi(this._dio);
  final Dio _dio;

  Future<VersionInfo> getVersionInfo() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/version');
      return VersionInfo.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final versionApiProvider = Provider<VersionApi>((ref) {
  return VersionApi(ref.watch(dioProvider));
});

final versionInfoProvider = FutureProvider.autoDispose<VersionInfo>((ref) {
  return ref.watch(versionApiProvider).getVersionInfo();
});
