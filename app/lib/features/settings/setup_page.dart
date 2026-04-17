import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/services/server_config.dart';
import '../../shared/theme/app_theme.dart';

class SetupPage extends StatefulWidget {
  const SetupPage({super.key});
  @override
  State<SetupPage> createState() => _SetupPageState();
}

class _SetupPageState extends State<SetupPage> {
  final _controller = TextEditingController();
  bool _testing = false;
  String? _result;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _test() async {
    final url = _controller.text.trim();
    if (url.isEmpty) return;
    setState(() { _testing = true; _result = null; });
    try {
      final api = ApiClient(baseUrl: url);
      final health = await api.health();
      setState(() {
        _result = 'ok:${health['sessions']} sessions, ${health['plugins']} providers';
        _testing = false;
      });
    } catch (e) {
      setState(() { _result = 'err:$e'; _testing = false; });
    }
  }

  Future<void> _connect() async {
    final url = _controller.text.trim();
    if (url.isEmpty) return;
    await context.read<ServerConfig>().setUrl(url);
  }

  bool get _isOk => _result?.startsWith('ok:') == true;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                // Logo
                Container(
                  width: 64, height: 64,
                  decoration: BoxDecoration(color: AppColors.accent, borderRadius: BorderRadius.circular(16)),
                  child: const Center(child: Text('N', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 32))),
                ),
                const SizedBox(height: 20),
                const Text('Welcome to NTC', style: TextStyle(fontSize: 22, fontWeight: FontWeight.w600)),
                const SizedBox(height: 6),
                const Text('Connect to your NTC server', style: TextStyle(color: AppColors.textMuted, fontSize: 14)),
                const SizedBox(height: 32),

                // URL input
                TextField(
                  controller: _controller,
                  decoration: const InputDecoration(
                    labelText: 'Server URL',
                    hintText: 'http://192.168.x.x:8640',
                    prefixIcon: Icon(Icons.dns, size: 20),
                  ),
                  keyboardType: TextInputType.url,
                  onSubmitted: (_) => _test(),
                ),
                const SizedBox(height: 16),

                // Test result
                if (_result != null)
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: _isOk ? AppColors.successSoft : AppColors.errorSoft,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Row(
                      children: [
                        Icon(_isOk ? Icons.check_circle : Icons.error,
                          color: _isOk ? AppColors.success : AppColors.error, size: 18),
                        const SizedBox(width: 8),
                        Expanded(child: Text(
                          _isOk ? _result!.substring(3) : _result!.substring(4),
                          style: TextStyle(fontSize: 12, color: _isOk ? AppColors.success : AppColors.error),
                        )),
                      ],
                    ),
                  ),

                const SizedBox(height: 20),

                // Buttons
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton(
                        onPressed: _testing ? null : _test,
                        style: OutlinedButton.styleFrom(
                          foregroundColor: AppColors.accent,
                          side: const BorderSide(color: AppColors.border),
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                        child: _testing
                            ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent))
                            : const Text('Test Connection'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: FilledButton(
                        onPressed: _isOk ? _connect : null,
                        style: FilledButton.styleFrom(
                          backgroundColor: AppColors.accent,
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                        child: const Text('Connect'),
                      ),
                    ),
                  ],
                ),

                const SizedBox(height: 32),
                const Text(
                  'Run the NTC server on your Mac:\nmake dev',
                  style: TextStyle(color: AppColors.textMuted, fontSize: 12, fontFamily: 'monospace'),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
