import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:pretty_dio_logger/pretty_dio_logger.dart';

// Builds a Dio instance pinned to the currently-configured server
// URL, automatically attaching the bearer token (if any) and
// surfacing 401 to the auth controller for forced sign-out.
//
// We rebuild the client whenever AuthState changes (server URL or
// token) — Riverpod handles the invalidation; consumers just
// `ref.watch(dioProvider)` and don't worry about staleness.
final dioProvider = Provider<Dio>((ref) {
  final auth = ref.watch(authControllerProvider);
  final baseUrl = switch (auth) {
    AuthLoggedOut(serverUrl: final s) => s,
    AuthLoggedIn(serverUrl: final s) => s,
    _ => '',
  };
  final token = switch (auth) {
    AuthLoggedIn(token: final t) => t,
    _ => null,
  };

  final dio = Dio(
    BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 8),
      receiveTimeout: const Duration(seconds: 30),
      headers: {
        'Accept': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      },
      validateStatus: (_) => true, // we throw ApiException ourselves
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      onResponse: (response, handler) {
        final status = response.statusCode ?? 0;
        if (status >= 200 && status < 300) {
          handler.next(response);
          return;
        }
        if (status == 401) {
          // Token revoked / expired — kick to login.
          ref.read(authControllerProvider.notifier).logout();
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
