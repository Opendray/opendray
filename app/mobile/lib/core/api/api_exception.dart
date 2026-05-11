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
