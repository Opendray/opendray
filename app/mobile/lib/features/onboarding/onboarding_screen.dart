import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/auth_api.dart';
import 'package:opendray/core/auth/auth_state.dart';

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
  final _controller = TextEditingController(text: 'http://');
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
    final normalized = _normalize(raw);
    setState(() {
      _busy = true;
      _error = null;
      _detected = null;
    });
    try {
      final h = await probeHealth(normalized);
      setState(() => _detected = 'opendray ${h.version} (${h.status})');
      await ref.read(authControllerProvider.notifier).setServerUrl(normalized);
      // Router redirects automatically on AuthState change.
    } on ApiException catch (e) {
      setState(() => _error = 'Server replied ${e.statusCode}: ${e.message}');
    } on Object catch (e) {
      setState(() => _error = 'Could not reach $normalized: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String _normalize(String raw) {
    var v = raw.trim();
    if (!v.startsWith('http://') && !v.startsWith('https://')) {
      v = 'https://$v';
    }
    while (v.endsWith('/')) {
      v = v.substring(0, v.length - 1);
    }
    return v;
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
                decoration: const InputDecoration(
                  labelText: 'Gateway URL',
                  hintText: 'https://opendray.example.com',
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
                    : const Text('Continue'),
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
