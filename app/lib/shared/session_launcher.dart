import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../core/api/api_client.dart';
import '../core/models/provider.dart';
import '../core/models/session.dart';
import '../features/dashboard/widgets/new_session_dialog.dart';
import 'theme/app_theme.dart';

/// Launches the "New Session" bottom sheet with an optional pre-filled cwd,
/// creates the session on confirm, starts it, and navigates into the session
/// page. Returns true if a session was created.
///
/// Use this from anywhere in the app (e.g. Files page "Create Session Here"
/// on a folder) so we have one canonical entry point for starting sessions.
Future<bool> launchNewSession(BuildContext context, {String? initialCwd}) async {
  final api = context.read<ApiClient>();
  List<ProviderInfo> providers;
  try {
    providers = await api.listProviders();
  } catch (_) {
    providers = [];
  }
  if (!context.mounted) return false;

  final result = await showModalBottomSheet<Map<String, dynamic>>(
    context: context,
    isScrollControlled: true,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => NewSessionDialog(
        providers: providers, initialCwd: initialCwd),
  );

  if (result == null || !context.mounted) return false;

  // Create the session — if this fails we're done.
  Session session;
  try {
    session = await ApiClient.describeErrors(() => api.createSession(
      cwd: result['cwd'] as String,
      sessionType: result['sessionType'] as String,
      name: result['name'] as String? ?? '',
      model: result['model'] as String? ?? '',
      extraArgs: List<String>.from(result['extraArgs'] ?? const []),
      claudeAccountId: result['claudeAccountId'] as String?,
      llmProviderId: result['llmProviderId'] as String?,
    ));
  } catch (e) {
    if (context.mounted) _errSnack(context, 'Create failed: $e');
    return false;
  }

  // Navigate immediately so the user lands on the session page; start it
  // separately so a start failure doesn't prevent them from seeing / retrying
  // the session. The session page has its own "Start Session" button and
  // shows the reason when start fails.
  if (context.mounted) context.push('/session/${session.id}');

  try {
    await ApiClient.describeErrors(() => api.startSession(session.id));
    return true;
  } catch (e) {
    if (context.mounted) _errSnack(context, 'Start failed: $e');
    return false;
  }
}

void _errSnack(BuildContext context, String msg) {
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(
    content: Text(msg),
    backgroundColor: AppColors.error,
    duration: const Duration(seconds: 6),
    action: SnackBarAction(
      label: 'Dismiss',
      textColor: Colors.white,
      onPressed: () =>
          ScaffoldMessenger.of(context).hideCurrentSnackBar(),
    ),
  ));
}
