// ChangeCredentialsScreen — full-screen form for rotating the
// operator's admin username + password from mobile.
//
// Form fields:
//   - Current password (required, verified server-side first)
//   - New username   (optional, prefilled with current)
//   - New password   (≥12 chars)
//   - Confirm        (must match new)
//
// On submit the server validates the current password, hashes the
// new one, atomically writes ~/.opendray/secrets/admin.key,
// revokes ALL existing tokens, and returns a fresh token issued
// under the new credentials. We stash that new token in
// AuthController so the operator stays signed in without
// re-prompting.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/auth_api.dart';
import 'package:opendray/core/auth/auth_state.dart';

class ChangeCredentialsScreen extends ConsumerStatefulWidget {
  const ChangeCredentialsScreen({super.key});

  @override
  ConsumerState<ChangeCredentialsScreen> createState() =>
      _ChangeCredentialsScreenState();
}

class _ChangeCredentialsScreenState
    extends ConsumerState<ChangeCredentialsScreen> {
  final _currentCtrl = TextEditingController();
  final _newUserCtrl = TextEditingController();
  final _newPassCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _submitting = false;
  String? _error;
  bool _obscureCurrent = true;
  bool _obscureNew = true;

  @override
  void initState() {
    super.initState();
    // Prefill the new-user field with the current username — most
    // operators are rotating only the password, not the user.
    final auth = ref.read(authControllerProvider);
    if (auth is AuthLoggedIn) {
      _newUserCtrl.text = auth.username;
    }
  }

  @override
  void dispose() {
    _currentCtrl.dispose();
    _newUserCtrl.dispose();
    _newPassCtrl.dispose();
    _confirmCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final api = ref.read(authApiProvider);
      final res = await api.changeCredentials(
        currentPassword: _currentCtrl.text,
        newUser: _newUserCtrl.text.trim().isEmpty
            ? null
            : _newUserCtrl.text.trim(),
        newPassword: _newPassCtrl.text,
      );
      // Server issued a fresh token under the new credentials —
      // stash it so the user stays signed in. Without this, every
      // request after the rotation 401s.
      await ref.read(authControllerProvider.notifier).setLoggedIn(
            token: res.token,
            username: res.username,
            expiresAt: res.expiresAt,
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Credentials updated.'),
          duration: Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      Navigator.of(context).pop();
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.statusCode == 401
              ? 'Current password is wrong.'
              : e.message;
          _submitting = false;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _submitting = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Change credentials')),
      body: SafeArea(
        bottom: false,
        child: Form(
          key: _formKey,
          autovalidateMode: AutovalidateMode.onUserInteraction,
          child: ListView(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
            children: [
              Text(
                'Verify your current password, then pick new credentials. '
                'All other signed-in sessions will be revoked.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
              const SizedBox(height: 16),
              _label('Current password'),
              const SizedBox(height: 4),
              TextFormField(
                controller: _currentCtrl,
                obscureText: _obscureCurrent,
                autocorrect: false,
                decoration: InputDecoration(
                  suffixIcon: IconButton(
                    icon: Icon(_obscureCurrent
                        ? Icons.visibility_off_outlined
                        : Icons.visibility_outlined),
                    onPressed: () => setState(
                        () => _obscureCurrent = !_obscureCurrent),
                  ),
                ),
                validator: (v) =>
                    (v == null || v.isEmpty) ? 'Required' : null,
              ),
              const SizedBox(height: 16),
              _label('New username'),
              const SizedBox(height: 4),
              TextFormField(
                controller: _newUserCtrl,
                autocorrect: false,
                textCapitalization: TextCapitalization.none,
                decoration: const InputDecoration(),
                style: const TextStyle(fontFamily: 'monospace'),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? 'Required' : null,
              ),
              const SizedBox(height: 16),
              _label('New password'),
              const SizedBox(height: 4),
              TextFormField(
                controller: _newPassCtrl,
                obscureText: _obscureNew,
                autocorrect: false,
                decoration: InputDecoration(
                  helperText: 'At least 8 characters',
                  suffixIcon: IconButton(
                    icon: Icon(_obscureNew
                        ? Icons.visibility_off_outlined
                        : Icons.visibility_outlined),
                    onPressed: () =>
                        setState(() => _obscureNew = !_obscureNew),
                  ),
                ),
                validator: (v) {
                  if (v == null || v.isEmpty) return 'Required';
                  if (v.length < 8) {
                    return 'Must be at least 8 characters';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 16),
              _label('Confirm new password'),
              const SizedBox(height: 4),
              TextFormField(
                controller: _confirmCtrl,
                obscureText: _obscureNew,
                autocorrect: false,
                decoration: const InputDecoration(),
                validator: (v) {
                  if (v != _newPassCtrl.text) {
                    return "Doesn't match the new password";
                  }
                  return null;
                },
              ),
              if (_error != null) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.error.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(6),
                    border: Border.all(
                      color: theme.colorScheme.error.withValues(alpha: 0.4),
                    ),
                  ),
                  child: Text(
                    _error!,
                    style: TextStyle(
                      color: theme.colorScheme.error,
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 20),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: _submitting
                          ? null
                          : () => Navigator.of(context).pop(),
                      child: const Text('Cancel'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: _submitting ? null : _submit,
                      icon: _submitting
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child:
                                  CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.check, size: 18),
                      label: Text(_submitting ? 'Saving…' : 'Update'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _label(String text) => Text(
        text,
        style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      );
}
