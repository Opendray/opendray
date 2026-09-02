import 'dart:io';

import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/auth/cf_access.dart';
import 'package:opendray/core/auth/cf_access_controller.dart';
import 'package:pretty_dio_logger/pretty_dio_logger.dart';

// How many times a transient request is replayed before the failure is
// allowed to reach the UI. Two covers the case this exists for — a
// socket the OS tore down while the app was suspended — without making
// a genuinely unreachable gateway take most of a minute to report.
const kMaxRetries = 2;

const _retryBackoff = <Duration>[
  Duration(milliseconds: 300),
  Duration(milliseconds: 900),
];

const _retryAttemptKey = 'opendray.retryAttempt';

// Idle pooled connections are the whole problem: Dart keeps them alive
// across an app suspension and then hands out a socket the OS already
// killed. Expiring them sooner shrinks that window; the retry below
// covers what still slips through.
const _poolIdleTimeout = Duration(seconds: 5);

/// Whether the failure means "the connection broke" rather than "the
/// server answered and disliked the request". Only the former is worth
/// replaying — a 4xx/5xx is a real answer and must reach the caller.
bool isTransientFailure(DioException e) {
  switch (e.type) {
    case DioExceptionType.connectionTimeout:
    case DioExceptionType.sendTimeout:
    case DioExceptionType.receiveTimeout:
    case DioExceptionType.connectionError:
      return true;
    case DioExceptionType.unknown:
      return e.error is SocketException || e.error is HttpException;
    case DioExceptionType.badResponse:
    case DioExceptionType.cancel:
    case DioExceptionType.badCertificate:
      return false;
  }
}

/// Whether replaying the request is safe. Reads only: re-sending a POST
/// could create a second session, and re-sending a streaming download
/// (which opts out of timeouts) would restart a large transfer.
bool isReplayable(RequestOptions options) {
  final method = options.method.toUpperCase();
  if (method != 'GET' && method != 'HEAD') return false;
  if (options.receiveTimeout == Duration.zero) return false;
  return true;
}

/// Builds the app's HTTP client. Split out from the provider so tests
/// can drive it with a stub adapter.
Dio buildDio({
  required String baseUrl,
  String? token,
  String? cfCookie,
  void Function()? onUnauthorized,
  void Function(String host)? onAccessChallenge,
}) {
  final cookieHeader = cfCookieHeader(cfCookie);
  final dio = Dio(
    BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 8),
      receiveTimeout: const Duration(seconds: 30),
      headers: {
        'Accept': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
        // Cloudflare Access reads this at the edge; the gateway
        // never sees it. Absent on a LAN deployment, where there is
        // no Access in the path.
        if (cookieHeader != null) 'Cookie': cookieHeader,
      },
      validateStatus: (_) => true, // we throw ApiException ourselves
    ),
  )..httpClientAdapter = IOHttpClientAdapter(
      createHttpClient: () => HttpClient()..idleTimeout = _poolIdleTimeout,
    );

  dio.interceptors.add(
    InterceptorsWrapper(
      onResponse: (response, handler) {
        // Checked before the status code, because a challenge
        // usually arrives as a perfectly healthy 200 carrying the
        // Access login page instead of our JSON.
        if (isAccessChallenge(response)) {
          final host = response.realUri.host;
          onAccessChallenge?.call(host);
          handler.reject(
            DioException(
              requestOptions: response.requestOptions,
              response: response,
              type: DioExceptionType.badResponse,
              error: AccessChallengeException(
                statusCode: response.statusCode ?? 0,
                host: host,
              ),
            ),
          );
          return;
        }
        final status = response.statusCode ?? 0;
        if (status >= 200 && status < 300) {
          handler.next(response);
          return;
        }
        if (status == 401) {
          // Token revoked / expired — kick to login.
          onUnauthorized?.call();
        }
        final body = response.data;
        final message = (body is Map && body['error'] is String)
            ? body['error'] as String
            : '${response.requestOptions.method} '
                '${response.requestOptions.path} failed ($status)';
        handler.reject(
          DioException(
            requestOptions: response.requestOptions,
            response: response,
            type: DioExceptionType.badResponse,
            error: ApiException(
              statusCode: status,
              message: message,
              body: body,
            ),
          ),
        );
      },
      // Transient connection failures are routine on mobile — waking
      // from sleep, roaming between APs, the gateway restarting behind
      // a deploy. Without this a single blip becomes a full-screen
      // "failed to load" even though the very next attempt would work.
      onError: (e, handler) async {
        final options = e.requestOptions;
        final attempt = (options.extra[_retryAttemptKey] as int?) ?? 0;
        if (attempt >= kMaxRetries ||
            !isTransientFailure(e) ||
            !isReplayable(options)) {
          handler.next(e);
          return;
        }
        await Future<void>.delayed(_retryBackoff[attempt]);
        options.extra[_retryAttemptKey] = attempt + 1;
        try {
          handler.resolve(await dio.fetch<dynamic>(options));
        } on DioException catch (retryError) {
          handler.next(retryError);
        }
      },
    ),
  );

  dio.interceptors.add(
    PrettyDioLogger(
      requestBody: true,
      responseBody: false,
      requestHeader: false,
      responseHeader: false,
      compact: true,
    ),
  );

  return dio;
}

// Builds a Dio instance pinned to the currently-configured server
// URL, automatically attaching the bearer token (if any) and
// surfacing 401 to the auth controller for forced sign-out.
//
// We rebuild the client whenever AuthState changes (server URL or
// token) — Riverpod handles the invalidation; consumers just
// `ref.watch(dioProvider)` and don't worry about staleness.
final dioProvider = Provider<Dio>((ref) {
  final auth = ref.watch(authControllerProvider);
  // Watched, not read: a fresh Access cookie has to rebuild the
  // client, otherwise every request keeps going out with the dead
  // one until something else happens to invalidate the provider.
  ref.watch(cfAccessControllerProvider);
  final cfCookie = ref.read(cfAccessControllerProvider.notifier).cookie;
  final baseUrl = switch (auth) {
    AuthLoggedOut(serverUrl: final s) => s,
    AuthLoggedIn(serverUrl: final s) => s,
    _ => '',
  };
  final token = switch (auth) {
    AuthLoggedIn(token: final t) => t,
    _ => null,
  };

  return buildDio(
    baseUrl: baseUrl,
    token: token,
    cfCookie: cfCookie,
    onUnauthorized: () => ref.read(authControllerProvider.notifier).logout(),
    onAccessChallenge: (host) => ref
        .read(cfAccessControllerProvider.notifier)
        .challenged(reason: host),
  );
});

ApiException toApiException(Object error) {
  if (error is ApiException) return error;
  if (error is DioException) {
    if (error.error is ApiException) return error.error! as ApiException;
    return ApiException(
      statusCode: error.response?.statusCode ?? 0,
      message: error.message ?? 'Network error',
      body: error.response?.data,
    );
  }
  return ApiException(statusCode: 0, message: error.toString());
}
