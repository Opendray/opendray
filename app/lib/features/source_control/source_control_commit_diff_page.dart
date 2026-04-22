import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/theme/app_theme.dart';
import 'widgets/multi_file_diff_view.dart';

/// Read-only view of a single commit's file changes. Backed by
/// `/api/source-control/{plugin}/diff?mode=commit&commit=SHA` — same
/// multi-file diff payload shape the Changes tab already renders, so
/// we reuse MultiFileDiffView verbatim.
class SourceControlCommitDiffPage extends StatefulWidget {
  const SourceControlCommitDiffPage({
    super.key,
    required this.pluginName,
    required this.repo,
    required this.sha,
    required this.subject,
  });

  final String pluginName;
  final String repo;
  final String sha;
  final String subject;

  @override
  State<SourceControlCommitDiffPage> createState() =>
      _SourceControlCommitDiffPageState();
}

class _SourceControlCommitDiffPageState
    extends State<SourceControlCommitDiffPage> {
  List<Map<String, dynamic>>? _files;
  bool _loading = false;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() { _loading = true; _error = null; });
    try {
      final result = await ApiClient.describeErrors(() => _api.scDiff(
            widget.pluginName,
            repo: widget.repo,
            mode: 'commit',
            commit: widget.sha,
          ));
      if (!mounted) return;
      final files = ((result['files'] as List?) ?? const [])
          .cast<Map<String, dynamic>>();
      setState(() { _files = files; _loading = false; });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() { _loading = false; _error = e.message; });
    }
  }

  @override
  Widget build(BuildContext context) {
    final short = widget.sha.length >= 7
        ? widget.sha.substring(0, 7)
        : widget.sha;
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(short,
                style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 13,
                    color: AppColors.accent)),
            Text(widget.subject,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12)),
          ],
        ),
        actions: [
          IconButton(
            tooltip: context.tr('Refresh'),
            icon: const Icon(Icons.refresh, size: 20),
            onPressed: _loading ? null : _load,
          ),
        ],
      ),
      body: _body(context),
    );
  }

  Widget _body(BuildContext context) {
    if (_loading && _files == null) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_error != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Icon(Icons.error_outline,
                color: AppColors.error, size: 32),
            const SizedBox(height: 12),
            Text(_error!,
                textAlign: TextAlign.center,
                style: const TextStyle(
                    color: AppColors.error, fontSize: 12)),
            const SizedBox(height: 16),
            TextButton.icon(
              onPressed: _load,
              icon: const Icon(Icons.refresh, size: 16),
              label: Text(context.tr('Retry')),
            ),
          ]),
        ),
      );
    }
    return MultiFileDiffView(
      files: _files ?? const [],
      emptyMessage: context.tr('No file changes in this commit.'),
    );
  }
}
