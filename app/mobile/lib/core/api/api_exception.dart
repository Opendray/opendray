// Wraps a non-2xx response from the gateway. The HTTP layer
// throws this; UI layers `catch (ApiException)` to surface the
// message and special-case 401 (token expired / revoked).
class ApiException implements Exception {
  ApiException({
    required this.statusCode,
    required this.message,
    this.body,
  });

  final int statusCode;
  final String message;
  final Object? body;

  bool get isUnauthorized => statusCode == 401;

  @override
  String toString() => 'ApiException($statusCode): $message';
}

/// Cloudflare Access blocked the request before it reached the
/// gateway. Distinct from [ApiException] so callers can tell "the
/// edge wants you to sign in again" apart from "opendray said no" —
/// they need completely different recovery (SSO WebView vs. login
/// screen), and conflating them sends the operator to a screen that
/// cannot fix their problem.
class AccessChallengeException extends ApiException {
  AccessChallengeException({required super.statusCode, required this.host})
      : super(message: 'Cloudflare Access authentication required');

  /// Host that issued the challenge, for the error banner.
  final String host;

  @override
  String toString() => 'AccessChallengeException($host)';
}
