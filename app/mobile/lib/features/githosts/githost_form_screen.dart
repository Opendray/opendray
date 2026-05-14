import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/githosts_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Single form for both Add and Edit. Token field semantics differ:
// on Add it's required (no other way to provision). On Edit it's
// optional — empty leaves the existing token intact, useful when
// the operator just wants to rename or toggle without re-typing the
// PAT they pasted from the issuer's web UI.
class GitHostFormScreen extends ConsumerStatefulWidget {
  const GitHostFormScreen({this.existing, super.key});
  final GitHost? existing;

  @override
  ConsumerState<GitHostFormScreen> createState() => _GitHostFormScreenState();
}

class _GitHostFormScreenState extends ConsumerState<GitHostFormScreen> {
  static List<(String, String)> _kinds() => [
        ('github', t.githosts.form.kinds.github),
        ('gitlab', t.githosts.form.kinds.gitlab),
        ('bitbucket', t.githosts.form.kinds.bitbucket),
        ('gitea', t.githosts.form.kinds.gitea),
        ('custom', t.githosts.form.kinds.custom),
      ];

  late final TextEditingController _host;
  late final TextEditingController _name;
  final _token = TextEditingController();
  late String _kind;
  late bool _enabled;
  bool _saving = false;
  String? _error;

  bool get _isEdit => widget.existing != null;

  @override
  void initState() {
    super.initState();
    final ex = widget.existing;
    _host = TextEditingController(text: ex?.host ?? '');
    _name = TextEditingController(text: ex?.name ?? '');
    _kind = ex?.kind ?? 'github';
    _enabled = ex?.enabled ?? true;
  }

  @override
  void dispose() {
    _host.dispose();
    _name.dispose();
    _token.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_saving) return;
    final host = _host.text.trim();
    final name = _name.text.trim();
    final token = _token.text.trim();
    if (host.isEmpty) {
      setState(() => _error = t.githosts.form.validateHost);
      return;
    }
    if (name.isEmpty) {
      setState(() => _error = t.githosts.form.validateName);
      return;
    }
    if (!_isEdit && token.isEmpty) {
      setState(() => _error = t.githosts.form.validateTokenRequired);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    try {
      if (_isEdit) {
        final ex = widget.existing!;
        await ref.read(gitHostsApiProvider).update(
              ex.id,
              kind: _kind != ex.kind ? _kind : null,
              host: host != ex.host ? host : null,
              name: name != ex.name ? name : null,
              // Empty token on edit means "leave it alone" (per API
              // docs). Only send when the operator actually typed a
              // new one.
              token: token.isEmpty ? null : token,
              enabled: _enabled != ex.enabled ? _enabled : null,
            );
      } else {
        await ref.read(gitHostsApiProvider).create(
              kind: _kind,
              host: host,
              name: name,
              token: token,
            );
      }
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(_isEdit
              ? t.githosts.form.snackUpdated
              : t.githosts.form.snackAdded),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      navigator.pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.githosts.form.saveFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.githosts.form.saveFailedGeneric(error: e.toString());
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit
            ? t.githosts.form.appBarEdit(name: widget.existing!.name)
            : t.githosts.form.appBarNew),
        actions: [
          TextButton(
            onPressed: _saving ? null : _submit,
            child: Text(_saving
                ? t.githosts.form.saving
                : (_isEdit ? t.githosts.form.save : t.githosts.form.add)),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Text(t.githosts.form.kindLabel, style: muted),
          const SizedBox(height: 6),
          DropdownButtonFormField<String>(
            initialValue: _kind,
            items: [
              for (final (val, label) in _kinds())
                DropdownMenuItem(value: val, child: Text(label)),
            ],
            onChanged: (v) => setState(() => _kind = v ?? _kind),
            decoration: const InputDecoration(border: OutlineInputBorder()),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _host,
            autocorrect: false,
            keyboardType: TextInputType.url,
            decoration: InputDecoration(
              labelText: t.githosts.form.hostLabel,
              hintText: _kind == 'github' ? 'api.github.com' : 'gitlab.example.com',
              helperText: 'API base or canonical hostname.',
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _name,
            autocorrect: false,
            decoration: InputDecoration(
              labelText: t.githosts.form.nameLabel,
              hintText: t.githosts.form.nameHint,
              helperText: t.githosts.form.nameHelper,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _token,
            autocorrect: false,
            decoration: InputDecoration(
              labelText: _isEdit
                  ? t.githosts.form.tokenLabelKeep
                  : t.githosts.form.tokenLabel,
              hintText: _kind == 'github' ? 'ghp_…' : 'glpat-… or PAT',
              helperText: _isEdit
                  ? t.githosts.form.tokenPreviewHint(
                      preview: widget.existing!.tokenMask.isEmpty
                          ? t.githosts.form.tokenPreviewNone
                          : widget.existing!.tokenMask,
                    )
                  : t.githosts.form.tokenHintNew,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(t.common.enabled),
            subtitle: Text(
              _enabled
                  ? t.githosts.form.enabledHelper
                  : t.githosts.form.pausedSubtitle,
              style: muted,
            ),
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ],
        ],
      ),
    );
  }
}
