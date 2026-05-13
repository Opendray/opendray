import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/auth_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Username + password login. Calls /api/v1/auth/mobile-login which
// returns a token with mobile_token_ttl (default 30 days, vs the
// 24h browser-token TTL). The token is persisted to secure storage
// by AuthController.setLoggedIn.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _userCtrl = TextEditingController();
  final _passCtrl = TextEditingController();
  bool _busy = false;
  bool _obscurePass = true;
  String? _error;

  @override
  void dispose() {
    _userCtrl.dispose();
    _passCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final username = _userCtrl.text.trim();
    final password = _passCtrl.text;
    if (username.isEmpty || password.isEmpty) {
      setState(() => _error = t.auth.errorRequired);
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final api = ref.read(authApiProvider);
      final res = await api.mobileLogin(
        username: username,
        password: password,
      );
      await ref.read(authControllerProvider.notifier).setLoggedIn(
            token: res.token,
            username: res.username,
            expiresAt: res.expiresAt,
          );
      // Router redirects to /home on AuthLoggedIn.
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } on Object catch (e) {
      setState(() => _error = t.auth.errorGeneric(error: e.toString()));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _changeServer() async {
    await ref.read(authControllerProvider.notifier).resetServer();
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider);
    final serverUrl = switch (auth) {
      AuthLoggedOut(serverUrl: final s) => s,
      _ => '',
    };
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 48, 24, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                t.auth.signInTitle,
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      serverUrl,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ),
                  TextButton(
                    onPressed: _busy ? null : _changeServer,
                    child: Text(t.auth.changeServer),
                  ),
                ],
              ),
              const SizedBox(height: 28),
              TextField(
                controller: _userCtrl,
                autocorrect: false,
                textCapitalization: TextCapitalization.none,
                textInputAction: TextInputAction.next,
                decoration: InputDecoration(labelText: t.auth.username),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _passCtrl,
                autocorrect: false,
                obscureText: _obscurePass,
                textInputAction: TextInputAction.go,
                onSubmitted: (_) => _submit(),
                decoration: InputDecoration(
                  labelText: t.auth.password,
                  suffixIcon: IconButton(
                    icon: Icon(
                      _obscurePass
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined,
                    ),
                    onPressed: () =>
                        setState(() => _obscurePass = !_obscurePass),
                  ),
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Theme.of(context)
                        .colorScheme
                        .error
                        .withValues(alpha: 0.1),
                    border: Border.all(
                      color: Theme.of(context)
                          .colorScheme
                          .error
                          .withValues(alpha: 0.3),
                    ),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    _error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 28),
              FilledButton(
                onPressed: _busy ? null : _submit,
                child: _busy
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(t.auth.signIn),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
