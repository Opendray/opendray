import 'dart:io';

// Working out what the operator meant when they typed a gateway URL.
//
// Two deployments, two conventions, and getting them confused is what
// makes onboarding fail in a way nobody can debug from the message:
//
//   LAN       http://192.168.3.5:8770   no certificate, plain http
//   published https://gateway.example    TLS in front; plain http gets
//                                        a 301 to the https form
//
// So the scheme is a real decision, not a formality.

/// Adds a scheme to a URL typed without one and trims trailing noise.
///
/// A private or loopback address is a LAN gateway and gets http;
/// anything else is assumed to be published and gets https. Guessing
/// https for a LAN address would break the common self-hosted case;
/// guessing http for a hostname produces a redirect the app then has
/// to make sense of.
///
/// An explicit scheme is always left alone. If the operator says
/// http:// against a published host they get the redirect, and
/// [protocolUpgradeTarget] handles it from there.
String normalizeGatewayUrl(String raw) {
  var v = raw.trim();
  if (!v.startsWith('http://') && !v.startsWith('https://')) {
    v = '${_looksLocal(v) ? 'http' : 'https'}://$v';
  }
  return v.replaceAll(RegExp(r'/+$'), '');
}

/// The URL to retry when a redirect is nothing but a scheme upgrade.
///
/// TLS front ends answer plain http with a 301 to the https form of
/// the same request. That one is safe to act on without asking: same
/// host, same path, strictly more secure, and the server asked for
/// it. Every other redirect is something intercepting the request,
/// which is reported rather than followed, so returning null here is
/// what keeps the Access challenge visible.
String? protocolUpgradeTarget({required Uri from, String? location}) {
  if (from.scheme != 'http') return null;
  final loc = (location ?? '').trim();
  if (loc.isEmpty) return null;
  final target = Uri.tryParse(loc);
  if (target == null) return null;
  if (target.scheme != 'https') return null;
  if (target.host.toLowerCase() != from.host.toLowerCase()) return null;
  if (target.path != from.path) return null;
  return target.toString();
}

/// Rebuilds the base URL from an upgraded probe URL, so onboarding
/// persists `https://host` rather than `https://host/api/v1/health`.
String upgradedBaseUrl({required String baseUrl, required String upgraded}) {
  final base = Uri.parse(baseUrl.replaceAll(RegExp(r'/+$'), ''));
  final up = Uri.parse(upgraded);
  return Uri(
    scheme: up.scheme,
    host: up.host,
    port: base.hasPort ? base.port : null,
  ).toString().replaceAll(RegExp(r'/+$'), '');
}

/// Whether a host[:port] with no scheme is a LAN address.
bool _looksLocal(String hostPort) {
  // Parsed with a scheme bolted on, because a bare "host:8770" is
  // otherwise read as scheme "host".
  final host = Uri.tryParse('http://$hostPort')?.host ?? '';
  if (host.isEmpty) return false;
  final lower = host.toLowerCase();
  if (lower == 'localhost' || lower.endsWith('.local')) return true;
  final ip = InternetAddress.tryParse(host);
  if (ip == null) return false;
  if (ip.isLoopback || ip.isLinkLocal) return true;
  return _isPrivate(ip);
}

/// RFC 1918 / RFC 4193. dart:io has isLoopback and isLinkLocal but no
/// notion of a private range, so the ranges are spelled out.
bool _isPrivate(InternetAddress ip) {
  final b = ip.rawAddress;
  if (ip.type == InternetAddressType.IPv4 && b.length == 4) {
    if (b[0] == 10) return true;
    if (b[0] == 172 && b[1] >= 16 && b[1] <= 31) return true;
    if (b[0] == 192 && b[1] == 168) return true;
    return false;
  }
  // fc00::/7 — unique local addresses.
  if (b.isNotEmpty) return (b[0] & 0xfe) == 0xfc;
  return false;
}
