import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Wraps /api/v1/audit/log — cursor-paginated read of the gateway's
// audit log. Mobile's Activity tab uses this for an infinite-scroll
// feed; subject_kind / action / time-range filters are exposed but
// only the mobile-relevant subset is wired in the UI for now.
class AuditApi {
  AuditApi(this._dio);
  final Dio _dio;

  Future<AuditPage> log({
    String? subjectKind,
    String? subjectId,
    String? action,
    DateTime? since,
    DateTime? until,
    String? cursor,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/audit/log',
        queryParameters: {
          if (subjectKind != null && subjectKind.isNotEmpty)
            'subject_kind': subjectKind,
          if (subjectId != null && subjectId.isNotEmpty)
            'subject_id': subjectId,
          if (action != null && action.isNotEmpty) 'action': action,
          if (since != null) 'since': since.toUtc().toIso8601String(),
          if (until != null) 'until': until.toUtc().toIso8601String(),
          if (cursor != null && cursor.isNotEmpty) 'cursor': cursor,
          'limit': limit,
        },
      );
      return AuditPage.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final auditApiProvider = Provider<AuditApi>((ref) {
  return AuditApi(ref.watch(dioProvider));
});
