import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/auth_api.dart';
import 'package:opendray/core/api/gateway_url.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/auth/cf_access_controller.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/auth/cf_access_login_screen.dart';

// First-run screen. The user types the gateway URL here; we
// validate by calling /api/v1/health on it before persisting.
//
// Why probe-then-persist instead of "trust the URL and recover
// on 404 later": typos at this step are common ("https://" vs
// "http://", missing ports, internal-vs-external hostname). A
// fast probe surfaces them immediately, in a context the user
// understands.
class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  // Deliberately empty. It used to be pre-filled with "http://",
  // which meant typing a published hostname after it produced a
  // cleartext URL and a redirect the app could not explain.
  // normalizeGatewayUrl picks the scheme from the address instead.
  final _controller = TextEditingController();
  bool _busy = false;
  String? _error;
  String? _detected;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _continue() async {
    final raw = _controller.text.trim();
    if (raw.isEmpty || raw == 'http://' || raw == 'https://') {
      setState(() => _error = 'Enter your gateway URL');
      return;
    }
    final normalized = normalizeGatewayUrl(raw);
    setState(() {
      _busy = true;
      _error = null;
      _detected = null;
    });
    try {
      await _probeAndPersist(normalized);
    } on AccessChallengeException catch (e) {
      // The gateway is behind Cloudflare Access and this device has
      // no Access session yet. Onboarding drives the SSO itself
      // rather than deferring to AccessGate: there is no server URL
      // persisted yet, so the gate has nothing to sign in to.
      //
      // e.baseUrl, not what was typed: an http:// entry was upgraded
      // to https before the challenge came back, and the Secure
      // CF_Authorization cookie is unreadable through an http:// URL.
      setState(() => _error = t.access.challengeBanner(host: e.host));
      await _runAccessSso(e.baseUrl ?? normalized);
    } on ApiException catch (e) {
      setState(() => _error = 'Server replied ${e.statusCode}: ${e.message}');
    } on Object catch (e) {
      setState(() => _error = 'Could not reach $normalized: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _probeAndPersist(String normalized) async {
    final probe = await probeGateway(
      normalized,
      cfCookie: ref.read(cfAccessControllerProvider.notifier).cookie,
    );
    if (!mounted) return;
    final h = probe.health;
    setState(() => _detected = 'opendray ${h.version} (${h.status})');
    // probe.baseUrl, not the typed one: the gateway may have upgraded
    // us to https, and persisting the http form would earn that
    // redirect on every request for the life of the install.
    await ref
        .read(authControllerProvider.notifier)
        .setServerUrl(probe.baseUrl);
    // Router redirects automatically on AuthState change.
  }

  Future<void> _runAccessSso(String normalized) async {
    if (!mounted) return;
    final cookie = await CfAccessLoginScreen.show(context, normalized);
    if (cookie == null || !mounted) return;
    await ref.read(cfAccessControllerProvider.notifier).save(cookie);
    if (!mounted) return;
    setState(() => _error = null);
    try {
      await _probeAndPersist(normalized);
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Could not reach $normalized: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 48, 24, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'opendray',
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
              ),
              const SizedBox(height: 8),
              Text(
                'Enter your gateway URL to get started.',
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 36),
              TextField(
                controller: _controller,
                autocorrect: false,
                keyboardType: TextInputType.url,
                textInputAction: TextInputAction.go,
                onSubmitted: (_) => _continue(),
                decoration: InputDecoration(
                  labelText: t.onboarding.gatewayLabel,
                  hintText: t.onboarding.gatewayHint,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'For self-hosted deployments this is the URL you '
                'configured under [admin] base_url.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (_error != null) ...[
                const SizedBox(height: 16),
                _ErrorBanner(message: _error!),
              ],
              if (_detected != null) ...[
                const SizedBox(height: 16),
                _SuccessBanner(message: _detected!),
              ],
              const SizedBox(height: 28),
              FilledButton(
                onPressed: _busy ? null : _continue,
                child: _busy
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(t.onboarding.kContinue),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.error.withValues(alpha: 0.1),
        border: Border.all(
          color: Theme.of(context).colorScheme.error.withValues(alpha: 0.3),
        ),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        message,
        style: TextStyle(color: Theme.of(context).colorScheme.error),
      ),
    );
  }
}

class _SuccessBanner extends StatelessWidget {
  const _SuccessBanner({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.green.withValues(alpha: 0.1),
        border: Border.all(color: Colors.green.withValues(alpha: 0.3)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(message, style: const TextStyle(color: Colors.greenAccent)),
    );
  }
}
