import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Round Table API client (experimental) — a cross-vendor AI GROUP CHAT.
// Members are the seated providers (claude/codex/antigravity/grok/opencode)
// plus the operator. The operator @mentions who should reply; each mentioned
// member reads the whole thread and answers in character. Mirrors
// app/shared/src/lib/roundtable.ts + app/web/src/components/roundtable/*.
// Wraps /api/v1/round-tables/*.

// Seat providers with a headless worker path (must match the backend's
// worker.AgentWorker.buildCommand switch). A standalone gemini seat has no
// headless path yet.
const seatProviders = <String>[
  'claude',
  'codex',
  'antigravity',
  'grok',
  'opencode',
];

// Vendor family behind each seat — the diversity is the whole point.
// opencode is provider-agnostic, so it names the CLI itself.
const seatVendor = <String, String>{
  'claude': 'Anthropic',
  'codex': 'OpenAI',
  'antigravity': 'Google Gemini',
  'grok': 'xAI Grok',
  'opencode': 'OpenCode',
};

// Seat providers that support opendray multi-account selection (a per-seat
// account pin honoured by the backend worker). Mirrors the backend's
// roundtable.providerHasAccounts.
const seatSupportsAccount = <String>{'claude', 'antigravity'};

// Default model per seat, pre-filled so nothing has to be typed. codex's own
// config default (gpt-5.4) is rejected on a plain ChatGPT plan.
const seatModelDefault = <String, String>{'codex': 'gpt-5.4-mini'};

class Seat {
  const Seat({
    required this.provider,
    this.model = '',
    this.accountId = '',
    this.persona = '',
  });

  factory Seat.fromJson(Map<String, dynamic> j) => Seat(
        provider: j['provider']?.toString() ?? '',
        model: j['model']?.toString() ?? '',
        accountId: j['account_id']?.toString() ?? '',
        persona: j['persona']?.toString() ?? '',
      );

  final String provider;
  final String model;
  final String accountId;
  final String persona;

  Map<String, dynamic> toJson() => {
        'provider': provider,
        if (model.isNotEmpty) 'model': model,
        if (accountId.isNotEmpty) 'account_id': accountId,
        if (persona.isNotEmpty) 'persona': persona,
      };
}

// One step of the role-based execution plan.
class PlanStep {
  const PlanStep({
    required this.assignee,
    required this.task,
    this.model = '',
    this.accountId = '',
    this.status = 'pending',
    this.sessionId = '',
  });

  factory PlanStep.fromJson(Map<String, dynamic> j) => PlanStep(
        assignee: j['assignee']?.toString() ?? '',
        model: j['model']?.toString() ?? '',
        accountId: j['account_id']?.toString() ?? '',
        task: j['task']?.toString() ?? '',
        status: j['status']?.toString() ?? 'pending',
        sessionId: j['session_id']?.toString() ?? '',
      );

  final String assignee;
  final String model;
  final String accountId;
  final String task;
  final String status; // pending | running | done
  final String sessionId;

  Map<String, dynamic> toJson() => {
        'assignee': assignee,
        if (model.isNotEmpty) 'model': model,
        if (accountId.isNotEmpty) 'account_id': accountId,
        'task': task,
        'status': status,
        if (sessionId.isNotEmpty) 'session_id': sessionId,
      };

  PlanStep copyWith({String? assignee, String? task}) => PlanStep(
        assignee: assignee ?? this.assignee,
        model: model,
        accountId: accountId,
        task: task ?? this.task,
        status: status,
        sessionId: sessionId,
      );
}

class RoundTable {
  const RoundTable({
    required this.id,
    required this.topic,
    required this.cwd,
    required this.seats,
    required this.framing,
    required this.plan,
    required this.status,
    required this.resultingSessionId,
    required this.origin,
    required this.updatedAt,
  });

  factory RoundTable.fromJson(Map<String, dynamic> j) {
    final rawSeats = j['seats'];
    final seats = rawSeats is List
        ? rawSeats
            .whereType<Map<String, dynamic>>()
            .map(Seat.fromJson)
            .toList()
        : <Seat>[];
    return RoundTable(
      id: j['id']?.toString() ?? '',
      topic: j['topic']?.toString() ?? '',
      cwd: j['cwd']?.toString() ?? '',
      seats: seats,
      framing: j['framing']?.toString() ?? '',
      plan: (j['plan'] is List)
          ? (j['plan'] as List)
              .whereType<Map<String, dynamic>>()
              .map(PlanStep.fromJson)
              .toList()
          : <PlanStep>[],
      status: j['status']?.toString() ?? 'active',
      resultingSessionId: j['resulting_session_id']?.toString() ?? '',
      origin: j['origin']?.toString() ?? 'operator',
      updatedAt: DateTime.tryParse(j['updated_at']?.toString() ?? '') ??
          DateTime.now(),
    );
  }

  final String id;
  final String topic;
  final String cwd;
  final List<Seat> seats;
  // Table-level directive shared by all members (topic + role relationships).
  final String framing;
  // Role-based execution plan (ordered steps). Empty until drafted.
  final List<PlanStep> plan;
  final String status; // active | closed
  final String resultingSessionId;
  final String origin;
  final DateTime updatedAt;

  bool get isClosed => status == 'closed';
}

class RtMessage {
  const RtMessage({
    required this.id,
    required this.role,
    required this.seatProvider,
    required this.seatModel,
    required this.kind,
    required this.content,
    required this.mentions,
    required this.createdAt,
  });

  factory RtMessage.fromJson(Map<String, dynamic> j) {
    final rawMentions = j['mentions'];
    return RtMessage(
      id: j['id']?.toString() ?? '',
      role: j['role']?.toString() ?? '',
      seatProvider: j['seat_provider']?.toString() ?? '',
      seatModel: j['seat_model']?.toString() ?? '',
      kind: j['kind']?.toString() ?? 'message',
      content: j['content']?.toString() ?? '',
      mentions: rawMentions is List
          ? rawMentions.map((e) => e.toString()).toList()
          : <String>[],
      createdAt: DateTime.tryParse(j['created_at']?.toString() ?? '') ??
          DateTime.now(),
    );
  }

  final String id;
  final String role; // operator | seat | system
  final String seatProvider;
  final String seatModel;
  final String kind; // message | summary
  final String content;
  final List<String> mentions;
  final DateTime createdAt;

  bool get isOperator => role == 'operator';
  bool get isSystem => role == 'system';
  bool get isSummary => kind == 'summary';
}

class SeatModelOption {
  const SeatModelOption({required this.value, required this.label});

  factory SeatModelOption.fromJson(Map<String, dynamic> j) => SeatModelOption(
        value: j['value']?.toString() ?? '',
        label: j['label']?.toString() ?? '',
      );

  final String value; // passed to --model ("" = CLI default)
  final String label;
}

class RoundtableApi {
  RoundtableApi(this._dio);
  final Dio _dio;

  Future<List<RoundTable>> list({String? cwd}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/round-tables',
        queryParameters: {if (cwd != null && cwd.isNotEmpty) 'cwd': cwd},
      );
      final raw = res.data?['round_tables'];
      return raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(RoundTable.fromJson)
              .toList()
          : <RoundTable>[];
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<(RoundTable, List<RtMessage>)> get(String id) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/round-tables/$id');
      final rt = RoundTable.fromJson(
        (res.data?['round_table'] as Map<String, dynamic>?) ?? const {},
      );
      final raw = res.data?['messages'];
      final msgs = raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(RtMessage.fromJson)
              .toList()
          : <RtMessage>[];
      return (rt, msgs);
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Selectable models per provider — antigravity/opencode enumerated live from
  // their CLIs; claude/codex/grok curated.
  Future<Map<String, List<SeatModelOption>>> seatModels() async {
    try {
      final res = await _dio
          .get<Map<String, dynamic>>('/api/v1/round-tables/models');
      final raw = res.data?['models'];
      final out = <String, List<SeatModelOption>>{};
      if (raw is Map) {
        raw.forEach((key, value) {
          if (value is List) {
            out[key.toString()] = value
                .whereType<Map<String, dynamic>>()
                .map(SeatModelOption.fromJson)
                .toList();
          }
        });
      }
      return out;
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<RoundTable> create({
    required List<Seat> seats,
    String? cwd,
    String framing = '',
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/round-tables',
        data: {
          if (cwd != null && cwd.isNotEmpty) 'cwd': cwd,
          if (framing.isNotEmpty) 'framing': framing,
          'seats': seats.map((s) => s.toJson()).toList(),
        },
      );
      return RoundTable.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Reassign roles (seats) and/or re-frame a live table. Only the fields
  // provided are changed.
  Future<RoundTable> update(
    String id, {
    List<Seat>? seats,
    String? framing,
    String? cwd,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/round-tables/$id',
        data: {
          if (seats != null) 'seats': seats.map((s) => s.toJson()).toList(),
          if (framing != null) 'framing': framing,
          if (cwd != null) 'cwd': cwd,
        },
      );
      return RoundTable.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Post a message. @mentioned members reply asynchronously — poll get() while
  // replies land.
  Future<void> postMessage(String id, String content) async {
    try {
      await _dio.post<void>(
        '/api/v1/round-tables/$id/messages',
        data: {'content': content},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // A member drafts a role-assigned execution plan (async).
  Future<void> draftPlan(String id, {String provider = ''}) async {
    try {
      await _dio.post<void>(
        '/api/v1/round-tables/$id/plan/draft',
        data: {'provider': provider},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Replace the plan (operator edits).
  Future<RoundTable> setPlan(String id, List<PlanStep> steps) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/round-tables/$id/plan',
        data: {'steps': steps.map((s) => s.toJson()).toList()},
      );
      return RoundTable.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Launch a real session to carry out one plan step; returns its id.
  Future<String> runPlanStep(
    String id,
    int index, {
    String cwd = '',
    String accountId = '',
    List<String> args = const [],
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/round-tables/$id/plan/run',
        data: {
          'index': index,
          if (cwd.isNotEmpty) 'cwd': cwd,
          if (accountId.isNotEmpty) 'account_id': accountId,
          'args': args,
        },
      );
      return res.data?['session_id']?.toString() ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Resume a paused auto-discussion for another burst.
  Future<void> continueDiscussion(String id) async {
    try {
      await _dio.post<void>('/api/v1/round-tables/$id/continue');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> summarize(String id, {String provider = ''}) async {
    try {
      await _dio.post<void>(
        '/api/v1/round-tables/$id/summarize',
        data: {'provider': provider},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Spawn a real agent session (full tool access) to implement the discussion.
  // Returns the new session id.
  Future<String> handoff(
    String id, {
    required String provider,
    String? cwd,
    String model = '',
    String accountId = '',
    bool forceNew = false,
    List<String> args = const [],
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/round-tables/$id/handoff',
        data: {
          'provider': provider,
          if (cwd != null && cwd.isNotEmpty) 'cwd': cwd,
          if (model.isNotEmpty) 'model': model,
          if (accountId.isNotEmpty) 'account_id': accountId,
          'force_new': forceNew,
          'args': args,
        },
      );
      return res.data?['session_id']?.toString() ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> close(String id) async {
    try {
      await _dio.post<void>('/api/v1/round-tables/$id/close');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Flip a closed chat back to active so the operator can resume it.
  Future<void> reopen(String id) async {
    try {
      await _dio.post<void>('/api/v1/round-tables/$id/reopen');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/round-tables/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final roundtableApiProvider = Provider<RoundtableApi>((ref) {
  return RoundtableApi(ref.watch(dioProvider));
});

// List of round tables, newest first.
final roundTablesProvider =
    FutureProvider.autoDispose<List<RoundTable>>((ref) {
  return ref.watch(roundtableApiProvider).list();
});

// Selectable seat models per provider (fetched once; drives the create sheet).
final seatModelsProvider =
    FutureProvider.autoDispose<Map<String, List<SeatModelOption>>>((ref) {
  return ref.watch(roundtableApiProvider).seatModels();
});
