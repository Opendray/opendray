class Session {
  final String id;
  final String name;
  final String sessionType;
  final String? claudeSessionId;
  final String? claudeAccountId;
  final String cwd;
  final String status;
  final String model;
  final int? pid;
  final List<String> extraArgs;
  final Map<String, String> envOverrides;
  final double totalCostUsd;
  final int inputTokens;
  final int outputTokens;
  final DateTime createdAt;
  final DateTime lastActiveAt;

  const Session({
    required this.id,
    required this.name,
    required this.sessionType,
    this.claudeSessionId,
    this.claudeAccountId,
    required this.cwd,
    required this.status,
    required this.model,
    this.pid,
    this.extraArgs = const [],
    this.envOverrides = const {},
    this.totalCostUsd = 0,
    this.inputTokens = 0,
    this.outputTokens = 0,
    required this.createdAt,
    required this.lastActiveAt,
  });

  bool get isRunning => status == 'running';

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      id: json['id'] as String,
      name: json['name'] as String? ?? '',
      sessionType: json['sessionType'] as String? ?? 'claude',
      claudeSessionId: json['claudeSessionId'] as String?,
      claudeAccountId: json['claudeAccountId'] as String?,
      cwd: json['cwd'] as String,
      status: json['status'] as String? ?? 'stopped',
      model: json['model'] as String? ?? '',
      pid: json['pid'] as int?,
      extraArgs: (json['extraArgs'] as List<dynamic>?)?.cast<String>() ?? [],
      envOverrides: (json['envOverrides'] as Map<String, dynamic>?)?.cast<String, String>() ?? {},
      totalCostUsd: (json['totalCostUsd'] as num?)?.toDouble() ?? 0,
      inputTokens: json['inputTokens'] as int? ?? 0,
      outputTokens: json['outputTokens'] as int? ?? 0,
      createdAt: DateTime.parse(json['createdAt'] as String),
      lastActiveAt: DateTime.parse(json['lastActiveAt'] as String),
    );
  }
}
