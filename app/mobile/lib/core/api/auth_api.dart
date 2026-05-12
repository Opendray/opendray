import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Calls into /api/v1/health and /api/v1/auth/mobile-login. The
// onboarding screen uses health() for URL validation; the login
// screen uses mobileLogin() to obtain a 30-day bearer token.
//
// `health()` is the only call we make against an unconfigured /
// untrusted server URL — it MUST be safe regardless of what the
// user types in onboarding.
class AuthApi {
  AuthApi(this._dio);
  final Dio _dio;

  Future<HealthResponse> health({String? baseUrlOverride}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/health',
        options: baseUrlOverride != null
            ? Options(headers: {'baseUrl-override': true})
            : null,
      );
      return HealthResponse.fromJson(res.data ?? {});
    } catch (e) {
      throw toApiException(e);
    }
  }

  Future<MobileLoginResponse> mobileLogin({
    required String username,
    required String password,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/auth/mobile-login',
        data: {'username': username, 'password': password},
      );
      return MobileLoginResponse.fromJson(res.data ?? {});
    } catch (e) {
      throw toApiException(e);
    }
  }

  // POST /api/v1/auth/change-credentials — rotates the operator's
  // username + password. Server returns a fresh token issued
  // under the new credentials so the client stays logged in
  // without re-prompting for password immediately. All other
  // tokens are revoked server-side; an attacker holding a stolen
  // bearer would be kicked out on the next request.
  Future<MobileLoginResponse> changeCredentials({
    required String currentPassword,
    required String newPassword,
    String? newUser,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/auth/change-credentials',
        data: {
          'current_password': currentPassword,
          'new_password': newPassword,
          if (newUser != null && newUser.isNotEmpty) 'new_user': newUser,
        },
      );
      return MobileLoginResponse.fromJson(res.data ?? {});
    } catch (e) {
      throw toApiException(e);
    }
  }
}

final authApiProvider = Provider<AuthApi>((ref) {
  return AuthApi(ref.watch(dioProvider));
});

// Onboarding-only Dio: hits an arbitrary user-typed server URL
// without requiring it to be persisted to AuthState first.
Dio buildOnboardingDio(String baseUrl) {
  return Dio(
    BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 6),
      receiveTimeout: const Duration(seconds: 6),
      headers: {'Accept': 'application/json'},
      validateStatus: (_) => true,
    ),
  );
}

Future<HealthResponse> probeHealth(String baseUrl) async {
  final dio = buildOnboardingDio(baseUrl);
  try {
    final res = await dio.get<Map<String, dynamic>>('/api/v1/health');
    final status = res.statusCode ?? 0;
    if (status < 200 || status >= 300) {
      throw toApiException(
        DioException(
          requestOptions: res.requestOptions,
          response: res,
          type: DioExceptionType.badResponse,
        ),
      );
    }
    return HealthResponse.fromJson(res.data ?? {});
  } catch (e) {
    throw toApiException(e);
  } finally {
    dio.close();
  }
}
