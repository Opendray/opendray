import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/gateway_url.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/auth/cf_access.dart';

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
//
// [cfCookie] is threaded through because onboarding is where a
// Cloudflare Access challenge is *most* likely: the operator has
// just typed a public hostname, and the very first probe is the one
// the edge bounces.
Dio buildOnboardingDio(String baseUrl, {String? cfCookie}) {
  final cookie = cfCookieHeader(cfCookie);
  return Dio(
    BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 6),
      receiveTimeout: const Duration(seconds: 6),
      headers: {
        'Accept': 'application/json',
        if (cookie != null) 'Cookie': cookie,
      },
      validateStatus: (_) => true,
      // Same reasoning as buildDio: onboarding is the most likely
      // place to meet an Access challenge, and following the redirect
      // lands on an identity provider's HTML instead of reporting the
      // challenge, which surfaces as an unexplained "Network error".
      followRedirects: false,
    ),
  );
}

/// What a successful probe learned about a candidate gateway.
class GatewayProbe {
  const GatewayProbe({required this.health, required this.baseUrl});

  final HealthResponse health;

  /// The URL that actually answered. Differs from the one probed when
  /// the gateway redirected http to https, in which case this is the
  /// one worth persisting: saving the http form would earn the same
  /// redirect on every request from then on.
  final String baseUrl;
}

/// Probes /api/v1/health on a candidate gateway URL.
///
/// Throws [AccessChallengeException] when Cloudflare Access answered
/// instead of the gateway, so onboarding can offer SSO rather than
/// reporting the login page as a broken server.
///
/// Follows exactly one kind of redirect: a same-host, same-path
/// upgrade from http to https, which is what a TLS front end answers
/// plain http with. Everything else is reported, because everything
/// else is somebody intercepting the request.
Future<GatewayProbe> probeGateway(
  String baseUrl, {
  String? cfCookie,
  @visibleForTesting
  Dio Function(String baseUrl, {String? cfCookie})? dioFactory,
}) async {
  final first =
      await _probeOnce(baseUrl, cfCookie: cfCookie, dioFactory: dioFactory);
  if (first.health != null) {
    return GatewayProbe(health: first.health!, baseUrl: baseUrl);
  }
  final upgraded = upgradedBaseUrl(baseUrl: baseUrl, upgraded: first.upgrade!);
  final second =
      await _probeOnce(upgraded, cfCookie: cfCookie, dioFactory: dioFactory);
  if (second.health != null) {
    return GatewayProbe(health: second.health!, baseUrl: upgraded);
  }
  // Two upgrades in a row means a redirect loop, not a front end
  // doing its job. Report the second answer rather than looping.
  throw ApiException(
    statusCode: 0,
    message: 'Gateway kept redirecting $baseUrl to itself',
  );
}

class _ProbeResult {
  const _ProbeResult.ok(this.health) : upgrade = null;
  const _ProbeResult.upgradeTo(this.upgrade) : health = null;
  final HealthResponse? health;
  final String? upgrade;
}

Future<_ProbeResult> _probeOnce(
  String baseUrl, {
  String? cfCookie,
  Dio Function(String baseUrl, {String? cfCookie})? dioFactory,
}) async {
  final dio = (dioFactory ?? buildOnboardingDio)(baseUrl, cfCookie: cfCookie);
  try {
    // get<dynamic>, never get<Map<String, dynamic>>. Dio casts the
    // decoded body to the type argument, and that cast runs INSIDE
    // the await -- before any line below it. A challenge answers with
    // an HTML login page, so asking for a Map threw a _TypeError with
    // a null message, which toApiException rendered as the useless
    // "Network error" while every check here sat unreachable. This
    // client has no interceptors (the main one in dio_provider does,
    // and those DO run before the cast), so the shape of the response
    // has to be our problem, not Dio's.
    final res = await dio.get<dynamic>('/api/v1/health');
    if (isAccessChallenge(res)) {
      throw AccessChallengeException(
        statusCode: res.statusCode ?? 0,
        host: res.realUri.host,
        baseUrl: baseUrl,
      );
    }
    final status = res.statusCode ?? 0;
    if (status >= 300 && status < 400) {
      final target = protocolUpgradeTarget(
        from: res.realUri,
        location: res.headers.value('location'),
      );
      if (target != null) return _ProbeResult.upgradeTo(target);
    }
    if (status < 200 || status >= 300) {
      throw toApiException(
        DioException(
          requestOptions: res.requestOptions,
          response: res,
          type: DioExceptionType.badResponse,
        ),
      );
    }
    final data = res.data;
    if (data is! Map<String, dynamic>) {
      // A 2xx that is not JSON is not our gateway. Say what actually
      // came back rather than letting a cast failure downstream turn
      // it into an unexplained network error.
      final ct = res.headers.value(Headers.contentTypeHeader) ?? 'none';
      throw ApiException(
        statusCode: status,
        message: 'Not an opendray gateway: expected JSON, got $ct',
      );
    }
    return _ProbeResult.ok(HealthResponse.fromJson(data));
  } catch (e) {
    throw toApiException(e);
  } finally {
    dio.close();
  }
}
