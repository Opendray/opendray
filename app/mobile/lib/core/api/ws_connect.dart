import 'package:dio/dio.dart';
import 'package:opendray/core/auth/cf_access.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// One place to open a WebSocket against the gateway.
//
// The bearer token goes in the `Authorization` header, not in the
// query string. The gateway accepts both (internal/auth: bearerToken
// reads the header first, then `?token=`), and the browser client has
// to keep using the query form because the WebSocket API cannot set
// headers -- but a native client has no such excuse, and a token in a
// URL ends up in every proxy access log it passes through. Publishing
// the gateway through Cloudflare makes that a real log, on someone
// else's disk, rather than a line in our own.
//
// The same handshake carries the Cloudflare Access cookie, because
// Access gates the WebSocket upgrade exactly like any other request.

/// `ws://`/`wss://` URI for a gateway path.
Uri gatewayWsUri({required String serverUrl, required String path}) {
  final base = serverUrl.replaceAll(RegExp(r'/+$'), '');
  final wsBase = base.startsWith('https')
      ? base.replaceFirst('https', 'wss')
      : base.replaceFirst('http', 'ws');
  return Uri.parse('$wsBase$path');
}

/// Handshake headers. `Origin` is deliberately absent: the gateway's
/// same-origin check treats a missing Origin as a non-browser client,
/// which is what we are.
Map<String, String> gatewayWsHeaders({
  required String token,
  String? cfCookie,
}) {
  final cookie = cfCookieHeader(cfCookie);
  return {
    'Authorization': 'Bearer $token',
    if (cookie != null) 'Cookie': cookie,
  };
}

/// Opens an authenticated WebSocket to [path] on [serverUrl].
WebSocketChannel connectGatewayWs({
  required String serverUrl,
  required String path,
  required String token,
  String? cfCookie,
}) {
  return IOWebSocketChannel.connect(
    gatewayWsUri(serverUrl: serverUrl, path: path),
    headers: gatewayWsHeaders(token: token, cfCookie: cfCookie),
  );
}

/// How many times a dropped gateway WebSocket is retried before the
/// UI stops trying and reports the failure.
///
/// The cap only means anything because the attempt counter is reset
/// on the first byte received rather than on the socket object being
/// created: the latter says nothing about whether the handshake was
/// accepted, and resetting there makes every attempt look like the
/// first one, so the loop never ends.
const kMaxReconnectAttempts = 5;

/// Delay before reconnect attempt [attempt] (1-based).
///
/// Exponential from 500ms, so five attempts span roughly 8 seconds
/// rather than hammering a gateway that is restarting behind a
/// deploy, or an Access gate that is going to keep saying no until
/// the operator signs in again.
Duration reconnectBackoff(int attempt) {
  final n = attempt < 1 ? 1 : attempt;
  return Duration(milliseconds: 500 * (1 << (n - 1)));
}

/// Re-checks Access over HTTP after a WebSocket failure.
///
/// A handshake that Cloudflare Access rejects surfaces to the client
/// as a plain socket error: the upgrade simply does not happen, and
/// there is no response object to inspect the way the Dio pipeline
/// does. That makes an expired Access cookie indistinguishable from
/// "the gateway is down" at the WS layer, and an operator sitting on
/// a session terminal when the cookie expires would see nothing but
/// a reconnect loop, with no way back to the sign-in.
///
/// So we ask a path that *can* answer: one cheap GET through the
/// guarded client, whose response runs the same challenge detection
/// as everything else and raises the sign-in through AccessGate.
/// Result and errors are both discarded on purpose; the interceptor
/// has already done the only work that matters.
Future<void> probeForAccessChallenge(Dio dio) async {
  try {
    await dio.get<dynamic>('/api/v1/health');
  } on Object catch (_) {
    // Nothing to do here. Either the probe was challenged (in which
    // case the interceptor already told CfAccessController) or the
    // gateway is genuinely unreachable, which the WS error the
    // caller is already showing describes better than we could.
  }
}
