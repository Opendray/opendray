import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/models/session.dart';
import '../../shared/session_launcher.dart';
import '../../shared/theme/app_theme.dart';
import 'widgets/session_card.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});
  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  List<Session> _sessions = [];
  bool _loading = true;
  String? _error;
  Timer? _pollTimer;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
    _pollTimer = Timer.periodic(const Duration(seconds: 5), (_) => _load());
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final sessions = await _api.listSessions();
      if (mounted) setState(() { _sessions = sessions; _loading = false; _error = null; });
    } catch (e) {
      if (mounted) setState(() { _loading = false; _error = e.toString(); });
    }
  }

  Future<void> _showCreateDialog() => launchNewSession(context);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            Container(
              width: 28, height: 28,
              decoration: BoxDecoration(color: AppColors.accent, borderRadius: BorderRadius.circular(7)),
              child: const Center(child: Text('N', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14))),
            ),
            const SizedBox(width: 10),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('NTC', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                Text('${_sessions.length} sessions', style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
              ],
            ),
          ],
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: FilledButton.icon(
              onPressed: _showCreateDialog,
              icon: const Icon(Icons.add, size: 16),
              label: const Text('New', style: TextStyle(fontSize: 13)),
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.accent,
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
              ),
            ),
          ),
        ],
      ),
      body: _loading && _sessions.isEmpty
          ? const Center(child: CircularProgressIndicator(color: AppColors.accent))
          : _error != null && _sessions.isEmpty
          ? Center(child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(mainAxisSize: MainAxisSize.min, children: [
                const Icon(Icons.cloud_off, color: AppColors.error, size: 40),
                const SizedBox(height: 12),
                const Text('Cannot connect to server', style: TextStyle(fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text('Check Settings → Server URL', style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
                const SizedBox(height: 16),
                FilledButton(onPressed: _load, style: FilledButton.styleFrom(backgroundColor: AppColors.accent), child: const Text('Retry')),
              ]),
            ))
          : _sessions.isEmpty
              ? _buildEmpty()
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView.separated(
                    padding: const EdgeInsets.all(16),
                    itemCount: _sessions.length,
                    separatorBuilder: (_, _) => const SizedBox(height: 10),
                    itemBuilder: (_, i) => SessionCard(
                      session: _sessions[i],
                      onTap: () => context.push('/session/${_sessions[i].id}'),
                      onStart: () async { await _api.startSession(_sessions[i].id); _load(); },
                      onStop: () async { await _api.stopSession(_sessions[i].id); _load(); },
                      onDelete: () async { await _api.deleteSession(_sessions[i].id); _load(); },
                    ),
                  ),
                ),
    );
  }

  Widget _buildEmpty() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 64, height: 64,
            decoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(16)),
            child: const Icon(Icons.terminal, size: 32, color: AppColors.textMuted),
          ),
          const SizedBox(height: 16),
          const Text('No sessions', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          const Text('Create a session to start', style: TextStyle(color: AppColors.textMuted, fontSize: 13)),
          const SizedBox(height: 20),
          FilledButton(
            onPressed: _showCreateDialog,
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: const Text('Create Session'),
          ),
        ],
      ),
    );
  }
}
