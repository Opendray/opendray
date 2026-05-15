///
/// Generated file. Do not edit.
///
// coverage:ignore-file
// ignore_for_file: type=lint, unused_import
// dart format off

import 'package:flutter/widgets.dart';
import 'package:intl/intl.dart';
import 'package:slang/generated.dart';
import 'strings.g.dart';

// Path: <root>
class TranslationsZh extends Translations with BaseTranslations<AppLocale, Translations> {
	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	TranslationsZh({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  $meta = meta ?? TranslationMetadata(
		    locale: AppLocale.zh,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ),
		  super(cardinalResolver: cardinalResolver, ordinalResolver: ordinalResolver) {
		super.$meta.setFlatMapFunction($meta.getTranslation); // copy base translations to super.$meta
		$meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <zh>.
	@override final TranslationMetadata<AppLocale, Translations> $meta;

	/// Access flat map
	@override dynamic operator[](String key) => $meta.getTranslation(key) ?? super.$meta.getTranslation(key);

	late final TranslationsZh _root = this; // ignore: unused_field

	@override 
	TranslationsZh $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => TranslationsZh(meta: meta ?? this.$meta);

	// Translations
	@override late final _TranslationsCommonZh common = _TranslationsCommonZh._(_root);
	@override late final _TranslationsAuthZh auth = _TranslationsAuthZh._(_root);
	@override late final _TranslationsNavZh nav = _TranslationsNavZh._(_root);
	@override late final _TranslationsWebZh web = _TranslationsWebZh._(_root);
	@override late final _TranslationsMoreZh more = _TranslationsMoreZh._(_root);
	@override late final _TranslationsSessionsZh sessions = _TranslationsSessionsZh._(_root);
	@override late final _TranslationsMcpZh mcp = _TranslationsMcpZh._(_root);
	@override late final _TranslationsProvidersZh providers = _TranslationsProvidersZh._(_root);
	@override late final _TranslationsIntegrationsZh integrations = _TranslationsIntegrationsZh._(_root);
	@override late final _TranslationsMemoryWorkersZh memoryWorkers = _TranslationsMemoryWorkersZh._(_root);
	@override late final _TranslationsMemoryCleanupZh memoryCleanup = _TranslationsMemoryCleanupZh._(_root);
	@override late final _TranslationsProjectZh project = _TranslationsProjectZh._(_root);
	@override late final _TranslationsBackupsZh backups = _TranslationsBackupsZh._(_root);
	@override late final _TranslationsBackupTargetsZh backupTargets = _TranslationsBackupTargetsZh._(_root);
	@override late final _TranslationsBackupSchedulesZh backupSchedules = _TranslationsBackupSchedulesZh._(_root);
	@override late final _TranslationsBackupTargetEditorZh backupTargetEditor = _TranslationsBackupTargetEditorZh._(_root);
	@override late final _TranslationsGithostsZh githosts = _TranslationsGithostsZh._(_root);
	@override late final _TranslationsChannelsZh channels = _TranslationsChannelsZh._(_root);
	@override late final _TranslationsOnboardingZh onboarding = _TranslationsOnboardingZh._(_root);
	@override late final _TranslationsSkillsZh skills = _TranslationsSkillsZh._(_root);
	@override late final _TranslationsCustomTasksZh customTasks = _TranslationsCustomTasksZh._(_root);
	@override late final _TranslationsNotesPageZh notesPage = _TranslationsNotesPageZh._(_root);
	@override late final _TranslationsMemoryZh memory = _TranslationsMemoryZh._(_root);
	@override late final _TranslationsAboutZh about = _TranslationsAboutZh._(_root);
	@override late final _TranslationsSettingsZh settings = _TranslationsSettingsZh._(_root);
}

// Path: common
class _TranslationsCommonZh extends TranslationsCommonEn {
	_TranslationsCommonZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get ok => '确定';
	@override String get cancel => '取消';
	@override String get save => '保存';
	@override String get delete => '删除';
	@override String get edit => '编辑';
	@override String get back => '返回';
	@override String get done => '完成';
	@override String get close => '关闭';
	@override String get retry => '重试';
	@override String get copy => '复制';
	@override String get enabled => '已启用';
	@override String get refresh => '刷新';
}

// Path: auth
class _TranslationsAuthZh extends TranslationsAuthEn {
	_TranslationsAuthZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get signInTitle => '登录';
	@override String get changeServer => '更换';
	@override String get username => '用户名';
	@override String get password => '密码';
	@override String get signIn => '登录';
	@override String get signingIn => '登录中…';
	@override String get subtitle => '请使用运维账号登录。';
	@override String get errorRequired => '请输入用户名和密码';
	@override String errorGeneric({required Object error}) => '登录失败：${error}';
	@override String get errorFallback => '登录失败';
}

// Path: nav
class _TranslationsNavZh extends TranslationsNavEn {
	_TranslationsNavZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get sessions => '会话';
	@override String get memory => '记忆';
	@override String get notes => '笔记';
	@override String get more => '更多';
	@override String get activity => '活动';
	@override String get providers => '提供方';
	@override String get channels => '频道';
	@override String get integrations => '集成';
	@override String get plugins => '插件';
	@override String get backups => '备份';
	@override String get settings => '设置';
	@override String get tutorial => '教程';
	@override String get workspace => '工作区';
}

// Path: web
class _TranslationsWebZh extends TranslationsWebEn {
	_TranslationsWebZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get brand => 'opendray';
	@override String get loading => '加载中…';
	@override late final _TranslationsWebTopbarZh topbar = _TranslationsWebTopbarZh._(_root);
	@override late final _TranslationsWebSessionsZh sessions = _TranslationsWebSessionsZh._(_root);
	@override late final _TranslationsWebMemoryZh memory = _TranslationsWebMemoryZh._(_root);
	@override late final _TranslationsWebConflictsZh conflicts = _TranslationsWebConflictsZh._(_root);
	@override late final _TranslationsWebMemoryHealthZh memoryHealth = _TranslationsWebMemoryHealthZh._(_root);
	@override late final _TranslationsWebMemoryWorkersZh memoryWorkers = _TranslationsWebMemoryWorkersZh._(_root);
	@override late final _TranslationsWebCleanupInboxZh cleanupInbox = _TranslationsWebCleanupInboxZh._(_root);
	@override late final _TranslationsWebProjectZh project = _TranslationsWebProjectZh._(_root);
	@override late final _TranslationsWebMemoryInspectorZh memoryInspector = _TranslationsWebMemoryInspectorZh._(_root);
	@override late final _TranslationsWebNotesZh notes = _TranslationsWebNotesZh._(_root);
	@override late final _TranslationsWebActivityZh activity = _TranslationsWebActivityZh._(_root);
	@override late final _TranslationsWebProvidersZh providers = _TranslationsWebProvidersZh._(_root);
	@override late final _TranslationsWebChannelsZh channels = _TranslationsWebChannelsZh._(_root);
	@override late final _TranslationsWebIntegrationsZh integrations = _TranslationsWebIntegrationsZh._(_root);
	@override late final _TranslationsWebPluginsZh plugins = _TranslationsWebPluginsZh._(_root);
	@override late final _TranslationsWebBackupsZh backups = _TranslationsWebBackupsZh._(_root);
	@override late final _TranslationsWebServerSettingsZh serverSettings = _TranslationsWebServerSettingsZh._(_root);
	@override late final _TranslationsWebSettingsZh settings = _TranslationsWebSettingsZh._(_root);
	@override late final _TranslationsWebLogViewerZh logViewer = _TranslationsWebLogViewerZh._(_root);
	@override late final _TranslationsWebPathInputZh pathInput = _TranslationsWebPathInputZh._(_root);
	@override late final _TranslationsWebMemoryAmbientZh memoryAmbient = _TranslationsWebMemoryAmbientZh._(_root);
	@override late final _TranslationsWebNoteEditorZh noteEditor = _TranslationsWebNoteEditorZh._(_root);
	@override late final _TranslationsWebExportZh export = _TranslationsWebExportZh._(_root);
}

// Path: more
class _TranslationsMoreZh extends TranslationsMoreEn {
	_TranslationsMoreZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '更多';
	@override late final _TranslationsMoreIdentityZh identity = _TranslationsMoreIdentityZh._(_root);
	@override late final _TranslationsMoreSectionsZh sections = _TranslationsMoreSectionsZh._(_root);
	@override late final _TranslationsMoreItemsZh items = _TranslationsMoreItemsZh._(_root);
	@override String get signOut => '退出登录';
}

// Path: sessions
class _TranslationsSessionsZh extends TranslationsSessionsEn {
	_TranslationsSessionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '会话';
	@override String get refresh => '刷新';
	@override String get actions => '操作';
	@override String get spawn => '创建';
	@override late final _TranslationsSessionsFiltersZh filters = _TranslationsSessionsFiltersZh._(_root);
	@override late final _TranslationsSessionsCardZh card = _TranslationsSessionsCardZh._(_root);
	@override late final _TranslationsSessionsEmptyZh empty = _TranslationsSessionsEmptyZh._(_root);
	@override String get errorTitle => '加载会话失败';
	@override late final _TranslationsSessionsRelativeZh relative = _TranslationsSessionsRelativeZh._(_root);
	@override late final _TranslationsSessionsDetailZh detail = _TranslationsSessionsDetailZh._(_root);
	@override late final _TranslationsSessionsTerminalZh terminal = _TranslationsSessionsTerminalZh._(_root);
	@override late final _TranslationsSessionsActionZh action = _TranslationsSessionsActionZh._(_root);
	@override late final _TranslationsSessionsDirPickerZh dirPicker = _TranslationsSessionsDirPickerZh._(_root);
	@override late final _TranslationsSessionsInspectorZh inspector = _TranslationsSessionsInspectorZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetZh spawnSheet = _TranslationsSessionsSpawnSheetZh._(_root);
}

// Path: mcp
class _TranslationsMcpZh extends TranslationsMcpEn {
	_TranslationsMcpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP';
	@override String get newServer => '新建服务器';
	@override String get addSecret => '添加密钥';
	@override String get editConfig => '编辑配置';
	@override String get viewRawConfig => '查看原始配置';
	@override String get copyId => '复制 ID';
	@override String copiedSnack({required Object id}) => '已复制 ${id}';
	@override String get deleteServerTitle => '删除 MCP 服务器？';
	@override String get deleteSecretTitle => '删除密钥？';
	@override late final _TranslationsMcpErrorPrefixZh errorPrefix = _TranslationsMcpErrorPrefixZh._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
	@override late final _TranslationsMcpEditorZh editor = _TranslationsMcpEditorZh._(_root);
	@override late final _TranslationsMcpSecretZh secret = _TranslationsMcpSecretZh._(_root);
	@override late final _TranslationsMcpPopupZh popup = _TranslationsMcpPopupZh._(_root);
	@override late final _TranslationsMcpKvZh kv = _TranslationsMcpKvZh._(_root);
	@override String deleteServerBody({required Object id}) => '移除 ${id} 的密钥库目录。引用此服务器的会话将无法启动。';
	@override String deleteServerSnack({required Object id}) => '已删除 ${id}。';
	@override String serversCount({required Object count}) => '服务器（${count}）';
	@override String secretsCount({required Object count}) => '密钥（${count}）';
	@override String get emptyServers => '未注册任何 MCP 服务器。点击「新建服务器」添加一个。';
	@override String get emptySecrets => '暂无密钥。添加一个，将敏感的 env / headers 注入 MCP 服务器，无需放在 JSON 里。';
	@override String get noVaultFileYet => '尚无密钥库文件 — 添加密钥时会创建。';
	@override String get tapToReplaceHint => '点击替换 · 长按 / 垃圾桶 删除';
	@override String get failedToLoad => '加载 MCP 状态失败';
	@override String get serverCreatedSnack => 'MCP 服务器已创建。';
	@override String get serverUpdatedSnack => 'MCP 服务器已更新。';
	@override String get envHeading => '环境变量';
	@override String get encryptionAes => 'AES-GCM 加密（密钥存于 OS keychain）';
	@override String get encryptionPlaintext => '明文 — keychain 不可用';
	@override String toggleEnabledSnack({required Object name}) => '${name} 已启用。';
	@override String toggleDisabledSnack({required Object name}) => '${name} 已停用。';
}

// Path: providers
class _TranslationsProvidersZh extends TranslationsProvidersEn {
	_TranslationsProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '提供商';
	@override String get configSaved => '提供商配置已更新。';
	@override String saveFailedApi({required Object error}) => '保存失败：${error}';
	@override String saveFailedGeneric({required Object error}) => '保存失败：${error}';
	@override String get reload => '重新加载';
	@override late final _TranslationsProvidersErrorPrefixZh errorPrefix = _TranslationsProvidersErrorPrefixZh._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
	@override late final _TranslationsProvidersAccountsZh accounts = _TranslationsProvidersAccountsZh._(_root);
	@override String get configFallbackTitle => '提供商配置';
	@override String get saving => '保存中…';
	@override String get save => '保存';
	@override String get configLoadFailed => '加载提供商失败';
	@override String get argsHelper => '以空格分隔的 CLI 参数。';
	@override String get listEmptyHeadline => '未加载任何提供商。';
	@override String get listEmptyBody => '网关在启动时从插件目录解析提供商。如果有遗漏，请检查日志。';
	@override String get listLoadFailed => '加载提供商失败';
	@override String get cliSectionHeader => 'CLI 提供商';
	@override String enabledSnack({required Object name}) => '${name} 已启用。';
	@override String disabledSnack({required Object name}) => '${name} 已停用。';
}

// Path: integrations
class _TranslationsIntegrationsZh extends TranslationsIntegrationsEn {
	_TranslationsIntegrationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '集成';
	@override String get register => '注册';
	@override String get registerDialogTitle => '注册集成';
	@override String get edit => '编辑';
	@override String editTitle({required Object name}) => '编辑 ${name}';
	@override String get enabledLabel => '已启用';
	@override String get iSavedIt => '我已保存';
	@override String apiKeyForName({required Object name}) => '${name} 的 API key';
	@override String apiKeySubtitleRegister({required Object routePrefix}) => '将其交给集成方，使其能够通过 /api/v1/${routePrefix}/… 进行认证。';
	@override String copiedRequestId({required Object id}) => '已复制 request_id ${id}';
	@override String get updateOk => '集成已更新。';
	@override String registerFailedApi({required Object error}) => '注册失败：${error}';
	@override String registerFailedGeneric({required Object error}) => '注册失败：${error}';
	@override String updateFailedApi({required Object error}) => '更新失败：${error}';
	@override String updateFailedGeneric({required Object error}) => '更新失败：${error}';
	@override String get deleteTitle => '删除集成？';
	@override String deletedSnack({required Object name}) => '已删除 ${name}。';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String get rotateKey => '轮换密钥';
	@override String get rotateConfirmTitle => '轮换 API key？';
	@override String get rotate => '轮换';
	@override String newApiKeyTitle({required Object name}) => '${name} 的新 API key';
	@override String get newApiKeySubtitle => '将其交给集成方。旧密钥已失效。';
	@override String rotateFailedApi({required Object error}) => '轮换失败：${error}';
	@override String rotateFailedGeneric({required Object error}) => '轮换失败：${error}';
	@override String get deleteBody => '移除该注册并吊销 API key。使用旧 key 的进行中请求将开始失败。';
	@override String rotateBody({required Object name}) => '为 ${name} 生成新 API key 并立即让旧 key 失效。';
	@override String get appBarFallback => '集成';
	@override String get tooltipMore => '更多';
	@override String get tooltipReadOnly => '系统集成 — 只读';
	@override String get kvRoutePrefix => '路由前缀';
	@override String get kvBaseUrl => 'Base URL';
	@override String get kvScopes => '范围';
	@override String get kvVersion => '版本';
	@override String get kvLastHealthPing => '最近健康检查';
	@override String get kvCreated => '创建于';
	@override String get kvKeyRotated => 'Key 轮换于';
	@override String detailLoadFailed({required Object error}) => '加载集成失败：${error}';
	@override String get callsLoadFailed => '加载调用失败';
	@override String get noMatchingCalls => '日志中暂无匹配的调用。';
	@override String get directionAll => '全部';
	@override String get directionInbound => '入站';
	@override String get directionOutbound => '出站';
	@override late final _TranslationsIntegrationsFormZh form = _TranslationsIntegrationsFormZh._(_root);
	@override String get emptyState => '在 Web 管理端注册：集成 → 新建。';
	@override String get sectionRegistered => '已注册';
	@override String get sectionSystem => '系统';
	@override String get listLoadFailed => '加载集成失败';
}

// Path: memoryWorkers
class _TranslationsMemoryWorkersZh extends TranslationsMemoryWorkersEn {
	_TranslationsMemoryWorkersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆工作器';
	@override String savedSnack({required Object label}) => '${label} 已保存';
	@override String saveFailed({required Object error}) => '保存失败：${error}';
	@override String testFailed({required Object error}) => '测试调用失败：${error}';
	@override String get workerLabel => '工作器';
	@override String get summarizerHttp => '摘要器（HTTP）';
	@override String get agentCliPrint => 'Agent（CLI --print）';
	@override String get cliLabel => 'CLI';
	@override String get cliClaude => 'Claude';
	@override String get cliGemini => 'Gemini';
	@override String get claudeAccountLabel => 'Claude 账号';
	@override String get claudeAccountDefault => '默认';
	@override String get test => '测试';
	@override String get intro => '每个记忆系统的 LLM 触点都可以独立服务 — 由本地 summarizer 端点（LM Studio / OpenAI 兼容）或在 --print 模式下生成无头 Claude / Gemini agent 来处理。高质量叙事任务（gitactivity、transcript）适合 agent 工作器；高频任务（gatekeeper）按设计保留在本地端点上。';
	@override String get errorTitle => '端点不可达';
	@override String get errorDetail => '/api/v1/memory/workers 路由在 M25 中是新增的 — opendray 二进制可能需要重启以挂载这些路由并运行迁移 0029。';
	@override String get summarizerOnlyBadge => '仅 summarizer';
	@override String get summarizerInfo => '使用注册表默认 summarizer 提供商。在 Web 管理端选择具体行。';
	@override String get agentWarning => 'Agent 模式每次调用都会生成无头 CLI。延迟约 5-15 秒（相比 summarizer 约 1 秒）；成本从 CPU 转移到你的 Claude / Gemini 配额。';
	@override String get noCalls24h => '过去 24 小时没有调用。';
	@override String testOkSnack({required Object label, required Object duration}) => '${label} OK — ${duration}ms';
	@override String testFailedReturnedSnack({required Object label, required Object error}) => '${label} 失败：${error}';
	@override String get unknownError => '未知';
	@override late final _TranslationsMemoryWorkersTasksZh tasks = _TranslationsMemoryWorkersTasksZh._(_root);
}

// Path: memoryCleanup
class _TranslationsMemoryCleanupZh extends TranslationsMemoryCleanupEn {
	_TranslationsMemoryCleanupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆清理';
	@override String approveFailed({required Object error}) => '批准失败：${error}';
	@override String rejectFailed({required Object error}) => '拒绝失败：${error}';
	@override String loadFailed({required Object error}) => '加载失败：${error}';
	@override String get reject => '拒绝';
}

// Path: project
class _TranslationsProjectZh extends TranslationsProjectEn {
	_TranslationsProjectZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '项目';
	@override String get pickFirst => '请先选择一个项目。';
	@override String loadFailed({required Object error}) => '加载失败：${error}';
	@override String projectsLoadFailed({required Object error}) => '加载项目列表失败：${error}';
	@override String get projectLabel => '项目';
	@override String get resetTooltip => '重置项目记忆';
	@override String get append => '追加';
	@override String get appendDialogTitle => '追加日志条目';
	@override String get titleFieldLabel => '标题（可选）';
	@override String get contentFieldLabel => '内容（Markdown）';
	@override String appendFailed({required Object error}) => '失败：${error}';
	@override String approveFailed({required Object error}) => '批准失败：${error}';
	@override String rejectFailed({required Object error}) => '拒绝失败：${error}';
	@override String cleanupFailed({required Object error}) => '清理失败：${error}';
	@override String get resetConfirmTitle => '重置项目记忆？';
	@override String get alsoDeleteScanner => '同时删除扫描器文档';
	@override String get alsoDeletePgvector => '同时删除 pgvector 记忆';
	@override String get deleteForever => '永久删除';
	@override String resetDoneSnack({required Object parts}) => '已重置：${parts}';
	@override String resetFailed({required Object error}) => '重置失败：${error}';
	@override String docSavedSnack({required Object kind}) => '${kind} 已保存';
	@override String docSaveFailed({required Object error}) => '保存失败：${error}';
	@override String docHintTemplate({required Object kind}) => '以 Markdown 编写 ${kind}…';
	@override String get deleteEntryTooltip => '删除条目';
	@override String get agentReason => 'Agent 原因';
	@override String get reject => '拒绝';
	@override String get approve => '批准';
	@override String replaceConfirmTitle({required Object kind}) => '替换当前 ${kind}？';
	@override String replaceKind({required Object kind}) => '替换 ${kind}';
	@override String get reason => '原因';
	@override String get willMergeInto => '将合并到';
}

// Path: backups
class _TranslationsBackupsZh extends TranslationsBackupsEn {
	_TranslationsBackupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份';
	@override String get runConfirmTitle => '立即运行备份？';
	@override String get runConfirmBody => '向本地目标触发一次新的转储。任务在服务端运行；此列表会随进度刷新。';
	@override String get run => '运行';
	@override String get runNow => '立即运行';
	@override String get queueing => '入队中…';
	@override String queuedSnack({required Object id}) => '备份已入队（${id}）。监控进度中…';
	@override String runFailedApi({required Object error}) => '运行失败：${error}';
	@override String runFailedGeneric({required Object error}) => '运行失败：${error}';
	@override String rowSucceededSnack({required Object bytes}) => '备份成功（${bytes}）。';
	@override String rowFailedSnack({required Object error}) => '备份失败：${error}';
	@override String get unknownError => '未知错误';
	@override String get detailTitle => '备份详情';
	@override String get deleteTitle => '删除备份？';
	@override String deleteBody({required Object target}) => '从 ${target} 移除二进制文件，并在索引中标记该行为已删除。';
	@override String deletedSnack({required Object id}) => '已删除 ${id}。';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String get menuSchedules => '计划';
	@override String get menuTargets => '目标';
	@override late final _TranslationsBackupsKvZh kv = _TranslationsBackupsKvZh._(_root);
	@override late final _TranslationsBackupsEmptyMissingDepsZh emptyMissingDeps = _TranslationsBackupsEmptyMissingDepsZh._(_root);
	@override late final _TranslationsBackupsEmptyNoTargetsZh emptyNoTargets = _TranslationsBackupsEmptyNoTargetsZh._(_root);
	@override late final _TranslationsBackupsEmptyNoBackupsZh emptyNoBackups = _TranslationsBackupsEmptyNoBackupsZh._(_root);
	@override String get restartToActivate => '重启 opendray 以激活备份';
	@override String get passphraseSaved => '你的密语已保存。网关仅在启动时加载，因此更改需重启后才生效。';
	@override String get keyFileLabel => '密钥文件';
	@override String get configuredViaLabel => '配置方式';
	@override late final _TranslationsBackupsWizardZh wizard = _TranslationsBackupsWizardZh._(_root);
	@override String get statusReady => '备份就绪';
	@override String get statusCannot => '备份无法运行';
	@override String get overviewTargets => '目标';
	@override String get overviewSchedules => '计划';
	@override String get overviewBackups => '备份';
	@override String get failedToLoad => '加载备份失败';
	@override String get envVarConfigured => 'OPENDRAY_BACKUP_KEY 环境变量';
	@override String get savedConfirmCheckbox => '我已将密语保存到密码管理器';
	@override String get pgDumpMissing => 'pg_dump 不在 PATH 中。请安装 postgresql-client 并重启 opendray。';
	@override late final _TranslationsBackupsEncryptionZh encryption = _TranslationsBackupsEncryptionZh._(_root);
}

// Path: backupTargets
class _TranslationsBackupTargetsZh extends TranslationsBackupTargetsEn {
	_TranslationsBackupTargetsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份目标';
	@override String get newTarget => '新建目标';
	@override String get testConnection => '测试连接';
	@override String get editConfig => '编辑配置';
	@override String get viewRawConfig => '查看原始配置';
	@override String configDialogTitle({required Object kind}) => '${kind} 配置';
	@override String get deleteTitle => '删除目标？';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
}

// Path: backupSchedules
class _TranslationsBackupSchedulesZh extends TranslationsBackupSchedulesEn {
	_TranslationsBackupSchedulesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份计划';
	@override String get newButton => '新建';
	@override String get deleteTitle => '删除计划？';
	@override String get targetLabel => '目标';
	@override String get intervalLabel => '间隔';
	@override String get retentionLabel => '保留（最近 N 个）';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
	@override String get noTargets => '未配置任何备份目标。请从 Web 管理端或「目标」屏添加。';
	@override String get okMsgCreate => '计划已创建。';
	@override String get okMsgUpdate => '计划已更新。';
	@override String get okMsgDelete => '计划已删除。';
	@override String get errorPrefixCreate => '创建失败';
	@override String get errorPrefixUpdate => '更新失败';
	@override String get errorPrefixDelete => '删除失败';
	@override String deleteBody({required Object targetId}) => '移除目标 ${targetId} 的定期规格。已存在的备份不受影响。';
	@override String get emptyList => '暂无计划。\n点击「新建」创建一个。';
	@override String get validatePickTarget => '请选择一个目标。';
	@override String get validateInterval => '间隔必须大于 0。';
	@override String get formTitleEdit => '编辑计划';
	@override String get formTitleNew => '新建计划';
	@override String get saveButtonEdit => '保存';
	@override String get saveButtonNew => '创建';
	@override String get targetFixedHint => '目标一旦创建即不可改。';
	@override String get enabledOn => '调度器将按周期运行。';
	@override String get enabledOff => '已暂停 — 重新启用前不会自动运行。';
	@override String get loadFailedTitle => '加载计划失败';
	@override String get pausedBadge => '已暂停';
	@override String everyInterval({required Object interval}) => '每 ${interval}';
	@override String keepRetention({required Object n}) => '· 保留 ${n}';
	@override String nextRun({required Object when}) => '· 下次 ${when}';
	@override String lastRun({required Object when}) => '· 上次 ${when}';
}

// Path: backupTargetEditor
class _TranslationsBackupTargetEditorZh extends TranslationsBackupTargetEditorEn {
	_TranslationsBackupTargetEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get useHttps => '使用 HTTPS';
	@override String get pathStyle => '路径风格寻址';
	@override String get pathStyleSubtitle => '旧版 / MinIO';
	@override late final _TranslationsBackupTargetEditorKindsZh kinds = _TranslationsBackupTargetEditorKindsZh._(_root);
	@override String get formTitleEdit => '编辑目标';
	@override String get formTitleNew => '新建备份目标';
	@override String idHintAuto({required Object prefix}) => '自动：${prefix}-1';
	@override String get idHelper => '小写字母、数字、连字符。默认为下一个可用槽。';
	@override String get enabledOn => '定期和临时备份可使用此目标。';
	@override String get enabledOff => '服务器将拒绝向此处写入备份。';
	@override String get saving => '保存中…';
	@override String get create => '创建';
	@override String get rootDirLabel => '根目录';
	@override String get rootDirHint => '留空 = cfg.backup.local_dir (~/.opendray/backups)';
	@override String get hostLabel => '主机';
	@override String get portLabel => '端口';
	@override String get shareLabel => '共享';
	@override String get shareHint => '顶层共享名';
	@override String get shareSampleHint => 'Claude_Workspace';
	@override String get userLabel => '用户';
	@override String get passwordLabel => '密码';
	@override String get passwordHintKeepCurrent => '留空 = 保留当前值';
	@override String get passwordHintKeep => '留空 = 保留';
	@override String get pathPrefixLabel => '路径前缀';
	@override String get pathPrefixHintShareRoot => '共享根下的子文件夹（可选）';
	@override String get pathPrefixHintBaseUrl => 'Base URL 下的子文件夹（可选）';
	@override String get pathPrefixHintObjectKey => '对象键前缀（可选）';
	@override String get pathPrefixHintSshFolder => '绝对路径或相对用户主目录（可选）';
	@override String get pathPrefixHintRemoteRoot => '远端根下的子文件夹（可选）';
	@override String get endpointLabel => '端点';
	@override String get regionLabel => '区域';
	@override String get bucketLabel => '存储桶';
	@override String get accessKeyLabel => 'Access Key';
	@override String get secretKeyLabel => 'Secret Key';
	@override String get secretKeyHintEdit => '留空 = 保留当前值。已 AES-256-GCM 加密存储。';
	@override String get secretKeyHintNew => '已 AES-256-GCM 加密存储；不会回显。';
	@override String get baseUrlLabel => 'Base URL';
	@override String get baseUrlHint => '完整 URL 包含路径。Nextcloud：https://cloud.example/remote.php/dav/files/<user>';
	@override String get sftpPasswordHintEdit => '留空 = 保留。如果密码 + 私钥同时存在，私钥优先。';
	@override String get sftpPasswordHintNew => '密码或私钥二选一。两者同时存在时，密码仅作回退。';
	@override String get privateKeyLabel => '私钥（PEM）';
	@override String get privateKeyHintEdit => '留空 = 保留。粘贴 OpenSSH/PEM 内容。';
	@override String get privateKeyHintNew => '粘贴 OpenSSH/PEM 私钥内容。多行输入 — 保留 BEGIN/END 标记。';
	@override String get hostKeyLabel => 'Host key（pinning）';
	@override String get hostKeyHint => 'OpenSSH 格式的服务器公钥。`ssh-keyscan <host>` 获取。留空 = 不 pinning（局域网外不推荐）。';
	@override String get rcloneNote => '需要 opendray 主机上安装 rclone CLI。首次需运行 `rclone config` 交互式认证云账户。';
	@override String get rcloneRemoteLabel => '远端名';
	@override String get rcloneRemoteHint => '来自 `rclone config` 的名字（不带冒号）。';
	@override String get rcloneBinaryLabel => '二进制路径';
	@override String get rcloneBinaryHint => '覆盖 `which rclone`。留空 = PATH 查找。';
	@override String get rcloneConfigLabel => '配置路径';
	@override String get rcloneConfigHint => '覆盖 --config。留空 = rclone 默认。';
}

// Path: githosts
class _TranslationsGithostsZh extends TranslationsGithostsEn {
	_TranslationsGithostsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Git 主机';
	@override String get addHost => '添加主机';
	@override String get deleteTitle => '删除 Git 主机？';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
	@override late final _TranslationsGithostsErrorPrefixZh errorPrefix = _TranslationsGithostsErrorPrefixZh._(_root);
	@override late final _TranslationsGithostsFormZh form = _TranslationsGithostsFormZh._(_root);
	@override String deleteBody({required Object host}) => '移除该凭据。试图列出 ${host} 的 PR 的会话将回退到未认证 API。';
	@override String deletedSnack({required Object name}) => '已删除 ${name}。';
	@override String enabledSnack({required Object name}) => '${name} 已启用。';
	@override String disabledSnack({required Object name}) => '${name} 已停用。';
	@override String get emptyList => '未配置任何 Git 主机。\n\n添加一个凭据，让网关可以列出你仓库的 pull request。';
	@override String get failedToLoad => '加载 Git 主机失败';
}

// Path: channels
class _TranslationsChannelsZh extends TranslationsChannelsEn {
	_TranslationsChannelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '通道';
	@override String get kNew => '新建';
	@override String get sendTest => '发送测试消息';
	@override String get editConfig => '编辑配置';
	@override String get editNotifications => '编辑通知';
	@override String get viewRawConfig => '查看原始配置';
	@override String get copyChannelId => '复制通道 ID';
	@override String copiedSnack({required Object id}) => '已复制 ${id}';
	@override String createdSnack({required Object kind}) => '已创建 ${kind} 通道。';
	@override String createFailedApi({required Object error}) => '创建失败：${error}';
	@override String createFailedGeneric({required Object error}) => '创建失败：${error}';
	@override String get deleteTitle => '删除通道？';
	@override late final _TranslationsChannelsConfigDialogZh configDialog = _TranslationsChannelsConfigDialogZh._(_root);
	@override late final _TranslationsChannelsWebhookDialogZh webhookDialog = _TranslationsChannelsWebhookDialogZh._(_root);
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
	@override late final _TranslationsChannelsNotificationsZh notifications = _TranslationsChannelsNotificationsZh._(_root);
	@override late final _TranslationsChannelsPopupZh popup = _TranslationsChannelsPopupZh._(_root);
	@override late final _TranslationsChannelsBadgesZh badges = _TranslationsChannelsBadgesZh._(_root);
	@override String capsLabel({required Object list}) => '· 能力：${list}';
	@override String get bridgeWebOnly => 'Bridge 通道仅 Web 端';
	@override String get bridgeEmptyAdd => '在 Web 管理端添加：通道 → 新建。';
	@override String get deleteBody => '停止该通道并移除其配置。仍在传输中的通知会被静默丢弃。';
	@override late final _TranslationsChannelsSnacksZh snacks = _TranslationsChannelsSnacksZh._(_root);
	@override late final _TranslationsChannelsErrorPrefixZh errorPrefix = _TranslationsChannelsErrorPrefixZh._(_root);
	@override String get failedToLoad => '加载通道失败';
	@override late final _TranslationsChannelsKindsZh kinds = _TranslationsChannelsKindsZh._(_root);
}

// Path: onboarding
class _TranslationsOnboardingZh extends TranslationsOnboardingEn {
	_TranslationsOnboardingZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get gatewayLabel => '网关 URL';
	@override String get gatewayHint => 'https://opendray.example.com';
	@override String get kContinue => '继续';
}

// Path: skills
class _TranslationsSkillsZh extends TranslationsSkillsEn {
	_TranslationsSkillsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '技能';
	@override String get newSkill => '新建技能';
	@override String customizingBuiltin({required Object id}) => '自定义内置 ${id}';
	@override String get idLabel => 'Id（slug）';
	@override String get idHint => '例如：tdd-guide';
	@override String get bodyLabel => '正文（Markdown）';
	@override String loadFailedApi({required Object error}) => '加载失败：${error}';
	@override String loadFailedGeneric({required Object error}) => '加载失败：${error}';
	@override String get idRequired => '必须填写 Id。';
	@override String get bodyRequired => '正文不能为空。';
	@override String get snackCreated => '技能已创建。';
	@override String get snackOverride => '已保存为库覆盖。';
	@override String get snackUpdated => '技能已更新。';
	@override String saveFailedApi({required Object error}) => '保存失败：${error}';
	@override String saveFailedGeneric({required Object error}) => '保存失败：${error}';
	@override String get resetTitle => '重置为内置？';
	@override String get deleteTitle => '删除技能？';
	@override String resetBody({required Object id}) => '移除 ${id} 的库覆盖。会话将回退到内置正文。';
	@override String get resetButton => '重置';
	@override String resetSnack({required Object id}) => '已将 ${id} 重置为内置。';
	@override String deletedSnack({required Object id}) => '已删除 ${id}。';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String deleteBody({required Object id}) => '从库中移除 ${id}。引用它的会话在恢复前会失败。';
	@override String get newSkillTitle => '新建技能';
	@override String customizeTitle({required Object id}) => '自定义 ${id}';
	@override String editTitle({required Object id}) => '编辑 ${id}';
	@override String get resetTooltip => '重置为内置';
	@override String get deleteTooltip => '删除';
	@override String get saving => '保存中…';
	@override String get saveOverride => '保存覆盖';
	@override String get overrideBanner => '保存会以相同 id 创建一个库覆盖。会话将使用此正文而非内置版本，直到你重置。';
	@override String get idHelper => '小写字母 / 数字 / 横线。创建后锁定。';
	@override String get emptyList => '未配置任何技能。网关附带内置技能（planner、code-reviewer 等）。';
	@override String get failedToLoad => '加载技能失败';
}

// Path: customTasks
class _TranslationsCustomTasksZh extends TranslationsCustomTasksEn {
	_TranslationsCustomTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '自定义任务';
	@override String get newTask => '新建任务';
	@override String get deleteTitle => '删除任务？';
	@override String deletedSnack({required Object name}) => '已删除 ${name}。';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String get popupEdit => '编辑';
	@override String get popupDelete => '删除';
	@override String get nameHint => '例如：backend-tests';
	@override String get commandHint => '/run pnpm test --filter backend';
	@override String get descriptionHint => '在任务名下方显示的一行说明。';
	@override String get scopeGlobal => '全局';
	@override String get scopeProject => '项目';
	@override String get cwdHint => '/Users/you/projects/backend';
	@override String get snackCreated => '任务已创建。';
	@override String get snackUpdated => '任务已更新。';
	@override String get deleteBody => '从目录中移除任务。已插入到会话中的实例不受影响。';
	@override String get introBanner => '定义自己的斜杠命令。它们会与内置任务一起出现在会话任务选择器中。';
	@override String get validateNameRequired => '必须填写名称';
	@override String get validateCommandRequired => '必须填写命令';
	@override String get validateProjectCwd => '项目范围任务需要绝对 cwd 路径';
	@override String get appBarEdit => '编辑自定义任务';
	@override String get appBarNew => '新建自定义任务';
	@override String get fieldName => '名称';
	@override String get nameHelper => '在检查器的任务选择器中显示。';
	@override String get fieldCommand => '命令';
	@override String get commandHelper => '选择时插入到会话的文本。可以是 CLI 命令或 Claude 斜杠命令。';
	@override String get fieldDescription => '描述（可选）';
	@override String get fieldScope => '范围';
	@override String get globalScopeHint => '从任何会话可见，不论 cwd。';
	@override String get projectScopeHint => '仅当会话的 cwd 匹配以下路径时可见。';
	@override String get fieldProjectCwd => '项目 cwd';
	@override String get cwdHelper => '绝对路径。以此 cwd 启动的会话将看到该任务。';
	@override String get saving => '保存中…';
	@override String get save => '保存';
	@override String get create => '创建';
	@override String get failedToLoad => '加载自定义任务失败';
}

// Path: notesPage
class _TranslationsNotesPageZh extends TranslationsNotesPageEn {
	_TranslationsNotesPageZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '笔记';
	@override String get newButton => '新建';
	@override String get newNoteDialogTitle => '新建笔记';
	@override String get searchHint => '搜索整个仓库…';
	@override String get up => '上级';
	@override String get copyPath => '复制路径';
	@override String get open => '打开';
	@override String copiedSnack({required Object path}) => '已复制 ${path}';
	@override String get deleteTitle => '删除笔记？';
	@override String deletedSnack({required Object path}) => '已删除 ${path}';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String createFailedApi({required Object error}) => '创建失败：${error}';
	@override String createFailedGeneric({required Object error}) => '创建失败：${error}';
	@override String get pathLabel => '相对仓库的路径';
	@override String get pathHint => 'personal/scratch.md';
	@override String get create => '创建';
	@override String get popupDelete => '删除';
	@override String get deleteBody => '此操作不可撤销。仓库的 git 同步会同时移除网关主机上的文件。';
	@override String emptyFilterMatch({required Object query}) => '未找到匹配「${query}」的笔记。';
	@override String get emptyVault => '仓库为空。点击 + 创建第一条笔记。';
	@override String emptyFolder({required Object path}) => '文件夹「${path}」为空。';
	@override String get validatePath => '必须填写路径';
	@override String get validatePathDots => '路径不能包含「..」';
	@override String get pathHelper => '缺失时自动追加 .md。';
	@override late final _TranslationsNotesPageEditorZh editor = _TranslationsNotesPageEditorZh._(_root);
}

// Path: memory
class _TranslationsMemoryZh extends TranslationsMemoryEn {
	_TranslationsMemoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆';
	@override String get more => '更多';
	@override String get workers => '记忆工作器';
	@override String get kNew => '新建';
	@override String get searchHint => '搜索…';
	@override String get projectLabel => '项目';
	@override String get filterHint => '按名称或路径筛选…';
	@override String get copied => '已复制';
	@override String get copyTooltip => '复制文本';
	@override late final _TranslationsMemoryDeleteAllConfirmZh deleteAllConfirm = _TranslationsMemoryDeleteAllConfirmZh._(_root);
	@override String deletedSnackOne({required Object n}) => '已删除 ${n} 条记忆';
	@override String deletedSnackOther({required Object n}) => '已删除 ${n} 条记忆';
	@override String bulkDeleteFailedApi({required Object error}) => '批量删除失败：${error}';
	@override String bulkDeleteFailedGeneric({required Object error}) => '批量删除失败：${error}';
	@override late final _TranslationsMemoryDeleteOneZh deleteOne = _TranslationsMemoryDeleteOneZh._(_root);
	@override late final _TranslationsMemoryScopeZh scope = _TranslationsMemoryScopeZh._(_root);
	@override late final _TranslationsMemoryCreateZh create = _TranslationsMemoryCreateZh._(_root);
}

// Path: about
class _TranslationsAboutZh extends TranslationsAboutEn {
	_TranslationsAboutZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '关于';
	@override String get loading => '加载中…';
	@override late final _TranslationsAboutSectionsZh sections = _TranslationsAboutSectionsZh._(_root);
	@override late final _TranslationsAboutFieldsZh fields = _TranslationsAboutFieldsZh._(_root);
	@override String copied({required Object label}) => '已复制 ${label}';
	@override String get copyTooltip => '复制';
	@override late final _TranslationsAboutCopyLabelsZh copyLabels = _TranslationsAboutCopyLabelsZh._(_root);
	@override String get tagline => 'opendray mobile — 多 CLI 网关控制。\n源码：github.com/Opendray/opendray_v2';
}

// Path: settings
class _TranslationsSettingsZh extends TranslationsSettingsEn {
	_TranslationsSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置';
	@override late final _TranslationsSettingsLanguageZh language = _TranslationsSettingsLanguageZh._(_root);
	@override late final _TranslationsSettingsAppearanceZh appearance = _TranslationsSettingsAppearanceZh._(_root);
	@override late final _TranslationsSettingsAccountZh account = _TranslationsSettingsAccountZh._(_root);
	@override late final _TranslationsSettingsGatewayZh gateway = _TranslationsSettingsGatewayZh._(_root);
	@override late final _TranslationsSettingsChangeCredentialsZh changeCredentials = _TranslationsSettingsChangeCredentialsZh._(_root);
	@override late final _TranslationsSettingsLogViewerZh logViewer = _TranslationsSettingsLogViewerZh._(_root);
	@override late final _TranslationsSettingsServerSettingsZh serverSettings = _TranslationsSettingsServerSettingsZh._(_root);
}

// Path: web.topbar
class _TranslationsWebTopbarZh extends TranslationsWebTopbarEn {
	_TranslationsWebTopbarZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get expandSidebar => '展开侧边栏';
	@override String get collapseSidebar => '收起侧边栏';
	@override String get search => '搜索';
	@override String get openPalette => '打开命令面板';
	@override String get theme => '主题';
	@override String themeLabel({required Object mode}) => '主题：${mode}';
	@override String get appearance => '外观';
	@override String get themeLight => '浅色';
	@override String get themeDark => '深色';
	@override String get themeSystem => '跟随系统';
	@override String get language => '语言';
	@override String get languageEnglish => 'English';
	@override String get languageChinese => '中文';
	@override String get signedInAs => '登录账号';
	@override String get tokenExpires => '令牌到期';
	@override String get signOut => '退出登录';
}

// Path: web.sessions
class _TranslationsWebSessionsZh extends TranslationsWebSessionsEn {
	_TranslationsWebSessionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebSessionsListZh list = _TranslationsWebSessionsListZh._(_root);
	@override late final _TranslationsWebSessionsTabsZh tabs = _TranslationsWebSessionsTabsZh._(_root);
	@override late final _TranslationsWebSessionsPageZh page = _TranslationsWebSessionsPageZh._(_root);
	@override late final _TranslationsWebSessionsEmptyZh empty = _TranslationsWebSessionsEmptyZh._(_root);
	@override late final _TranslationsWebSessionsHeaderZh header = _TranslationsWebSessionsHeaderZh._(_root);
	@override late final _TranslationsWebSessionsTerminalZh terminal = _TranslationsWebSessionsTerminalZh._(_root);
	@override late final _TranslationsWebSessionsSpawnZh spawn = _TranslationsWebSessionsSpawnZh._(_root);
	@override late final _TranslationsWebSessionsAccountSwitcherZh accountSwitcher = _TranslationsWebSessionsAccountSwitcherZh._(_root);
	@override late final _TranslationsWebSessionsInspectorZh inspector = _TranslationsWebSessionsInspectorZh._(_root);
	@override late final _TranslationsWebSessionsEndedZh ended = _TranslationsWebSessionsEndedZh._(_root);
	@override late final _TranslationsWebSessionsFileBrowserZh fileBrowser = _TranslationsWebSessionsFileBrowserZh._(_root);
}

// Path: web.memory
class _TranslationsWebMemoryZh extends TranslationsWebMemoryEn {
	_TranslationsWebMemoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆';
	@override String get subtitle => '浏览、搜索并编辑 Agent 通过 opendray-memory MCP 服务器存储的记忆。';
	@override String get navProject => '项目';
	@override String get navCleanupInbox => '清理收件箱';
	@override String get navWorkers => 'Workers';
	@override String get navConfiguration => '配置 →';
}

// Path: web.conflicts
class _TranslationsWebConflictsZh extends TranslationsWebConflictsEn {
	_TranslationsWebConflictsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '跨层冲突';
	@override String get subtitle => '每日 detector 在 facts/plan/goal/journal 之间发现的矛盾。';
	@override String get loading => '正在加载冲突…';
	@override String get empty => '无待处理冲突。点击"立即检测"运行一次按需扫描。';
	@override String get pickCwd => '选一个项目查看其冲突。';
	@override String get detectNow => '立即检测';
	@override String detected({required Object count}) => '新发现 ${count} 条冲突';
	@override String get accept => '采纳';
	@override String get dismiss => '驳回';
	@override String get accepted => '已采纳 — 别忘了实际修正';
	@override String get dismissed => '已驳回';
	@override late final _TranslationsWebConflictsSeverityZh severity = _TranslationsWebConflictsSeverityZh._(_root);
}

// Path: web.memoryHealth
class _TranslationsWebMemoryHealthZh extends TranslationsWebMemoryHealthEn {
	_TranslationsWebMemoryHealthZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object days}) => '记忆健康 — 最近 ${days} 天';
	@override String get subtitle => '本项目两套记忆子系统的运行情况聚合。';
	@override String get loading => '正在加载健康快照…';
	@override String get errorLoading => '加载健康快照失败。';
	@override String get pickCwd => '选一个项目查看其记忆健康。';
	@override String get newFacts => '新增 facts';
	@override String newFactsHint({required Object total}) => '累计 ${total} 条';
	@override String get captureFires => 'Capture 触发次数';
	@override String captureFiresHint({required Object stored, required Object deduped}) => '存入 ${stored} · 去重 ${deduped}';
	@override String get newJournal => '新增日志条目';
	@override String newJournalHint({required Object total}) => '累计 ${total} 条';
	@override String get planAge => '计划最近更新';
	@override String planAgeHint({required Object count}) => '${count} 条 plan-drift 提案待审';
	@override String get planAgeHintNone => '无 plan-drift 提案待审';
	@override String get goalAge => '目标最近更新';
	@override String get pending => '待审提案';
	@override String pendingHint({required Object days}) => '最久 ${days} 天';
	@override String topHit({required Object hits}) => '命中最多 · ${hits} 次';
	@override String zeroHit({required Object count}) => '${count} 条超过 7 天未命中的 fact — 清理候选。';
	@override String get never => '从未';
	@override String get today => '今日';
	@override String daysAgo_one({required Object count}) => '${count} 天前';
	@override String daysAgo_other({required Object count}) => '${count} 天前';
}

// Path: web.memoryWorkers
class _TranslationsWebMemoryWorkersZh extends TranslationsWebMemoryWorkersEn {
	_TranslationsWebMemoryWorkersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Memory workers';
	@override String get loading => '正在加载 worker 配置…';
	@override String get errorTitle => '无法访问该接口。';
	@override String get errorDescription => '/api/v1/memory/workers 路由在 M25 中新增 — opendray 二进制可能需要重启以挂载它们并执行 migration 0029。';
	@override String get intro => 'Memory 子系统的每个 LLM 接入点都可由本地 <1>summarizer</1>（LM Studio / OpenAI 兼容）端点服务，或通过以 <5>--print</5> 模式启动的无头 <3>Claude / Gemini agent</3> 服务。叙事性任务（gitactivity、transcript）从 agent worker 中获益；高频任务（gatekeeper）按设计仍走本地端点。';
	@override String get enabledBadge => '已启用';
	@override String get disabledBadge => '已禁用';
	@override String get summarizerOnlyBadge => '仅 summarizer';
	@override String callsCount({required Object count}) => '${count} 次调用 · 24 小时';
	@override String avgMs({required Object ms}) => '平均 ${ms}ms';
	@override String errorsCount({required Object count}) => '${count} 次错误';
	@override String get workerLabel => 'Worker';
	@override String get summarizerHttp => 'Summarizer (HTTP)';
	@override String get agentCliPrint => 'Agent (CLI --print)';
	@override String get summarizerProviderLabel => 'Summarizer Provider';
	@override String get registryDefault => '注册表默认值';
	@override String get cliLabel => 'CLI';
	@override String get selectPlaceholder => '选择';
	@override String get cliClaude => 'Claude';
	@override String get cliGemini => 'Gemini';
	@override String get claudeAccountLabel => 'Claude 账号';
	@override String get claudeAccountDefault => '默认';
	@override String get agentWarning => 'Agent 模式每次调用都会启动一个无头 CLI。延迟从 <1>~1s</1>（summarizer）上升到 <3>~5-15s</3>；成本从 CPU 转移到 Claude/Gemini 配额。';
	@override String get enabledCheckbox => '启用';
	@override String get testButton => '测试';
	@override String get saveButton => '保存';
	@override String recentCalls({required Object count}) => '最近调用 (${count})';
	@override String get tableWhen => '时间';
	@override String get tableWorker => 'worker';
	@override String get tableMs => 'ms';
	@override String get tableOk => 'ok';
	@override String savedToast({required Object label}) => '${label} 已更新';
	@override String get saveFailedToast => '保存失败';
	@override String testOkToast({required Object label, required Object ms}) => '${label} OK — ${ms}ms';
	@override String testFailedToast({required Object label}) => '${label} 失败';
	@override String get testCallFailedToast => '测试调用失败';
	@override String get unknownError => '未知错误';
	@override late final _TranslationsWebMemoryWorkersTasksZh tasks = _TranslationsWebMemoryWorkersTasksZh._(_root);
}

// Path: web.cleanupInbox
class _TranslationsWebCleanupInboxZh extends TranslationsWebCleanupInboxEn {
	_TranslationsWebCleanupInboxZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get emptyTitle => '清理收件箱为空';
	@override String get emptyDescription => '目前所有项目均无待处理的清理决策。LLM librarian 要么尚未跑过这些记忆，要么判断全部仍是关键内容。';
	@override String get title => '清理收件箱';
	@override String get subtitle => 'LLM memory librarian 的跨项目待处理决策。批准 stale → 删除，批准 duplicate → 合并，批准 keep → 一段时间内冻结该条目不再被重新判断。';
	@override String get globalScope => '(全局)';
	@override String get orphanBadge => '孤立';
	@override String get orphanTitle => 'scope_key 被截断（老旧的镜像导入数据）。不是可导航的项目。';
	@override String get openProject => '打开项目';
	@override String get mergeIntoPrefix => '→ 合并到';
	@override String get reasonPrefix => '原因：';
	@override String get executeButton => '执行';
	@override String get confirmKeepButton => '确认保留';
	@override String get rejectButton => '驳回';
	@override String get approvedKeptToast => '已保留';
	@override String approvedExecutedToast({required Object verdict}) => '已执行 ${verdict}';
	@override String get approveFailedToast => '批准失败';
	@override String get rejectedToast => '已驳回 — 记忆保留';
	@override String get rejectFailedToast => '驳回失败';
}

// Path: web.project
class _TranslationsWebProjectZh extends TranslationsWebProjectEn {
	_TranslationsWebProjectZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectPickerZh picker = _TranslationsWebProjectPickerZh._(_root);
	@override String get noCwd => '选择一个项目以管理其记忆。';
	@override late final _TranslationsWebProjectHeaderZh header = _TranslationsWebProjectHeaderZh._(_root);
	@override late final _TranslationsWebProjectTabsZh tabs = _TranslationsWebProjectTabsZh._(_root);
	@override late final _TranslationsWebProjectDocLabelZh docLabel = _TranslationsWebProjectDocLabelZh._(_root);
	@override late final _TranslationsWebProjectVerdictLabelZh verdictLabel = _TranslationsWebProjectVerdictLabelZh._(_root);
	@override late final _TranslationsWebProjectEditorZh editor = _TranslationsWebProjectEditorZh._(_root);
	@override late final _TranslationsWebProjectReadonlyZh readonly = _TranslationsWebProjectReadonlyZh._(_root);
	@override late final _TranslationsWebProjectJournalZh journal = _TranslationsWebProjectJournalZh._(_root);
	@override late final _TranslationsWebProjectInboxZh inbox = _TranslationsWebProjectInboxZh._(_root);
	@override late final _TranslationsWebProjectCleanupZh cleanup = _TranslationsWebProjectCleanupZh._(_root);
	@override late final _TranslationsWebProjectResetZh reset = _TranslationsWebProjectResetZh._(_root);
}

// Path: web.memoryInspector
class _TranslationsWebMemoryInspectorZh extends TranslationsWebMemoryInspectorEn {
	_TranslationsWebMemoryInspectorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryInspectorStatusZh status = _TranslationsWebMemoryInspectorStatusZh._(_root);
	@override String get statusBody => '这是网关当前用于每次 <1>memory_search</1> / <3>memory_store</3> 调用的 embedder。如果与上方配置不一致，说明有未保存的更改 — 点击 Save 后重启服务即可生效。';
	@override late final _TranslationsWebMemoryInspectorScopeZh scope = _TranslationsWebMemoryInspectorScopeZh._(_root);
	@override late final _TranslationsWebMemoryInspectorSearchZh search = _TranslationsWebMemoryInspectorSearchZh._(_root);
	@override late final _TranslationsWebMemoryInspectorRecordsZh records = _TranslationsWebMemoryInspectorRecordsZh._(_root);
	@override late final _TranslationsWebMemoryInspectorRowZh row = _TranslationsWebMemoryInspectorRowZh._(_root);
	@override late final _TranslationsWebMemoryInspectorToastsZh toasts = _TranslationsWebMemoryInspectorToastsZh._(_root);
	@override late final _TranslationsWebMemoryInspectorBulkDeleteZh bulkDelete = _TranslationsWebMemoryInspectorBulkDeleteZh._(_root);
	@override late final _TranslationsWebMemoryInspectorAddMemZh addMem = _TranslationsWebMemoryInspectorAddMemZh._(_root);
	@override late final _TranslationsWebMemoryInspectorPickerZh picker = _TranslationsWebMemoryInspectorPickerZh._(_root);
	@override late final _TranslationsWebMemoryInspectorMigrationBannerZh migrationBanner = _TranslationsWebMemoryInspectorMigrationBannerZh._(_root);
	@override late final _TranslationsWebMemoryInspectorReembedZh reembed = _TranslationsWebMemoryInspectorReembedZh._(_root);
}

// Path: web.notes
class _TranslationsWebNotesZh extends TranslationsWebNotesEn {
	_TranslationsWebNotesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '笔记';
	@override late final _TranslationsWebNotesHeaderZh header = _TranslationsWebNotesHeaderZh._(_root);
	@override late final _TranslationsWebNotesLeftZh left = _TranslationsWebNotesLeftZh._(_root);
	@override late final _TranslationsWebNotesTagsZh tags = _TranslationsWebNotesTagsZh._(_root);
	@override late final _TranslationsWebNotesTreeZh tree = _TranslationsWebNotesTreeZh._(_root);
	@override late final _TranslationsWebNotesOutlineZh outline = _TranslationsWebNotesOutlineZh._(_root);
	@override late final _TranslationsWebNotesNewNoteZh newNote = _TranslationsWebNotesNewNoteZh._(_root);
	@override late final _TranslationsWebNotesEmptyZh empty = _TranslationsWebNotesEmptyZh._(_root);
	@override late final _TranslationsWebNotesPickerZh picker = _TranslationsWebNotesPickerZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncZh vaultSync = _TranslationsWebNotesVaultSyncZh._(_root);
	@override late final _TranslationsWebNotesSyncBadgeZh syncBadge = _TranslationsWebNotesSyncBadgeZh._(_root);
}

// Path: web.activity
class _TranslationsWebActivityZh extends TranslationsWebActivityEn {
	_TranslationsWebActivityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '活动';
	@override String get subtitle => '按调用维度审计每个由注册集成发起的 API 请求。包括入站调用（第三方应用以集成 API key 调用 opendray）和出站代理调用（admin → opendray 代理 → 集成）。本管理端 UI 直接发起的调用不会被记录。';
	@override String get refresh => '刷新';
	@override String get refreshTooltip => '刷新';
	@override late final _TranslationsWebActivityFiltersZh filters = _TranslationsWebActivityFiltersZh._(_root);
	@override String callsCount_one({required Object count}) => '${count} 次调用';
	@override String callsCount_other({required Object count}) => '${count} 次调用';
	@override String get loading => '加载中…';
	@override late final _TranslationsWebActivityTableZh table = _TranslationsWebActivityTableZh._(_root);
	@override late final _TranslationsWebActivityEmptyZh empty = _TranslationsWebActivityEmptyZh._(_root);
	@override late final _TranslationsWebActivityEventsZh events = _TranslationsWebActivityEventsZh._(_root);
}

// Path: web.providers
class _TranslationsWebProvidersZh extends TranslationsWebProvidersEn {
	_TranslationsWebProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProvidersListZh list = _TranslationsWebProvidersListZh._(_root);
	@override late final _TranslationsWebProvidersDetailZh detail = _TranslationsWebProvidersDetailZh._(_root);
	@override late final _TranslationsWebProvidersConfigFormZh configForm = _TranslationsWebProvidersConfigFormZh._(_root);
	@override late final _TranslationsWebProvidersClaudeAccountsZh claudeAccounts = _TranslationsWebProvidersClaudeAccountsZh._(_root);
}

// Path: web.channels
class _TranslationsWebChannelsZh extends TranslationsWebChannelsEn {
	_TranslationsWebChannelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '频道';
	@override String get subtitle => '双向消息集成。每个频道的出站通知按其 <1>notify_on</1> 过滤。';
	@override String get newButton => '新建频道';
	@override String get loading => '加载中…';
	@override late final _TranslationsWebChannelsEmptyZh empty = _TranslationsWebChannelsEmptyZh._(_root);
	@override late final _TranslationsWebChannelsCardZh card = _TranslationsWebChannelsCardZh._(_root);
	@override late final _TranslationsWebChannelsToastsZh toasts = _TranslationsWebChannelsToastsZh._(_root);
	@override late final _TranslationsWebChannelsDialogZh dialog = _TranslationsWebChannelsDialogZh._(_root);
	@override late final _TranslationsWebChannelsNotificationsZh notifications = _TranslationsWebChannelsNotificationsZh._(_root);
	@override late final _TranslationsWebChannelsBridgeZh bridge = _TranslationsWebChannelsBridgeZh._(_root);
	@override late final _TranslationsWebChannelsSetupZh setup = _TranslationsWebChannelsSetupZh._(_root);
}

// Path: web.integrations
class _TranslationsWebIntegrationsZh extends TranslationsWebIntegrationsEn {
	_TranslationsWebIntegrationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '集成';
	@override String get subtitle => '调用 opendray 的外部应用。通过 <1>/api/v1/proxy/&lt;prefix&gt;/…</1> 反向代理，并通过 WS 端点订阅事件。';
	@override String get register => '注册';
	@override String get loading => '加载中…';
	@override late final _TranslationsWebIntegrationsTabsZh tabs = _TranslationsWebIntegrationsTabsZh._(_root);
	@override late final _TranslationsWebIntegrationsEmptyZh empty = _TranslationsWebIntegrationsEmptyZh._(_root);
	@override String get groupSystem => '系统（由 opendray 管理）';
	@override String get groupOperator => '用户注册';
	@override late final _TranslationsWebIntegrationsCardZh card = _TranslationsWebIntegrationsCardZh._(_root);
	@override late final _TranslationsWebIntegrationsRegisterDialogZh register_dialog = _TranslationsWebIntegrationsRegisterDialogZh._(_root);
	@override late final _TranslationsWebIntegrationsRevealZh reveal = _TranslationsWebIntegrationsRevealZh._(_root);
	@override late final _TranslationsWebIntegrationsEditDialogZh edit_dialog = _TranslationsWebIntegrationsEditDialogZh._(_root);
	@override late final _TranslationsWebIntegrationsProxyZh proxy = _TranslationsWebIntegrationsProxyZh._(_root);
}

// Path: web.plugins
class _TranslationsWebPluginsZh extends TranslationsWebPluginsEn {
	_TranslationsWebPluginsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '检查器插件';
	@override String get subtitle => '配置在会话打开时右侧检查器面板呈现的数据源。每个插件都是管理员级别且在所有会话间共享。点击章节标题可折叠。';
	@override late final _TranslationsWebPluginsCommonZh common = _TranslationsWebPluginsCommonZh._(_root);
	@override late final _TranslationsWebPluginsMcpZh mcp = _TranslationsWebPluginsMcpZh._(_root);
	@override late final _TranslationsWebPluginsMcpSecretsZh mcpSecrets = _TranslationsWebPluginsMcpSecretsZh._(_root);
	@override late final _TranslationsWebPluginsSkillsZh skills = _TranslationsWebPluginsSkillsZh._(_root);
	@override late final _TranslationsWebPluginsCustomTasksZh customTasks = _TranslationsWebPluginsCustomTasksZh._(_root);
	@override late final _TranslationsWebPluginsGitHostsZh gitHosts = _TranslationsWebPluginsGitHostsZh._(_root);
}

// Path: web.backups
class _TranslationsWebBackupsZh extends TranslationsWebBackupsEn {
	_TranslationsWebBackupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份';
	@override String get subtitle => '写入到可插拔目标的加密 PostgreSQL dump。配置计划与保留策略，或触发一次性备份作为快速安全网。';
	@override String get exportData => '导出数据';
	@override String get loading => '加载中…';
	@override String get loadStatusFailedToast => '加载备份状态失败';
	@override late final _TranslationsWebBackupsTabsZh tabs = _TranslationsWebBackupsTabsZh._(_root);
	@override late final _TranslationsWebBackupsInventoryZh inventory = _TranslationsWebBackupsInventoryZh._(_root);
	@override late final _TranslationsWebBackupsRestartZh restart = _TranslationsWebBackupsRestartZh._(_root);
	@override late final _TranslationsWebBackupsSetupZh setup = _TranslationsWebBackupsSetupZh._(_root);
	@override late final _TranslationsWebBackupsGeneratedZh generated = _TranslationsWebBackupsGeneratedZh._(_root);
	@override late final _TranslationsWebBackupsStatusZh status = _TranslationsWebBackupsStatusZh._(_root);
	@override late final _TranslationsWebBackupsBackupsTabZh backupsTab = _TranslationsWebBackupsBackupsTabZh._(_root);
	@override late final _TranslationsWebBackupsRestoreZh restore = _TranslationsWebBackupsRestoreZh._(_root);
	@override late final _TranslationsWebBackupsSchedulesTabZh schedulesTab = _TranslationsWebBackupsSchedulesTabZh._(_root);
	@override late final _TranslationsWebBackupsNewScheduleZh newSchedule = _TranslationsWebBackupsNewScheduleZh._(_root);
	@override late final _TranslationsWebBackupsTargetsTabZh targetsTab = _TranslationsWebBackupsTargetsTabZh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorZh targetEditor = _TranslationsWebBackupsTargetEditorZh._(_root);
}

// Path: web.serverSettings
class _TranslationsWebServerSettingsZh extends TranslationsWebServerSettingsEn {
	_TranslationsWebServerSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsSectionsZh sections = _TranslationsWebServerSettingsSectionsZh._(_root);
	@override String get loading => '正在加载服务器设置…';
	@override String loadFailed({required Object message}) => '加载失败：${message}';
	@override String get noConfigFlag => 'opendray 启动时未指定 -config，设置仅从环境变量加载，无法在此编辑。';
	@override String get resetButton => '重置';
	@override String get resetButtonTitle => '丢弃此分区中未保存的修改';
	@override String resetConfirm({required Object section}) => '将"${section}"重置为上次保存的值？';
	@override String get badgeRestartRequired => '需要重启';
	@override String get badgeUnsaved => '未保存';
	@override String get saveButton => '保存修改';
	@override String get saveToastTitle => '设置已保存';
	@override String get saveToastDesc => '点击「重启」以应用。';
	@override String get saveErrorTitle => '保存失败';
	@override String get dangerousConfirm => '您更改了监听地址 / 管理员账号 / 管理员密码。重启后可能需要重新登录或使用新地址。是否继续？';
	@override String get unsavedHint => '有未保存的修改';
	@override String get savedHint => '所有修改已保存';
	@override String get searchPlaceholder => '筛选字段…';
	@override late final _TranslationsWebServerSettingsRestartZh restart = _TranslationsWebServerSettingsRestartZh._(_root);
	@override late final _TranslationsWebServerSettingsFormGroupsZh formGroups = _TranslationsWebServerSettingsFormGroupsZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsZh fields = _TranslationsWebServerSettingsFieldsZh._(_root);
	@override late final _TranslationsWebServerSettingsLiveTailZh liveTail = _TranslationsWebServerSettingsLiveTailZh._(_root);
	@override late final _TranslationsWebServerSettingsMemoryInspectorCardZh memoryInspectorCard = _TranslationsWebServerSettingsMemoryInspectorCardZh._(_root);
	@override String get localOnnxBanner => '需要使用 <1>-tags local_onnx</1> 编译二进制。标准构建在选择此后端时会返回明确的 stub 错误。设置步骤参见 <3>记忆 → 本地 ONNX</3> 教程。';
	@override late final _TranslationsWebServerSettingsStringListZh stringList = _TranslationsWebServerSettingsStringListZh._(_root);
	@override late final _TranslationsWebServerSettingsHttpHelpersZh httpHelpers = _TranslationsWebServerSettingsHttpHelpersZh._(_root);
	@override late final _TranslationsWebServerSettingsProbeZh probe = _TranslationsWebServerSettingsProbeZh._(_root);
	@override late final _TranslationsWebServerSettingsBackupZh backup = _TranslationsWebServerSettingsBackupZh._(_root);
	@override late final _TranslationsWebServerSettingsTargetRowZh targetRow = _TranslationsWebServerSettingsTargetRowZh._(_root);
}

// Path: web.settings
class _TranslationsWebSettingsZh extends TranslationsWebSettingsEn {
	_TranslationsWebSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置';
	@override String get subtitle => '工作区、账号与网关配置。';
	@override late final _TranslationsWebSettingsGroupsZh groups = _TranslationsWebSettingsGroupsZh._(_root);
	@override late final _TranslationsWebSettingsItemsZh items = _TranslationsWebSettingsItemsZh._(_root);
	@override late final _TranslationsWebSettingsHealthZh health = _TranslationsWebSettingsHealthZh._(_root);
	@override late final _TranslationsWebSettingsBreadcrumbZh breadcrumb = _TranslationsWebSettingsBreadcrumbZh._(_root);
	@override late final _TranslationsWebSettingsAppearanceZh appearance = _TranslationsWebSettingsAppearanceZh._(_root);
	@override late final _TranslationsWebSettingsFontZh font = _TranslationsWebSettingsFontZh._(_root);
	@override late final _TranslationsWebSettingsAccountZh account = _TranslationsWebSettingsAccountZh._(_root);
	@override late final _TranslationsWebSettingsChangeCredentialsZh changeCredentials = _TranslationsWebSettingsChangeCredentialsZh._(_root);
	@override late final _TranslationsWebSettingsSystemZh system = _TranslationsWebSettingsSystemZh._(_root);
	@override late final _TranslationsWebSettingsAboutZh about = _TranslationsWebSettingsAboutZh._(_root);
}

// Path: web.logViewer
class _TranslationsWebLogViewerZh extends TranslationsWebLogViewerEn {
	_TranslationsWebLogViewerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get filterPlaceholder => '过滤…';
	@override String get debugTooltip => 'Debug 计数';
	@override String get infoTooltip => 'Info 计数';
	@override String get warnTooltip => 'Warn 计数';
	@override String get errorTooltip => 'Error 计数';
	@override String get streaming => '正在流式传输';
	@override String get disconnected => '已断开';
	@override String get live => '实时';
	@override String get offline => '离线';
	@override String get pauseTooltip => '暂停自动滚动';
	@override String get resumeTooltip => '恢复自动滚动';
	@override String get clearTooltip => '清空本地视图（服务端 ring 不受影响）';
	@override String get downloadTooltip => '下载完整 ring 为 .log 文件';
	@override String get emptyWaiting => '等待日志记录…';
	@override String emptyFiltered({required Object query}) => '没有匹配 "${query}" 的记录';
}

// Path: web.pathInput
class _TranslationsWebPathInputZh extends TranslationsWebPathInputEn {
	_TranslationsWebPathInputZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get testButton => '测试';
	@override String get testTooltip => '解析并检查此路径';
	@override String get notFound => '未找到 ·';
	@override String get childrenSuffix => '项';
	@override String get expectedDirectory => '· 期望是目录';
}

// Path: web.memoryAmbient
class _TranslationsWebMemoryAmbientZh extends TranslationsWebMemoryAmbientEn {
	_TranslationsWebMemoryAmbientZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryAmbientHeaderZh header = _TranslationsWebMemoryAmbientHeaderZh._(_root);
	@override String get loading => '加载中…';
	@override late final _TranslationsWebMemoryAmbientProvidersZh providers = _TranslationsWebMemoryAmbientProvidersZh._(_root);
	@override late final _TranslationsWebMemoryAmbientRulesZh rules = _TranslationsWebMemoryAmbientRulesZh._(_root);
	@override late final _TranslationsWebMemoryAmbientProfilesZh profiles = _TranslationsWebMemoryAmbientProfilesZh._(_root);
	@override late final _TranslationsWebMemoryAmbientCostZh cost = _TranslationsWebMemoryAmbientCostZh._(_root);
}

// Path: web.noteEditor
class _TranslationsWebNoteEditorZh extends TranslationsWebNoteEditorEn {
	_TranslationsWebNoteEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get source => '源码';
	@override String get preview => '预览';
	@override String tagTitle({required Object tag}) => '标签 #${tag}';
	@override String get emptyNote => '空白笔记。切换到源码标签开始书写。';
	@override String get saveFailedToast => '保存失败';
	@override late final _TranslationsWebNoteEditorStatusZh status = _TranslationsWebNoteEditorStatusZh._(_root);
}

// Path: web.export
class _TranslationsWebExportZh extends TranslationsWebExportEn {
	_TranslationsWebExportZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '导出数据';
	@override String get subtitle => '把选中的逻辑实体打成一份一次性 zip 包。服务器上保留 24 小时后自动回收。';
	@override String get backToBackups => '← 备份';
	@override late final _TranslationsWebExportSectionsZh sections = _TranslationsWebExportSectionsZh._(_root);
	@override late final _TranslationsWebExportFormZh form = _TranslationsWebExportFormZh._(_root);
	@override late final _TranslationsWebExportHistoryZh history = _TranslationsWebExportHistoryZh._(_root);
	@override late final _TranslationsWebExportImportZh import = _TranslationsWebExportImportZh._(_root);
	@override late final _TranslationsWebExportImportsZh imports = _TranslationsWebExportImportsZh._(_root);
}

// Path: more.identity
class _TranslationsMoreIdentityZh extends TranslationsMoreIdentityEn {
	_TranslationsMoreIdentityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get signedInAs => '登录账号';
	@override String get server => '服务器';
	@override String get tokenExpires => '令牌到期';
}

// Path: more.sections
class _TranslationsMoreSectionsZh extends TranslationsMoreSectionsEn {
	_TranslationsMoreSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get gateway => '网关';
	@override String get memory => '记忆';
	@override String get system => '系统';
}

// Path: more.items
class _TranslationsMoreItemsZh extends TranslationsMoreItemsEn {
	_TranslationsMoreItemsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMoreItemsIntegrationsZh integrations = _TranslationsMoreItemsIntegrationsZh._(_root);
	@override late final _TranslationsMoreItemsChannelsZh channels = _TranslationsMoreItemsChannelsZh._(_root);
	@override late final _TranslationsMoreItemsProvidersZh providers = _TranslationsMoreItemsProvidersZh._(_root);
	@override late final _TranslationsMoreItemsMcpZh mcp = _TranslationsMoreItemsMcpZh._(_root);
	@override late final _TranslationsMoreItemsSkillsZh skills = _TranslationsMoreItemsSkillsZh._(_root);
	@override late final _TranslationsMoreItemsGitHostsZh gitHosts = _TranslationsMoreItemsGitHostsZh._(_root);
	@override late final _TranslationsMoreItemsCustomTasksZh customTasks = _TranslationsMoreItemsCustomTasksZh._(_root);
	@override late final _TranslationsMoreItemsProjectMemoryZh projectMemory = _TranslationsMoreItemsProjectMemoryZh._(_root);
	@override late final _TranslationsMoreItemsCleanupInboxZh cleanupInbox = _TranslationsMoreItemsCleanupInboxZh._(_root);
	@override late final _TranslationsMoreItemsBackupsZh backups = _TranslationsMoreItemsBackupsZh._(_root);
	@override late final _TranslationsMoreItemsSettingsZh settings = _TranslationsMoreItemsSettingsZh._(_root);
	@override late final _TranslationsMoreItemsAboutZh about = _TranslationsMoreItemsAboutZh._(_root);
}

// Path: sessions.filters
class _TranslationsSessionsFiltersZh extends TranslationsSessionsFiltersEn {
	_TranslationsSessionsFiltersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get all => '全部';
	@override String get running => '运行中';
	@override String get idle => '空闲';
	@override String get ended => '已结束';
}

// Path: sessions.card
class _TranslationsSessionsCardZh extends TranslationsSessionsCardEn {
	_TranslationsSessionsCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String startedRelative({required Object provider, required Object when}) => '${provider} · ${when}启动';
}

// Path: sessions.empty
class _TranslationsSessionsEmptyZh extends TranslationsSessionsEmptyEn {
	_TranslationsSessionsEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get titleAll => '暂无会话';
	@override String titleFiltered({required Object filter}) => '没有匹配「${filter}」筛选的会话。';
	@override String get subtitleAll => '点击「创建」按钮新建一个。';
	@override String get subtitleFiltered => '试试其他筛选条件或下拉刷新。';
}

// Path: sessions.relative
class _TranslationsSessionsRelativeZh extends TranslationsSessionsRelativeEn {
	_TranslationsSessionsRelativeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String secondsAgo({required Object n}) => '${n} 秒前';
	@override String minutesAgo({required Object n}) => '${n} 分钟前';
	@override String hoursAgo({required Object n}) => '${n} 小时前';
	@override String daysAgo({required Object n}) => '${n} 天前';
}

// Path: sessions.detail
class _TranslationsSessionsDetailZh extends TranslationsSessionsDetailEn {
	_TranslationsSessionsDetailZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get fallbackTitle => '会话';
	@override String get refreshMetadata => '刷新元数据';
	@override String get inspector => '检查器（文件 / Git / 任务 / 历史 / 笔记）';
	@override String get projectMemory => '项目记忆（目标 / 计划 / 日志 / 收件箱）';
	@override String get actions => '操作';
	@override String started({required Object when}) => '${when} 启动';
	@override String startedEnded({required Object started, required Object ended}) => '${started} 启动  ·  ${ended} 结束';
	@override String idPrefix({required Object id}) => 'id: ${id}';
	@override String get errorTitle => '加载会话失败';
}

// Path: sessions.terminal
class _TranslationsSessionsTerminalZh extends TranslationsSessionsTerminalEn {
	_TranslationsSessionsTerminalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsSessionsTerminalSnackbarZh snackbar = _TranslationsSessionsTerminalSnackbarZh._(_root);
	@override late final _TranslationsSessionsTerminalImageSourceZh imageSource = _TranslationsSessionsTerminalImageSourceZh._(_root);
	@override late final _TranslationsSessionsTerminalKeyboardZh keyboard = _TranslationsSessionsTerminalKeyboardZh._(_root);
	@override late final _TranslationsSessionsTerminalConnectionZh connection = _TranslationsSessionsTerminalConnectionZh._(_root);
}

// Path: sessions.action
class _TranslationsSessionsActionZh extends TranslationsSessionsActionEn {
	_TranslationsSessionsActionZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get stop => '停止';
	@override String get stopping => '停止中…';
	@override String get stopDescription => '发送 SIGTERM，保留历史';
	@override String get restart => '重启';
	@override String get restarting => '重启中…';
	@override String get restartDescription => '重新启动 CLI 进程';
	@override String get delete => '删除';
	@override String get deleteDescription => '移除会话及其历史';
	@override String get deleteConfirm => '确定永久删除此会话吗？其环形缓冲区和历史将全部丢失。';
	@override late final _TranslationsSessionsActionErrorsZh errors = _TranslationsSessionsActionErrorsZh._(_root);
}

// Path: sessions.dirPicker
class _TranslationsSessionsDirPickerZh extends TranslationsSessionsDirPickerEn {
	_TranslationsSessionsDirPickerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get parent => '上级';
	@override String get newFolder => '新建文件夹';
	@override String get useThisFolder => '使用此文件夹';
	@override String get loading => '加载中…';
	@override String get empty => '此处没有子文件夹。\n选择此文件夹，或新建一个。';
	@override String createdSnack({required Object path}) => '已创建 ${path}';
	@override String mkdirFailedSnack({required Object error}) => '创建文件夹失败：${error}';
	@override late final _TranslationsSessionsDirPickerDialogZh dialog = _TranslationsSessionsDirPickerDialogZh._(_root);
}

// Path: sessions.inspector
class _TranslationsSessionsInspectorZh extends TranslationsSessionsInspectorEn {
	_TranslationsSessionsInspectorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsSessionsInspectorShellZh shell = _TranslationsSessionsInspectorShellZh._(_root);
	@override late final _TranslationsSessionsInspectorSharedZh shared = _TranslationsSessionsInspectorSharedZh._(_root);
	@override late final _TranslationsSessionsInspectorHistoryZh history = _TranslationsSessionsInspectorHistoryZh._(_root);
	@override late final _TranslationsSessionsInspectorFilesZh files = _TranslationsSessionsInspectorFilesZh._(_root);
	@override late final _TranslationsSessionsInspectorGitZh git = _TranslationsSessionsInspectorGitZh._(_root);
	@override late final _TranslationsSessionsInspectorTasksZh tasks = _TranslationsSessionsInspectorTasksZh._(_root);
	@override late final _TranslationsSessionsInspectorNotesZh notes = _TranslationsSessionsInspectorNotesZh._(_root);
}

// Path: sessions.spawnSheet
class _TranslationsSessionsSpawnSheetZh extends TranslationsSessionsSpawnSheetEn {
	_TranslationsSessionsSpawnSheetZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建会话';
	@override String get errorRequired => '需要指定提供商和工作目录';
	@override String errorGeneric({required Object error}) => '创建会话失败：${error}';
	@override String get cancel => '取消';
	@override String get spawn => '创建';
	@override String get providerLabel => '提供商';
	@override String get disabledSuffix => '（已停用）';
	@override String get cwdLabel => '工作目录';
	@override String get cwdHint => '/Users/you/projects/foo';
	@override String get cwdHelper => '网关主机上的绝对路径。';
	@override String get browse => '浏览';
	@override String get nameLabel => '名称（可选）';
	@override String get nameHint => '例如：backend-refactor';
	@override String get argsLabel => '额外参数（可选）';
	@override String get argsHint => '--continue --verbose';
	@override String get argsHelper => '以空格分隔；留空使用提供商默认值。';
	@override late final _TranslationsSessionsSpawnSheetBypassZh bypass = _TranslationsSessionsSpawnSheetBypassZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetNoProvidersZh noProviders = _TranslationsSessionsSpawnSheetNoProvidersZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetProviderLoadErrorZh providerLoadError = _TranslationsSessionsSpawnSheetProviderLoadErrorZh._(_root);
	@override late final _TranslationsSessionsSpawnSheetClaudeAccountZh claudeAccount = _TranslationsSessionsSpawnSheetClaudeAccountZh._(_root);
}

// Path: mcp.errorPrefix
class _TranslationsMcpErrorPrefixZh extends TranslationsMcpErrorPrefixEn {
	_TranslationsMcpErrorPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get delete => '删除失败';
	@override String get add => '添加失败';
	@override String get update => '更新失败';
	@override String get toggle => '切换失败';
}

// Path: mcp.editor
class _TranslationsMcpEditorZh extends TranslationsMcpEditorEn {
	_TranslationsMcpEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get nameHint => 'my-mcp-server';
	@override String get jsonHint => 'JSON 配置 — name、transport: stdio、command、args…';
	@override String get descriptionPlaceholder => '可选的一行说明';
	@override String get validateJsonObject => '正文必须是 JSON 对象';
	@override String validateJsonInvalid({required Object error}) => '无效的 JSON：${error}';
	@override String get appBarEdit => '编辑 MCP 服务器';
	@override String get appBarNew => '新建 MCP 服务器';
	@override String get idLockedHint => '编辑模式下锁定 — 需删除后重建以更改。';
	@override String get jsonLabel => '服务器 JSON';
	@override String get jsonSchemaHelp => 'Schema：transport 必须是 stdio、http 或 sse。stdio 需要 command + args。http/sse 需要 url + headers。用 \$secret:KEY 引用密钥库的密钥。';
	@override String get idLabel => 'id（URL 片段，小写字母数字 / 横线 / 下划线）';
	@override String get idRequired => 'id 必填';
	@override String get saving => '保存中…';
	@override String get save => '保存';
	@override String get create => '创建';
}

// Path: mcp.secret
class _TranslationsMcpSecretZh extends TranslationsMcpSecretEn {
	_TranslationsMcpSecretZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get keyLabel => '键';
	@override String get keyHint => 'GITHUB_TOKEN、OPENAI_KEY、…';
	@override String get valueLabel => '值';
	@override String get keyRequired => '必须填写键。';
	@override String get keyInvalid => '键必须匹配 [A-Za-z_][A-Za-z0-9_]* — 与 shell 环境变量规则相同。';
	@override String get valueRequired => '必须填写值。';
	@override String get replaceTitle => '替换密钥值';
	@override String get addTitle => '添加密钥';
	@override String get saveButton => '保存';
	@override String get addButton => '添加';
	@override String get helpRules => 'shell 环境变量规则：字母或 _ 开头，仅含字母 / 数字 / _。';
	@override String get replaceHint => '粘贴新值（旧值被擦除）';
	@override String get addHint => '粘贴密钥值';
	@override String addedSnack({required Object key}) => '已添加密钥 ${key}。';
	@override String updatedSnack({required Object key}) => '已更新密钥 ${key}。';
	@override String deletedSnack({required Object key}) => '已删除 ${key}。';
	@override String get deleteBody => '从加密的密钥库中移除该值。引用此密钥的 MCP 服务器在恢复前将无法启动。';
}

// Path: mcp.popup
class _TranslationsMcpPopupZh extends TranslationsMcpPopupEn {
	_TranslationsMcpPopupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get editConfigSubtitle => '完整 JSON 编辑器 — 仅限密钥库支持的服务器';
	@override String get viewRawSubtitle => '服务器 JSON 的只读查看器';
	@override String get deleteLabel => '删除';
}

// Path: mcp.kv
class _TranslationsMcpKvZh extends TranslationsMcpKvEn {
	_TranslationsMcpKvZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get transport => '传输';
	@override String get description => '描述';
	@override String get command => '命令';
	@override String get args => '参数';
	@override String get headers => 'Headers';
}

// Path: providers.errorPrefix
class _TranslationsProvidersErrorPrefixZh extends TranslationsProvidersErrorPrefixEn {
	_TranslationsProvidersErrorPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get toggle => '切换失败';
	@override String get rename => '重命名失败';
	@override String get delete => '删除失败';
}

// Path: providers.accounts
class _TranslationsProvidersAccountsZh extends TranslationsProvidersAccountsEn {
	_TranslationsProvidersAccountsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get rename => '重命名';
	@override String renameTitle({required Object name}) => '重命名 ${name}';
	@override String get displayNameLabel => '显示名';
	@override String get displayNameHint => '工作账号';
	@override String get deleteTitle => '删除账号？';
	@override String importFailedApi({required Object error}) => '导入失败：${error}';
	@override String importFailedGeneric({required Object error}) => '导入失败：${error}';
	@override String get enable => '启用';
	@override String get disable => '停用';
	@override String get deleteLabel => '删除';
	@override String get deleteBody => '移除该账号及其存储的 OAuth token。已使用此账号的会话保持运行，但重新认证会失败。';
	@override String deletedSnack({required Object name}) => '已删除 ${name}。';
	@override String get importSyncedSnack => '已同步 — 网关没有新账号。';
	@override String importedSnackOne({required Object n}) => '已导入 ${n} 个账号。';
	@override String importedSnackOther({required Object n}) => '已导入 ${n} 个账号。';
	@override String get importing => '同步中…';
	@override String get importLocal => '导入本地';
	@override String get addHint => '添加新账号仅可在网关主机上操作。';
	@override String get addBody => '新目录会自动出现在这里。OAuth 流程步骤参见文档。';
	@override String loadFailed({required Object error}) => '加载账号失败：${error}';
	@override String get intro => '以 Claude 提供商启动的会话会从这些账号中选择（或回退到环境变量）。';
	@override String enabledSnack({required Object name}) => '${name} 已启用。';
	@override String disabledSnack({required Object name}) => '${name} 已停用。';
	@override String renamedSnack({required Object name}) => '已重命名为 ${name}。';
}

// Path: integrations.form
class _TranslationsIntegrationsFormZh extends TranslationsIntegrationsFormEn {
	_TranslationsIntegrationsFormZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get validateRequired => '名称、Base URL、路由前缀必填。';
	@override String get fieldName => '名称';
	@override String get fieldNameHint => 'My Bot';
	@override String get fieldBaseUrl => 'Base URL';
	@override String get fieldRoutePrefix => '路由前缀';
	@override String get routePrefixHelper => '可通过 /api/v1/<前缀>/... 访问';
	@override String get fieldScopes => '范围（可选）';
	@override String get scopesHelper => '逗号分隔。留空 = 服务器默认。';
	@override String get fieldVersion => '版本（可选）';
	@override String get validateBaseUrl => '必须填写 Base URL。';
	@override String get editFieldScopes => '范围';
	@override String get editScopesHelper => '逗号分隔。';
	@override String get editFieldVersion => '版本';
	@override String get apiKeyWarn => '此 key 只显示这一次。';
	@override String get copyCopied => '已复制';
	@override String get copyCopy => '复制';
}

// Path: memoryWorkers.tasks
class _TranslationsMemoryWorkersTasksZh extends TranslationsMemoryWorkersTasksEn {
	_TranslationsMemoryWorkersTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsMemoryWorkersTasksGatekeeperZh gatekeeper = _TranslationsMemoryWorkersTasksGatekeeperZh._(_root);
	@override late final _TranslationsMemoryWorkersTasksCleanerZh cleaner = _TranslationsMemoryWorkersTasksCleanerZh._(_root);
	@override late final _TranslationsMemoryWorkersTasksGitactivityZh gitactivity = _TranslationsMemoryWorkersTasksGitactivityZh._(_root);
	@override late final _TranslationsMemoryWorkersTasksTranscriptZh transcript = _TranslationsMemoryWorkersTasksTranscriptZh._(_root);
}

// Path: backups.kv
class _TranslationsBackupsKvZh extends TranslationsBackupsKvEn {
	_TranslationsBackupsKvZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get status => '状态';
	@override String get target => '目标';
	@override String get triggeredBy => '触发者';
	@override String get started => '开始';
	@override String get finished => '完成';
	@override String get size => '大小';
	@override String get encrypted => '已加密';
	@override String get targetPath => '目标路径';
	@override String get error => '错误';
	@override String get yes => '是';
	@override String get no => '否';
}

// Path: backups.emptyMissingDeps
class _TranslationsBackupsEmptyMissingDepsZh extends TranslationsBackupsEmptyMissingDepsEn {
	_TranslationsBackupsEmptyMissingDepsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get headline => '备份暂时无法运行';
	@override String get body => '安装 postgresql-client 并重启 opendray。';
}

// Path: backups.emptyNoTargets
class _TranslationsBackupsEmptyNoTargetsZh extends TranslationsBackupsEmptyNoTargetsEn {
	_TranslationsBackupsEmptyNoTargetsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get headline => '未配置任何备份目标';
	@override String get body => '打开「更多」菜单 → 目标，添加一个目的地（本地 / S3 / SMB / SFTP / WebDAV / rclone）。然后返回并点击「立即运行」。';
}

// Path: backups.emptyNoBackups
class _TranslationsBackupsEmptyNoBackupsZh extends TranslationsBackupsEmptyNoBackupsEn {
	_TranslationsBackupsEmptyNoBackupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get headline => '暂无备份';
	@override String get body => '点击「立即运行」生成一次新快照，或打开「计划」设置定期运行。';
}

// Path: backups.wizard
class _TranslationsBackupsWizardZh extends TranslationsBackupsWizardEn {
	_TranslationsBackupsWizardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置备份';
	@override String get intro => '选择一个主密语。opendray 用它通过 AES-256-GCM 加密每一份备份。丢失密语就丢失数据 — 无法恢复。';
	@override String get saving => '保存中…';
	@override String get generateAndSave => '生成并保存';
	@override String get savePassphrase => '保存密语';
	@override String get generateHint => '服务器生成密码学级别随机密语，你复制到密码管理器，然后确认。';
	@override String get helperRecommended => '建议：从密码管理器生成 40+ 字符';
	@override String get saveNowHeader => '立即保存这个密语';
	@override String get saveNowBody => '此处只显示一次。之后无法从 opendray 取回。';
}

// Path: backups.encryption
class _TranslationsBackupsEncryptionZh extends TranslationsBackupsEncryptionEn {
	_TranslationsBackupsEncryptionZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get checkAgain => '重新检查';
	@override String get generate => '生成';
	@override String get paste => '粘贴';
	@override String get random256bit => '256 位随机密钥';
	@override String get passphraseLabel => '你的密语';
	@override String get passphraseHint => '至少 20 个字符';
	@override String get passphraseCopied => '密语已复制到剪贴板';
}

// Path: backupTargetEditor.kinds
class _TranslationsBackupTargetEditorKindsZh extends TranslationsBackupTargetEditorKindsEn {
	_TranslationsBackupTargetEditorKindsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsBackupTargetEditorKindsLocalZh local = _TranslationsBackupTargetEditorKindsLocalZh._(_root);
	@override late final _TranslationsBackupTargetEditorKindsSmbZh smb = _TranslationsBackupTargetEditorKindsSmbZh._(_root);
	@override late final _TranslationsBackupTargetEditorKindsWebdavZh webdav = _TranslationsBackupTargetEditorKindsWebdavZh._(_root);
	@override late final _TranslationsBackupTargetEditorKindsSftpZh sftp = _TranslationsBackupTargetEditorKindsSftpZh._(_root);
	@override late final _TranslationsBackupTargetEditorKindsS3Zh s3 = _TranslationsBackupTargetEditorKindsS3Zh._(_root);
	@override late final _TranslationsBackupTargetEditorKindsRcloneZh rclone = _TranslationsBackupTargetEditorKindsRcloneZh._(_root);
}

// Path: githosts.errorPrefix
class _TranslationsGithostsErrorPrefixZh extends TranslationsGithostsErrorPrefixEn {
	_TranslationsGithostsErrorPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get toggle => '切换失败';
	@override String get delete => '删除失败';
}

// Path: githosts.form
class _TranslationsGithostsFormZh extends TranslationsGithostsFormEn {
	_TranslationsGithostsFormZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get kindLabel => '类型';
	@override String get hostLabel => '主机';
	@override String get nameLabel => '名称';
	@override String get nameHint => 'work-github、personal-gitlab、…';
	@override late final _TranslationsGithostsFormKindsZh kinds = _TranslationsGithostsFormKindsZh._(_root);
	@override String get validateHost => '必须填写主机。';
	@override String get validateName => '必须填写名称。';
	@override String get snackAdded => '主机已添加。';
	@override String get snackUpdated => '主机已更新。';
	@override String saveFailedApi({required Object error}) => '保存失败：${error}';
	@override String saveFailedGeneric({required Object error}) => '保存失败：${error}';
	@override String get saving => '保存中…';
	@override String get save => '保存';
	@override String get add => '添加';
	@override String get nameHelper => '在 PR 列表中显示的名字。';
	@override String get tokenLabelKeep => 'Token（留空 = 保留现有）';
	@override String get tokenLabel => 'Token';
	@override String get tokenHintKeep => '留空 = 保留现有。';
	@override String get tokenHintNew => '粘贴个人访问令牌。';
	@override String get enabledHelper => '可供会话用于 PR / 远端查找。';
	@override String get validateTokenRequired => '添加主机时必须填写 Token。';
	@override String appBarEdit({required Object name}) => '编辑 ${name}';
	@override String get appBarNew => '添加 Git 主机';
	@override String tokenPreviewHint({required Object preview}) => '当前预览：${preview}';
	@override String get tokenPreviewNone => '（无）';
	@override String get pausedSubtitle => '已暂停 — 会话跳过此主机。';
}

// Path: channels.configDialog
class _TranslationsChannelsConfigDialogZh extends TranslationsChannelsConfigDialogEn {
	_TranslationsChannelsConfigDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object kind}) => '${kind} 配置';
}

// Path: channels.webhookDialog
class _TranslationsChannelsWebhookDialogZh extends TranslationsChannelsWebhookDialogEn {
	_TranslationsChannelsWebhookDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object kind}) => '${kind} Webhook URL';
	@override String get copiedSnack => '已复制 Webhook URL。';
}

// Path: channels.notifications
class _TranslationsChannelsNotificationsZh extends TranslationsChannelsNotificationsEn {
	_TranslationsChannelsNotificationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '通知偏好';
	@override String get notifyOn => '通知时机';
	@override String get repeatPolicy => '重复策略';
	@override String get cooldownWindow => '冷却时间';
	@override String get includeSnippet => '包含终端片段';
	@override String get snippetLengthCap => '片段长度上限';
	@override String get notifyOnAll => '所有会话事件。';
	@override String get notifyOnEmpty => '未选择事件 — 已静音外发通知。';
	@override String get snippetHelper => '在每条通知中嵌入终端最近的内容。';
	@override String get snippetNoCap => '无上限';
	@override String snippetChars({required Object n}) => '${n} 字符';
	@override String get updatedSnack => '通知偏好已更新。';
	@override late final _TranslationsChannelsNotificationsModesZh modes = _TranslationsChannelsNotificationsModesZh._(_root);
}

// Path: channels.popup
class _TranslationsChannelsPopupZh extends TranslationsChannelsPopupEn {
	_TranslationsChannelsPopupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get enable => '启用';
	@override String get disable => '停用';
	@override String get mute => '静音';
	@override String get unmute => '取消静音';
	@override String get deleteLabel => '删除';
}

// Path: channels.badges
class _TranslationsChannelsBadgesZh extends TranslationsChannelsBadgesEn {
	_TranslationsChannelsBadgesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get running => '运行中';
	@override String get starting => '启动中…';
	@override String get disabled => '已停用';
	@override String get muted => '已静音';
}

// Path: channels.snacks
class _TranslationsChannelsSnacksZh extends TranslationsChannelsSnacksEn {
	_TranslationsChannelsSnacksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get testDispatched => '测试消息已发送。';
	@override String get channelEnabled => '通道已启用。';
	@override String get channelDisabled => '通道已停用。';
	@override String get channelMuted => '通道已静音。';
	@override String get channelUnmuted => '通道已取消静音。';
	@override String get configUpdated => '通道配置已更新。';
	@override String get channelDeleted => '通道已删除。';
}

// Path: channels.errorPrefix
class _TranslationsChannelsErrorPrefixZh extends TranslationsChannelsErrorPrefixEn {
	_TranslationsChannelsErrorPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get test => '测试失败';
	@override String get toggle => '切换失败';
	@override String get muteToggle => '静音切换失败';
	@override String get update => '更新失败';
	@override String get delete => '删除失败';
}

// Path: channels.kinds
class _TranslationsChannelsKindsZh extends TranslationsChannelsKindsEn {
	_TranslationsChannelsKindsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsChannelsKindsTelegramZh telegram = _TranslationsChannelsKindsTelegramZh._(_root);
	@override late final _TranslationsChannelsKindsSlackZh slack = _TranslationsChannelsKindsSlackZh._(_root);
	@override late final _TranslationsChannelsKindsDiscordZh discord = _TranslationsChannelsKindsDiscordZh._(_root);
	@override late final _TranslationsChannelsKindsFeishuZh feishu = _TranslationsChannelsKindsFeishuZh._(_root);
	@override late final _TranslationsChannelsKindsDingtalkZh dingtalk = _TranslationsChannelsKindsDingtalkZh._(_root);
	@override late final _TranslationsChannelsKindsWecomZh wecom = _TranslationsChannelsKindsWecomZh._(_root);
	@override late final _TranslationsChannelsKindsWechatZh wechat = _TranslationsChannelsKindsWechatZh._(_root);
}

// Path: notesPage.editor
class _TranslationsNotesPageEditorZh extends TranslationsNotesPageEditorEn {
	_TranslationsNotesPageEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get markdownHint => 'Markdown…';
	@override String get saving => '保存中…';
	@override String get autosave => '随输入自动保存';
	@override String loadFailedApi({required Object error}) => '加载失败：${error}';
	@override String loadFailedGeneric({required Object error}) => '加载失败：${error}';
	@override String saveFailedApi({required Object error}) => '保存失败：${error}';
	@override String saveFailedGeneric({required Object error}) => '保存失败：${error}';
	@override String savedAt({required Object time}) => '${time} 已保存';
}

// Path: memory.deleteAllConfirm
class _TranslationsMemoryDeleteAllConfirmZh extends TranslationsMemoryDeleteAllConfirmEn {
	_TranslationsMemoryDeleteAllConfirmZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '删除此范围内所有记忆？';
	@override String get deleteAll => '全部删除';
}

// Path: memory.deleteOne
class _TranslationsMemoryDeleteOneZh extends TranslationsMemoryDeleteOneEn {
	_TranslationsMemoryDeleteOneZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '删除该记忆？';
	@override String get body => '此操作不可撤销。';
}

// Path: memory.scope
class _TranslationsMemoryScopeZh extends TranslationsMemoryScopeEn {
	_TranslationsMemoryScopeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get project => '项目';
	@override String get global => '全局';
}

// Path: memory.create
class _TranslationsMemoryCreateZh extends TranslationsMemoryCreateEn {
	_TranslationsMemoryCreateZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get textLabel => '文本';
	@override String get scopeKeyLabel => '范围键（项目 cwd）';
	@override String get scopeKeyHint => '/Users/you/projects/foo';
	@override String get submit => '创建';
}

// Path: about.sections
class _TranslationsAboutSectionsZh extends TranslationsAboutSectionsEn {
	_TranslationsAboutSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get app => '应用';
	@override String get server => '服务器';
}

// Path: about.fields
class _TranslationsAboutFieldsZh extends TranslationsAboutFieldsEn {
	_TranslationsAboutFieldsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get app => '应用';
	@override String get version => '版本';
	@override String versionFormat({required Object version, required Object build}) => '${version}（build ${build}）';
	@override String get package => '包名';
	@override String get url => 'URL';
	@override String get signedInAs => '登录账号';
	@override String get tokenExpires => '令牌到期';
}

// Path: about.copyLabels
class _TranslationsAboutCopyLabelsZh extends TranslationsAboutCopyLabelsEn {
	_TranslationsAboutCopyLabelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get version => '版本';
	@override String get serverUrl => '服务器 URL';
}

// Path: settings.language
class _TranslationsSettingsLanguageZh extends TranslationsSettingsLanguageEn {
	_TranslationsSettingsLanguageZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '语言';
	@override String get system => '跟随系统';
	@override String get systemSubtitle => '跟随手机的语言设置';
	@override String get english => 'English';
	@override String get chinese => '中文';
}

// Path: settings.appearance
class _TranslationsSettingsAppearanceZh extends TranslationsSettingsAppearanceEn {
	_TranslationsSettingsAppearanceZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '外观';
	@override String get system => '跟随系统';
	@override String get systemSubtitle => '跟随手机的外观设置';
	@override String get light => '浅色';
	@override String get lightSubtitle => '始终使用浅色主题';
	@override String get dark => '深色';
	@override String get darkSubtitle => '始终使用深色主题';
}

// Path: settings.account
class _TranslationsSettingsAccountZh extends TranslationsSettingsAccountEn {
	_TranslationsSettingsAccountZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '账户';
	@override String get changeCredentials => '修改凭据';
	@override String get changeCredentialsSubtitle => '用户名和密码';
}

// Path: settings.gateway
class _TranslationsSettingsGatewayZh extends TranslationsSettingsGatewayEn {
	_TranslationsSettingsGatewayZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get section => '网关';
	@override String get serverSettings => '服务器设置';
	@override String get serverSettingsSubtitle => '监听地址、日志、凭据库、内存、存储路径…';
	@override String get liveLogs => '实时日志';
	@override String get liveLogsSubtitle => '查看网关实时日志 — 与 Web 管理端同源';
}

// Path: settings.changeCredentials
class _TranslationsSettingsChangeCredentialsZh extends TranslationsSettingsChangeCredentialsEn {
	_TranslationsSettingsChangeCredentialsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '修改凭据';
	@override String get explanation => '验证当前密码，然后选择新凭据。其他已登录会话将全部失效。';
	@override String get currentPassword => '当前密码';
	@override String get newUsername => '新用户名';
	@override String get newPassword => '新密码';
	@override String get confirmPassword => '确认新密码';
	@override String get validatorRequired => '必填';
	@override String get passwordHelper => '至少 8 个字符';
	@override String get passwordTooShort => '至少需要 8 个字符';
	@override String get passwordMismatch => '与新密码不一致';
	@override String get updatedSnack => '凭据已更新。';
	@override String get wrongCurrent => '当前密码不正确。';
	@override String get saving => '保存中…';
	@override String get update => '更新';
}

// Path: settings.logViewer
class _TranslationsSettingsLogViewerZh extends TranslationsSettingsLogViewerEn {
	_TranslationsSettingsLogViewerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '实时日志';
	@override String get reconnect => '重新连接';
	@override String get copyBuffer => '复制缓冲';
	@override String get clearLocal => '清除本地视图';
	@override String get copiedSnack => '已将缓冲复制到剪贴板';
	@override String get filterHint => '筛选子串…';
	@override late final _TranslationsSettingsLogViewerLevelsZh levels = _TranslationsSettingsLogViewerLevelsZh._(_root);
}

// Path: settings.serverSettings
class _TranslationsSettingsServerSettingsZh extends TranslationsSettingsServerSettingsEn {
	_TranslationsSettingsServerSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '服务器设置';
	@override String get reloadTooltip => '从服务器重新加载';
	@override String get restartTooltip => '重启网关';
	@override String get restartConfirmTitle => '重启 opendray？';
	@override String get restartConfirmBody => '网关将自我 exec。手机应用可能短暂断开连接。';
	@override String get restart => '重启';
	@override String get restartQueuedSnack => '已请求重启。稍后下拉刷新。';
	@override String restartFailedApi({required Object error}) => '重启失败：${error}';
	@override String restartFailedGeneric({required Object error}) => '重启失败：${error}';
	@override String loadedFrom({required Object path}) => '加载自：${path}';
	@override String get restartHint => '大部分配置需要重启网关后生效。重启按钮在 AppBar 中。';
	@override String get savedNeedsRestart => '已保存。重启网关以生效。';
	@override String get savedSimple => '已保存。';
	@override String get changesNeedRestart => '此配置的修改需重启网关。';
	@override String get loadFailed => '加载服务器设置失败';
	@override late final _TranslationsSettingsServerSettingsSectionsZh sections = _TranslationsSettingsServerSettingsSectionsZh._(_root);
	@override late final _TranslationsSettingsServerSettingsSectionDescriptionsZh sectionDescriptions = _TranslationsSettingsServerSettingsSectionDescriptionsZh._(_root);
	@override late final _TranslationsSettingsServerSettingsFieldsZh fields = _TranslationsSettingsServerSettingsFieldsZh._(_root);
	@override String validateInteger({required Object field}) => '「${field}」必须是整数';
	@override String validateNumber({required Object field}) => '「${field}」必须是数字';
}

// Path: web.sessions.list
class _TranslationsWebSessionsListZh extends TranslationsWebSessionsListEn {
	_TranslationsWebSessionsListZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '会话';
	@override String get countSeparator => '·';
	@override String get newAria => '创建新会话';
	@override String get newTooltip => '新建会话';
	@override String get loading => '加载中…';
	@override String get emptyTitle => '暂无会话。';
	@override String emptyHint({required Object kbd}) => '按 ${kbd} 创建。';
	@override String endedHeader({required Object count}) => '已结束 (${count})';
	@override String get clearAll => '清空全部';
	@override String confirmClearAll({required Object count}) => '确定移除全部 ${count} 个已结束的会话?';
	@override String confirmTerminate({required Object name}) => '终止并移除 ${name}?';
	@override String childPromoted({required Object count}) => '其 ${count} 个子任务会话将被提升为顶级。';
	@override String childPromotedPlural({required Object count}) => '其 ${count} 个子任务会话将被提升为顶级。';
	@override String footer({required Object live, required Object ended}) => '${live} 运行中 · ${ended} 已结束';
	@override late final _TranslationsWebSessionsListRowZh row = _TranslationsWebSessionsListRowZh._(_root);
	@override String get deleteFailedToast => '删除失败';
}

// Path: web.sessions.tabs
class _TranslationsWebSessionsTabsZh extends TranslationsWebSessionsTabsEn {
	_TranslationsWebSessionsTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get closeAria => '关闭标签并移除会话';
	@override String get closeTitle => '关闭标签并移除会话';
}

// Path: web.sessions.page
class _TranslationsWebSessionsPageZh extends TranslationsWebSessionsPageEn {
	_TranslationsWebSessionsPageZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get removedToast => '会话已移除';
	@override String get removeFailedToast => '移除失败';
	@override String get stoppedToast => '会话已停止';
	@override String get stopFailedToast => '停止失败';
	@override String get restartedToast => '会话已重启';
	@override String get restartFailedToast => '重启失败';
	@override String confirmCloseTabTitle({required Object name}) => '停止并移除 "${name}"?';
	@override String get confirmCloseTabDescription => 'CLI 进程将被终止并删除该记录。';
	@override String get confirmCloseTabConfirm => '停止并移除';
	@override String confirmRemoveTitle({required Object name}) => '移除 ${name}?';
	@override String get confirmRemoveTitleFallback => '移除会话?';
	@override String get confirmRemoveDescription => '这将删除该记录。';
	@override String get confirmRemoveConfirm => '移除';
}

// Path: web.sessions.empty
class _TranslationsWebSessionsEmptyZh extends TranslationsWebSessionsEmptyEn {
	_TranslationsWebSessionsEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '未打开任何会话';
	@override String hint({required Object kbdN, required Object kbdW, required Object kbdRange}) => '从列表中挑选一个会话，或新建一个。快捷键：${kbdN} 新建，${kbdW} 关闭，${kbdRange} 切换。';
	@override String get spawn => '创建会话';
}

// Path: web.sessions.header
class _TranslationsWebSessionsHeaderZh extends TranslationsWebSessionsHeaderEn {
	_TranslationsWebSessionsHeaderZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loadingSession => '正在加载会话…';
	@override String get showList => '显示会话列表';
	@override String get hideList => '隐藏会话列表';
	@override String get showInspector => '显示检查器';
	@override String get hideInspector => '隐藏检查器';
	@override String get attachImage => '附加图片';
	@override String get attachImageTooltip => '附加图片（或直接粘贴 / 拖入终端）';
	@override String get restart => '重启';
	@override String get restarting => '重启中…';
	@override String get remove => '移除';
	@override String get removing => '移除中…';
	@override String get stop => '停止';
	@override String get stopping => '停止中…';
	@override String pid({required Object pid}) => 'pid ${pid}';
}

// Path: web.sessions.terminal
class _TranslationsWebSessionsTerminalZh extends TranslationsWebSessionsTerminalEn {
	_TranslationsWebSessionsTerminalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get uploadingToast => '正在上传图片…';
	@override String get uploadedToast => '图片已附加';
	@override String get uploadFailedToast => '上传失败';
	@override String get uploadInvalidTypeToast => '仅支持图片文件';
	@override String get dropToAttach => '释放以附加图片';
}

// Path: web.sessions.spawn
class _TranslationsWebSessionsSpawnZh extends TranslationsWebSessionsSpawnEn {
	_TranslationsWebSessionsSpawnZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '创建会话';
	@override String get description => '在已注册的 Provider 下启动一个 CLI 会话。';
	@override String get provider => 'Provider';
	@override String get claudeAccount => 'Claude 账号';
	@override String get loadingAccounts => '正在加载账号…';
	@override String get noAccounts => '尚未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。';
	@override String get kDefault => '默认';
	@override String get defaultTooltip => '使用系统 keychain / 环境变量';
	@override String get tokenEmptyBadge => '·未填';
	@override String get tokenMissingTooltip => '未设置 token — 请先在 Providers 面板中填入';
	@override String get multiAccountHint => '已配置多个账号 — 请为本次会话挑选一个。';
	@override String get cwd => '工作目录';
	@override String get cwdPlaceholder => '/Users/you/projects/foo';
	@override String get browse => '浏览';
	@override String get nameLabel => '名称（可选）';
	@override String get namePlaceholder => 'claude in pet-tracker';
	@override String get argsLabel => 'CLI 参数（每行一个）';
	@override String get bypassClaude => '跳过权限提示';
	@override String get bypassCodex => '跳过所有批准与沙盒 (--dangerously-bypass-approvals-and-sandbox)';
	@override String get bypassGemini => 'YOLO 模式 (--yolo)';
	@override String get bypassOnHint => '本次会话将以更高的自主权运行。';
	@override String get bypassOffHint => '关闭 — 确认与提示按正常流程处理。';
	@override String get errorPickProvider => '请选择一个 Provider。';
	@override String get errorCwdRequired => '请填写工作目录。';
	@override String get cancel => '取消';
	@override String get submit => '创建';
	@override String get submitting => '创建中…';
	@override String get spawnedToast => '会话已创建';
	@override String spawnedDescription({required Object provider, required Object pid}) => '${provider} · pid ${pid}';
	@override String get pidFallback => '—';
}

// Path: web.sessions.accountSwitcher
class _TranslationsWebSessionsAccountSwitcherZh extends TranslationsWebSessionsAccountSwitcherEn {
	_TranslationsWebSessionsAccountSwitcherZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get tooltip => '切换 Claude 账号（将重启 CLI 进程）';
	@override String get currentDefault => '默认';
	@override String get menuTitle => '切换 Claude 账号';
	@override String get defaultName => '默认';
	@override String get defaultSubtitle => 'CLI 的系统 keychain / 环境变量';
	@override String get tokenEmpty => '·未填';
	@override String get confirmSwitch => '切换账号将重启 Claude CLI 进程。CLI 内部进行中的对话状态将丢失。继续？';
	@override String get switchedToast => '账号已切换';
	@override String switchedDescription({required Object account, required Object pid}) => '当前使用 @${account} · pid ${pid}';
	@override String get switchedDefault => '默认';
	@override String get switchFailedToast => '切换失败';
}

// Path: web.sessions.inspector
class _TranslationsWebSessionsInspectorZh extends TranslationsWebSessionsInspectorEn {
	_TranslationsWebSessionsInspectorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebSessionsInspectorTabsZh tabs = _TranslationsWebSessionsInspectorTabsZh._(_root);
}

// Path: web.sessions.ended
class _TranslationsWebSessionsEndedZh extends TranslationsWebSessionsEndedEn {
	_TranslationsWebSessionsEndedZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get bufferUnavailable => '[缓冲区不可用]';
	@override String get readOnlyBanner => '[会话已结束 — 只读缓冲区]';
}

// Path: web.sessions.fileBrowser
class _TranslationsWebSessionsFileBrowserZh extends TranslationsWebSessionsFileBrowserEn {
	_TranslationsWebSessionsFileBrowserZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '选择工作目录';
	@override String get description => '浏览网关主机的文件系统并选择一个文件夹。';
	@override String get parent => '上级目录';
	@override String get home => '家目录';
	@override String get refresh => '刷新';
	@override String get pathPlaceholder => '/Users/you/projects';
	@override String get loading => '加载中…';
	@override String get empty => '目录为空。';
	@override String get newFolder => '新建文件夹';
	@override String get newFolderPlaceholder => 'new-folder-name';
	@override String get create => '创建';
	@override String get cancel => '取消';
	@override String get useThisFolder => '使用此文件夹';
	@override String get createdToast => '文件夹已创建';
	@override String get mkdirFailedToast => '创建失败';
	@override String get homeFailedToast => '读取家目录失败';
}

// Path: web.conflicts.severity
class _TranslationsWebConflictsSeverityZh extends TranslationsWebConflictsSeverityEn {
	_TranslationsWebConflictsSeverityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get low => '低';
	@override String get medium => '中';
	@override String get high => '高';
}

// Path: web.memoryWorkers.tasks
class _TranslationsWebMemoryWorkersTasksZh extends TranslationsWebMemoryWorkersTasksEn {
	_TranslationsWebMemoryWorkersTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebMemoryWorkersTasksGatekeeperZh gatekeeper = _TranslationsWebMemoryWorkersTasksGatekeeperZh._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksCleanerZh cleaner = _TranslationsWebMemoryWorkersTasksCleanerZh._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksGitactivityZh gitactivity = _TranslationsWebMemoryWorkersTasksGitactivityZh._(_root);
	@override late final _TranslationsWebMemoryWorkersTasksTranscriptZh transcript = _TranslationsWebMemoryWorkersTasksTranscriptZh._(_root);
}

// Path: web.project.picker
class _TranslationsWebProjectPickerZh extends TranslationsWebProjectPickerEn {
	_TranslationsWebProjectPickerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '选择项目';
	@override String get subtitle => '项目记忆按工作目录划分。选择一个以管理它的目标、计划、日志和清理队列。';
	@override String get pathPlaceholder => '/path/to/your/project';
	@override String get browse => '浏览';
	@override String get browseTooltip => '浏览网关主机的文件系统';
	@override String get open => '打开';
	@override String get recentLabel => '最近的项目（来自已存记忆）：';
	@override String get orphanTooltip => '看上去是被截断的 scope_key（老旧镜像导入 bug）。可能没有项目文档。';
	@override String get orphanBadge => '孤立';
}

// Path: web.project.header
class _TranslationsWebProjectHeaderZh extends TranslationsWebProjectHeaderEn {
	_TranslationsWebProjectHeaderZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String docsCount_one({required Object count}) => '${count} 份文档';
	@override String docsCount_other({required Object count}) => '${count} 份文档';
	@override String journalEntries_one({required Object count}) => '${count} 条日志';
	@override String journalEntries_other({required Object count}) => '${count} 条日志';
	@override String pendingProposals_one({required Object count}) => '${count} 条待处理提案';
	@override String pendingProposals_other({required Object count}) => '${count} 条待处理提案';
	@override String cleanupPending({required Object count}) => '${count} 条待清理';
}

// Path: web.project.tabs
class _TranslationsWebProjectTabsZh extends TranslationsWebProjectTabsEn {
	_TranslationsWebProjectTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get health => '健康';
	@override String get goal => '目标';
	@override String get plan => '计划';
	@override String get tech => '技术栈';
	@override String get activity => '活动';
	@override String get journal => '日志';
	@override String get inbox => '收件箱';
	@override String get conflicts => '冲突';
	@override String get cleanup => '清理';
}

// Path: web.project.docLabel
class _TranslationsWebProjectDocLabelZh extends TranslationsWebProjectDocLabelEn {
	_TranslationsWebProjectDocLabelZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get goal => '目标';
	@override String get plan => '计划';
	@override String get tech_stack => '技术栈';
	@override String get recent_activity => '最近活动';
}

// Path: web.project.verdictLabel
class _TranslationsWebProjectVerdictLabelZh extends TranslationsWebProjectVerdictLabelEn {
	_TranslationsWebProjectVerdictLabelZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get stale => '删除';
	@override String get duplicate => '合并';
	@override String get keep => '保留';
}

// Path: web.project.editor
class _TranslationsWebProjectEditorZh extends TranslationsWebProjectEditorEn {
	_TranslationsWebProjectEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get updatedBy => '更新者';
	@override String noDocSet({required Object label}) => '尚未设置${label}。';
	@override String get save => '保存';
	@override String get saveFailedToast => '保存失败';
	@override String savedToast({required Object label}) => '${label}已保存';
	@override String get goalPlaceholder => '我们在做什么？一段文字。每个 agent 在 spawn 时都会读取。';
	@override String get planPlaceholder => '当前计划 — 现在在做什么、下一步是什么。随进度更新。';
}

// Path: web.project.readonly
class _TranslationsWebProjectReadonlyZh extends TranslationsWebProjectReadonlyEn {
	_TranslationsWebProjectReadonlyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebProjectReadonlyTechStackZh tech_stack = _TranslationsWebProjectReadonlyTechStackZh._(_root);
	@override late final _TranslationsWebProjectReadonlyRecentActivityZh recent_activity = _TranslationsWebProjectReadonlyRecentActivityZh._(_root);
	@override String noneCaptured({required Object label}) => '尚未捕获${label}。';
	@override String get generatedBy => '生成者';
	@override String get lastRefresh => '最近刷新';
}

// Path: web.project.journal
class _TranslationsWebProjectJournalZh extends TranslationsWebProjectJournalEn {
	_TranslationsWebProjectJournalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get empty => '暂无日志条目。每次会话结束都会自动追加一条。';
}

// Path: web.project.inbox
class _TranslationsWebProjectInboxZh extends TranslationsWebProjectInboxEn {
	_TranslationsWebProjectInboxZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get emptyTitle => '收件箱为空。';
	@override String get emptyHint => 'Agent 通过 `project_goal_set` / `project_plan_set` MCP 工具在这里提交提案。';
	@override String approvedToast({required Object label}) => '${label}已更新';
	@override String get approveFailedToast => '批准失败';
	@override String get rejectedToast => '已驳回';
	@override String get rejectFailedToast => '驳回失败';
	@override String get sessionPrefix => 'ses';
	@override String warning({required Object label}) => '批准将完全替换当前${label}。';
	@override String get warningSuffix => '请检查下方 diff；这不是合并。';
	@override String get current => '当前';
	@override String get proposed => '提议';
	@override String get emptyBody => '(空)';
	@override String get approve => '批准';
	@override String get reject => '驳回';
	@override String confirmDialogTitle({required Object label}) => '替换${label}?';
	@override String confirmDialogDescription({required Object label}) => '当前${label}将被提议内容覆盖。无法通过本界面撤销（可手动改回）。';
	@override String get confirmCancel => '取消';
	@override String get confirmReplace => '确认替换';
}

// Path: web.project.cleanup
class _TranslationsWebProjectCleanupZh extends TranslationsWebProjectCleanupEn {
	_TranslationsWebProjectCleanupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get hint => 'LLM librarian 为该项目的记忆提出保留 / 过期 / 重复 的判定。删除前需你批准。';
	@override String get runNow => '立即运行清理';
	@override String runSucceededToast({required Object decided, required Object scanned}) => '清理已运行：${decided} 条决策入队（扫描了 ${scanned} 条）';
	@override String get runFailedToast => '清理运行失败';
	@override String get empty => '暂无待处理决策。要么没有记忆达到判定年龄，要么上次运行判断全部仍是关键内容。';
	@override String get mergeIntoPrefix => '→ 合并到';
	@override String get reasonPrefix => '原因：';
	@override String get executeButton => '执行';
	@override String get confirmKeepButton => '确认保留';
	@override String get rejectButton => '驳回';
	@override String approvedExecutedToast({required Object label}) => '已执行${label}';
	@override String get approveFailedToast => '批准失败';
	@override String get rejectedToast => '已驳回 — 记忆保留';
	@override String get rejectFailedToast => '驳回失败';
}

// Path: web.project.reset
class _TranslationsWebProjectResetZh extends TranslationsWebProjectResetEn {
	_TranslationsWebProjectResetZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get button => '重置';
	@override String get dialogTitle => '重置项目记忆?';
	@override String get dialogDescription => '删除该 cwd 下存储的所有项目上下文。不可撤销。';
	@override String get alwaysDeleted => '始终删除：目标、计划、提案、日志、清理决策。';
	@override String get alsoDeleteScannerLabel => '同时删除 scanner 文档';
	@override String get alsoDeleteScannerSuffix => '(tech_stack + recent_activity)。';
	@override String get alsoDeleteScannerHint => '下次 spawn 会自动重建 — 通常不勾选即可。';
	@override String get alsoDeleteMemoriesLabel => '同时删除 pgvector 记忆';
	@override String get alsoDeleteMemoriesSuffix => '（该 scope_key 下的）。';
	@override String get alsoDeleteMemoriesHint => 'Agent 存储的长期事实（用户偏好、项目事实）。无法恢复。';
	@override String get cancel => '取消';
	@override String get deleteForever => '永久删除';
	@override String successToast({required Object summary}) => '重置：已删除 ${summary}';
	@override late final _TranslationsWebProjectResetSummaryZh summary = _TranslationsWebProjectResetSummaryZh._(_root);
	@override String get failedToast => '重置失败';
}

// Path: web.memoryInspector.status
class _TranslationsWebMemoryInspectorStatusZh extends TranslationsWebMemoryInspectorStatusEn {
	_TranslationsWebMemoryInspectorStatusZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '当前 embedder';
	@override String get unavailable => '不可用';
	@override String get probing => '探测中…';
	@override String dimensions({required Object dim, required Object state}) => '${dim} 维 · ${state}';
	@override String get enabled => '已启用';
	@override String get disabled => '已禁用';
	@override String get testButton => '测试 embedder';
}

// Path: web.memoryInspector.scope
class _TranslationsWebMemoryInspectorScopeZh extends TranslationsWebMemoryInspectorScopeEn {
	_TranslationsWebMemoryInspectorScopeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Scope';
	@override String get scopeKey => 'Scope key';
	@override String get scopeKeyIgnored => '(global 时忽略)';
	@override String get scopeKeyCwd => '(项目的 cwd)';
	@override String get scopeKeySession => '(session id)';
	@override String get placeholderProject => '/path/to/project (cwd)';
	@override String get placeholderSession => 'session id';
	@override String get syncMd => '同步 .md';
	@override String get syncTooltip => '把 Claude 的 <cwd>/.claude/memory/*.md 重新摄取到 pgvector';
	@override late final _TranslationsWebMemoryInspectorScopeValuesZh values = _TranslationsWebMemoryInspectorScopeValuesZh._(_root);
}

// Path: web.memoryInspector.search
class _TranslationsWebMemoryInspectorSearchZh extends TranslationsWebMemoryInspectorSearchEn {
	_TranslationsWebMemoryInspectorSearchZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get placeholder => '语义搜索查询（Enter 运行；为空则浏览）';
	@override String get run => '搜索';
	@override String get clear => '清空';
	@override String get failedToast => '搜索失败';
}

// Path: web.memoryInspector.records
class _TranslationsWebMemoryInspectorRecordsZh extends TranslationsWebMemoryInspectorRecordsEn {
	_TranslationsWebMemoryInspectorRecordsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get noMemories => '暂无记忆';
	@override String matches_one({required Object count}) => '${count} 条匹配';
	@override String matches_other({required Object count}) => '${count} 条匹配';
	@override String memories_one({required Object count}) => '${count} 条记忆';
	@override String memories_other({required Object count}) => '${count} 条记忆';
	@override String get scopeGlobalSuffix => '（全局）';
	@override String scopeInSuffix({required Object scope}) => '（${scope}：';
	@override String get addButton => '添加记忆';
	@override String get addTooltip => '手动在此 scope 创建一条记忆';
	@override String get deleteAll => '全部删除';
	@override String get deleteAllTooltip => '删除该 scope/scope_key 下的全部记忆';
	@override String get loading => '加载中…';
	@override String get enterScopeKeyHint => '输入 scope key 以浏览记忆。';
	@override String noMatchesForQuery({required Object query}) => '未找到匹配 "${query}"';
	@override String get noMemoriesInScope => '此 scope 暂无记忆。';
}

// Path: web.memoryInspector.row
class _TranslationsWebMemoryInspectorRowZh extends TranslationsWebMemoryInspectorRowEn {
	_TranslationsWebMemoryInspectorRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String simBadge({required Object value}) => '相似度 ${value}';
	@override String hits_one({required Object count}) => '命中 ${count} 次';
	@override String hits_other({required Object count}) => '命中 ${count} 次';
	@override String lastHitTooltip({required Object relative}) => '最近命中 ${relative}';
	@override String get editPlaceholder => '记忆文本 — Cmd/Ctrl+Enter 保存 · Esc 取消';
	@override String get saveTooltip => '保存 (Cmd/Ctrl+Enter)';
	@override String get cancelTooltip => '取消 (Esc)';
	@override String get editTooltip => '编辑该记忆';
	@override String get deleteTooltip => '删除该记忆';
	@override String get emptyError => '记忆文本不能为空';
	@override String deleteConfirm({required Object id}) => '删除记忆 ${id}? 不可恢复。';
}

// Path: web.memoryInspector.toasts
class _TranslationsWebMemoryInspectorToastsZh extends TranslationsWebMemoryInspectorToastsEn {
	_TranslationsWebMemoryInspectorToastsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get deleted => '记忆已删除';
	@override String get deleteFailed => '删除失败';
	@override String bulkDeleted_one({required Object count}) => '已从此 scope 删除 ${count} 条记忆';
	@override String bulkDeleted_other({required Object count}) => '已从此 scope 删除 ${count} 条记忆';
	@override String get bulkDeleteFailed => '批量删除失败';
	@override String get created => '记忆已创建';
	@override String get createFailed => '创建失败';
	@override String get updated => '记忆已更新';
	@override String get updateFailed => '更新失败';
	@override String migrated({required Object reembed, required Object examined, required Object to}) => '已迁移 ${reembed}/${examined} 条记忆到 ${to}';
	@override String get migrationFailed => '迁移失败';
	@override String syncIngested_one({required Object count}) => '已摄取 ${count} 个新记忆文件';
	@override String syncIngested_other({required Object count}) => '已摄取 ${count} 个新记忆文件';
	@override String get syncEmpty => '没有需要同步的新 .md 文件';
	@override String get syncEmptyDescription => '已是最新，或该 cwd 没有 Claude memory 目录。';
	@override String get syncFailed => '同步失败';
	@override String testOk({required Object embedder, required Object dim}) => 'Embedder OK：${embedder} · ${dim} 维';
	@override String testOkDescription({required Object preview}) => 'vector_preview = [${preview}…]';
	@override String get testFailed => 'Embedder 探测失败';
}

// Path: web.memoryInspector.bulkDelete
class _TranslationsWebMemoryInspectorBulkDeleteZh extends TranslationsWebMemoryInspectorBulkDeleteEn {
	_TranslationsWebMemoryInspectorBulkDeleteZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '删除此 scope 的全部记忆?';
	@override String get description => '这是一次 SQL 操作 — 该 scope 下全部记忆将被原子性删除。通过 Claude 镜像摄取的记忆会在下次 <1>同步 .md</1> 时重新出现；其余内容永久消失。';
	@override String get scope => 'Scope';
	@override String get scopeKey => 'Scope key';
	@override String get currentlyVisible => '当前可见';
	@override String items_one({required Object count}) => '${count} 条记忆';
	@override String items_other({required Object count}) => '${count} 条记忆';
	@override String get cancel => '取消';
	@override String get deleteAll => '全部删除';
}

// Path: web.memoryInspector.addMem
class _TranslationsWebMemoryInspectorAddMemZh extends TranslationsWebMemoryInspectorAddMemEn {
	_TranslationsWebMemoryInspectorAddMemZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '添加记忆';
	@override String get description => '手动创建一条记忆。Agent 会通过 <1>memory_store</1> MCP 工具自动创建；此表单用于运维想跳过 agent 直接录入事实的场景。';
	@override String get textLabel => '文本';
	@override String get textPlaceholder => '纯文本。Embedder 在存储时把它转成向量；agent 通过 memory_search 取回。';
	@override String get cancel => '取消';
	@override String get create => '创建';
}

// Path: web.memoryInspector.picker
class _TranslationsWebMemoryInspectorPickerZh extends TranslationsWebMemoryInspectorPickerEn {
	_TranslationsWebMemoryInspectorPickerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get button => '选择';
	@override String get buttonTooltip => '从已保存的 scope key 或活跃会话中选择';
	@override String get loading => '加载中…';
	@override String empty({required Object scope}) => '${scope} 暂无已保存的 key 或活跃会话。';
	@override String get savedHeader => '已保存的记忆';
	@override String get activeHeader => '活跃会话';
}

// Path: web.memoryInspector.migrationBanner
class _TranslationsWebMemoryInspectorMigrationBannerZh extends TranslationsWebMemoryInspectorMigrationBannerEn {
	_TranslationsWebMemoryInspectorMigrationBannerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String headline_one({required Object count}) => '${count} 条记忆不会出现在搜索结果中';
	@override String headline_other({required Object count}) => '${count} 条记忆不会出现在搜索结果中';
	@override String subtitle({required Object summary, required Object current}) => '${summary} — 当前 embedder 为 <1>${current}</1>。pgvector 按 embedder 分区其相似度索引，旧条目在重嵌入前不会被检索到。';
	@override String summaryItem({required Object count, required Object name}) => '${count} 条在 ${name}';
	@override String get migrateButton => '迁移';
}

// Path: web.memoryInspector.reembed
class _TranslationsWebMemoryInspectorReembedZh extends TranslationsWebMemoryInspectorReembedEn {
	_TranslationsWebMemoryInspectorReembedZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '重新嵌入记忆';
	@override String get description => '对存储在其他 embedder 下的记忆重新计算向量，使它们再次可被搜索到。';
	@override String get targetEmbedder => '目标 embedder';
	@override String get onName => '在';
	@override String get totalToReembed => '待重嵌入总数';
	@override String get explainer => '每条记忆的文本会重新发送到当前 embedder；新向量原地替换旧向量。ID、scope、scope_key、metadata 与时间戳保持不变。搜索结果立即生效 — 无需重启。';
	@override String get reportExamined => '已检查';
	@override String get reportReembedded => '已重嵌入';
	@override String get reportFailed => '失败';
	@override String get reportFrom => '来源';
	@override String errors_one({required Object count}) => '${count} 个错误';
	@override String errors_other({required Object count}) => '${count} 个错误';
	@override String get done => '完成';
	@override String get cancel => '取消';
	@override String get reembedding => '重嵌入中…';
	@override String reembedTotal({required Object total}) => '重嵌入 ${total} 条';
}

// Path: web.notes.header
class _TranslationsWebNotesHeaderZh extends TranslationsWebNotesHeaderEn {
	_TranslationsWebNotesHeaderZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get outline => '大纲';
	@override String get showOutline => '显示大纲';
	@override String get hideOutline => '隐藏大纲';
	@override String get today => '今天';
	@override String get todayTooltip => '打开或创建今天的日志笔记';
	@override String get kNew => '新建';
}

// Path: web.notes.left
class _TranslationsWebNotesLeftZh extends TranslationsWebNotesLeftEn {
	_TranslationsWebNotesLeftZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get tree => '目录树';
	@override String get tags => '标签';
	@override String get filterNotes => '过滤笔记…';
	@override String get filterTags => '过滤标签…';
	@override String get filteredBy => '已筛选';
	@override String get clearTagTooltip => '清除标签筛选';
	@override String get expandAll => '全部展开';
	@override String get expandAllTooltip => '展开全部文件夹';
	@override String get collapseAll => '全部收起';
	@override String get collapseAllTooltip => '收起全部文件夹';
	@override String get loading => '加载中…';
	@override String footer({required Object visible, required Object total}) => '${visible} / ${total} 条笔记';
}

// Path: web.notes.tags
class _TranslationsWebNotesTagsZh extends TranslationsWebNotesTagsEn {
	_TranslationsWebNotesTagsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get emptyVault => 'vault 中暂无标签。';
	@override String noMatches({required Object query}) => '未找到匹配 "${query}"。';
}

// Path: web.notes.tree
class _TranslationsWebNotesTreeZh extends TranslationsWebNotesTreeEn {
	_TranslationsWebNotesTreeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get empty => 'vault 为空。';
}

// Path: web.notes.outline
class _TranslationsWebNotesOutlineZh extends TranslationsWebNotesOutlineEn {
	_TranslationsWebNotesOutlineZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '大纲';
	@override String get empty => '此笔记没有标题。添加 <1>## 标题</1> 行以填充大纲。';
}

// Path: web.notes.newNote
class _TranslationsWebNotesNewNoteZh extends TranslationsWebNotesNewNoteEn {
	_TranslationsWebNotesNewNoteZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get prompt => '新笔记路径（相对 vault，必须以 .md 结尾）';
	@override String defaultPath({required Object date}) => 'library/notes-${date}.md';
	@override String get errorMustEndMd => '路径必须以 .md 结尾';
	@override String get createdToast => '笔记已创建';
	@override String get createFailedToast => '创建失败';
}

// Path: web.notes.empty
class _TranslationsWebNotesEmptyZh extends TranslationsWebNotesEmptyEn {
	_TranslationsWebNotesEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '未选择笔记';
	@override String get hint => '从左侧目录树挑选一条笔记，直接跳到今天的日志，或新建一条。AI 生成的项目文档位于 <1>projects/</1>；个人草稿位于 <3>personal/</3>。';
	@override String get today => '今天的日志笔记';
	@override String get kNew => '新建笔记';
}

// Path: web.notes.picker
class _TranslationsWebNotesPickerZh extends TranslationsWebNotesPickerEn {
	_TranslationsWebNotesPickerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get browseAria => '浏览文件夹';
	@override String matches_one({required Object count}) => '${count} 个匹配';
	@override String matches_other({required Object count}) => '${count} 个匹配';
	@override String foldersInVault({required Object count}) => 'vault 中 ${count} 个文件夹';
	@override String noMatch({required Object value}) => '未找到匹配的文件夹。直接保存即可使用 <1>${value}</1>（首次写入时懒创建）。';
}

// Path: web.notes.vaultSync
class _TranslationsWebNotesVaultSyncZh extends TranslationsWebNotesVaultSyncEn {
	_TranslationsWebNotesVaultSyncZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Vault 同步';
	@override String get description => '把 notes vault 作为 git 仓库进行 commit、pull 与 push。认证使用网关主机的 git 凭据（SSH agent / credential helper）。';
	@override String get reading => '正在读取 vault 状态…';
	@override late final _TranslationsWebNotesVaultSyncInitZh init = _TranslationsWebNotesVaultSyncInitZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncBranchZh branch = _TranslationsWebNotesVaultSyncBranchZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncActionZh action = _TranslationsWebNotesVaultSyncActionZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncCommitZh commit = _TranslationsWebNotesVaultSyncCommitZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncFileListZh fileList = _TranslationsWebNotesVaultSyncFileListZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncRemoteZh remote = _TranslationsWebNotesVaultSyncRemoteZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncHistoryZh history = _TranslationsWebNotesVaultSyncHistoryZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncConflictZh conflict = _TranslationsWebNotesVaultSyncConflictZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncAuthZh auth = _TranslationsWebNotesVaultSyncAuthZh._(_root);
	@override late final _TranslationsWebNotesVaultSyncAutoSyncZh autoSync = _TranslationsWebNotesVaultSyncAutoSyncZh._(_root);
}

// Path: web.notes.syncBadge
class _TranslationsWebNotesSyncBadgeZh extends TranslationsWebNotesSyncBadgeEn {
	_TranslationsWebNotesSyncBadgeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get syncLabel => '同步';
	@override String get initLabel => '初始化';
	@override String get initTooltip => 'vault 尚未初始化为 git 仓库';
	@override String get conflictLabel => '冲突';
	@override String get conflictTooltip => 'vault 存在未解决的冲突 — 点击进入恢复';
	@override String get syncFallback => 'sync';
	@override String tooltip({required Object branch, required Object files, required Object ahead, required Object behind}) => '分支 ${branch} · ${files} 处改动 · 领先 ${ahead} · 落后 ${behind}';
	@override String get tooltipAutoOn => ' · 自动同步已开启';
	@override String tooltipLastError({required Object error}) => ' · 上次错误：${error}';
	@override String get branchPlaceholder => '—';
}

// Path: web.activity.filters
class _TranslationsWebActivityFiltersZh extends TranslationsWebActivityFiltersEn {
	_TranslationsWebActivityFiltersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get integration => '集成';
	@override String get direction => '方向';
	@override String get status => '状态';
	@override String get allIntegrations => '所有集成';
	@override String get all => '全部';
	@override String get inbound => '入站';
	@override String get outbound => '出站';
	@override String get allStatuses => '所有状态';
	@override String get status2 => '2xx 成功';
	@override String get status3 => '3xx 重定向';
	@override String get status4 => '4xx 客户端错误';
	@override String get status5 => '5xx 服务端错误';
}

// Path: web.activity.table
class _TranslationsWebActivityTableZh extends TranslationsWebActivityTableEn {
	_TranslationsWebActivityTableZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get time => '时间';
	@override String get integration => '集成';
	@override String get directionTitle => '方向';
	@override String get method => '方法';
	@override String get path => '路径';
	@override String get status => '状态';
	@override String get duration => '耗时';
	@override String get inboundAria => '入站';
	@override String get outboundAria => '出站';
}

// Path: web.activity.empty
class _TranslationsWebActivityEmptyZh extends TranslationsWebActivityEmptyEn {
	_TranslationsWebActivityEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get filtered => '没有调用符合当前筛选条件。';
	@override String get title => '尚未记录任何 API 调用';
	@override String get description => '当第三方应用以其集成 API key 调用 opendray 时，每次请求都会被记录在这里。';
	@override String get stepWithIntegrations => '在你的第三方应用中使用已有集成的 API key';
	@override String get stepRegister => '在 集成 → 新建 中注册一个集成';
	@override String get stepCallEndpoint => '调用任意接口，例如 <1>POST /api/v1/sessions</1>';
	@override String get stepAppears => '调用会在几秒内出现在这里';
	@override String get footnote => '你在本管理端 UI 中发起的调用不会被记录 — 仅集成归属的流量会被记录。';
}

// Path: web.activity.events
class _TranslationsWebActivityEventsZh extends TranslationsWebActivityEventsEn {
	_TranslationsWebActivityEventsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '正在加载事件…';
	@override String get empty => '尚无事件。';
	@override String get emptyFiltered => '没有匹配的事件。';
	@override String get loadOlder => '加载更早的事件';
	@override String get today => '今天';
	@override String get yesterday => '昨天';
}

// Path: web.providers.list
class _TranslationsWebProvidersListZh extends TranslationsWebProvidersListEn {
	_TranslationsWebProvidersListZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Providers';
	@override String get loading => '加载中…';
	@override String get disabledBadge => '已禁用';
	@override String get noneSelected => '未选择任何 Provider。';
}

// Path: web.providers.detail
class _TranslationsWebProvidersDetailZh extends TranslationsWebProvidersDetailEn {
	_TranslationsWebProvidersDetailZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get enabled => '已启用';
	@override String get disabled => '已禁用';
	@override String toggleAria({required Object name}) => '切换 ${name}';
	@override String get configuration => '配置';
	@override String get noConfig => '此 Provider 没有可配置字段。';
	@override String get executable => 'executable:';
	@override String get manifestHash => 'manifest_hash:';
	@override String get reset => '重置';
	@override String get save => '保存更改';
	@override String get saving => '保存中…';
	@override String get savedToast => 'Provider 配置已保存';
	@override String get saveFailedToast => '保存失败';
	@override String get toggleFailedToast => '切换失败';
	@override late final _TranslationsWebProvidersDetailCapsZh caps = _TranslationsWebProvidersDetailCapsZh._(_root);
}

// Path: web.providers.configForm
class _TranslationsWebProvidersConfigFormZh extends TranslationsWebProvidersConfigFormEn {
	_TranslationsWebProvidersConfigFormZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get selectPlaceholder => '选择…';
	@override String get defaultOption => '(默认)';
	@override String get switchOn => '开';
	@override String get switchOff => '关';
	@override String get showSecret => '显示密钥';
	@override String get hideSecret => '隐藏密钥';
}

// Path: web.providers.claudeAccounts
class _TranslationsWebProvidersClaudeAccountsZh extends TranslationsWebProvidersClaudeAccountsEn {
	_TranslationsWebProvidersClaudeAccountsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Claude 账号';
	@override String get tutorialTooltip => '打开多账号教程章节';
	@override String get importLocal => '导入本地';
	@override String get importLocalTooltip => '扫描网关主机上的 ~/.claude-accounts/ 目录并注册新的目录。该按钮仅在网关主机环境下工作 — 详见教程。';
	@override String get importedNothingToast => '无需导入 — 账号已同步。';
	@override String importedToast_one({required Object count}) => '已从 ~/.claude-accounts 导入 ${count} 个账号';
	@override String importedToast_other({required Object count}) => '已从 ~/.claude-accounts 导入 ${count} 个账号';
	@override String get importFailedToast => '导入失败';
	@override String get addingTitle => '添加新账号。';
	@override String get addingBodyPrefix => '在网关主机执行：';
	@override String get addingBodySuffix => 'opendray 的文件系统监听会自动注册新目录，或点击 <1>导入本地</1> 立即扫描。';
	@override String get architectureLink => '架构与完整指南 →';
	@override String get loading => '加载中…';
	@override String get empty => '尚无 Claude 账号。在网关主机执行上面的 shell 命令，然后点击 <1>导入本地</1> 扫描。';
	@override String get noTokenYet => '尚无 token';
	@override String get configDir => 'config_dir:';
	@override String get tokenPath => 'token_path:';
	@override String get toggleFailedToast => '切换失败';
	@override String removeConfirm({required Object name}) => '移除账号 "${name}"?';
	@override String get removedToast => '账号已移除';
	@override String get removeFailedToast => '移除失败';
	@override String toggleAria({required Object name}) => '切换 ${name}';
	@override String removeAria({required Object name}) => '移除 ${name}';
}

// Path: web.channels.empty
class _TranslationsWebChannelsEmptyZh extends TranslationsWebChannelsEmptyEn {
	_TranslationsWebChannelsEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '暂无频道';
	@override String get description => '内置类型：Telegram · Slack · Discord · 飞书 · 钉钉 · 企业微信。挑选一个并填入凭证，或使用 <1>bridge</1> 通过 WebSocket 接入自定义平台。';
}

// Path: web.channels.card
class _TranslationsWebChannelsCardZh extends TranslationsWebChannelsCardEn {
	_TranslationsWebChannelsCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get running => '运行中';
	@override String get starting => '启动中…';
	@override String get disabled => '已禁用';
	@override String get muted => '已静音';
	@override String get tokenLabel => 'token:';
	@override String get chatIdLabel => 'chat_id:';
	@override String get channelIdLabel => 'channel_id:';
	@override String get notifyOnLabel => 'notify_on:';
	@override String get webhookLabel => 'webhook:';
	@override String get copyWebhookTooltip => '复制 webhook URL';
	@override String get webhookCopiedToast => '已复制 webhook URL';
	@override String get setup => '配置';
	@override String get setupTooltip => '查看适配器连接信息和示例代码';
	@override String get test => '测试';
	@override String get testNotRunningTooltip => '频道必须处于运行状态';
	@override String get testBridgeTooltip => 'Bridge 频道无法从管理端测试 — 请先连接一个适配器';
	@override String get editAria => '编辑频道';
	@override String get editTooltip => '编辑频道配置';
	@override String get deleteAria => '删除频道';
	@override String get bridgeSuffix => '(bridge)';
}

// Path: web.channels.toasts
class _TranslationsWebChannelsToastsZh extends TranslationsWebChannelsToastsEn {
	_TranslationsWebChannelsToastsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get testSent => '测试消息已发送';
	@override String get testFailed => '测试失败';
	@override String deleteConfirm({required Object id}) => '删除频道 ${id}?';
	@override String get deleted => '频道已删除';
	@override String get created => '频道已创建';
	@override String get updated => '频道已更新';
}

// Path: web.channels.dialog
class _TranslationsWebChannelsDialogZh extends TranslationsWebChannelsDialogEn {
	_TranslationsWebChannelsDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get editTitle => '编辑频道';
	@override String get createTitle => '注册频道';
	@override String get descriptionBridge => '外部适配器（Python/Node/...）通过 WebSocket 连接并出示此 token。';
	@override String get descriptionDefault => '配置消息集成。';
	@override String get kindLabel => '类型';
	@override String get kindImmutable => '（不可更改 — 如需更换类型请删除后重建）';
	@override String get enabledLabel => '启用';
	@override String get enabledBridgeHint => '（立即开始接受适配器连接）';
	@override String get enabledWebhookHint => '（立即开始接收 webhook）';
	@override String get enabledDefaultHint => '（立即启动）';
	@override String get cancel => '取消';
	@override String get save => '保存';
	@override String get saving => '保存中…';
	@override String get create => '创建';
	@override String get creating => '创建中…';
	@override String unknownKind({required Object kind}) => '未知类型：${kind}';
	@override String get nameRequired => 'name 不能为空';
	@override String get tokenRequired => 'token 不能为空';
	@override String topicIdsNumeric({required Object value}) => 'Topic ID 必须是数字（收到 ${value}）';
	@override String fieldRequired({required Object label}) => '${label} 不能为空';
	@override String get cooldownInvalid => 'Cooldown 必须是非负整数秒';
	@override String get snippetCapInvalid => 'Snippet cap 必须是非负数字';
}

// Path: web.channels.notifications
class _TranslationsWebChannelsNotificationsZh extends TranslationsWebChannelsNotificationsEn {
	_TranslationsWebChannelsNotificationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get sectionTitle => '会话通知';
	@override String get notifyOnLabel => '通知触发条件';
	@override String get hintAll => '接收全部会话事件。点击标签可退订。';
	@override String get hintNone => '未选择任何事件 — 出站通知已静音。';
	@override String hintSome({required Object total, required Object selected}) => '只选择了 ${total} 中的 ${selected} 个 topic。';
	@override String get repeatPolicyLabel => '重复策略';
	@override String get cooldownLabel => '冷却时长';
	@override String get onceReplyHint => '在该聊天中以非命令文本回复会重置抑制 — opendray 会把你的回复转发到会话的 stdin 并重新启用通知。';
	@override String get terminalSnippetLabel => '终端片段';
	@override String get embedSnippetLabel => '在 idle 通知中嵌入最近的终端画面';
	@override String get snippetExplainer => '开启后，idle 卡片会包含一段代码块片段，呈现用户在网页终端中会看到的内容 — Claude TUI 自身的装饰（状态 spinner、"bypass permissions" 提示、分隔线）会被自动过滤。';
	@override late final _TranslationsWebChannelsNotificationsModesZh modes = _TranslationsWebChannelsNotificationsModesZh._(_root);
	@override late final _TranslationsWebChannelsNotificationsCooldownsZh cooldowns = _TranslationsWebChannelsNotificationsCooldownsZh._(_root);
	@override late final _TranslationsWebChannelsNotificationsSnippetCapsZh snippetCaps = _TranslationsWebChannelsNotificationsSnippetCapsZh._(_root);
}

// Path: web.channels.bridge
class _TranslationsWebChannelsBridgeZh extends TranslationsWebChannelsBridgeEn {
	_TranslationsWebChannelsBridgeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get nameLabel => 'Bridge 名称';
	@override String get namePlaceholder => 'wechat / discord-custom / whatsapp...';
	@override String get nameHint => '适配器的人类可读标签。会显示在频道列表中。';
	@override String get tokenLabel => '适配器 token';
	@override String get regenerateTooltip => '重新生成';
	@override String get copyTooltip => '复制';
	@override String get tokenCopiedToast => '已复制 token';
	@override String get tokenHint => '适配器通过 WS register 帧发送此 token（也可作为 <1>X-Bridge-Token</1> header）。';
	@override String get capsLabel => '接受的能力（可选白名单）';
	@override String get capsHint => '为空 = 接受适配器声明的任意能力。已选 = 即使适配器提供更多能力，也只允许这些。';
	@override String get afterCreate => '点击 <1>创建</1> 后，适配器设置对话框会自动打开，里面包含 WebSocket URL 和可直接复制的 Python / Node / wscat 起步代码。';
}

// Path: web.channels.setup
class _TranslationsWebChannelsSetupZh extends TranslationsWebChannelsSetupEn {
	_TranslationsWebChannelsSetupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object name}) => '适配器设置 — ${name}';
	@override String get description => '运行一个任意语言的适配器，使用这些凭证通过 WebSocket 连接到 opendray。opendray 会通过它路由会话通知和 slash 命令。';
	@override String get wsUrlLabel => 'WebSocket URL';
	@override String get tokenLabel => '适配器 token';
	@override String authInfo({required Object frame}) => '<1>认证：</1>通过 <3>X-Bridge-Token</3> header、<5>?token=</5> query 参数或 <7>Authorization: Bearer …</7> 之一发送 token。首个 WS 帧必须是 <9>${frame}</9>。完整协议见仓库中的 <11>docs/bridge-protocol.md</11>。';
	@override String get pythonInstall => '安装：<1>pip install websockets</1>。运行：<3>python adapter.py</3>。';
	@override String get nodeInstall => '安装：<1>npm i ws</1>。运行：<3>node adapter.mjs</3>。';
	@override String get wscatInstall => '安装：<1>npm i -g wscat</1>。连接后，粘贴上方的 JSON 注册帧，然后手动发送后续帧。';
	@override String get close => '关闭';
	@override String get copyHide => '隐藏';
	@override String get copyShow => '显示';
	@override String copyLabelToast({required Object label}) => '已复制 ${label}';
	@override String get copyCode => '复制';
	@override String get copied => '已复制';
	@override String get codeCopiedToast => '已复制代码';
}

// Path: web.integrations.tabs
class _TranslationsWebIntegrationsTabsZh extends TranslationsWebIntegrationsTabsEn {
	_TranslationsWebIntegrationsTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get registered => '已注册';
	@override String get console => '反向代理';
}

// Path: web.integrations.empty
class _TranslationsWebIntegrationsEmptyZh extends TranslationsWebIntegrationsEmptyEn {
	_TranslationsWebIntegrationsEmptyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '暂无集成';
	@override String get description => '注册一个外部应用，给它一个受限的 API key。它的代码不需要进入本仓库。';
	@override String get register => '注册集成';
}

// Path: web.integrations.card
class _TranslationsWebIntegrationsCardZh extends TranslationsWebIntegrationsCardEn {
	_TranslationsWebIntegrationsCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get managedBadge => '受管';
	@override String get managedTooltip => 'opendray 管理这个集成。编辑或轮换它的 key 会导致正在运行的会话（mcp.json 中持有旧 bearer）失效。';
	@override String get consumerBadge => 'consumer';
	@override String get consumerTooltip => '仅消费型集成 — 没有可供探测的 HTTP 服务';
	@override String get disabledBadge => '已禁用';
	@override String get consumerOnlyHint => '消费 opendray 的 API。未挂载反向代理。';
	@override String lastProbed({required Object relative}) => '${relative} 探测过';
	@override String rotated({required Object relative}) => '${relative} 轮换过';
	@override String get managedReadOnly => '只读 — opendray 把它的 key 烤进每次 spawn 的 mcp.json';
	@override String get managedReadOnlyTooltip => 'opendray 管理此行。如需重置：删除 ~/.opendray/memory.key 后重启，或直接通过 SQL 删除此行 — 下次启动时会重新引导。';
	@override String get editAria => '编辑集成';
	@override String get editTooltip => '编辑 scopes / base URL / version';
	@override String get rotateKey => '轮换 key';
	@override String get deleteAria => '删除集成';
	@override String rotateConfirm({required Object name}) => '轮换 "${name}" 的 API key? 当前 key 将立即失效。';
	@override String deleteConfirm({required Object name}) => '删除集成 ${name}?';
	@override String get removedToast => '集成已移除';
}

// Path: web.integrations.register_dialog
class _TranslationsWebIntegrationsRegisterDialogZh extends TranslationsWebIntegrationsRegisterDialogEn {
	_TranslationsWebIntegrationsRegisterDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '注册集成';
	@override String get description => '签发一次性 API key。关闭前请复制它 — opendray 不会再显示明文。';
	@override String get nameLabel => '名称';
	@override String get namePlaceholder => 'PetTracker';
	@override String get modeHint => '如果是 <1>仅消费型</1> 集成（第三方应用调用 opendray API 但不暴露自身服务），保留下面两个字段为空。两个都填则是 <3>反向代理型</3> 集成。';
	@override String get baseUrlLabel => 'Base URL';
	@override String get optionalSuffix => '(可选)';
	@override String get baseUrlPlaceholder => 'http://192.168.3.42:8080';
	@override String get routePrefixLabel => 'Route prefix';
	@override String get routePrefixPlaceholder => 'pet-tracker';
	@override String routePrefixHint({required Object prefix}) => '可通过 <1>/api/v1/proxy/${prefix}/*</1> 访问。';
	@override String get routePrefixPlaceholderToken => '<prefix>';
	@override String get versionLabel => 'Version（可选）';
	@override String get versionPlaceholder => '0.1.0';
	@override String get scopesLabel => 'Scopes';
	@override String get scopesIntro => '选择该集成允许调用的 API 范围。每个开关映射到一个 Bearer token 声明 — opendray 会拒绝任何越权请求。';
	@override String get errorNameRequired => 'Name 不能为空。';
	@override String get errorBothOrNeither => 'base_url 和 route_prefix 必须成对设置。同时填入 = 反向代理集成；同时留空 = 仅消费型集成。';
	@override String get cancel => '取消';
	@override String get submit => '注册';
	@override String get submitting => '注册中…';
}

// Path: web.integrations.reveal
class _TranslationsWebIntegrationsRevealZh extends TranslationsWebIntegrationsRevealEn {
	_TranslationsWebIntegrationsRevealZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get titleIssued => 'API key 已签发';
	@override String get titleRotated => 'API key 已轮换';
	@override String get description => '这是明文 key 唯一一次显示的机会。立即复制并更新所有消费端 — 旧 key（如有）将不再有效。';
	@override String get discardAria => '丢弃新 key';
	@override String get discardTooltip => '丢弃新 key（轮换已经发生 — 旧 key 同样失效）';
	@override String get discardConfirm => '确定丢弃新 key? 轮换已经使旧 key 失效 — 丢弃意味着此集成将没有任何可用 key，直到再次轮换。';
	@override String get copy => '复制';
	@override String get copied => '已复制';
	@override String get updateHint => '<1>请用此新 key 更新每个消费端应用。</1> 旧 key 已在服务端失效，下次请求会返回 <3>401 unauthorized</3>。';
	@override String get acknowledge => '我已复制 key 并会更新消费端应用。我了解 opendray 不会再显示它。';
	@override String get discard => '丢弃';
	@override String get done => '完成';
}

// Path: web.integrations.edit_dialog
class _TranslationsWebIntegrationsEditDialogZh extends TranslationsWebIntegrationsEditDialogEn {
	_TranslationsWebIntegrationsEditDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object name}) => '编辑集成 · ${name}';
	@override String get description => '修改 scopes、version 或 base URL。Name 和 route prefix 不可更改 — 如需修改请删除并重新注册。';
	@override String get nameLabel => 'Name';
	@override String get routePrefixLabel => 'Route prefix';
	@override String get consumerOnlyLabel => '(仅消费型)';
	@override String get baseUrlLabel => 'Base URL';
	@override String get baseUrlConsumerSuffix => '(仅消费型 — 留空)';
	@override String get baseUrlProxySuffix => '(反向代理目标)';
	@override String get baseUrlConsumerPlaceholder => '(留空 — 此集成消费 opendray API)';
	@override String get baseUrlProxyPlaceholder => 'http://127.0.0.1:8080';
	@override String get consumerHint => '这是一个仅消费型集成。在此修改 base URL 还需要 route prefix；请删除后重新注册。';
	@override String get versionLabel => 'Version';
	@override String get versionPlaceholder => '0.1.0';
	@override String get scopesLabel => 'Scopes';
	@override String get scopesIntro => '收窄或放宽此集成 API key 授权的 API 范围。已颁发的 token 不受影响 — 新的 scope 集在下次请求时生效。';
	@override String get errorModeSwitch => '在仅消费型与反向代理型之间切换需要删除并重新注册 — name 和 route_prefix 无法原地修改。';
	@override String get updatedToast => '集成已更新';
	@override String get cancel => '取消';
	@override String get save => '保存更改';
}

// Path: web.integrations.proxy
class _TranslationsWebIntegrationsProxyZh extends TranslationsWebIntegrationsProxyEn {
	_TranslationsWebIntegrationsProxyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get emptyTitle => '尚未注册集成';
	@override String emptyDescription({required Object prefix}) => '请先注册一个集成；控制台通过 /api/v1/proxy/${prefix}/* 以 admin token 代理请求。';
	@override String get targetLabel => '目标';
	@override String get selectPlaceholder => '选择集成…';
	@override String get baseLabel => 'base:';
	@override String get history => '历史';
	@override String get historyEmpty => '此集成尚无历史请求';
	@override String get send => '发送';
	@override String get sending => '发送中…';
	@override String get extraHeadersLabel => '额外 header（每行一条，Name: Value）';
	@override String get bodyLabel => 'Body';
	@override String get headers => 'Headers';
	@override String get body => 'Body';
	@override String get emptyBody => '(空)';
	@override String get requestFailed => '请求失败';
	@override String get stubText => '发送一个请求即可查看上游响应。';
	@override String get stubInjects => 'opendray 会注入 <1>X-Integration-ID</1>，并剥离你的 <3>Authorization</3> header。';
	@override String get prefixPlaceholder => '<prefix>';
}

// Path: web.plugins.common
class _TranslationsWebPluginsCommonZh extends TranslationsWebPluginsCommonEn {
	_TranslationsWebPluginsCommonZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get cancel => '取消';
	@override String get edit => '编辑';
	@override String get add => '添加';
	@override String get save => '保存';
	@override String get create => '创建';
}

// Path: web.plugins.mcp
class _TranslationsWebPluginsMcpZh extends TranslationsWebPluginsMcpEn {
	_TranslationsWebPluginsMcpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP 服务器';
	@override String description({required Object KEY}) => '注入到每次 spawn（claude / codex）的 Model Context Protocol 服务器。Vault 条目位于 <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>；env / headers 中以 <3>\$${KEY}</3> 引用的密钥来自下方 <5>MCP secrets</5>。';
	@override String get newServer => '新建服务器';
	@override String get empty => '尚无 MCP 服务器。添加一个以为 agent 会话暴露额外工具。';
	@override late final _TranslationsWebPluginsMcpColumnsZh columns = _TranslationsWebPluginsMcpColumnsZh._(_root);
	@override String get noUrl => '无 URL';
	@override String get noCommand => '无 command';
	@override String deleteConfirm({required Object id}) => '删除 MCP 服务器 "${id}"?';
	@override String get removedToast => 'MCP 服务器已移除';
	@override String get deleteFailedToast => '删除失败';
	@override String get toggleFailedToast => '切换失败';
	@override late final _TranslationsWebPluginsMcpEditorZh editor = _TranslationsWebPluginsMcpEditorZh._(_root);
}

// Path: web.plugins.mcpSecrets
class _TranslationsWebPluginsMcpSecretsZh extends TranslationsWebPluginsMcpSecretsEn {
	_TranslationsWebPluginsMcpSecretsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP 密钥';
	@override String get encryptedBadge => '已加密';
	@override String get plaintextBadge => '明文';
	@override String get encryptedTooltip => 'AES-GCM 在磁盘上加密；密钥位于操作系统 keychain';
	@override String get plaintextTooltip => '操作系统 keychain 不可用 — 文件以明文存储。请检查网关日志。';
	@override String description({required Object KEY}) => '在任意 <3>mcp.json</3> 中以 <1>\$${KEY}</1> 占位符引用的值在 spawn 时会被替换。<5>已保存的值不会通过 API 返回</5> — 可以覆盖或删除，但无法读回。';
	@override String descriptionStored({required Object path}) => ' 存储于 <1>${path}</1>。';
	@override String get addSecret => '添加密钥';
	@override String empty({required Object KEY}) => '暂无已存密钥。添加后即可在 MCP 服务器配置中以 <1>\$${KEY}</1> 引用。';
	@override late final _TranslationsWebPluginsMcpSecretsColumnsZh columns = _TranslationsWebPluginsMcpSecretsColumnsZh._(_root);
	@override String get editTooltip => '覆盖已存的值';
	@override String deleteConfirm({required Object key}) => '删除密钥 "${key}"? 任何引用 \$${key} 的 mcp.json 在你重新设置之前都会回退到字面占位符。';
	@override String get removedToast => '密钥已移除';
	@override String get deleteFailedToast => '删除失败';
	@override late final _TranslationsWebPluginsMcpSecretsEditorZh editor = _TranslationsWebPluginsMcpSecretsEditorZh._(_root);
}

// Path: web.plugins.skills
class _TranslationsWebPluginsSkillsZh extends TranslationsWebPluginsSkillsEn {
	_TranslationsWebPluginsSkillsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Agent skills';
	@override String get description => '作为 Tier 1 索引注入到 Claude 会话的可复用能力 — agent 通过 <1>opendray skill describe &lt;id&gt;</1> 按需加载完整 SKILL.md。内置 skill 随二进制发布但可被 <3>自定义</3> — 你的修改保存到 <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> 并覆盖内置版本。点击重置可还原。';
	@override String get newSkill => '新建 skill';
	@override String get empty => '未找到任何 skill。';
	@override late final _TranslationsWebPluginsSkillsColumnsZh columns = _TranslationsWebPluginsSkillsColumnsZh._(_root);
	@override String get noDescription => '无描述';
	@override String get builtinBadge => '内置';
	@override String get builtinTooltip => '嵌入 opendray 二进制 — 点击自定义可在 vault 中覆盖';
	@override String get vaultBadge => 'vault';
	@override String get overridesBuiltin => '覆盖内置';
	@override String get overridesBuiltinTooltip => '此 vault skill 覆盖了同 id 的内置版本';
	@override String get customize => '自定义';
	@override String get customizeTooltip => '打开 SKILL.md 并保存为 vault 覆盖';
	@override String get editTooltip => '编辑此 vault skill';
	@override String get resetTooltip => '删除 vault 覆盖并回退到内置版本';
	@override String get reset => '重置';
	@override String resetConfirm({required Object id}) => '将 "${id}" 重置为内置版本? 这会删除你的 vault SKILL.md 并回退到嵌入副本。';
	@override String deleteConfirm({required Object id}) => '从 vault 删除 skill "${id}"? 这会移除该 SKILL.md 文件。';
	@override String get removedToast => 'Skill 已移除';
	@override String get deleteFailedToast => '删除失败';
	@override late final _TranslationsWebPluginsSkillsEditorZh editor = _TranslationsWebPluginsSkillsEditorZh._(_root);
}

// Path: web.plugins.customTasks
class _TranslationsWebPluginsCustomTasksZh extends TranslationsWebPluginsCustomTasksEn {
	_TranslationsWebPluginsCustomTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '自定义任务';
	@override String get description => '在 Tasks 选项卡中以点选即运行的方式呈现的快捷方式。留空 cwd 即为所有会话可见的全局任务，或填写绝对路径以限定 scope。';
	@override String get addTask => '添加任务';
	@override String get empty => '尚无自定义任务。';
	@override late final _TranslationsWebPluginsCustomTasksColumnsZh columns = _TranslationsWebPluginsCustomTasksColumnsZh._(_root);
	@override String get globalScope => '全局';
	@override String deleteConfirm({required Object name}) => '删除自定义任务 "${name}"?';
	@override String get removedToast => '任务已移除';
	@override String get deleteFailedToast => '删除失败';
	@override late final _TranslationsWebPluginsCustomTasksDialogZh dialog = _TranslationsWebPluginsCustomTasksDialogZh._(_root);
}

// Path: web.plugins.gitHosts
class _TranslationsWebPluginsGitHostsZh extends TranslationsWebPluginsGitHostsEn {
	_TranslationsWebPluginsGitHostsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Git 主机';
	@override String get description => '每个主机一个 token — 被 Git 选项卡用来拉取 pull request，<1>也被 Notes vault sync</1> 使用（当其 remote 通过 HTTPS 指向同一主机上的私有仓库时）。支持 GitHub.com、自托管 GitHub Enterprise、Gitea 与 GitLab。';
	@override String get addHost => '添加主机';
	@override String get empty => '尚未配置任何 git 主机。\n添加一个以在检查器 Git 选项卡中启用 PR 列表。';
	@override late final _TranslationsWebPluginsGitHostsColumnsZh columns = _TranslationsWebPluginsGitHostsColumnsZh._(_root);
	@override String get statusEnabled => '已启用';
	@override String get statusDisabled => '已禁用';
	@override String deleteConfirm({required Object host}) => '移除 git 主机 ${host}? 对该主机的 PR 查询将停止工作。';
	@override String get removedToast => 'Git 主机已移除';
	@override String get deleteFailedToast => '删除失败';
	@override late final _TranslationsWebPluginsGitHostsDialogZh dialog = _TranslationsWebPluginsGitHostsDialogZh._(_root);
}

// Path: web.backups.tabs
class _TranslationsWebBackupsTabsZh extends TranslationsWebBackupsTabsEn {
	_TranslationsWebBackupsTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get backups => '备份';
	@override String get schedules => '计划';
	@override String get targets => '目标';
}

// Path: web.backups.inventory
class _TranslationsWebBackupsInventoryZh extends TranslationsWebBackupsInventoryEn {
	_TranslationsWebBackupsInventoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份里包含什么？';
	@override String summary({required Object tables, required Object rows}) => '${tables} 张表共 ${rows} 行';
	@override String get description => '每次备份是一个对下方所有表执行 <1>pg_dump --format=custom</1> 的产物，外加 <3>manifest.json</3> 和（可选的）<5>config.toml</5>。计数是实时的；bundle 捕获的是备份发生那一刻的数据。';
	@override String get loadFailedToast => '加载清单失败';
	@override String get rowsLabel => '行';
}

// Path: web.backups.restart
class _TranslationsWebBackupsRestartZh extends TranslationsWebBackupsRestartEn {
	_TranslationsWebBackupsRestartZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '请重启 opendray 以激活备份';
	@override String get description => '你的口令已保存。网关仅在启动时加载它，因此该功能在进程重启之前保持关闭。';
	@override String get keyFile => '密钥文件：';
	@override String get configuredVia => '配置方式：';
	@override String get envVar => 'OPENDRAY_BACKUP_KEY 环境变量';
	@override String get checkAgain => '再次检查';
}

// Path: web.backups.setup
class _TranslationsWebBackupsSetupZh extends TranslationsWebBackupsSetupEn {
	_TranslationsWebBackupsSetupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置备份';
	@override String get description => '选择一个主口令。opendray 会用它加密每个备份 blob。<1>丢失它意味着你的备份无法恢复</1>，请在继续之前把它保存到密码管理器（Vaultwarden、1Password 等）。';
	@override String get generate => '生成';
	@override String get pasteOwn => '粘贴自定义';
	@override String get generateTitle => '256 位随机密钥';
	@override String get generateHint => '服务端生成一个加密随机的口令并仅显示一次。你必须在继续前复制它 — 没有恢复路径。';
	@override String get pasteLabel => '你的口令';
	@override String get pastePlaceholder => '至少 20 个字符';
	@override String get pasteHint => '建议：使用密码管理器生成 40 个以上字符。';
	@override String get savesTo => '保存到：';
	@override String get saving => '保存中…';
	@override String get generateAndSave => '生成并保存';
	@override String get save => '保存';
}

// Path: web.backups.generated
class _TranslationsWebBackupsGeneratedZh extends TranslationsWebBackupsGeneratedEn {
	_TranslationsWebBackupsGeneratedZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '立即保存此口令';
	@override String get description => '这是 <1>唯一一次</1> 显示。opendray 与任何其他地方都将无法取回。请在继续前复制到密码管理器。';
	@override String get copy => '复制';
	@override String get copiedToast => '口令已复制到剪贴板';
	@override String get copyFailedToast => '复制失败 — 请手动选中并复制';
	@override String get savedTo => '已保存到：';
	@override String get ack => '我已将此口令保存到密码管理器';
	@override String get kContinue => '继续';
}

// Path: web.backups.status
class _TranslationsWebBackupsStatusZh extends TranslationsWebBackupsStatusEn {
	_TranslationsWebBackupsStatusZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get keyFingerprint => '密钥指纹：';
	@override String get pgDump => 'pg_dump：';
	@override String get pgDumpUnavailable => '不可用';
	@override String get pgDumpHint => '备份在 pg_dump 进入 PATH 之前无法运行（也可通过 <1>backup.pg_dump_path</1> 设置绝对路径）。请安装与你的服务器主版本匹配的 <3>postgresql-client</3> 并重启。';
}

// Path: web.backups.backupsTab
class _TranslationsWebBackupsBackupsTabZh extends TranslationsWebBackupsBackupsTabEn {
	_TranslationsWebBackupsBackupsTabZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get backupNow => '立即备份';
	@override String get triggering => '触发中…';
	@override String get includeConfig => '包含 config.toml';
	@override String get restoreFromFile => '从文件恢复';
	@override String get refresh => '刷新';
	@override String get queuedToast => '备份已排队';
	@override String get triggerFailedToast => '触发失败';
	@override String get listFailedToast => '加载备份列表失败';
	@override String deleteConfirm({required Object id}) => '删除备份 ${id}? 该 blob 将从目标中移除。';
	@override String get deletedToast => '备份已删除';
	@override String get deleteFailedToast => '删除失败';
	@override String get empty => '暂无备份。点击上方 "立即备份" 进行第一次。';
	@override late final _TranslationsWebBackupsBackupsTabColumnsZh columns = _TranslationsWebBackupsBackupsTabColumnsZh._(_root);
	@override String get downloadTooltip => '下载';
	@override String get deleteTooltip => '删除';
}

// Path: web.backups.restore
class _TranslationsWebBackupsRestoreZh extends TranslationsWebBackupsRestoreEn {
	_TranslationsWebBackupsRestoreZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '从备份 bundle 恢复';
	@override String get bundleLabel => '加密 bundle（.tar.gz.enc）';
	@override String get targetDsnLabel => '目标数据库 DSN';
	@override String get targetDsnHint => '（留空 = opendray 自己的 DB — 危险）';
	@override String get targetDsnPlaceholder => 'postgres://user:pass@host:5432/dbname';
	@override String get cleanLabel => '--clean --if-exists（先 drop 现有 schema；恢复到已填充的 DB 上时必需）';
	@override String get auditNoteLabel => '审计备注（可选）';
	@override String get auditNotePlaceholder => '恢复原因 — 会出现在 slog 中';
	@override String get ownDbWarning => '你正在恢复到 <1>opendray 自己的数据库</1>。启用 "--clean" 时会 drop 每张表并按字面回放备份 — 不可逆。请输入 <3>I understand</3> 以继续。';
	@override String get confirmPlaceholder => 'I understand';
	@override String get confirmSentinel => 'I understand';
	@override String get pgRestoreOutput => 'pg_restore 输出（最后 8 KiB）';
	@override String get noPgRestoreOutput => '（无 pg_restore 输出）';
	@override String get pickFileToast => '请先选择一个 bundle 文件';
	@override String get succeededToast => '恢复成功';
	@override String replayedDescription({required Object bytes, required Object id}) => '已回放 ${bytes}，来自 manifest ${id}';
	@override String get failedToast => '恢复失败';
	@override String get restoring => '恢复中…';
	@override String get restore => '恢复';
}

// Path: web.backups.schedulesTab
class _TranslationsWebBackupsSchedulesTabZh extends TranslationsWebBackupsSchedulesTabEn {
	_TranslationsWebBackupsSchedulesTabZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '周期备份。调度器每 30 秒轮询一次，并运行最旧的到期计划。';
	@override String get newSchedule => '新建计划';
	@override String get loadFailedToast => '加载计划失败';
	@override String deleteConfirm({required Object id}) => '删除计划 ${id}?';
	@override String get deletedToast => '计划已删除';
	@override String get deleteFailedToast => '删除失败';
	@override String get toggleFailedToast => '切换失败';
	@override String get empty => '暂无计划。添加一个以进行周期性自动备份。';
	@override late final _TranslationsWebBackupsSchedulesTabColumnsZh columns = _TranslationsWebBackupsSchedulesTabColumnsZh._(_root);
	@override String keepCount({required Object count}) => '${count} 个备份';
	@override String get deleteTooltip => '删除';
}

// Path: web.backups.newSchedule
class _TranslationsWebBackupsNewScheduleZh extends TranslationsWebBackupsNewScheduleEn {
	_TranslationsWebBackupsNewScheduleZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建备份计划';
	@override String get targetLabel => '目标';
	@override String get everyHoursLabel => '每隔（小时）';
	@override String get keepLastNLabel => '保留最近 N 个';
	@override String get enableImmediately => '立即启用';
	@override String get createdToast => '计划已创建';
	@override String get createFailedToast => '创建失败';
	@override String get creating => '创建中…';
	@override String get create => '创建';
}

// Path: web.backups.targetsTab
class _TranslationsWebBackupsTargetsTabZh extends TranslationsWebBackupsTargetsTabEn {
	_TranslationsWebBackupsTargetsTabZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '存储目标。v1 支持 <1>local</1>（opendray 主机磁盘）与 <3>smb</3>（任意 SMB / CIFS 共享，如 UNAS 或群晖）。';
	@override String get newTarget => '新建目标';
	@override String get listFailedToast => '加载目标列表失败';
	@override String deleteConfirm({required Object id}) => '删除目标 ${id}? 引用它的计划会阻止删除。';
	@override String get deletedToast => '目标已删除';
	@override String get deleteFailedToast => '删除失败';
	@override String get connectionOkToast => '连接成功';
	@override String get connectionFailedToast => '连接失败';
	@override String get testFailedToast => '测试失败';
	@override late final _TranslationsWebBackupsTargetsTabColumnsZh columns = _TranslationsWebBackupsTargetsTabColumnsZh._(_root);
	@override String get on => '开';
	@override String get off => '关';
	@override String get test => '测试';
	@override String get testing => '测试中…';
	@override String get deleteTooltip => '删除';
}

// Path: web.backups.targetEditor
class _TranslationsWebBackupsTargetEditorZh extends TranslationsWebBackupsTargetEditorEn {
	_TranslationsWebBackupsTargetEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建备份目标';
	@override String get kindPicker => '你想备份到哪里？';
	@override String get idLabel => 'ID（可选）';
	@override String get idPlaceholder => '留空则自动生成，例如 tgt_xxx';
	@override String get createdToast => '目标已创建';
	@override String get createFailedToast => '创建失败';
	@override String get creating => '创建中…';
	@override String get create => '创建目标';
	@override String get enableImmediately => '立即启用（否则保存为禁用 — 适合 "先配置好，稍后开启"）';
	@override late final _TranslationsWebBackupsTargetEditorLocalZh local = _TranslationsWebBackupsTargetEditorLocalZh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorSmbZh smb = _TranslationsWebBackupsTargetEditorSmbZh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorS3Zh s3 = _TranslationsWebBackupsTargetEditorS3Zh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorWebdavZh webdav = _TranslationsWebBackupsTargetEditorWebdavZh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorSftpZh sftp = _TranslationsWebBackupsTargetEditorSftpZh._(_root);
	@override late final _TranslationsWebBackupsTargetEditorRcloneZh rclone = _TranslationsWebBackupsTargetEditorRcloneZh._(_root);
}

// Path: web.serverSettings.sections
class _TranslationsWebServerSettingsSectionsZh extends TranslationsWebServerSettingsSectionsEn {
	_TranslationsWebServerSettingsSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsSectionsGeneralZh general = _TranslationsWebServerSettingsSectionsGeneralZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsLoggingZh logging = _TranslationsWebServerSettingsSectionsLoggingZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsSessionsZh sessions = _TranslationsWebServerSettingsSectionsSessionsZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsVaultZh vault = _TranslationsWebServerSettingsSectionsVaultZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsMcpZh mcp = _TranslationsWebServerSettingsSectionsMcpZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsMemoryZh memory = _TranslationsWebServerSettingsSectionsMemoryZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsMemoryAmbientZh memoryAmbient = _TranslationsWebServerSettingsSectionsMemoryAmbientZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsBackupZh backup = _TranslationsWebServerSettingsSectionsBackupZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsClaudeZh claude = _TranslationsWebServerSettingsSectionsClaudeZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsCodexZh codex = _TranslationsWebServerSettingsSectionsCodexZh._(_root);
	@override late final _TranslationsWebServerSettingsSectionsGeminiZh gemini = _TranslationsWebServerSettingsSectionsGeminiZh._(_root);
}

// Path: web.serverSettings.restart
class _TranslationsWebServerSettingsRestartZh extends TranslationsWebServerSettingsRestartEn {
	_TranslationsWebServerSettingsRestartZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get button => '重启服务器';
	@override String get buttonTitle => '对网关进程执行 self-exec';
	@override String get dirtyConfirm => '您有未保存的修改。重启将使用「上次保存」的配置，是否继续？';
	@override String get confirm => '重启 opendray 网关？所有打开的终端会话将自动重新连接。';
	@override String get overlay => '正在重启服务器…';
	@override String waiting({required Object tick}) => '等待 /health · ${tick}s';
	@override String get timedOutTitle => '重启超时';
	@override String get timedOutDesc => 'Health 接口未恢复。请查看服务器日志。';
	@override String get successToast => '服务器已重启';
}

// Path: web.serverSettings.formGroups
class _TranslationsWebServerSettingsFormGroupsZh extends TranslationsWebServerSettingsFormGroupsEn {
	_TranslationsWebServerSettingsFormGroupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get network => '网络';
	@override String get operatorAccount => '操作员账号';
	@override String get memoryConfiguration => '配置';
	@override String get memoryHttp => 'HTTP 后端（当 backend=http 时使用）';
	@override String get memoryLocal => '本地 ONNX（当 backend=local 时使用）';
	@override String get backupStatus => '状态';
	@override String get backupWhere => '备份目标位置';
	@override String get backupSchedules => '计划任务';
	@override String get backupWhatsInside => '备份里有什么？';
}

// Path: web.serverSettings.fields
class _TranslationsWebServerSettingsFieldsZh extends TranslationsWebServerSettingsFieldsEn {
	_TranslationsWebServerSettingsFieldsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebServerSettingsFieldsListenAddressZh listenAddress = _TranslationsWebServerSettingsFieldsListenAddressZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsUsernameZh username = _TranslationsWebServerSettingsFieldsUsernameZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsPasswordZh password = _TranslationsWebServerSettingsFieldsPasswordZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsTokenTTLZh tokenTTL = _TranslationsWebServerSettingsFieldsTokenTTLZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogLevelZh logLevel = _TranslationsWebServerSettingsFieldsLogLevelZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogFormatZh logFormat = _TranslationsWebServerSettingsFieldsLogFormatZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsLogFileZh logFile = _TranslationsWebServerSettingsFieldsLogFileZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsIdleThresholdZh idleThreshold = _TranslationsWebServerSettingsFieldsIdleThresholdZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsIdlePollIntervalZh idlePollInterval = _TranslationsWebServerSettingsFieldsIdlePollIntervalZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsVaultRootZh vaultRoot = _TranslationsWebServerSettingsFieldsVaultRootZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsNotesDirectoryZh notesDirectory = _TranslationsWebServerSettingsFieldsNotesDirectoryZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsSkillsDirectoryZh skillsDirectory = _TranslationsWebServerSettingsFieldsSkillsDirectoryZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGitRootZh gitRoot = _TranslationsWebServerSettingsFieldsGitRootZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsPersonalPrefixZh personalPrefix = _TranslationsWebServerSettingsFieldsPersonalPrefixZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsProjectsPrefixZh projectsPrefix = _TranslationsWebServerSettingsFieldsProjectsPrefixZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsRegistryRootZh registryRoot = _TranslationsWebServerSettingsFieldsRegistryRootZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsSecretsFileZh secretsFile = _TranslationsWebServerSettingsFieldsSecretsFileZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryBackendZh memoryBackend = _TranslationsWebServerSettingsFieldsMemoryBackendZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryStoreZh memoryStore = _TranslationsWebServerSettingsFieldsMemoryStoreZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryTopKZh memoryTopK = _TranslationsWebServerSettingsFieldsMemoryTopKZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryThresholdZh memoryThreshold = _TranslationsWebServerSettingsFieldsMemoryThresholdZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryScopeZh memoryScope = _TranslationsWebServerSettingsFieldsMemoryScopeZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryBaseUrlZh memoryBaseUrl = _TranslationsWebServerSettingsFieldsMemoryBaseUrlZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryModelZh memoryModel = _TranslationsWebServerSettingsFieldsMemoryModelZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryApiKeyZh memoryApiKey = _TranslationsWebServerSettingsFieldsMemoryApiKeyZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryLocalModelZh memoryLocalModel = _TranslationsWebServerSettingsFieldsMemoryLocalModelZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryLibraryPathZh memoryLibraryPath = _TranslationsWebServerSettingsFieldsMemoryLibraryPathZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryModelPathZh memoryModelPath = _TranslationsWebServerSettingsFieldsMemoryModelPathZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryTokenizerPathZh memoryTokenizerPath = _TranslationsWebServerSettingsFieldsMemoryTokenizerPathZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenZh memoryMaxSeqLen = _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeHistoryRootsZh claudeHistoryRoots = _TranslationsWebServerSettingsFieldsClaudeHistoryRootsZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsClaudeAccountsDirZh claudeAccountsDir = _TranslationsWebServerSettingsFieldsClaudeAccountsDirZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsCodexSessionsRootZh codexSessionsRoot = _TranslationsWebServerSettingsFieldsCodexSessionsRootZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGeminiTmpRootZh geminiTmpRoot = _TranslationsWebServerSettingsFieldsGeminiTmpRootZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsGeminiProjectsFileZh geminiProjectsFile = _TranslationsWebServerSettingsFieldsGeminiProjectsFileZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupLocalDirZh backupLocalDir = _TranslationsWebServerSettingsFieldsBackupLocalDirZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupExportDirZh backupExportDir = _TranslationsWebServerSettingsFieldsBackupExportDirZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupPgDumpPathZh backupPgDumpPath = _TranslationsWebServerSettingsFieldsBackupPgDumpPathZh._(_root);
	@override late final _TranslationsWebServerSettingsFieldsBackupPgRestorePathZh backupPgRestorePath = _TranslationsWebServerSettingsFieldsBackupPgRestorePathZh._(_root);
}

// Path: web.serverSettings.liveTail
class _TranslationsWebServerSettingsLiveTailZh extends TranslationsWebServerSettingsLiveTailEn {
	_TranslationsWebServerSettingsLiveTailZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get heading => '实时日志';
	@override String get description => '内存中的环形缓冲区（最近约 2,000 条）。重启后清空。';
}

// Path: web.serverSettings.memoryInspectorCard
class _TranslationsWebServerSettingsMemoryInspectorCardZh extends TranslationsWebServerSettingsMemoryInspectorCardEn {
	_TranslationsWebServerSettingsMemoryInspectorCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get heading => '检查器';
	@override String get description => '在专门页面浏览、搜索、编辑已存储的记忆。';
	@override String get openButton => '打开记忆 →';
}

// Path: web.serverSettings.stringList
class _TranslationsWebServerSettingsStringListZh extends TranslationsWebServerSettingsStringListEn {
	_TranslationsWebServerSettingsStringListZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get noneDefault => '（无 — 使用内置默认值）';
	@override String get addPath => '添加路径';
	@override String get removeTitle => '移除';
}

// Path: web.serverSettings.httpHelpers
class _TranslationsWebServerSettingsHttpHelpersZh extends TranslationsWebServerSettingsHttpHelpersEn {
	_TranslationsWebServerSettingsHttpHelpersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get autoDetected => '启动时自动检测到';
	@override String modelCount({required Object count}) => '${count} 个模型 — 点击使用';
	@override String get presets => '预设：';
	@override String get testConnection => '测试连接';
	@override late final _TranslationsWebServerSettingsHttpHelpersPresetTipZh presetTip = _TranslationsWebServerSettingsHttpHelpersPresetTipZh._(_root);
}

// Path: web.serverSettings.probe
class _TranslationsWebServerSettingsProbeZh extends TranslationsWebServerSettingsProbeEn {
	_TranslationsWebServerSettingsProbeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String unreachable({required Object error}) => '✗ 不可达：${error}';
	@override String get connectionFailed => '连接失败';
	@override String reachable({required Object detected, required Object total, required Object embedding}) => '✓ 可达 ${detected}· 共 ${total} 个模型 · ${embedding} 个嵌入';
	@override String modelMissing({required Object model}) => '⚠ 配置的模型 ${model} 不在列表中。从下方嵌入模型中选一个，或修正名称。';
	@override String get embeddingModelsLabel => '嵌入模型：';
	@override String moreModels({required Object count}) => '还有 ${count} 个';
	@override String get noEmbeddingFound => '⚠ 没有模型名包含 "embed"。该端点可能未加载嵌入模型 — 请检查本地服务。';
	@override String get configuredTitle => '当前已配置';
	@override String get applyTitle => '点击应用';
}

// Path: web.serverSettings.backup
class _TranslationsWebServerSettingsBackupZh extends TranslationsWebServerSettingsBackupEn {
	_TranslationsWebServerSettingsBackupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get featureDisabledTitle => '功能已禁用';
	@override String get featureDisabledHint => '在 opendray 的环境变量中设置 <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3>，然后重启。主密码仅来自环境变量 — 永不写入 config.toml。';
	@override String get statusRowLabel => '状态';
	@override String get enabledHealthy => '已启用 · 健康';
	@override String get enabledDegraded => '已启用 · 异常';
	@override String get keyFingerprintLabel => '密钥指纹';
	@override String get keyFingerprintHint => '记录到 Vaultwarden — 丢失会锁死所有先前的备份';
	@override String get pgDumpLabel => 'pg_dump';
	@override String get pgDumpUnavailable => '不可用';
	@override String get pgRestoreLabel => 'pg_restore';
	@override String get pgRestoreNotResolved => '（未解析）';
	@override String get openBackups => '打开备份页 →';
	@override String get openExport => '打开导出 / 导入 →';
	@override String get whereDesc => '每个目标都是备份块可写入的一个地方。opendray 支持 <1>本地磁盘</1>、<3>SMB/CIFS</3>（Windows / NAS）、<5>S3 兼容</5>（AWS、R2、B2、MinIO、阿里云 OSS、腾讯云 COS …）、<7>WebDAV</7>（Nextcloud、群晖、坚果云）、<9>SFTP</9>，外加 <11>rclone</11> 透传，接入另外 70+ 个后端（Google Drive、OneDrive、Dropbox、百度云、阿里云盘 …）。';
	@override String get loading => '加载中…';
	@override String get noTargets => '尚未添加目标。添加一个开始备份。';
	@override String get addTarget => '添加目标';
	@override String get noSchedulesHint => '没有循环计划。在 <1>/backups → 计划任务</1> 添加一个以自动执行备份。';
	@override late final _TranslationsWebServerSettingsBackupScheduleHeadersZh scheduleHeaders = _TranslationsWebServerSettingsBackupScheduleHeadersZh._(_root);
	@override String every({required Object interval}) => '每 ${interval}';
	@override String backupsKeep({required Object count}) => '${count} 份备份';
	@override String get stateEnabled => '已启用';
	@override String get statePaused => '已暂停';
	@override String get manageSchedules => '在 /backups → 计划任务 管理 →';
	@override String get whatsInsideDesc => '每份备份都是所有 opendray 表（sessions、integrations、memories、audit_log 等）的 <1>pg_dump --format=custom</1>，加上一个 <3>manifest.json</3>，以及（可选）当前的 <5>config.toml</5>。在 <7>备份页</7> 打开「备份里有什么？」面板可以看到带行数的实时清单。';
	@override String get advancedToggle => '高级选项（路径与客户端二进制）— 需要重启';
}

// Path: web.serverSettings.targetRow
class _TranslationsWebServerSettingsTargetRowZh extends TranslationsWebServerSettingsTargetRowEn {
	_TranslationsWebServerSettingsTargetRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get on => '开';
	@override String get off => '关';
	@override String get test => '测试';
	@override String get testing => '测试中…';
	@override String get delete => '删除';
	@override String connectionOk({required Object id}) => '${id}：连接正常';
	@override String get connectionFailedTitle => '连接失败';
	@override String get testFailedTitle => '测试失败';
	@override String deleteConfirm({required Object id}) => '删除目标 "${id}"？引用它的计划任务将阻止删除。';
	@override String get deleteSuccess => '目标已删除';
	@override String get deleteFailedTitle => '删除失败';
	@override String get unknownError => '未知错误';
}

// Path: web.settings.groups
class _TranslationsWebSettingsGroupsZh extends TranslationsWebSettingsGroupsEn {
	_TranslationsWebSettingsGroupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get workspace => '工作区';
	@override String get server => '服务器';
	@override String get system => '系统';
}

// Path: web.settings.items
class _TranslationsWebSettingsItemsZh extends TranslationsWebSettingsItemsEn {
	_TranslationsWebSettingsItemsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get appearance => '外观';
	@override String get font => '字号';
	@override String get account => '账号';
	@override String get status => '状态';
	@override String get about => '关于';
}

// Path: web.settings.health
class _TranslationsWebSettingsHealthZh extends TranslationsWebSettingsHealthEn {
	_TranslationsWebSettingsHealthZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get connecting => '连接中…';
	@override String get dbOk => 'db 正常';
	@override String get dbDown => 'db 异常';
}

// Path: web.settings.breadcrumb
class _TranslationsWebSettingsBreadcrumbZh extends TranslationsWebSettingsBreadcrumbEn {
	_TranslationsWebSettingsBreadcrumbZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get server => '服务器';
}

// Path: web.settings.appearance
class _TranslationsWebSettingsAppearanceZh extends TranslationsWebSettingsAppearanceEn {
	_TranslationsWebSettingsAppearanceZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '外观';
	@override String get description => '选择 opendray 的外观风格。';
	@override late final _TranslationsWebSettingsAppearanceOptionsZh options = _TranslationsWebSettingsAppearanceOptionsZh._(_root);
}

// Path: web.settings.font
class _TranslationsWebSettingsFontZh extends TranslationsWebSettingsFontEn {
	_TranslationsWebSettingsFontZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '字号';
	@override String get description => '缩放整个界面。按浏览器保存。';
	@override late final _TranslationsWebSettingsFontOptionsZh options = _TranslationsWebSettingsFontOptionsZh._(_root);
}

// Path: web.settings.account
class _TranslationsWebSettingsAccountZh extends TranslationsWebSettingsAccountEn {
	_TranslationsWebSettingsAccountZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '账号';
	@override String get description => '运维与当前 bearer token。';
	@override String get username => '用户名';
	@override String get tokenExpires => 'Token 过期';
	@override String get changeCredentials => '修改凭证';
}

// Path: web.settings.changeCredentials
class _TranslationsWebSettingsChangeCredentialsZh extends TranslationsWebSettingsChangeCredentialsEn {
	_TranslationsWebSettingsChangeCredentialsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '修改凭证';
	@override String get description => '先验证当前密码，再选择新凭证。所有其它已登录会话都将被吊销。';
	@override String get currentPassword => '当前密码';
	@override String get newUsername => '新用户名';
	@override String get newPassword => '新密码';
	@override String get newPasswordHint => '至少 8 个字符。';
	@override String get confirm => '确认新密码';
	@override String get errorTooShort => '新密码至少 8 个字符。';
	@override String get errorMismatch => '新密码和确认不一致。';
	@override String get errorWrongPassword => '当前密码不正确。';
	@override String get cancel => '取消';
	@override String get update => '更新';
	@override String get saving => '保存中…';
}

// Path: web.settings.system
class _TranslationsWebSettingsSystemZh extends TranslationsWebSettingsSystemEn {
	_TranslationsWebSettingsSystemZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '系统状态';
	@override String get description => '来自网关 /health 接口的实时状态。';
	@override String get status => '状态';
	@override String get version => '版本';
	@override String get uptime => '运行时长';
	@override String get database => '数据库';
	@override String get reachable => '可达';
	@override String get unreachable => '不可达';
}

// Path: web.settings.about
class _TranslationsWebSettingsAboutZh extends TranslationsWebSettingsAboutEn {
	_TranslationsWebSettingsAboutZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '关于';
	@override String get description => 'opendray v2 — 面向 AI agent CLI 的多路复用 + 集成网关。源码采用 Apache 2.0 协议。';
}

// Path: web.memoryAmbient.header
class _TranslationsWebMemoryAmbientHeaderZh extends TranslationsWebMemoryAmbientHeaderEn {
	_TranslationsWebMemoryAmbientHeaderZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '环境记忆 — 自动捕获与注入';
	@override String get body => 'opendray 每 10 秒轮询所有运行中的 agent 会话，通过可配置的 LLM 提取持久事实，去重后存入共享记忆池。配置由哪个 LLM 做提取（Provider）、何时触发提取（Capture rule）、以及在 spawn 时把什么内容（如果有）拼接到 agent 的 system prompt（Injection profile）。';
}

// Path: web.memoryAmbient.providers
class _TranslationsWebMemoryAmbientProvidersZh extends TranslationsWebMemoryAmbientProvidersEn {
	_TranslationsWebMemoryAmbientProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Summarizer Providers';
	@override String get addButton => '添加 provider';
	@override String get intro => '至少需要一个已启用的 provider 才能真正触发捕获。本地选项（Ollama、LM Studio、Integration）让你的会话内容不出外网。';
	@override String get empty => '尚未配置 provider。';
	@override late final _TranslationsWebMemoryAmbientProvidersRowZh row = _TranslationsWebMemoryAmbientProvidersRowZh._(_root);
	@override late final _TranslationsWebMemoryAmbientProvidersDialogZh dialog = _TranslationsWebMemoryAmbientProvidersDialogZh._(_root);
}

// Path: web.memoryAmbient.rules
class _TranslationsWebMemoryAmbientRulesZh extends TranslationsWebMemoryAmbientRulesEn {
	_TranslationsWebMemoryAmbientRulesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '捕获规则';
	@override String get addButton => '添加规则';
	@override String get intro => '每条规则表示 "当此 trigger 触发时，对新的会话消息做总结并存储持久事实。" 单会话规则覆盖全局默认。v1 内置 4 种 trigger 类型。';
	@override String get empty => '尚无捕获规则。添加一条以启用自动捕获。';
	@override late final _TranslationsWebMemoryAmbientRulesRowZh row = _TranslationsWebMemoryAmbientRulesRowZh._(_root);
	@override late final _TranslationsWebMemoryAmbientRulesDialogZh dialog = _TranslationsWebMemoryAmbientRulesDialogZh._(_root);
}

// Path: web.memoryAmbient.profiles
class _TranslationsWebMemoryAmbientProfilesZh extends TranslationsWebMemoryAmbientProfilesEn {
	_TranslationsWebMemoryAmbientProfilesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Injection Profiles';
	@override String get addButton => '添加 profile';
	@override String get intro => 'spawn 时，opendray 会把最近的项目记忆作为一段 markdown banner 拼接到 agent 的 system prompt — 前提是配置了 profile。没有 profile 时，模型仍可按需调用 memory_search。';
	@override String get empty => '尚无 injection profile。spawn 时不会自动注入记忆 — 模型仍可使用 memory_search。';
	@override late final _TranslationsWebMemoryAmbientProfilesRowZh row = _TranslationsWebMemoryAmbientProfilesRowZh._(_root);
	@override late final _TranslationsWebMemoryAmbientProfilesDialogZh dialog = _TranslationsWebMemoryAmbientProfilesDialogZh._(_root);
}

// Path: web.memoryAmbient.cost
class _TranslationsWebMemoryAmbientCostZh extends TranslationsWebMemoryAmbientCostEn {
	_TranslationsWebMemoryAmbientCostZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Token 成本（总计）';
	@override String get intro => '按 provider 聚合自 <1>memory_summarizer_calls</1>。本地 provider（Ollama、LM Studio、Integration）按 \$0 计价 — 硬件成本由运维承担。';
	@override String get empty => '暂无已启用的 provider — 没有成本数据。';
	@override late final _TranslationsWebMemoryAmbientCostColumnsZh columns = _TranslationsWebMemoryAmbientCostColumnsZh._(_root);
}

// Path: web.noteEditor.status
class _TranslationsWebNoteEditorStatusZh extends TranslationsWebNoteEditorStatusEn {
	_TranslationsWebNoteEditorStatusZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get saveFailed => '保存失败';
	@override String get saving => '保存中…';
	@override String get unsaved => '未保存';
	@override String get newNote => '新笔记';
	@override String get saved => '已保存';
}

// Path: web.export.sections
class _TranslationsWebExportSectionsZh extends TranslationsWebExportSectionsEn {
	_TranslationsWebExportSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get export => '导出';
	@override String get import => '导入';
}

// Path: web.export.form
class _TranslationsWebExportFormZh extends TranslationsWebExportFormEn {
	_TranslationsWebExportFormZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get scope => '范围';
	@override String get memories => '记忆';
	@override String get memoriesHint => '跨 CLI 持久化的记忆行（text + scope + metadata）。向量被省略；导入端重嵌入。';
	@override String get integrations => '集成';
	@override String get customTasks => '自定义任务';
	@override String get customTasksHint => '在 Inspector 的 Tasks 标签里展示的运维自定义任务。';
	@override late final _TranslationsWebExportFormIntegrationOptionsZh integrationOptions = _TranslationsWebExportFormIntegrationOptionsZh._(_root);
	@override String get confirmWarning => '输入 <1>I understand</1> 以确认。opendray 当前只存 bcrypt 哈希 — 选择明文也不会导出任何明文（该选项为将来保留明文缓存的版本而预留）。';
	@override String get confirmPlaceholder => 'I understand';
	@override String get confirmSentinel => 'i understand';
	@override String get footnote => '审计日志与会话记录不在范围内 — 由 /backups（运维 dump）覆盖。';
	@override String get building => '构建中…';
	@override String get create => '创建导出';
	@override String get readyToast => '导出就绪';
	@override String readyDescription({required Object bytes}) => '${bytes} 字节';
	@override String get failedToast => '导出失败';
}

// Path: web.export.history
class _TranslationsWebExportHistoryZh extends TranslationsWebExportHistoryEn {
	_TranslationsWebExportHistoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get empty => '暂无导出。请使用上面的表单创建一个。';
	@override String get title => '历史';
	@override late final _TranslationsWebExportHistoryColumnsZh columns = _TranslationsWebExportHistoryColumnsZh._(_root);
	@override String get download => '下载';
	@override String get deleteTooltip => '删除';
	@override String get listFailedToast => '加载导出列表失败';
	@override String get downloadFailedToast => '下载失败';
	@override String get noTokenToast => '没有下载 token（已过期？）';
	@override String deleteConfirm({required Object id}) => '删除导出 ${id}?';
	@override String get deletedToast => '导出已删除';
	@override String get deleteFailedToast => '删除失败';
	@override String get scopeEmpty => '(空)';
}

// Path: web.export.import
class _TranslationsWebExportImportZh extends TranslationsWebExportImportEn {
	_TranslationsWebExportImportZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get intro => '将一个导出 bundle（zip）回放到当前数据库。冲突项（id 一致，或 integrations 的 route_prefix 唯一冲突）默认 <1>跳过</1>。记忆条目会被标记为 <3>embedder=imported_v1</3>，需要做一次重嵌入后搜索才能返回；可在 <5>Memory → 维护</5> 触发。集成以 <7>enabled=false</7> 导入并使用非 bcrypt 的占位 key — 启用前请轮换。';
	@override String get memoryLink => 'Memory → 维护';
	@override String get bundleLabel => 'Bundle (.zip)';
	@override String get memoriesLabel => '记忆';
	@override String get integrationsLabel => '集成（仅元数据 — 不会导入 key）';
	@override String get customTasksLabel => '自定义任务';
	@override String get importing => '导入中…';
	@override String get importBundle => '导入 bundle';
	@override String get pickFileToast => '请先选择一个 bundle 文件';
	@override String get doneToast => '导入完成';
	@override String get finishedWithErrors => '导入完成但有错误';
	@override String get failedToast => '导入失败';
	@override late final _TranslationsWebExportImportSummaryCardZh summaryCard = _TranslationsWebExportImportSummaryCardZh._(_root);
}

// Path: web.export.imports
class _TranslationsWebExportImportsZh extends TranslationsWebExportImportsEn {
	_TranslationsWebExportImportsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载中…';
	@override String get empty => '暂无导入。';
	@override String get title => '历史';
	@override late final _TranslationsWebExportImportsColumnsZh columns = _TranslationsWebExportImportsColumnsZh._(_root);
	@override String get noneCounts => '(无)';
	@override String get listFailedToast => '加载导入列表失败';
}

// Path: more.items.integrations
class _TranslationsMoreItemsIntegrationsZh extends TranslationsMoreItemsIntegrationsEn {
	_TranslationsMoreItemsIntegrationsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '集成';
	@override String get subtitle => 'API 调用方 — 近期活动与错误率';
}

// Path: more.items.channels
class _TranslationsMoreItemsChannelsZh extends TranslationsMoreItemsChannelsEn {
	_TranslationsMoreItemsChannelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '通道';
	@override String get subtitle => '通知目的地';
}

// Path: more.items.providers
class _TranslationsMoreItemsProvidersZh extends TranslationsMoreItemsProvidersEn {
	_TranslationsMoreItemsProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '提供商';
	@override String get subtitle => 'Claude / Codex / Gemini CLI 状态';
}

// Path: more.items.mcp
class _TranslationsMoreItemsMcpZh extends TranslationsMoreItemsMcpEn {
	_TranslationsMoreItemsMcpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP';
	@override String get subtitle => 'Model Context Protocol 服务与密钥';
}

// Path: more.items.skills
class _TranslationsMoreItemsSkillsZh extends TranslationsMoreItemsSkillsEn {
	_TranslationsMoreItemsSkillsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '技能';
	@override String get subtitle => 'Agent SKILL.md 库（内置 + 库）';
}

// Path: more.items.gitHosts
class _TranslationsMoreItemsGitHostsZh extends TranslationsMoreItemsGitHostsEn {
	_TranslationsMoreItemsGitHostsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Git 主机';
	@override String get subtitle => 'GitHub / GitLab 等的 PAT 凭据';
}

// Path: more.items.customTasks
class _TranslationsMoreItemsCustomTasksZh extends TranslationsMoreItemsCustomTasksEn {
	_TranslationsMoreItemsCustomTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '自定义任务';
	@override String get subtitle => '会话任务选择器中显示的斜杠命令';
}

// Path: more.items.projectMemory
class _TranslationsMoreItemsProjectMemoryZh extends TranslationsMoreItemsProjectMemoryEn {
	_TranslationsMoreItemsProjectMemoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '项目目标 / 计划 / 日志';
	@override String get subtitle => '按 cwd 的记忆层 2-4 + 代理提案';
}

// Path: more.items.cleanupInbox
class _TranslationsMoreItemsCleanupInboxZh extends TranslationsMoreItemsCleanupInboxEn {
	_TranslationsMoreItemsCleanupInboxZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '清理收件箱';
	@override String get subtitle => '跨项目的 LLM 提议删除 / 合并';
}

// Path: more.items.backups
class _TranslationsMoreItemsBackupsZh extends TranslationsMoreItemsBackupsEn {
	_TranslationsMoreItemsBackupsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份';
	@override String get subtitle => '最新备份状态与立即运行';
}

// Path: more.items.settings
class _TranslationsMoreItemsSettingsZh extends TranslationsMoreItemsSettingsEn {
	_TranslationsMoreItemsSettingsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '设置';
	@override String get subtitle => '语言、外观、账户';
}

// Path: more.items.about
class _TranslationsMoreItemsAboutZh extends TranslationsMoreItemsAboutEn {
	_TranslationsMoreItemsAboutZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '关于';
	@override String get subtitle => '构建版本与服务器信息';
}

// Path: sessions.terminal.snackbar
class _TranslationsSessionsTerminalSnackbarZh extends TranslationsSessionsTerminalSnackbarEn {
	_TranslationsSessionsTerminalSnackbarZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String imagePickerFailed({required Object error}) => '图片选择失败：${error}';
	@override String get uploadingImage => '正在上传图片…';
	@override String imageAttached({required Object path}) => '已附加图片：${path}';
	@override String uploadFailed({required Object status, required Object message}) => '上传失败（${status}）：${message}';
	@override String uploadFailedGeneric({required Object error}) => '上传失败：${error}';
}

// Path: sessions.terminal.imageSource
class _TranslationsSessionsTerminalImageSourceZh extends TranslationsSessionsTerminalImageSourceEn {
	_TranslationsSessionsTerminalImageSourceZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get photoLibrary => '相册';
	@override String get takePhoto => '拍照';
}

// Path: sessions.terminal.keyboard
class _TranslationsSessionsTerminalKeyboardZh extends TranslationsSessionsTerminalKeyboardEn {
	_TranslationsSessionsTerminalKeyboardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get copyBuffer => '复制缓冲';
	@override String get paste => '粘贴';
	@override String get attachImage => '附加图片';
	@override String get enter => '回车';
}

// Path: sessions.terminal.connection
class _TranslationsSessionsTerminalConnectionZh extends TranslationsSessionsTerminalConnectionEn {
	_TranslationsSessionsTerminalConnectionZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get connecting => '连接中…';
	@override String get connected => '已连接';
	@override String get reconnecting => '重连中…';
	@override String reconnectingWithError({required Object error}) => '重连中（${error}）…';
	@override String get disconnected => '已断开';
	@override String disconnectedWithError({required Object error}) => '已断开（${error}）';
	@override String get ended => '会话已结束';
}

// Path: sessions.action.errors
class _TranslationsSessionsActionErrorsZh extends TranslationsSessionsActionErrorsEn {
	_TranslationsSessionsActionErrorsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String stop({required Object error}) => '停止失败：${error}';
	@override String start({required Object error}) => '重启失败：${error}';
	@override String delete({required Object error}) => '删除失败：${error}';
}

// Path: sessions.dirPicker.dialog
class _TranslationsSessionsDirPickerDialogZh extends TranslationsSessionsDirPickerDialogEn {
	_TranslationsSessionsDirPickerDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '新建文件夹';
	@override String get hint => '文件夹名';
	@override String get create => '创建';
}

// Path: sessions.inspector.shell
class _TranslationsSessionsInspectorShellZh extends TranslationsSessionsInspectorShellEn {
	_TranslationsSessionsInspectorShellZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '检查器';
	@override String loadError({required Object error}) => '加载会话失败：${error}';
	@override late final _TranslationsSessionsInspectorShellTabsZh tabs = _TranslationsSessionsInspectorShellTabsZh._(_root);
}

// Path: sessions.inspector.shared
class _TranslationsSessionsInspectorSharedZh extends TranslationsSessionsInspectorSharedEn {
	_TranslationsSessionsInspectorSharedZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get refresh => '刷新';
	@override String inserted({required Object text}) => '已插入：${text}';
	@override String insertFailedApi({required Object status, required Object message}) => '插入失败（${status}）：${message}';
	@override String insertFailedGeneric({required Object error}) => '插入失败：${error}';
	@override String insertFailedShort({required Object error}) => '插入失败：${error}';
}

// Path: sessions.inspector.history
class _TranslationsSessionsInspectorHistoryZh extends TranslationsSessionsInspectorHistoryEn {
	_TranslationsSessionsInspectorHistoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get insertIntoTerminal => '插入到终端';
	@override String get searchHint => '搜索提示…';
}

// Path: sessions.inspector.files
class _TranslationsSessionsInspectorFilesZh extends TranslationsSessionsInspectorFilesEn {
	_TranslationsSessionsInspectorFilesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get insertAtRef => '作为 @引用 插入';
	@override String get insertPath => '插入路径';
	@override String get insertPathSubtitle => '原样粘贴绝对路径';
	@override String get readContent => '读取内容';
	@override String get readContentSubtitle => '最多 256 KiB 纯文本';
	@override String readFailedApi({required Object status, required Object message}) => '读取失败（${status}）：${message}';
	@override String readFailedGeneric({required Object error}) => '读取失败：${error}';
	@override String get parent => '上级';
	@override String get backToCwd => '返回会话目录';
}

// Path: sessions.inspector.git
class _TranslationsSessionsInspectorGitZh extends TranslationsSessionsInspectorGitEn {
	_TranslationsSessionsInspectorGitZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get insertAtRef => '作为 @引用 插入';
	@override String get insertPath => '插入路径';
	@override String get showDiff => '查看 diff';
	@override String diffFailedApi({required Object status, required Object message}) => 'Diff 失败（${status}）：${message}';
	@override String diffFailedGeneric({required Object error}) => 'Diff 失败：${error}';
	@override String get insertHash => '插入哈希';
	@override String get showFullPatch => '查看完整 patch';
	@override String showFailedApi({required Object status, required Object message}) => '查看失败（${status}）：${message}';
	@override String showFailedGeneric({required Object error}) => '查看失败：${error}';
	@override String get tabStatus => '状态';
	@override String get tabLog => '日志';
}

// Path: sessions.inspector.tasks
class _TranslationsSessionsInspectorTasksZh extends TranslationsSessionsInspectorTasksEn {
	_TranslationsSessionsInspectorTasksZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get runCommand => '运行命令';
	@override String get insertCommand => '插入命令';
	@override String get insertCommandSubtitle => '粘贴但不回车，方便编辑';
}

// Path: sessions.inspector.notes
class _TranslationsSessionsInspectorNotesZh extends TranslationsSessionsInspectorNotesEn {
	_TranslationsSessionsInspectorNotesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String insertedAt({required Object path}) => '已插入：@${path}';
	@override String get myNotes => '我的笔记';
	@override String get projectDocs => '项目文档';
	@override String get insertAtRefTooltip => '作为 @引用 插入';
	@override String get insertAtRefShort => '插入 @引用';
	@override String draftHint({required Object project}) => '# ${project}\n\n想法、待办、为 agent 提供的上下文…';
	@override String createFailed({required Object error}) => '创建失败：${error}';
	@override String saveFailed({required Object error}) => '保存失败：${error}';
	@override String get changeLocationTooltip => '更改项目文档位置';
	@override String get filenameHint => '文件名（例如：spec 或 design.md）';
	@override String get create => '创建';
	@override String get filterHint => '筛选…';
	@override String get locationDialogTitle => '项目文档位置';
	@override String loadFailedApi({required Object error}) => '加载失败：${error}';
	@override String loadFailedGeneric({required Object error}) => '加载失败：${error}';
	@override String saveFailedApi({required Object error}) => '保存失败：${error}';
	@override String saveFailedGeneric({required Object error}) => '保存失败：${error}';
	@override String insertFailedApi({required Object error}) => '插入失败：${error}';
	@override String insertFailedGeneric({required Object error}) => '插入失败：${error}';
	@override String createFailedApi({required Object error}) => '创建失败：${error}';
	@override String createFailedGeneric({required Object error}) => '创建失败：${error}';
	@override String get personalHint => '个人草稿 — 随输入自动保存。AI agent 不会写入这里。';
	@override String get projectDocsHint => '架构 / 规范 / 决策 / 计划 / 回顾 — 通常由 agent 撰写或维护。';
	@override String get mappingCleared => '映射已清除 — 使用默认值';
	@override String mappedTo({required Object path}) => '已映射到 ${path}';
	@override String get cancelTooltip => '取消';
	@override String get newDocTooltip => '新建文档';
	@override String get noProjectMapping => '无法为此会话解析项目映射。检查网关是否配置了笔记库，以及会话的 cwd 是否已设置。';
	@override String get emptyProjectDocs => '暂无项目文档。点击 + 创建一个，或让 AI agent 根据提示生成。';
	@override String emptyFilterMatch({required Object query}) => '未找到匹配「${query}」的内容。';
	@override String get locationDialogHelp => '将此会话的 cwd 固定到笔记库下的某个文件夹。留空 = 重置。';
	@override String get sessionCwd => '会话 cwd';
	@override String get projectDocsPath => '相对笔记库的项目文档路径';
	@override String get locationStoredHint => '存储于 <vault>/.opendray-projects.json — 与笔记库其余部分一起 git 同步。';
	@override String pinnedHint({required Object path, required Object defaultPath}) => '已固定到 ${path}/（覆盖 ${defaultPath}）。AI agent 也会在此撰写文档。';
	@override String get noProjectMapping2 => '（无项目映射）';
	@override String get clearOverride => '清除覆盖';
	@override String get save => '保存';
}

// Path: sessions.spawnSheet.bypass
class _TranslationsSessionsSpawnSheetBypassZh extends TranslationsSessionsSpawnSheetBypassEn {
	_TranslationsSessionsSpawnSheetBypassZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get labelClaude => '绕过权限';
	@override String get labelCodex => '跳过批准与沙盒';
	@override String get labelGemini => 'YOLO 模式';
	@override String get subtitleOn => '此会话将以提升的自主权运行。';
	@override String get subtitleOff => '关闭 — 确认和提示按正常方式处理。';
}

// Path: sessions.spawnSheet.noProviders
class _TranslationsSessionsSpawnSheetNoProvidersZh extends TranslationsSessionsSpawnSheetNoProvidersEn {
	_TranslationsSessionsSpawnSheetNoProvidersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '未配置任何提供商';
	@override String get message => '网关没有启用任何 CLI 提供商。在 Web 管理端的「提供商」中配置，或编辑 config.toml 的 [providers] 段，然后点击重新加载。';
	@override String get reload => '重新加载';
}

// Path: sessions.spawnSheet.providerLoadError
class _TranslationsSessionsSpawnSheetProviderLoadErrorZh extends TranslationsSessionsSpawnSheetProviderLoadErrorEn {
	_TranslationsSessionsSpawnSheetProviderLoadErrorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '无法加载提供商';
	@override String get networkError => '网络错误';
	@override String serverPrefix({required Object code}) => '服务器 ${code}';
	@override String format({required Object prefix, required Object message}) => '${prefix}：${message}';
}

// Path: sessions.spawnSheet.claudeAccount
class _TranslationsSessionsSpawnSheetClaudeAccountZh extends TranslationsSessionsSpawnSheetClaudeAccountEn {
	_TranslationsSessionsSpawnSheetClaudeAccountZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Claude 账号';
	@override String get helperMulti => '已配置多个账号 — 为此会话选择一个。';
	@override String get helperSingle => '选择已配置的账号，或使用默认（环境变量 / 系统）。';
	@override String get kDefault => '默认（环境变量 / 系统）';
	@override String get disabledSuffix => '（已停用）';
	@override String get noTokenSuffix => '（无令牌）';
	@override String get noneHint => '未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。在 Web 管理端的「设置 → 账号」中添加账号。';
	@override String errorHint({required Object error}) => '无法加载 Claude 账号（${error}）。会话将以网关默认配置启动。';
}

// Path: memoryWorkers.tasks.gatekeeper
class _TranslationsMemoryWorkersTasksGatekeeperZh extends TranslationsMemoryWorkersTasksGatekeeperEn {
	_TranslationsMemoryWorkersTasksGatekeeperZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '守门员';
	@override String get description => '每次 memory_store 写入前的过滤器。高频（目标 <500ms） — 仅 summarizer。';
}

// Path: memoryWorkers.tasks.cleaner
class _TranslationsMemoryWorkersTasksCleanerZh extends TranslationsMemoryWorkersTasksCleanerEn {
	_TranslationsMemoryWorkersTasksCleanerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '清理馆员';
	@override String get description => '定期 LLM 馆员。判断旧记忆为保留 / 过期 / 重复。';
}

// Path: memoryWorkers.tasks.gitactivity
class _TranslationsMemoryWorkersTasksGitactivityZh extends TranslationsMemoryWorkersTasksGitactivityEn {
	_TranslationsMemoryWorkersTasksGitactivityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Git 活动摘要器';
	@override String get description => 'git log → 每 24 小时的 2-3 段叙事。天然适合 agent 工作器。';
}

// Path: memoryWorkers.tasks.transcript
class _TranslationsMemoryWorkersTasksTranscriptZh extends TranslationsMemoryWorkersTasksTranscriptEn {
	_TranslationsMemoryWorkersTasksTranscriptZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '会话记录摘要器';
	@override String get description => '会话结束时的「agent 做了什么」摘要。天然适合 agent 工作器。';
}

// Path: backupTargetEditor.kinds.local
class _TranslationsBackupTargetEditorKindsLocalZh extends TranslationsBackupTargetEditorKindsLocalEn {
	_TranslationsBackupTargetEditorKindsLocalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '本地磁盘';
	@override String get description => '运行 opendray 的机器上的文件夹';
}

// Path: backupTargetEditor.kinds.smb
class _TranslationsBackupTargetEditorKindsSmbZh extends TranslationsBackupTargetEditorKindsSmbEn {
	_TranslationsBackupTargetEditorKindsSmbZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'SMB 共享';
	@override String get description => 'Windows 共享 + 多数家用 NAS 设备';
}

// Path: backupTargetEditor.kinds.webdav
class _TranslationsBackupTargetEditorKindsWebdavZh extends TranslationsBackupTargetEditorKindsWebdavEn {
	_TranslationsBackupTargetEditorKindsWebdavZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'WebDAV';
	@override String get description => '自托管云盘 + 文件共享服务';
}

// Path: backupTargetEditor.kinds.sftp
class _TranslationsBackupTargetEditorKindsSftpZh extends TranslationsBackupTargetEditorKindsSftpEn {
	_TranslationsBackupTargetEditorKindsSftpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'SFTP';
	@override String get description => '任何可 SSH 访问的服务器';
}

// Path: backupTargetEditor.kinds.s3
class _TranslationsBackupTargetEditorKindsS3Zh extends TranslationsBackupTargetEditorKindsS3En {
	_TranslationsBackupTargetEditorKindsS3Zh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'S3 / 兼容';
	@override String get description => 'Amazon S3 + S3 兼容存储桶（MinIO、R2、B2）';
}

// Path: backupTargetEditor.kinds.rclone
class _TranslationsBackupTargetEditorKindsRcloneZh extends TranslationsBackupTargetEditorKindsRcloneEn {
	_TranslationsBackupTargetEditorKindsRcloneZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'rclone（任意）';
	@override String get description => '通过 rclone CLI 访问 OneDrive、Google Drive、Dropbox';
}

// Path: githosts.form.kinds
class _TranslationsGithostsFormKindsZh extends TranslationsGithostsFormKindsEn {
	_TranslationsGithostsFormKindsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get github => 'GitHub';
	@override String get gitlab => 'GitLab';
	@override String get bitbucket => 'Bitbucket';
	@override String get gitea => 'Gitea';
	@override String get custom => '自定义';
}

// Path: channels.notifications.modes
class _TranslationsChannelsNotificationsModesZh extends TranslationsChannelsNotificationsModesEn {
	_TranslationsChannelsNotificationsModesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get onceLabel => '每会话一次';
	@override String get onceDescription => '空闲时触发一次，回复或结束前不再触发。';
	@override String get cooldownLabel => '时间窗冷却';
	@override String get cooldownDescription => '在所选时间窗内抑制重复。';
	@override String get everyLabel => '每次事件（嘈杂）';
	@override String get everyDescription => '不抑制 — 仅适合低频通道。';
}

// Path: channels.kinds.telegram
class _TranslationsChannelsKindsTelegramZh extends TranslationsChannelsKindsTelegramEn {
	_TranslationsChannelsKindsTelegramZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '通过 @BotFather 创建机器人。opendray 长轮询 getUpdates 并通过 REST 发送。原生支持按钮和 reply_to_message。';
	@override String get botTokenLabel => '机器人 Token';
	@override String get botTokenHint => '从 @BotFather 获取。存储于通道配置；仅管理员 API 可见。';
	@override String get chatIdLabel => '默认 chat ID';
	@override String get chatIdPlaceholder => '42（可选 — 没有 ReplyCtx 时使用）';
}

// Path: channels.kinds.slack
class _TranslationsChannelsKindsSlackZh extends TranslationsChannelsKindsSlackEn {
	_TranslationsChannelsKindsSlackZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => 'Socket Mode — 无需公网 webhook。需要 bot OAuth token（xoxb-）和带 connections:write 的 app-level token（xapp-）。';
	@override String get botTokenLabel => 'Bot token（xoxb-…）';
	@override String get botTokenHint => 'OAuth & Permissions → Bot User OAuth Token。需要 chat:write。';
	@override String get appTokenLabel => 'App-level token（xapp-…）';
	@override String get appTokenHint => 'Settings → Basic Information → App-Level Tokens。范围：connections:write。';
	@override String get channelIdLabel => '默认 channel ID';
	@override String get channelIdPlaceholder => 'C0123ABC456（可选）';
}

// Path: channels.kinds.discord
class _TranslationsChannelsKindsDiscordZh extends TranslationsChannelsKindsDiscordEn {
	_TranslationsChannelsKindsDiscordZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '通过 Discord Developer Portal 创建机器人，启用 MESSAGE CONTENT INTENT。连接 Gateway WS — 无需公网 URL。';
	@override String get botTokenLabel => 'Bot token';
	@override String get botTokenPlaceholder => '来自 Discord Developer Portal 的 Bot token';
	@override String get botTokenHint => 'Application → Bot → Reset Token。邀请机器人时勾选 send_messages + embed_links。';
	@override String get channelIdLabel => '默认 channel ID';
	@override String get channelIdPlaceholder => '123456789012345678（可选）';
}

// Path: channels.kinds.feishu
class _TranslationsChannelsKindsFeishuZh extends TranslationsChannelsKindsFeishuEn {
	_TranslationsChannelsKindsFeishuZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '应用级凭据。入站走事件订阅 webhook。下方生成公网 webhook URL — 粘贴到飞书开放平台控制台。';
	@override String get afterCreateHint => '在通道卡上打开 webhook URL，粘贴到飞书开放平台 → 事件订阅 → Request URL。';
	@override String get appIdLabel => 'App ID';
	@override String get appSecretLabel => 'App secret';
	@override String get appSecretPlaceholder => '应用凭据 secret';
	@override String get verificationTokenLabel => 'Verification token';
	@override String get verificationTokenHint => '来自 事件订阅 → Verification Token。设置后，opendray 拒绝 token 不匹配的 webhook。';
	@override String get chatIdLabel => '默认 chat ID（oc_…）';
	@override String get chatIdPlaceholder => 'oc_xxxxxxxxxx（可选）';
}

// Path: channels.kinds.dingtalk
class _TranslationsChannelsKindsDingtalkZh extends TranslationsChannelsKindsDingtalkEn {
	_TranslationsChannelsKindsDingtalkZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '自定义群机器人。仅外发。群聊 → 机器人 → 添加 → 加签模式 → 复制 webhook + secret。';
	@override String get webhookUrlLabel => 'Webhook URL';
	@override String get secretLabel => '加签 secret';
	@override String get secretHint => '当机器人为「加签」安全模式时，将 secret 复制到这里。opendray 自动添加 timestamp + sign 参数。';
}

// Path: channels.kinds.wecom
class _TranslationsChannelsKindsWecomZh extends TranslationsChannelsKindsWecomEn {
	_TranslationsChannelsKindsWecomZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '群机器人 webhook。仅外发（文本 + markdown）。群设置 → 群机器人 → 添加 → 复制 webhook URL。';
	@override String get webhookKeyLabel => 'Webhook key';
	@override String get webhookKeyPlaceholder => '「key=」查询参数值';
	@override String get webhookKeyHint => '或将整个 webhook URL 粘贴到下方字段 — 任一即可。';
	@override String get webhookUrlLabel => '或完整 webhook URL';
}

// Path: channels.kinds.wechat
class _TranslationsChannelsKindsWechatZh extends TranslationsChannelsKindsWechatEn {
	_TranslationsChannelsKindsWechatZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get description => '通过 WxPusher 推送到个人微信。仅外发 — 推送服务不转发用户回复。每个接收方需通过二维码订阅一次。';
	@override String get appTokenLabel => 'App token（AT_…）';
	@override String get appTokenHint => 'WxPusher → 应用管理 → 创建应用 → 复制 App Token。';
	@override String get uidsLabel => '接收方 UID（每行一个）';
	@override String get uidsHint => 'UID 或 topic ID 至少需要一个。';
	@override String get topicIdsLabel => 'Topic ID（每行一个）';
	@override String get urlLabel => '点击跳转 URL';
	@override String get urlHint => '设置后，点击微信通知会打开此页面。';
}

// Path: settings.logViewer.levels
class _TranslationsSettingsLogViewerLevelsZh extends TranslationsSettingsLogViewerLevelsEn {
	_TranslationsSettingsLogViewerLevelsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get all => '全部';
	@override String get debug => '调试';
	@override String get info => '信息';
	@override String get warn => '警告';
	@override String get error => '错误';
}

// Path: settings.serverSettings.sections
class _TranslationsSettingsServerSettingsSectionsZh extends TranslationsSettingsServerSettingsSectionsEn {
	_TranslationsSettingsServerSettingsSectionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get general => '通用';
	@override String get logging => '日志';
	@override String get sessions => '会话';
	@override String get vault => '凭据库';
	@override String get mcpRegistry => 'MCP 注册表';
	@override String get memory => '记忆';
	@override String get backup => '备份';
	@override String get storageClaude => '存储 · Claude';
	@override String get storageCodex => '存储 · Codex';
	@override String get storageGemini => '存储 · Gemini';
}

// Path: settings.serverSettings.sectionDescriptions
class _TranslationsSettingsServerSettingsSectionDescriptionsZh extends TranslationsSettingsServerSettingsSectionDescriptionsEn {
	_TranslationsSettingsServerSettingsSectionDescriptionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get general => '监听地址、管理员账号、令牌 TTL。';
	@override String get logging => '详细程度、格式、磁盘日志路径。';
	@override String get sessions => '空闲检测阈值。';
	@override String get vault => '笔记、技能、git 版本化的根目录。';
	@override String get mcpRegistry => 'MCP 服务器 + 密钥文件的凭据库路径。';
	@override String get memory => '跨 CLI 的持久记忆子系统。';
	@override String get backup => '加密的数据库备份 + 管理数据导出。密语保存在密钥文件（设置 → 备份）。';
	@override String get storageClaude => 'Claude 会话记录在磁盘的存放位置。';
	@override String get storageCodex => 'Codex 会话根目录。';
	@override String get storageGemini => '按项目的临时目录 + projects.json 路径。';
}

// Path: settings.serverSettings.fields
class _TranslationsSettingsServerSettingsFieldsZh extends TranslationsSettingsServerSettingsFieldsEn {
	_TranslationsSettingsServerSettingsFieldsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get listenAddress => '监听地址';
	@override String get adminUser => '管理员用户';
	@override String get adminUserHelper => '当未设置密钥文件或环境变量时生效。否则参见 设置 → 账户。';
	@override String get adminPassword => '管理员密码';
	@override String get adminPasswordHelper => '留空 = 保留。日常轮换请用 设置 → 账户（密钥文件支持，无需重启）。';
	@override String get tokenTtlWeb => '令牌 TTL（Web）';
	@override String get tokenTtlHelper => 'Go duration 字符串，如 24h、30m。';
	@override String get level => '级别';
	@override String get format => '格式';
	@override String get filePath => '文件路径';
	@override String get filePathHelper => '留空 = 仅 stdout。';
	@override String get idleThreshold => '空闲阈值';
	@override String get idleThresholdHelper => '会话被标记为空闲前的安静时长。Go duration。';
	@override String get idleCheckInterval => '空闲检查间隔';
	@override String get idleCheckHelper => '空闲回收器运行的频率。';
	@override String get root => '根目录';
	@override String get rootHelper => 'notes / skills / git_root 子路径的父目录。';
	@override String get notesPath => '笔记路径';
	@override String get skillsPath => '技能路径';
	@override String get gitRoot => 'Git 根';
	@override String get personalPrefix => '个人前缀';
	@override String get projectsPrefix => '项目前缀';
	@override String get registryRoot => '注册表根';
	@override String get secretsFile => '密钥文件';
	@override String get backend => '后端';
	@override String get store => '存储';
	@override String get defaultTopK => '默认 top-k';
	@override String get similarityThreshold => '相似度阈值';
	@override String get defaultScope => '默认范围';
	@override String get chromemHelper => '当 store=chromem 时。';
	@override String get preserveHelper => '留空 = 保留当前值。';
	@override String get localModelName => '本地模型名';
	@override String get localLibraryPath => '本地库路径';
	@override String get localModelPath => '本地模型路径';
	@override String get localTokenizerPath => '本地分词器路径';
	@override String get localMaxSeqLen => '本地最大序列长度';
	@override String get backupEnabled => '已启用';
	@override String get backupEnabledHelper => '即使打开此项，备份子系统仍需配置 OPENDRAY_BACKUP_KEY 或密钥文件才会运行。';
	@override String get backupLocalDir => '本地目录';
	@override String get backupExportDir => '导出目录';
	@override String get pathHelper => '留空 = 启动时从 PATH 解析。';
	@override String get accountsDir => '账号目录';
	@override String get accountsHelper => '各账号 .claude/ 子目录的父目录。留空 = ~/.claude-accounts。';
	@override String get sessionsRoot => '会话根目录';
	@override String get sessionsRootHelper => '留空 = ~/.codex/sessions。';
	@override String get listenHelper => '网关绑定的 host:port。需重启。';
	@override String get secretsHelper => 'AES-256-GCM 加密的密钥库。';
	@override String get backendHelper => 'auto 选择最佳可用；local 需要 ONNX。';
	@override String get similarityHelper => '0.0–1.0；低于此值的结果会被过滤。';
	@override String get chromemPath => 'chromem 路径';
	@override String get httpBaseUrl => 'HTTP base URL';
	@override String get httpModel => 'HTTP model';
	@override String get httpApiKey => 'HTTP api key';
	@override String get httpDimensions => 'HTTP dimensions';
	@override String get pgDumpPath => 'pg_dump 路径';
	@override String get pgRestorePath => 'pg_restore 路径';
	@override String get tmpRoot => '临时根目录';
	@override String get projectsJson => 'projects.json';
}

// Path: web.sessions.list.row
class _TranslationsWebSessionsListRowZh extends TranslationsWebSessionsListRowEn {
	_TranslationsWebSessionsListRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get deleteAria => '删除会话';
	@override String get titleRemoveHistory => '从历史中移除';
	@override String get titleTerminate => '终止并移除';
	@override String get titleRemove => '移除';
	@override String claudeAccountTitle({required Object label}) => 'Claude 账号：${label}';
}

// Path: web.sessions.inspector.tabs
class _TranslationsWebSessionsInspectorTabsZh extends TranslationsWebSessionsInspectorTabsEn {
	_TranslationsWebSessionsInspectorTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get files => '文件';
	@override String get git => 'Git';
	@override String get search => '搜索';
	@override String get tasks => '任务';
	@override String get history => '历史';
	@override String get notes => '笔记';
	@override String get memory => '记忆';
}

// Path: web.memoryWorkers.tasks.gatekeeper
class _TranslationsWebMemoryWorkersTasksGatekeeperZh extends TranslationsWebMemoryWorkersTasksGatekeeperEn {
	_TranslationsWebMemoryWorkersTasksGatekeeperZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Gatekeeper';
	@override String get description => '每次 memory_store 前的预写过滤器。高频（目标 <500ms） — 仅 summarizer。';
}

// Path: web.memoryWorkers.tasks.cleaner
class _TranslationsWebMemoryWorkersTasksCleanerZh extends TranslationsWebMemoryWorkersTasksCleanerEn {
	_TranslationsWebMemoryWorkersTasksCleanerZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Cleaner librarian';
	@override String get description => '周期性 LLM 整理。对旧记忆判定为保留 / 过期 / 重复。';
}

// Path: web.memoryWorkers.tasks.gitactivity
class _TranslationsWebMemoryWorkersTasksGitactivityZh extends TranslationsWebMemoryWorkersTasksGitactivityEn {
	_TranslationsWebMemoryWorkersTasksGitactivityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Git 活动总结器';
	@override String get description => 'git log → 每 24 小时生成 2-3 段叙事。天然适合 agent worker。';
}

// Path: web.memoryWorkers.tasks.transcript
class _TranslationsWebMemoryWorkersTasksTranscriptZh extends TranslationsWebMemoryWorkersTasksTranscriptEn {
	_TranslationsWebMemoryWorkersTasksTranscriptZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '会话记录总结器';
	@override String get description => '会话结束时的“agent 都做了什么”总结。天然适合 agent worker。';
}

// Path: web.project.readonly.tech_stack
class _TranslationsWebProjectReadonlyTechStackZh extends TranslationsWebProjectReadonlyTechStackEn {
	_TranslationsWebProjectReadonlyTechStackZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '技术栈与结构';
	@override String get empty => '在该项目运行一次 Claude 会话 — scanner 会在每次 spawn 时刷新。';
}

// Path: web.project.readonly.recent_activity
class _TranslationsWebProjectReadonlyRecentActivityZh extends TranslationsWebProjectReadonlyRecentActivityEn {
	_TranslationsWebProjectReadonlyRecentActivityZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '最近活动 (git → LLM)';
	@override String get empty => 'Git 活动总结每 24 小时运行；下次调度后再来看看。';
}

// Path: web.project.reset.summary
class _TranslationsWebProjectResetSummaryZh extends TranslationsWebProjectResetSummaryEn {
	_TranslationsWebProjectResetSummaryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String docs_one({required Object count}) => '${count} 份文档';
	@override String docs_other({required Object count}) => '${count} 份文档';
	@override String journal({required Object count}) => '${count} 条日志';
	@override String proposals_one({required Object count}) => '${count} 条提案';
	@override String proposals_other({required Object count}) => '${count} 条提案';
	@override String cleanup({required Object count}) => '${count} 条清理';
	@override String memories({required Object count}) => '${count} 条记忆';
}

// Path: web.memoryInspector.scope.values
class _TranslationsWebMemoryInspectorScopeValuesZh extends TranslationsWebMemoryInspectorScopeValuesEn {
	_TranslationsWebMemoryInspectorScopeValuesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get project => 'project';
	@override String get session => 'session';
	@override String get global => 'global';
}

// Path: web.notes.vaultSync.init
class _TranslationsWebNotesVaultSyncInitZh extends TranslationsWebNotesVaultSyncInitEn {
	_TranslationsWebNotesVaultSyncInitZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'vault 尚未初始化为 git 仓库';
	@override String get body => '初始化会在 vault 根目录运行 <1>git init -b main</1> 并加入一份合理的 <3>.gitignore</3>。之后你就可以提交笔记并配置 remote（GitHub / Gitea / GitLab）进行跨机同步。';
	@override String get button => '把 vault 初始化为 git 仓库';
	@override String get initToast => 'vault 已初始化为 git 仓库';
	@override String get initFailedToast => '初始化失败';
}

// Path: web.notes.vaultSync.branch
class _TranslationsWebNotesVaultSyncBranchZh extends TranslationsWebNotesVaultSyncBranchEn {
	_TranslationsWebNotesVaultSyncBranchZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get clean => '干净';
	@override String staged({required Object count}) => '${count} 已暂存';
	@override String modified({required Object count}) => '${count} 已修改';
	@override String untracked({required Object count}) => '${count} 未跟踪';
}

// Path: web.notes.vaultSync.action
class _TranslationsWebNotesVaultSyncActionZh extends TranslationsWebNotesVaultSyncActionEn {
	_TranslationsWebNotesVaultSyncActionZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get pull => '拉取';
	@override String get push => '推送';
	@override String get pullTitleNoRemote => '请先配置 remote';
	@override String get pullTitleHasUpstream => 'git pull --rebase --autostash';
	@override String get pullTitleNoUpstream => '拉取 origin 的 HEAD；隐式建立 tracking';
	@override String get pushTitleNoRemote => '请先配置 remote';
	@override String get pushTitleHasUpstream => 'git push -u origin HEAD';
	@override String get pushTitleNoUpstream => '首次推送 — 会将 upstream 设为 origin/HEAD';
	@override String get noRemote => '尚未配置 remote — pull/push 已禁用';
	@override String get noUpstream => '尚无 upstream tracking — 首次推送会自动建立。';
	@override String get pulledToast => '已拉取';
	@override String get pullFailedToast => '拉取失败';
	@override String get pushedToast => '已推送';
	@override String get pushFailedToast => '推送失败';
}

// Path: web.notes.vaultSync.commit
class _TranslationsWebNotesVaultSyncCommitZh extends TranslationsWebNotesVaultSyncCommitEn {
	_TranslationsWebNotesVaultSyncCommitZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '提交';
	@override String placeholder({required Object date}) => 'Notes: ${date}（默认）';
	@override String get commitAll => '提交全部';
	@override String get hint => '暂存所有变更（<1>git add .</1>）然后用此 message 提交。message 为空则使用带时间戳的默认值。';
	@override String committedToast({required Object hash}) => '已提交 ${hash}';
	@override String get commitFailedToast => '提交失败';
}

// Path: web.notes.vaultSync.fileList
class _TranslationsWebNotesVaultSyncFileListZh extends TranslationsWebNotesVaultSyncFileListEn {
	_TranslationsWebNotesVaultSyncFileListZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String title({required Object count}) => '工作树 · ${count}';
	@override String moreSuffix({required Object count}) => '+${count} 更多';
}

// Path: web.notes.vaultSync.remote
class _TranslationsWebNotesVaultSyncRemoteZh extends TranslationsWebNotesVaultSyncRemoteEn {
	_TranslationsWebNotesVaultSyncRemoteZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Remote（origin）';
	@override String get cancel => '取消';
	@override String get change => '更换';
	@override String get configure => '配置';
	@override String get empty => '尚未设置 remote。添加一个 HTTPS 或 SSH URL（例如 <1>git@github.com:you/notes.git</1> 或 <3>https://tea.linivek.online/you/notes.git</3>）以启用 push / pull。';
	@override String get urlLabel => 'URL（HTTPS 或 SSH）';
	@override String get urlPlaceholder => 'git@host:owner/notes.git';
	@override String get save => '保存';
	@override String get savedToast => 'Remote 已保存';
	@override String get saveFailedToast => '设置 remote 失败';
}

// Path: web.notes.vaultSync.history
class _TranslationsWebNotesVaultSyncHistoryZh extends TranslationsWebNotesVaultSyncHistoryEn {
	_TranslationsWebNotesVaultSyncHistoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '最近提交';
	@override String get loading => '加载中…';
	@override String get empty => '暂无提交。';
}

// Path: web.notes.vaultSync.conflict
class _TranslationsWebNotesVaultSyncConflictZh extends TranslationsWebNotesVaultSyncConflictEn {
	_TranslationsWebNotesVaultSyncConflictZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override late final _TranslationsWebNotesVaultSyncConflictKindsZh kinds = _TranslationsWebNotesVaultSyncConflictKindsZh._(_root);
	@override String headline({required Object kind}) => 'vault 存在暂停的 ${kind} 且有未解决的冲突';
	@override String explainer({required Object kind}) => '在 ${kind} 完成之前，pull、push 与 commit 都被阻塞。你可以选择 <1>中止</1>（把工作树恢复到 ${kind} 之前的状态 — 保留本地提交，丢弃远端提交），或 <3>强制重置到 remote</3>（丢弃所有本地提交 + 未提交修改；vault 变成 origin 的精确镜像）。';
	@override String conflictedHeader({required Object count}) => '冲突文件 · ${count}';
	@override String abort({required Object kind}) => '中止 ${kind}';
	@override String abortTitle({required Object kind}) => 'git ${kind} --abort';
	@override String get forceReset => '强制重置到 remote';
	@override String get forceResetTitle => 'git fetch && git reset --hard origin/<branch> && git clean -fd';
	@override String forceResetConfirm({required Object kind}) => '破坏性操作：将\n  • 中止进行中的 ${kind}\n  • 运行 git fetch origin\n  • reset --hard 到 origin/<branch>\n  • clean -fd（删除未跟踪文件）\n\n任何尚未推送到 origin 的本地提交以及任何未提交修改都将永久丢失。\n\n继续？';
	@override String abortedToast({required Object kind}) => '已中止 ${kind}';
	@override String get abortedDescription => '工作树已恢复到操作前状态。';
	@override String get abortFailedToast => '中止失败';
	@override String resetToast({required Object branch}) => '已重置到 ${branch}';
	@override String get resetDescription => '本地更改已丢弃；vault 与 remote 一致。';
	@override String get resetFailedToast => '重置失败';
}

// Path: web.notes.vaultSync.auth
class _TranslationsWebNotesVaultSyncAuthZh extends TranslationsWebNotesVaultSyncAuthEn {
	_TranslationsWebNotesVaultSyncAuthZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '认证';
	@override String httpsTokenOk({required Object host}) => '将使用 Plugins → Git hosts 中为 <1>${host}</1> 存的 token。✓';
	@override String httpsTokenMissing({required Object host}) => '<1>${host}</1> 上的 HTTPS remote，opendray 中没有配置 token。在你为其添加 token 之前，私有仓库的 push / pull 很可能失败。';
	@override String ssh({required Object host}) => '<1>${host}</1> 上的 SSH remote。认证使用网关主机的 <3>~/.ssh/</3>（ssh-agent、identity 文件、host config）。可在主机 shell 用 <5>ssh -T git@${host}</5> 验证。';
	@override String get configureTokenLink => '→ 配置 git host token';
}

// Path: web.notes.vaultSync.autoSync
class _TranslationsWebNotesVaultSyncAutoSyncZh extends TranslationsWebNotesVaultSyncAutoSyncEn {
	_TranslationsWebNotesVaultSyncAutoSyncZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get loading => '加载自动同步设置…';
	@override String get title => '自动同步';
	@override String get on => '开';
	@override String get runNow => '立即运行';
	@override String get runNowTooltip => '立即唤醒同步循环（跳过等待，然后运行所有到期的步骤）';
	@override String get configure => '配置';
	@override String get hide => '隐藏';
	@override String get enabled => '启用';
	@override String get enabledTooltipNoRemote => '请先配置 remote 才能启用自动同步';
	@override String get noRemoteHint => '尚无 remote — push/pull 将被跳过。';
	@override String get commitEvery => '提交间隔';
	@override String get commitEveryExamples => '示例：<1>30s</1>、<3>10m</3>、<5>2h</5>。最小 30s。';
	@override String get pullEvery => '拉取间隔';
	@override String get pullEveryHint => '仅在启用 Pull 时使用。';
	@override String get pushAfterCommit => '提交后 push';
	@override String get pullPeriodically => '周期性 pull';
	@override String get commitTemplateLabel => '提交 message 模板';
	@override String commitTemplatePlaceholder({required Object date}) => 'Auto-sync: ${date}（留空则使用默认）';
	@override String get saveSettings => '保存设置';
	@override String get discard => '丢弃';
	@override String get lastCommit => '最近 commit';
	@override String get lastPush => '最近 push';
	@override String get lastPull => '最近 pull';
	@override String get never => '从未';
	@override String get savedToast => '自动同步设置已保存';
	@override String get saveFailedToast => '保存失败';
	@override String get triggeredToast => '已触发自动同步';
	@override String get runFailedToast => '运行失败';
}

// Path: web.providers.detail.caps
class _TranslationsWebProvidersDetailCapsZh extends TranslationsWebProvidersDetailCapsEn {
	_TranslationsWebProvidersDetailCapsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get resume => 'resume';
	@override String get stream => 'stream';
	@override String get images => 'images';
	@override String get mcp => 'mcp';
}

// Path: web.channels.notifications.modes
class _TranslationsWebChannelsNotificationsModesZh extends TranslationsWebChannelsNotificationsModesEn {
	_TranslationsWebChannelsNotificationsModesZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get onceLabel => '每个会话仅一次（推荐）';
	@override String get onceHint => '当会话变为 idle 时通知一次，然后保持静默，直到会话结束或你通过该频道回复。';
	@override String get cooldownLabel => '时间窗口冷却';
	@override String get cooldownHint => '对同一 (会话, 事件) 在所选时间窗口内抑制重复通知。';
	@override String get everyLabel => '每次事件都通知（嘈杂）';
	@override String get everyHint => '不做抑制。仅用于低频频道。';
}

// Path: web.channels.notifications.cooldowns
class _TranslationsWebChannelsNotificationsCooldownsZh extends TranslationsWebChannelsNotificationsCooldownsEn {
	_TranslationsWebChannelsNotificationsCooldownsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get k60 => '1 分钟';
	@override String get k300 => '5 分钟';
	@override String get k900 => '15 分钟';
	@override String get k1800 => '30 分钟';
	@override String get k3600 => '1 小时';
}

// Path: web.channels.notifications.snippetCaps
class _TranslationsWebChannelsNotificationsSnippetCapsZh extends TranslationsWebChannelsNotificationsSnippetCapsEn {
	_TranslationsWebChannelsNotificationsSnippetCapsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get k0 => '不限制 — 拆分到多条消息（默认）';
	@override String get k1000 => '1000 字符（精简）';
	@override String get k3000 => '3000 字符';
	@override String get k6000 => '6000 字符';
	@override String get k12000 => '12000 字符';
}

// Path: web.plugins.mcp.columns
class _TranslationsWebPluginsMcpColumnsZh extends TranslationsWebPluginsMcpColumnsEn {
	_TranslationsWebPluginsMcpColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get name => '名称';
	@override String get transport => 'Transport';
	@override String get spec => '规范';
	@override String get enabled => '启用';
}

// Path: web.plugins.mcp.editor
class _TranslationsWebPluginsMcpEditorZh extends TranslationsWebPluginsMcpEditorEn {
	_TranslationsWebPluginsMcpEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get createTitle => '新建 MCP 服务器';
	@override String editTitle({required Object id}) => '编辑 MCP: ${id}';
	@override String description({required Object API_KEY}) => 'JSON 结构：stdio（默认）使用 <1>command</1>+<3>args</3>+<5>env</5>；sse / http 使用 <7>transport</7> +<9> url</9>+<11>headers</11>。以 <13>\$${API_KEY}</13> 引用密钥 — spawn 时会从密钥文件替换。';
	@override String get idLabel => 'ID';
	@override String get idPlaceholder => 'filesystem';
	@override String get idHint => '小写字母 / 数字 / 短横 / 下划线。同时作为目录名与默认 <1>name</1>。';
	@override String get bodyLabel => 'mcp.json';
	@override String invalidJson({required Object error}) => '无效的 JSON：${error}';
	@override String get createdToast => 'MCP 服务器已创建';
	@override String get savedToast => 'MCP 服务器已保存';
	@override String get createFailedToast => '创建失败';
	@override String get saveFailedToast => '保存失败';
}

// Path: web.plugins.mcpSecrets.columns
class _TranslationsWebPluginsMcpSecretsColumnsZh extends TranslationsWebPluginsMcpSecretsColumnsEn {
	_TranslationsWebPluginsMcpSecretsColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get key => 'Key';
	@override String get value => 'Value';
}

// Path: web.plugins.mcpSecrets.editor
class _TranslationsWebPluginsMcpSecretsEditorZh extends TranslationsWebPluginsMcpSecretsEditorEn {
	_TranslationsWebPluginsMcpSecretsEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get addTitle => '添加密钥';
	@override String updateTitle({required Object key}) => '更新 ${key}';
	@override String addDescription({required Object KEY}) => '若操作系统 keychain 可用，则在磁盘上加密存储。可在任意 mcp.json 的 env / headers / args / url 中以 \$${KEY} 引用。';
	@override String get editDescription => '输入新值以覆盖。旧值无法恢复。';
	@override String get keyLabel => 'Key';
	@override String get keyPlaceholder => 'BRAVE_API_KEY';
	@override String get keyPattern => '必须匹配 <1>[A-Za-z_][A-Za-z0-9_]*</1>';
	@override String get keyCollision => '已存在 — 请使用编辑，或选择另一个名称。';
	@override String get valueLabel => 'Value';
	@override String get valueHint => '输入时隐藏。已保存的值不会通过 API 返回。';
	@override String get addedToast => '密钥已添加';
	@override String get updatedToast => '密钥已更新';
	@override String get saveFailedToast => '保存失败';
}

// Path: web.plugins.skills.columns
class _TranslationsWebPluginsSkillsColumnsZh extends TranslationsWebPluginsSkillsColumnsEn {
	_TranslationsWebPluginsSkillsColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get description => '描述';
	@override String get source => '来源';
}

// Path: web.plugins.skills.editor
class _TranslationsWebPluginsSkillsEditorZh extends TranslationsWebPluginsSkillsEditorEn {
	_TranslationsWebPluginsSkillsEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get createTitle => '新建 skill';
	@override String customizeTitle({required Object id}) => '自定义内置：${id}';
	@override String editTitle({required Object id}) => '编辑 skill：${id}';
	@override String get customizeDescription => '你正在查看一个嵌入到 opendray 的内置 skill。保存会在相同 id 上创建一个 vault 覆盖 — 你的修改保存在 ~/.opendray/vault/skills/<id>/SKILL.md 中并遮盖内置版本，直到你点击重置。';
	@override String get editDescription => 'SKILL.md 格式 — frontmatter（name + description），然后是 markdown 指令。description 会出现在 agent 的 Tier 1 索引中。';
	@override String get idLabel => 'ID';
	@override String get idPlaceholder => 'my-helper';
	@override String get idHint => '小写字母 / 数字 / 短横 / 下划线。作为 <1>~/.opendray/vault/skills/&lt;id&gt;/</1> 下的目录名。';
	@override String get bodyLabel => 'SKILL.md';
	@override String get createdToast => 'Skill 已创建';
	@override String get savedToast => 'Skill 已保存';
	@override String get savedOverrideToast => '已保存为 vault 覆盖';
	@override String get createFailedToast => '创建失败';
	@override String get saveFailedToast => '保存失败';
	@override String get saveAsOverride => '保存为 vault 覆盖';
}

// Path: web.plugins.customTasks.columns
class _TranslationsWebPluginsCustomTasksColumnsZh extends TranslationsWebPluginsCustomTasksColumnsEn {
	_TranslationsWebPluginsCustomTasksColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get name => '名称';
	@override String get command => '命令';
	@override String get scope => 'Scope';
}

// Path: web.plugins.customTasks.dialog
class _TranslationsWebPluginsCustomTasksDialogZh extends TranslationsWebPluginsCustomTasksDialogEn {
	_TranslationsWebPluginsCustomTasksDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get addTitle => '添加自定义任务';
	@override String editTitle({required Object name}) => '编辑 ${name}';
	@override String get description => '命令会原样发送到会话的终端中，等同于在 prompt 处键入并回车。';
	@override String get nameLabel => '名称';
	@override String get namePlaceholder => 'dev';
	@override String get commandLabel => '命令';
	@override String get commandPlaceholder => 'docker compose up --build';
	@override String get descLabel => '描述（可选）';
	@override String get descPlaceholder => '启动开发环境并跟踪日志';
	@override String get cwdLabel => 'cwd scope（可选）';
	@override String get cwdPlaceholder => '/Users/me/projects/foo（留空 = 全局）';
	@override String get cwdHint => '留空 = 在每个会话中都可见。否则只有当会话的 cwd 与此绝对路径匹配时才显示。';
	@override String get addedToast => '任务已添加';
	@override String get updatedToast => '任务已更新';
	@override String get addFailedToast => '添加失败';
	@override String get updateFailedToast => '更新失败';
}

// Path: web.plugins.gitHosts.columns
class _TranslationsWebPluginsGitHostsColumnsZh extends TranslationsWebPluginsGitHostsColumnsEn {
	_TranslationsWebPluginsGitHostsColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get host => '主机';
	@override String get kind => '类型';
	@override String get token => 'Token';
	@override String get enabled => '启用';
}

// Path: web.plugins.gitHosts.dialog
class _TranslationsWebPluginsGitHostsDialogZh extends TranslationsWebPluginsGitHostsDialogEn {
	_TranslationsWebPluginsGitHostsDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get addTitle => '添加 git 主机';
	@override String editTitle({required Object host}) => '编辑 ${host}';
	@override String get description => 'Token 存储在网关上。仅用于只读 API 调用（列出 PR 等）。';
	@override String get kindLabel => '类型';
	@override String get kindGitHub => 'GitHub';
	@override String get kindGitea => 'Gitea';
	@override String get kindGitLab => 'GitLab';
	@override String get hostLabel => '主机';
	@override String get hostPlaceholder => 'github.com';
	@override String get displayNameLabel => '显示名称（可选）';
	@override String get displayNamePlaceholder => 'Personal';
	@override String get tokenLabel => 'Token';
	@override String get newTokenLabel => '新 token（留空表示保留）';
	@override String get tokenPlaceholder => 'ghp_… / gho_… / glpat-…';
	@override String get tokenPlaceholderEdit => '…';
	@override String get tokenHint => 'GitHub：带 <1>repo</1> scope 的 PAT。Gitea：带 <3>read:repository</3> 的 token。GitLab：带 <5>read_api</5> 的 PAT。';
	@override String get enabledLabel => '启用';
	@override String get addedToast => 'Git 主机已添加';
	@override String get updatedToast => 'Git 主机已更新';
	@override String get addFailedToast => '添加失败';
	@override String get updateFailedToast => '更新失败';
}

// Path: web.backups.backupsTab.columns
class _TranslationsWebBackupsBackupsTabColumnsZh extends TranslationsWebBackupsBackupsTabColumnsEn {
	_TranslationsWebBackupsBackupsTabColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get target => '目标';
	@override String get status => '状态';
	@override String get started => '开始';
	@override String get size => '大小';
	@override String get actions => '操作';
}

// Path: web.backups.schedulesTab.columns
class _TranslationsWebBackupsSchedulesTabColumnsZh extends TranslationsWebBackupsSchedulesTabColumnsEn {
	_TranslationsWebBackupsSchedulesTabColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get target => '目标';
	@override String get interval => '间隔';
	@override String get keep => '保留';
	@override String get nextRun => '下次运行';
	@override String get enabled => '启用';
	@override String get actions => '操作';
}

// Path: web.backups.targetsTab.columns
class _TranslationsWebBackupsTargetsTabColumnsZh extends TranslationsWebBackupsTargetsTabColumnsEn {
	_TranslationsWebBackupsTargetsTabColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get kind => '类型';
	@override String get config => '配置';
	@override String get enabled => '启用';
	@override String get actions => '操作';
}

// Path: web.backups.targetEditor.local
class _TranslationsWebBackupsTargetEditorLocalZh extends TranslationsWebBackupsTargetEditorLocalEn {
	_TranslationsWebBackupsTargetEditorLocalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get rootLabel => '根目录';
	@override String get rootHint => '留空 = cfg.backup.local_dir (~/.opendray/backups)';
	@override String get rootPlaceholder => '~/backups/opendray  或  /mnt/external-hdd/opendray';
}

// Path: web.backups.targetEditor.smb
class _TranslationsWebBackupsTargetEditorSmbZh extends TranslationsWebBackupsTargetEditorSmbEn {
	_TranslationsWebBackupsTargetEditorSmbZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get hostLabel => '主机';
	@override String get hostPlaceholder => '192.168.9.8';
	@override String get portLabel => '端口';
	@override String get shareLabel => 'Share';
	@override String get shareHint => 'SMB 服务器上的顶层共享名';
	@override String get sharePlaceholder => 'Claude_Workspace';
	@override String get userLabel => '用户';
	@override String get passwordLabel => '密码';
	@override String get pathPrefixLabel => '路径前缀';
	@override String get pathPrefixHint => '共享根下的子文件夹（可选）';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.s3
class _TranslationsWebBackupsTargetEditorS3Zh extends TranslationsWebBackupsTargetEditorS3En {
	_TranslationsWebBackupsTargetEditorS3Zh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get endpointLabel => 'Endpoint';
	@override String get endpointHint => '主机（不要带协议）。AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000';
	@override String get endpointPlaceholder => 's3.amazonaws.com';
	@override String get regionLabel => 'Region';
	@override String get regionHint => '仅 AWS；R2 用 \'auto\'';
	@override String get regionPlaceholder => 'us-east-1 / auto';
	@override String get bucketLabel => 'Bucket';
	@override String get bucketPlaceholder => 'opendray-backups';
	@override String get accessKeyLabel => 'Access key';
	@override String get secretKeyLabel => 'Secret key';
	@override String get secretKeyHint => 'AES-256-GCM 加密存储；不会被回显';
	@override String get pathPrefixLabel => 'Path prefix';
	@override String get pathPrefixHint => 'Object-key 前缀（可选）';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
	@override String get useHttps => '使用 HTTPS';
	@override String get pathStyle => 'Path-style 寻址（legacy / MinIO）';
}

// Path: web.backups.targetEditor.webdav
class _TranslationsWebBackupsTargetEditorWebdavZh extends TranslationsWebBackupsTargetEditorWebdavEn {
	_TranslationsWebBackupsTargetEditorWebdavZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get baseUrlLabel => 'Base URL';
	@override String get baseUrlHint => '包含任意路径的完整 URL。示例：https://cloud.example.com/remote.php/dav/files/me/（Nextcloud）、https://nas.local:5006/（群晖）、https://dav.jianguoyun.com/dav/（坚果云）';
	@override String get baseUrlPlaceholder => 'https://cloud.example.com/remote.php/dav/files/<user>/';
	@override String get userLabel => '用户';
	@override String get passwordLabel => '密码';
	@override String get pathPrefixLabel => '路径前缀';
	@override String get pathPrefixHint => 'Base URL 下的子文件夹（可选）';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
}

// Path: web.backups.targetEditor.sftp
class _TranslationsWebBackupsTargetEditorSftpZh extends TranslationsWebBackupsTargetEditorSftpEn {
	_TranslationsWebBackupsTargetEditorSftpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get hostLabel => '主机';
	@override String get hostPlaceholder => 'vps.example.com';
	@override String get portLabel => '端口';
	@override String get userLabel => '用户';
	@override String get passwordLabel => '密码';
	@override String get passwordHint => '密码或私钥二选一。若两者都填，密码会被作为私钥口令使用。';
	@override String get privateKeyLabel => '私钥（PEM）';
	@override String get privateKeyHint => '粘贴 OpenSSH/PEM 格式的私钥内容（例如 ~/.ssh/id_ed25519）。留空则仅用密码认证。';
	@override String get privateKeyPlaceholder => '-----BEGIN OPENSSH PRIVATE KEY-----...';
	@override String get hostKeyLabel => 'Host key（pinning）';
	@override String get hostKeyHint => 'OpenSSH 风格的服务器公钥（运行 `ssh-keyscan host` 获取）。留空则禁用 pinning（LAN 之外不推荐）。';
	@override String get hostKeyPlaceholder => 'ssh-ed25519 AAAA...';
	@override String get pathPrefixLabel => '路径前缀';
	@override String get pathPrefixHint => '绝对路径或相对家目录（可选）';
	@override String get pathPrefixPlaceholder => '/var/backups/opendray  或  opendray-backups';
}

// Path: web.backups.targetEditor.rclone
class _TranslationsWebBackupsTargetEditorRcloneZh extends TranslationsWebBackupsTargetEditorRcloneEn {
	_TranslationsWebBackupsTargetEditorRcloneZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get rcloneHint => '要求 opendray 主机上已安装 <1>rclone</1> CLI。请先通过 <3>rclone config</3> 配置 remote，然后在下面填写 remote 名。opendray 在内部调用 <5>rclone rcat / cat / lsd</5>。';
	@override String get remoteLabel => 'Remote name';
	@override String get remoteHint => '来自 `rclone config` 的名字（不带冒号）。例如：gdrive、onedrive、dropbox-personal、baidu-pan';
	@override String get remotePlaceholder => 'gdrive';
	@override String get pathPrefixLabel => '路径前缀';
	@override String get pathPrefixHint => 'Remote 根下的子文件夹（可选）';
	@override String get pathPrefixPlaceholder => 'opendray/backups';
	@override String get binaryPathLabel => '二进制路径';
	@override String get binaryPathHint => '覆盖 `which rclone`。留空则走 PATH 查找。';
	@override String get binaryPathPlaceholder => '/opt/homebrew/bin/rclone';
	@override String get configPathLabel => 'Config 路径';
	@override String get configPathHint => '覆盖 --config（默认 ~/.config/rclone/rclone.conf 或 ~/.rclone.conf）';
	@override String get configPathPlaceholder => '留空则使用 rclone 默认';
}

// Path: web.serverSettings.sections.general
class _TranslationsWebServerSettingsSectionsGeneralZh extends TranslationsWebServerSettingsSectionsGeneralEn {
	_TranslationsWebServerSettingsSectionsGeneralZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '通用';
	@override String get desc => '监听地址、操作员账号、令牌 TTL。';
}

// Path: web.serverSettings.sections.logging
class _TranslationsWebServerSettingsSectionsLoggingZh extends TranslationsWebServerSettingsSectionsLoggingEn {
	_TranslationsWebServerSettingsSectionsLoggingZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '日志';
	@override String get desc => '日志级别、格式与实时跟踪。';
}

// Path: web.serverSettings.sections.sessions
class _TranslationsWebServerSettingsSectionsSessionsZh extends TranslationsWebServerSettingsSectionsSessionsEn {
	_TranslationsWebServerSettingsSectionsSessionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '会话';
	@override String get desc => '空闲检测阈值。';
}

// Path: web.serverSettings.sections.vault
class _TranslationsWebServerSettingsSectionsVaultZh extends TranslationsWebServerSettingsSectionsVaultEn {
	_TranslationsWebServerSettingsSectionsVaultZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'Vault';
	@override String get desc => '笔记、技能与 git 版本化根目录。';
}

// Path: web.serverSettings.sections.mcp
class _TranslationsWebServerSettingsSectionsMcpZh extends TranslationsWebServerSettingsSectionsMcpEn {
	_TranslationsWebServerSettingsSectionsMcpZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => 'MCP 注册表';
	@override String get desc => '服务器注册表与密钥。';
}

// Path: web.serverSettings.sections.memory
class _TranslationsWebServerSettingsSectionsMemoryZh extends TranslationsWebServerSettingsSectionsMemoryEn {
	_TranslationsWebServerSettingsSectionsMemoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆';
	@override String get desc => '跨 CLI 的持久化记忆子系统。';
}

// Path: web.serverSettings.sections.memoryAmbient
class _TranslationsWebServerSettingsSectionsMemoryAmbientZh extends TranslationsWebServerSettingsSectionsMemoryAmbientEn {
	_TranslationsWebServerSettingsSectionsMemoryAmbientZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '记忆 · 环境感知';
	@override String get desc => '自动捕获对话进入记忆，并在 spawn 时注入。';
}

// Path: web.serverSettings.sections.backup
class _TranslationsWebServerSettingsSectionsBackupZh extends TranslationsWebServerSettingsSectionsBackupEn {
	_TranslationsWebServerSettingsSectionsBackupZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '备份';
	@override String get desc => '加密数据库备份、恢复，以及管理员数据导出。';
}

// Path: web.serverSettings.sections.claude
class _TranslationsWebServerSettingsSectionsClaudeZh extends TranslationsWebServerSettingsSectionsClaudeEn {
	_TranslationsWebServerSettingsSectionsClaudeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '存储 · Claude';
	@override String get desc => 'Claude 会话记录在磁盘上的位置。';
}

// Path: web.serverSettings.sections.codex
class _TranslationsWebServerSettingsSectionsCodexZh extends TranslationsWebServerSettingsSectionsCodexEn {
	_TranslationsWebServerSettingsSectionsCodexZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '存储 · Codex';
	@override String get desc => 'Codex 会话根目录。';
}

// Path: web.serverSettings.sections.gemini
class _TranslationsWebServerSettingsSectionsGeminiZh extends TranslationsWebServerSettingsSectionsGeminiEn {
	_TranslationsWebServerSettingsSectionsGeminiZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '存储 · Gemini';
	@override String get desc => 'Gemini 每项目 tmp 与 projects.json。';
}

// Path: web.serverSettings.fields.listenAddress
class _TranslationsWebServerSettingsFieldsListenAddressZh extends TranslationsWebServerSettingsFieldsListenAddressEn {
	_TranslationsWebServerSettingsFieldsListenAddressZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '监听地址';
	@override String get hint => 'HTTP 服务绑定的 host:port，例如：0.0.0.0:8770。';
}

// Path: web.serverSettings.fields.username
class _TranslationsWebServerSettingsFieldsUsernameZh extends TranslationsWebServerSettingsFieldsUsernameEn {
	_TranslationsWebServerSettingsFieldsUsernameZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '用户名';
	@override String get hint => '登录表单使用的账号。修改后下次请求会强制重新登录。';
}

// Path: web.serverSettings.fields.password
class _TranslationsWebServerSettingsFieldsPasswordZh extends TranslationsWebServerSettingsFieldsPasswordEn {
	_TranslationsWebServerSettingsFieldsPasswordZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '密码';
	@override String get hint => '留空保持当前密码不变；填值则会覆盖。';
	@override String get hideTitle => '隐藏';
	@override String get revealTitle => '显示';
}

// Path: web.serverSettings.fields.tokenTTL
class _TranslationsWebServerSettingsFieldsTokenTTLZh extends TranslationsWebServerSettingsFieldsTokenTTLEn {
	_TranslationsWebServerSettingsFieldsTokenTTLZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '令牌 TTL';
	@override String get hint => 'Bearer 令牌生命周期，使用 Go duration，如 "24h"、"30m"。留空 = 永不过期。';
}

// Path: web.serverSettings.fields.logLevel
class _TranslationsWebServerSettingsFieldsLogLevelZh extends TranslationsWebServerSettingsFieldsLogLevelEn {
	_TranslationsWebServerSettingsFieldsLogLevelZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '日志级别';
	@override String get hint => '低于此级别的日志将被丢弃。';
}

// Path: web.serverSettings.fields.logFormat
class _TranslationsWebServerSettingsFieldsLogFormatZh extends TranslationsWebServerSettingsFieldsLogFormatEn {
	_TranslationsWebServerSettingsFieldsLogFormatZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '格式';
	@override String get hint => '"text" 适合人读；"json" 便于机器解析。';
}

// Path: web.serverSettings.fields.logFile
class _TranslationsWebServerSettingsFieldsLogFileZh extends TranslationsWebServerSettingsFieldsLogFileEn {
	_TranslationsWebServerSettingsFieldsLogFileZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '日志文件';
	@override String get hint => '可选的文件路径。10MB 自动轮转，保留 5 个备份。留空 = 仅输出到 stderr。';
}

// Path: web.serverSettings.fields.idleThreshold
class _TranslationsWebServerSettingsFieldsIdleThresholdZh extends TranslationsWebServerSettingsFieldsIdleThresholdEn {
	_TranslationsWebServerSettingsFieldsIdleThresholdZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '空闲阈值';
	@override String get hint => '会话静默这么久后触发 session.idle。留空 = 30s。';
}

// Path: web.serverSettings.fields.idlePollInterval
class _TranslationsWebServerSettingsFieldsIdlePollIntervalZh extends TranslationsWebServerSettingsFieldsIdlePollIntervalEn {
	_TranslationsWebServerSettingsFieldsIdlePollIntervalZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '空闲轮询间隔';
	@override String get hint => '空闲检测器的唤醒频率。越低 = 延迟越低、唤醒越多。留空 = 5s。';
}

// Path: web.serverSettings.fields.vaultRoot
class _TranslationsWebServerSettingsFieldsVaultRootZh extends TranslationsWebServerSettingsFieldsVaultRootEn {
	_TranslationsWebServerSettingsFieldsVaultRootZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Vault 根目录';
	@override String get hint => '笔记、技能和 MCP 注册表的顶层目录。';
}

// Path: web.serverSettings.fields.notesDirectory
class _TranslationsWebServerSettingsFieldsNotesDirectoryZh extends TranslationsWebServerSettingsFieldsNotesDirectoryEn {
	_TranslationsWebServerSettingsFieldsNotesDirectoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '笔记目录';
	@override String get hint => '覆盖笔记位置。默认为 <vault root>/notes。';
}

// Path: web.serverSettings.fields.skillsDirectory
class _TranslationsWebServerSettingsFieldsSkillsDirectoryZh extends TranslationsWebServerSettingsFieldsSkillsDirectoryEn {
	_TranslationsWebServerSettingsFieldsSkillsDirectoryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '技能目录';
	@override String get hint => '覆盖技能位置。默认为 <vault root>/skills。';
}

// Path: web.serverSettings.fields.gitRoot
class _TranslationsWebServerSettingsFieldsGitRootZh extends TranslationsWebServerSettingsFieldsGitRootEn {
	_TranslationsWebServerSettingsFieldsGitRootZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Git 根目录';
	@override String get hint => 'Vault Sync 功能提交到的工作树。';
}

// Path: web.serverSettings.fields.personalPrefix
class _TranslationsWebServerSettingsFieldsPersonalPrefixZh extends TranslationsWebServerSettingsFieldsPersonalPrefixEn {
	_TranslationsWebServerSettingsFieldsPersonalPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '个人前缀';
	@override String get hint => '自动派生路径时用于个人笔记的文件夹名。默认 "personal"。';
}

// Path: web.serverSettings.fields.projectsPrefix
class _TranslationsWebServerSettingsFieldsProjectsPrefixZh extends TranslationsWebServerSettingsFieldsProjectsPrefixEn {
	_TranslationsWebServerSettingsFieldsProjectsPrefixZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '项目前缀';
	@override String get hint => '项目笔记的文件夹名。默认 "projects"。';
}

// Path: web.serverSettings.fields.registryRoot
class _TranslationsWebServerSettingsFieldsRegistryRootZh extends TranslationsWebServerSettingsFieldsRegistryRootEn {
	_TranslationsWebServerSettingsFieldsRegistryRootZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '注册表根目录';
	@override String get hint => '存放 MCP server JSON 定义的目录。默认为 <vault>/mcp。';
}

// Path: web.serverSettings.fields.secretsFile
class _TranslationsWebServerSettingsFieldsSecretsFileZh extends TranslationsWebServerSettingsFieldsSecretsFileEn {
	_TranslationsWebServerSettingsFieldsSecretsFileZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '密钥文件';
	@override String get hint => 'spawn 时替换进 MCP server 命令的 key=value 文件。';
}

// Path: web.serverSettings.fields.memoryBackend
class _TranslationsWebServerSettingsFieldsMemoryBackendZh extends TranslationsWebServerSettingsFieldsMemoryBackendEn {
	_TranslationsWebServerSettingsFieldsMemoryBackendZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '嵌入器后端';
	@override String get hint => '"auto" / "bm25" 使用纯 Go 关键词路径（无需 cgo）；"http" 调用任何兼容 OpenAI 的 /v1/embeddings（ollama / OpenAI / LocalAI）；"local" 进程内运行 ONNX sentence-transformer — 需要用 `-tags local_onnx` 编译的二进制。';
}

// Path: web.serverSettings.fields.memoryStore
class _TranslationsWebServerSettingsFieldsMemoryStoreZh extends TranslationsWebServerSettingsFieldsMemoryStoreEn {
	_TranslationsWebServerSettingsFieldsMemoryStoreZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '存储';
	@override String get hint => '"pgvector" 复用 opendray 已有的 PG + vector 扩展；v1 唯一选项。';
}

// Path: web.serverSettings.fields.memoryTopK
class _TranslationsWebServerSettingsFieldsMemoryTopKZh extends TranslationsWebServerSettingsFieldsMemoryTopKEn {
	_TranslationsWebServerSettingsFieldsMemoryTopKZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '默认 top-K';
	@override String get hint => 'agent 未指定时 memory_search 返回多少条命中。留空 = 5。';
}

// Path: web.serverSettings.fields.memoryThreshold
class _TranslationsWebServerSettingsFieldsMemoryThresholdZh extends TranslationsWebServerSettingsFieldsMemoryThresholdEn {
	_TranslationsWebServerSettingsFieldsMemoryThresholdZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '相似度阈值';
	@override String get hint => '分数低于此值的命中将被丢弃。留空 = 0.1（宽松 — BM25 稀疏向量很少超过 0.5）。';
}

// Path: web.serverSettings.fields.memoryScope
class _TranslationsWebServerSettingsFieldsMemoryScopeZh extends TranslationsWebServerSettingsFieldsMemoryScopeEn {
	_TranslationsWebServerSettingsFieldsMemoryScopeZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '默认作用域';
	@override String get hint => 'agent 未指定时 memory_store 使用的作用域。"project"（推荐）按 cwd 分组；"session" 按会话隔离；"global" 跨 cwd 共享。';
}

// Path: web.serverSettings.fields.memoryBaseUrl
class _TranslationsWebServerSettingsFieldsMemoryBaseUrlZh extends TranslationsWebServerSettingsFieldsMemoryBaseUrlEn {
	_TranslationsWebServerSettingsFieldsMemoryBaseUrlZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Base URL';
	@override String get hint => '如 ollama 用 "http://localhost:11434/v1"，OpenAI 用 "https://api.openai.com/v1"。';
}

// Path: web.serverSettings.fields.memoryModel
class _TranslationsWebServerSettingsFieldsMemoryModelZh extends TranslationsWebServerSettingsFieldsMemoryModelEn {
	_TranslationsWebServerSettingsFieldsMemoryModelZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '模型';
	@override String get hint => '如 ollama 用 "nomic-embed-text"，OpenAI 用 "text-embedding-3-small"。';
}

// Path: web.serverSettings.fields.memoryApiKey
class _TranslationsWebServerSettingsFieldsMemoryApiKeyZh extends TranslationsWebServerSettingsFieldsMemoryApiKeyEn {
	_TranslationsWebServerSettingsFieldsMemoryApiKeyZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'API 密钥';
	@override String get hint => 'ollama / 本地服务可留空；OpenAI / Voyage / 托管服务必填。';
}

// Path: web.serverSettings.fields.memoryLocalModel
class _TranslationsWebServerSettingsFieldsMemoryLocalModelZh extends TranslationsWebServerSettingsFieldsMemoryLocalModelEn {
	_TranslationsWebServerSettingsFieldsMemoryLocalModelZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '模型名';
	@override String get hint => '仅作显示用 — 出现在日志 / Inspector 中。如 "bge-m3"、"bge-small-en-v1.5"。';
}

// Path: web.serverSettings.fields.memoryLibraryPath
class _TranslationsWebServerSettingsFieldsMemoryLibraryPathZh extends TranslationsWebServerSettingsFieldsMemoryLibraryPathEn {
	_TranslationsWebServerSettingsFieldsMemoryLibraryPathZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '库路径';
	@override String get hint => '存放 libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux) 的目录。`brew install onnxruntime` 后即 /opt/homebrew/opt/onnxruntime/lib。';
}

// Path: web.serverSettings.fields.memoryModelPath
class _TranslationsWebServerSettingsFieldsMemoryModelPathZh extends TranslationsWebServerSettingsFieldsMemoryModelPathEn {
	_TranslationsWebServerSettingsFieldsMemoryModelPathZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '模型路径';
	@override String get hint => '.onnx 权重的绝对路径。从 HuggingFace 下载，如 Xenova/bge-m3 或 Xenova/bge-small-en-v1.5。';
}

// Path: web.serverSettings.fields.memoryTokenizerPath
class _TranslationsWebServerSettingsFieldsMemoryTokenizerPathZh extends TranslationsWebServerSettingsFieldsMemoryTokenizerPathEn {
	_TranslationsWebServerSettingsFieldsMemoryTokenizerPathZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'Tokenizer 路径';
	@override String get hint => 'tokenizer.json（HuggingFace 标准格式）的绝对路径 — 通常和模型放在一起。';
}

// Path: web.serverSettings.fields.memoryMaxSeqLen
class _TranslationsWebServerSettingsFieldsMemoryMaxSeqLenZh extends TranslationsWebServerSettingsFieldsMemoryMaxSeqLenEn {
	_TranslationsWebServerSettingsFieldsMemoryMaxSeqLenZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '最大序列长度';
	@override String get hint => '超过这个 token 数会被截断。bge-m3 默认 512。留空 = 512。';
}

// Path: web.serverSettings.fields.claudeHistoryRoots
class _TranslationsWebServerSettingsFieldsClaudeHistoryRootsZh extends TranslationsWebServerSettingsFieldsClaudeHistoryRootsEn {
	_TranslationsWebServerSettingsFieldsClaudeHistoryRootsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '历史根目录';
	@override String get hint => '扫描 Claude 每项目 JSONL 记录的目录。留空 = 扫描 ~/.claude/projects + 所有 ~/.claude-accounts/*/projects。';
}

// Path: web.serverSettings.fields.claudeAccountsDir
class _TranslationsWebServerSettingsFieldsClaudeAccountsDirZh extends TranslationsWebServerSettingsFieldsClaudeAccountsDirEn {
	_TranslationsWebServerSettingsFieldsClaudeAccountsDirZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '账号目录';
	@override String get hint => 'opendray 管理的 Claude 账号 ConfigDir 根目录。默认 ~/.claude-accounts。';
}

// Path: web.serverSettings.fields.codexSessionsRoot
class _TranslationsWebServerSettingsFieldsCodexSessionsRootZh extends TranslationsWebServerSettingsFieldsCodexSessionsRootEn {
	_TranslationsWebServerSettingsFieldsCodexSessionsRootZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '会话根目录';
	@override String get hint => '遍历 Codex rollout JSONL 文件的目录。默认 ~/.codex/sessions。';
}

// Path: web.serverSettings.fields.geminiTmpRoot
class _TranslationsWebServerSettingsFieldsGeminiTmpRootZh extends TranslationsWebServerSettingsFieldsGeminiTmpRootEn {
	_TranslationsWebServerSettingsFieldsGeminiTmpRootZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'tmp 目录';
	@override String get hint => '存放 Gemini 每项目 tmp 文件夹的根目录。默认 ~/.gemini/tmp。';
}

// Path: web.serverSettings.fields.geminiProjectsFile
class _TranslationsWebServerSettingsFieldsGeminiProjectsFileZh extends TranslationsWebServerSettingsFieldsGeminiProjectsFileEn {
	_TranslationsWebServerSettingsFieldsGeminiProjectsFileZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'projects.json';
	@override String get hint => 'Gemini cwd→短名映射文件的路径。默认 ~/.gemini/projects.json。';
}

// Path: web.serverSettings.fields.backupLocalDir
class _TranslationsWebServerSettingsFieldsBackupLocalDirZh extends TranslationsWebServerSettingsFieldsBackupLocalDirEn {
	_TranslationsWebServerSettingsFieldsBackupLocalDirZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '本地备份目录';
	@override String get hint => '自动创建的 `local` 目标的默认根目录。留空 = ~/.opendray/backups。需要重启。';
}

// Path: web.serverSettings.fields.backupExportDir
class _TranslationsWebServerSettingsFieldsBackupExportDirZh extends TranslationsWebServerSettingsFieldsBackupExportDirEn {
	_TranslationsWebServerSettingsFieldsBackupExportDirZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => '导出目录';
	@override String get hint => '一次性导出 zip 在磁盘上的暂存位置。留空 = ~/.opendray/exports。包将在 24 小时后自动过期。需要重启。';
}

// Path: web.serverSettings.fields.backupPgDumpPath
class _TranslationsWebServerSettingsFieldsBackupPgDumpPathZh extends TranslationsWebServerSettingsFieldsBackupPgDumpPathEn {
	_TranslationsWebServerSettingsFieldsBackupPgDumpPathZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'pg_dump 路径';
	@override String get hint => 'pg_dump 的绝对路径。主版本号必须 ≥ 服务器的。留空 = PATH 上的第一个 pg_dump。';
}

// Path: web.serverSettings.fields.backupPgRestorePath
class _TranslationsWebServerSettingsFieldsBackupPgRestorePathZh extends TranslationsWebServerSettingsFieldsBackupPgRestorePathEn {
	_TranslationsWebServerSettingsFieldsBackupPgRestorePathZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get label => 'pg_restore 路径';
	@override String get hint => '/backups/restore 流程使用的 pg_restore 绝对路径。同样的主版本号规则。';
}

// Path: web.serverSettings.httpHelpers.presetTip
class _TranslationsWebServerSettingsHttpHelpersPresetTipZh extends TranslationsWebServerSettingsHttpHelpersPresetTipEn {
	_TranslationsWebServerSettingsHttpHelpersPresetTipZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get ollama => '本地 ollama 守护进程';
	@override String get lmStudio => 'LM Studio 本地服务';
	@override String get openai => 'OpenAI 云端（需要 API key）';
}

// Path: web.serverSettings.backup.scheduleHeaders
class _TranslationsWebServerSettingsBackupScheduleHeadersZh extends TranslationsWebServerSettingsBackupScheduleHeadersEn {
	_TranslationsWebServerSettingsBackupScheduleHeadersZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get schedule => '计划';
	@override String get target => '目标';
	@override String get cadence => '频率';
	@override String get keep => '保留';
	@override String get state => '状态';
}

// Path: web.settings.appearance.options
class _TranslationsWebSettingsAppearanceOptionsZh extends TranslationsWebSettingsAppearanceOptionsEn {
	_TranslationsWebSettingsAppearanceOptionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get light => '浅色';
	@override String get lightDesc => '始终浅色';
	@override String get dark => '深色';
	@override String get darkDesc => '始终深色';
	@override String get system => '跟随系统';
	@override String get systemDesc => '跟随操作系统设置';
}

// Path: web.settings.font.options
class _TranslationsWebSettingsFontOptionsZh extends TranslationsWebSettingsFontOptionsEn {
	_TranslationsWebSettingsFontOptionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get compact => '紧凑';
	@override String get kDefault => '默认';
	@override String get comfy => '舒适';
	@override String get large => '大';
}

// Path: web.memoryAmbient.providers.row
class _TranslationsWebMemoryAmbientProvidersRowZh extends TranslationsWebMemoryAmbientProvidersRowEn {
	_TranslationsWebMemoryAmbientProvidersRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get defaultBadge => '★ 默认';
	@override String get makeDefault => '设为默认';
	@override String get test => '测试';
	@override String get testing => '测试中…';
	@override String get delete => '删除';
	@override String testOk({required Object name}) => '${name}：连接成功';
	@override String get testFailedToast => '测试失败';
	@override String deleteConfirm({required Object name}) => '删除 provider "${name}"?';
	@override String get deletedToast => 'Provider 已删除';
	@override String get deleteFailedToast => '删除失败';
	@override String get updateFailedToast => '更新失败';
	@override String madeDefaultToast({required Object name}) => '${name} 已设为默认';
}

// Path: web.memoryAmbient.providers.dialog
class _TranslationsWebMemoryAmbientProvidersDialogZh extends TranslationsWebMemoryAmbientProvidersDialogEn {
	_TranslationsWebMemoryAmbientProvidersDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '添加 summarizer provider';
	@override String get kindLabel => '类型';
	@override String get nameLabel => '名称';
	@override String get namePlaceholder => '例如 lmstudio-qwen';
	@override String get modelLabel => '模型';
	@override String get baseUrlLabel => 'Base URL';
	@override String get integrationNote => 'Integration 类型 provider 通过一个已注册的集成解析 base URL。请先在 Integrations 中配置；更高级的 wiring（extra_config）在本版本中仅 DB 配置。';
	@override String get apiKeyLabel => 'API key';
	@override String get apiKeyHint => 'AES-GCM 加密存储（使用 backup 主口令）。不会被回显；保存后只显示指纹。';
	@override String get makeDefaultLabel => '将此设为默认 provider';
	@override String get create => '创建';
	@override String get nameRequiredToast => '名称不能为空';
	@override String createdToast({required Object name}) => '已创建 Provider ${name}';
	@override String get createFailedToast => '创建失败';
}

// Path: web.memoryAmbient.rules.row
class _TranslationsWebMemoryAmbientRulesRowZh extends TranslationsWebMemoryAmbientRulesRowEn {
	_TranslationsWebMemoryAmbientRulesRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get globalDefault => '全局默认';
	@override String get scopeLabel => 'scope:';
	@override String get dedupLabel => 'dedup:';
	@override String get runNow => '立即运行';
	@override String get running => '运行中…';
	@override String get delete => '删除';
	@override String firedToast({required Object sessions}) => '规则已对 ${sessions} 个会话触发';
	@override String get runNowFailedToast => '立即运行失败';
	@override String deleteConfirm({required Object name}) => '删除规则 "${name}"?';
	@override String get deletedToast => '规则已删除';
	@override String get deleteFailedToast => '删除失败';
	@override late final _TranslationsWebMemoryAmbientRulesRowSummaryZh summary = _TranslationsWebMemoryAmbientRulesRowSummaryZh._(_root);
}

// Path: web.memoryAmbient.rules.dialog
class _TranslationsWebMemoryAmbientRulesDialogZh extends TranslationsWebMemoryAmbientRulesDialogEn {
	_TranslationsWebMemoryAmbientRulesDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '添加捕获规则';
	@override String get nameLabel => '名称';
	@override String get triggerLabel => 'Trigger';
	@override String get nLabel => 'N（消息条数）';
	@override String get idleLabel => 'Idle 秒数';
	@override String get kLabel => 'K（字符数）';
	@override String get scopeLabel => '目标 scope';
	@override String get scopeSession => 'session';
	@override String get scopeProject => 'project（推荐）';
	@override String get scopeGlobal => 'global';
	@override String get dedupLabel => '去重阈值（0.0 – 1.0）';
	@override String get dedupHint => '越高 = 去重越严格。0.85 是推荐的平衡点。';
	@override String get create => '创建';
	@override String get nameRequiredToast => '名称不能为空';
	@override String createdToast({required Object name}) => '已创建规则 ${name}';
	@override String get createFailedToast => '创建失败';
}

// Path: web.memoryAmbient.profiles.row
class _TranslationsWebMemoryAmbientProfilesRowZh extends TranslationsWebMemoryAmbientProfilesRowEn {
	_TranslationsWebMemoryAmbientProfilesRowZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get globalDefault => '全局默认';
	@override String get delete => '删除';
	@override String get deleteConfirm => '删除该 injection profile?';
	@override String get deletedToast => 'Profile 已删除';
	@override String get deleteFailedToast => '删除失败';
}

// Path: web.memoryAmbient.profiles.dialog
class _TranslationsWebMemoryAmbientProfilesDialogZh extends TranslationsWebMemoryAmbientProfilesDialogEn {
	_TranslationsWebMemoryAmbientProfilesDialogZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get title => '添加 injection profile';
	@override String get strategyLabel => '策略';
	@override String get kLabel => 'K（注入的 top memory 数）';
	@override String get hint => '每个 session_id 一个 profile（或全局默认）。单会话 profile 暂只能通过 API 添加；UI 当前只管理全局默认。';
	@override String get create => '创建';
	@override String get createdToast => 'Profile 已创建';
	@override String get createFailedToast => '创建失败';
}

// Path: web.memoryAmbient.cost.columns
class _TranslationsWebMemoryAmbientCostColumnsZh extends TranslationsWebMemoryAmbientCostColumnsEn {
	_TranslationsWebMemoryAmbientCostColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get provider => 'Provider';
	@override String get calls => '调用';
	@override String get inTokens => '输入 token';
	@override String get outTokens => '输出 token';
	@override String get usdEst => 'USD 估算';
}

// Path: web.export.form.integrationOptions
class _TranslationsWebExportFormIntegrationOptionsZh extends TranslationsWebExportFormIntegrationOptionsEn {
	_TranslationsWebExportFormIntegrationOptionsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get none => '无';
	@override String get noneHint => '完全跳过 integrations 表。';
	@override String get metadata => '仅元数据（推荐）';
	@override String get metadataHint => 'ID、name、route prefix、scopes — 不包含任何 API key 凭证。';
	@override String get plaintext => '包含明文 API key';
	@override String get plaintextHint => 'v1 仅 bcrypt：不存在可恢复的明文。Manifest 会记录此事实；不会泄露任何内容。';
}

// Path: web.export.history.columns
class _TranslationsWebExportHistoryColumnsZh extends TranslationsWebExportHistoryColumnsEn {
	_TranslationsWebExportHistoryColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get status => '状态';
	@override String get scope => '范围';
	@override String get size => '大小';
	@override String get expires => '过期';
	@override String get actions => '操作';
}

// Path: web.export.import.summaryCard
class _TranslationsWebExportImportSummaryCardZh extends TranslationsWebExportImportSummaryCardEn {
	_TranslationsWebExportImportSummaryCardZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get memories => '记忆';
	@override String get integrations => '集成';
	@override String get customTasks => '自定义任务';
	@override String get created => '已创建';
	@override String get skipped => '已跳过';
	@override String get failed => '失败';
}

// Path: web.export.imports.columns
class _TranslationsWebExportImportsColumnsZh extends TranslationsWebExportImportsColumnsEn {
	_TranslationsWebExportImportsColumnsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get id => 'ID';
	@override String get status => '状态';
	@override String get source => '来源';
	@override String get counts => '计数';
	@override String get when => '时间';
}

// Path: sessions.inspector.shell.tabs
class _TranslationsSessionsInspectorShellTabsZh extends TranslationsSessionsInspectorShellTabsEn {
	_TranslationsSessionsInspectorShellTabsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get files => '文件';
	@override String get git => 'Git';
	@override String get tasks => '任务';
	@override String get history => '历史';
	@override String get notes => '笔记';
}

// Path: web.notes.vaultSync.conflict.kinds
class _TranslationsWebNotesVaultSyncConflictKindsZh extends TranslationsWebNotesVaultSyncConflictKindsEn {
	_TranslationsWebNotesVaultSyncConflictKindsZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get rebase => 'rebase';
	@override String get merge => 'merge';
	@override String get cherryPick => 'cherry-pick';
	@override String get operation => 'operation';
}

// Path: web.memoryAmbient.rules.row.summary
class _TranslationsWebMemoryAmbientRulesRowSummaryZh extends TranslationsWebMemoryAmbientRulesRowSummaryEn {
	_TranslationsWebMemoryAmbientRulesRowSummaryZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String afterMessages({required Object n}) => '每 ${n} 条消息';
	@override String onIdle({required Object seconds}) => 'idle ≥ ${seconds}s';
	@override String kChars({required Object k}) => '≥ ${k} 字符';
	@override String get manual => '仅手动';
}

/// The flat map containing all translations for locale <zh>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on TranslationsZh {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'common.ok' => '确定',
			'common.cancel' => '取消',
			'common.save' => '保存',
			'common.delete' => '删除',
			'common.edit' => '编辑',
			'common.back' => '返回',
			'common.done' => '完成',
			'common.close' => '关闭',
			'common.retry' => '重试',
			'common.copy' => '复制',
			'common.enabled' => '已启用',
			'common.refresh' => '刷新',
			'auth.signInTitle' => '登录',
			'auth.changeServer' => '更换',
			'auth.username' => '用户名',
			'auth.password' => '密码',
			'auth.signIn' => '登录',
			'auth.signingIn' => '登录中…',
			'auth.subtitle' => '请使用运维账号登录。',
			'auth.errorRequired' => '请输入用户名和密码',
			'auth.errorGeneric' => ({required Object error}) => '登录失败：${error}',
			'auth.errorFallback' => '登录失败',
			'nav.sessions' => '会话',
			'nav.memory' => '记忆',
			'nav.notes' => '笔记',
			'nav.more' => '更多',
			'nav.activity' => '活动',
			'nav.providers' => '提供方',
			'nav.channels' => '频道',
			'nav.integrations' => '集成',
			'nav.plugins' => '插件',
			'nav.backups' => '备份',
			'nav.settings' => '设置',
			'nav.tutorial' => '教程',
			'nav.workspace' => '工作区',
			'web.brand' => 'opendray',
			'web.loading' => '加载中…',
			'web.topbar.expandSidebar' => '展开侧边栏',
			'web.topbar.collapseSidebar' => '收起侧边栏',
			'web.topbar.search' => '搜索',
			'web.topbar.openPalette' => '打开命令面板',
			'web.topbar.theme' => '主题',
			'web.topbar.themeLabel' => ({required Object mode}) => '主题：${mode}',
			'web.topbar.appearance' => '外观',
			'web.topbar.themeLight' => '浅色',
			'web.topbar.themeDark' => '深色',
			'web.topbar.themeSystem' => '跟随系统',
			'web.topbar.language' => '语言',
			'web.topbar.languageEnglish' => 'English',
			'web.topbar.languageChinese' => '中文',
			'web.topbar.signedInAs' => '登录账号',
			'web.topbar.tokenExpires' => '令牌到期',
			'web.topbar.signOut' => '退出登录',
			'web.sessions.list.title' => '会话',
			'web.sessions.list.countSeparator' => '·',
			'web.sessions.list.newAria' => '创建新会话',
			'web.sessions.list.newTooltip' => '新建会话',
			'web.sessions.list.loading' => '加载中…',
			'web.sessions.list.emptyTitle' => '暂无会话。',
			'web.sessions.list.emptyHint' => ({required Object kbd}) => '按 ${kbd} 创建。',
			'web.sessions.list.endedHeader' => ({required Object count}) => '已结束 (${count})',
			'web.sessions.list.clearAll' => '清空全部',
			'web.sessions.list.confirmClearAll' => ({required Object count}) => '确定移除全部 ${count} 个已结束的会话?',
			'web.sessions.list.confirmTerminate' => ({required Object name}) => '终止并移除 ${name}?',
			'web.sessions.list.childPromoted' => ({required Object count}) => '其 ${count} 个子任务会话将被提升为顶级。',
			'web.sessions.list.childPromotedPlural' => ({required Object count}) => '其 ${count} 个子任务会话将被提升为顶级。',
			'web.sessions.list.footer' => ({required Object live, required Object ended}) => '${live} 运行中 · ${ended} 已结束',
			'web.sessions.list.row.deleteAria' => '删除会话',
			'web.sessions.list.row.titleRemoveHistory' => '从历史中移除',
			'web.sessions.list.row.titleTerminate' => '终止并移除',
			'web.sessions.list.row.titleRemove' => '移除',
			'web.sessions.list.row.claudeAccountTitle' => ({required Object label}) => 'Claude 账号：${label}',
			'web.sessions.list.deleteFailedToast' => '删除失败',
			'web.sessions.tabs.closeAria' => '关闭标签并移除会话',
			'web.sessions.tabs.closeTitle' => '关闭标签并移除会话',
			'web.sessions.page.removedToast' => '会话已移除',
			'web.sessions.page.removeFailedToast' => '移除失败',
			'web.sessions.page.stoppedToast' => '会话已停止',
			'web.sessions.page.stopFailedToast' => '停止失败',
			'web.sessions.page.restartedToast' => '会话已重启',
			'web.sessions.page.restartFailedToast' => '重启失败',
			'web.sessions.page.confirmCloseTabTitle' => ({required Object name}) => '停止并移除 "${name}"?',
			'web.sessions.page.confirmCloseTabDescription' => 'CLI 进程将被终止并删除该记录。',
			'web.sessions.page.confirmCloseTabConfirm' => '停止并移除',
			'web.sessions.page.confirmRemoveTitle' => ({required Object name}) => '移除 ${name}?',
			'web.sessions.page.confirmRemoveTitleFallback' => '移除会话?',
			'web.sessions.page.confirmRemoveDescription' => '这将删除该记录。',
			'web.sessions.page.confirmRemoveConfirm' => '移除',
			'web.sessions.empty.title' => '未打开任何会话',
			'web.sessions.empty.hint' => ({required Object kbdN, required Object kbdW, required Object kbdRange}) => '从列表中挑选一个会话，或新建一个。快捷键：${kbdN} 新建，${kbdW} 关闭，${kbdRange} 切换。',
			'web.sessions.empty.spawn' => '创建会话',
			'web.sessions.header.loadingSession' => '正在加载会话…',
			'web.sessions.header.showList' => '显示会话列表',
			'web.sessions.header.hideList' => '隐藏会话列表',
			'web.sessions.header.showInspector' => '显示检查器',
			'web.sessions.header.hideInspector' => '隐藏检查器',
			'web.sessions.header.attachImage' => '附加图片',
			'web.sessions.header.attachImageTooltip' => '附加图片（或直接粘贴 / 拖入终端）',
			'web.sessions.header.restart' => '重启',
			'web.sessions.header.restarting' => '重启中…',
			'web.sessions.header.remove' => '移除',
			'web.sessions.header.removing' => '移除中…',
			'web.sessions.header.stop' => '停止',
			'web.sessions.header.stopping' => '停止中…',
			'web.sessions.header.pid' => ({required Object pid}) => 'pid ${pid}',
			'web.sessions.terminal.uploadingToast' => '正在上传图片…',
			'web.sessions.terminal.uploadedToast' => '图片已附加',
			'web.sessions.terminal.uploadFailedToast' => '上传失败',
			'web.sessions.terminal.uploadInvalidTypeToast' => '仅支持图片文件',
			'web.sessions.terminal.dropToAttach' => '释放以附加图片',
			'web.sessions.spawn.title' => '创建会话',
			'web.sessions.spawn.description' => '在已注册的 Provider 下启动一个 CLI 会话。',
			'web.sessions.spawn.provider' => 'Provider',
			'web.sessions.spawn.claudeAccount' => 'Claude 账号',
			'web.sessions.spawn.loadingAccounts' => '正在加载账号…',
			'web.sessions.spawn.noAccounts' => '尚未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。',
			'web.sessions.spawn.kDefault' => '默认',
			'web.sessions.spawn.defaultTooltip' => '使用系统 keychain / 环境变量',
			'web.sessions.spawn.tokenEmptyBadge' => '·未填',
			'web.sessions.spawn.tokenMissingTooltip' => '未设置 token — 请先在 Providers 面板中填入',
			'web.sessions.spawn.multiAccountHint' => '已配置多个账号 — 请为本次会话挑选一个。',
			'web.sessions.spawn.cwd' => '工作目录',
			'web.sessions.spawn.cwdPlaceholder' => '/Users/you/projects/foo',
			'web.sessions.spawn.browse' => '浏览',
			'web.sessions.spawn.nameLabel' => '名称（可选）',
			'web.sessions.spawn.namePlaceholder' => 'claude in pet-tracker',
			'web.sessions.spawn.argsLabel' => 'CLI 参数（每行一个）',
			'web.sessions.spawn.bypassClaude' => '跳过权限提示',
			'web.sessions.spawn.bypassCodex' => '跳过所有批准与沙盒 (--dangerously-bypass-approvals-and-sandbox)',
			'web.sessions.spawn.bypassGemini' => 'YOLO 模式 (--yolo)',
			'web.sessions.spawn.bypassOnHint' => '本次会话将以更高的自主权运行。',
			'web.sessions.spawn.bypassOffHint' => '关闭 — 确认与提示按正常流程处理。',
			'web.sessions.spawn.errorPickProvider' => '请选择一个 Provider。',
			'web.sessions.spawn.errorCwdRequired' => '请填写工作目录。',
			'web.sessions.spawn.cancel' => '取消',
			'web.sessions.spawn.submit' => '创建',
			'web.sessions.spawn.submitting' => '创建中…',
			'web.sessions.spawn.spawnedToast' => '会话已创建',
			'web.sessions.spawn.spawnedDescription' => ({required Object provider, required Object pid}) => '${provider} · pid ${pid}',
			'web.sessions.spawn.pidFallback' => '—',
			'web.sessions.accountSwitcher.tooltip' => '切换 Claude 账号（将重启 CLI 进程）',
			'web.sessions.accountSwitcher.currentDefault' => '默认',
			'web.sessions.accountSwitcher.menuTitle' => '切换 Claude 账号',
			'web.sessions.accountSwitcher.defaultName' => '默认',
			'web.sessions.accountSwitcher.defaultSubtitle' => 'CLI 的系统 keychain / 环境变量',
			'web.sessions.accountSwitcher.tokenEmpty' => '·未填',
			'web.sessions.accountSwitcher.confirmSwitch' => '切换账号将重启 Claude CLI 进程。CLI 内部进行中的对话状态将丢失。继续？',
			'web.sessions.accountSwitcher.switchedToast' => '账号已切换',
			'web.sessions.accountSwitcher.switchedDescription' => ({required Object account, required Object pid}) => '当前使用 @${account} · pid ${pid}',
			'web.sessions.accountSwitcher.switchedDefault' => '默认',
			'web.sessions.accountSwitcher.switchFailedToast' => '切换失败',
			'web.sessions.inspector.tabs.files' => '文件',
			'web.sessions.inspector.tabs.git' => 'Git',
			'web.sessions.inspector.tabs.search' => '搜索',
			'web.sessions.inspector.tabs.tasks' => '任务',
			'web.sessions.inspector.tabs.history' => '历史',
			'web.sessions.inspector.tabs.notes' => '笔记',
			'web.sessions.inspector.tabs.memory' => '记忆',
			'web.sessions.ended.bufferUnavailable' => '[缓冲区不可用]',
			'web.sessions.ended.readOnlyBanner' => '[会话已结束 — 只读缓冲区]',
			'web.sessions.fileBrowser.title' => '选择工作目录',
			'web.sessions.fileBrowser.description' => '浏览网关主机的文件系统并选择一个文件夹。',
			'web.sessions.fileBrowser.parent' => '上级目录',
			'web.sessions.fileBrowser.home' => '家目录',
			'web.sessions.fileBrowser.refresh' => '刷新',
			'web.sessions.fileBrowser.pathPlaceholder' => '/Users/you/projects',
			'web.sessions.fileBrowser.loading' => '加载中…',
			'web.sessions.fileBrowser.empty' => '目录为空。',
			'web.sessions.fileBrowser.newFolder' => '新建文件夹',
			'web.sessions.fileBrowser.newFolderPlaceholder' => 'new-folder-name',
			'web.sessions.fileBrowser.create' => '创建',
			'web.sessions.fileBrowser.cancel' => '取消',
			'web.sessions.fileBrowser.useThisFolder' => '使用此文件夹',
			'web.sessions.fileBrowser.createdToast' => '文件夹已创建',
			'web.sessions.fileBrowser.mkdirFailedToast' => '创建失败',
			'web.sessions.fileBrowser.homeFailedToast' => '读取家目录失败',
			'web.memory.title' => '记忆',
			'web.memory.subtitle' => '浏览、搜索并编辑 Agent 通过 opendray-memory MCP 服务器存储的记忆。',
			'web.memory.navProject' => '项目',
			'web.memory.navCleanupInbox' => '清理收件箱',
			'web.memory.navWorkers' => 'Workers',
			'web.memory.navConfiguration' => '配置 →',
			'web.conflicts.title' => '跨层冲突',
			'web.conflicts.subtitle' => '每日 detector 在 facts/plan/goal/journal 之间发现的矛盾。',
			'web.conflicts.loading' => '正在加载冲突…',
			'web.conflicts.empty' => '无待处理冲突。点击"立即检测"运行一次按需扫描。',
			'web.conflicts.pickCwd' => '选一个项目查看其冲突。',
			'web.conflicts.detectNow' => '立即检测',
			'web.conflicts.detected' => ({required Object count}) => '新发现 ${count} 条冲突',
			'web.conflicts.accept' => '采纳',
			'web.conflicts.dismiss' => '驳回',
			'web.conflicts.accepted' => '已采纳 — 别忘了实际修正',
			'web.conflicts.dismissed' => '已驳回',
			'web.conflicts.severity.low' => '低',
			'web.conflicts.severity.medium' => '中',
			'web.conflicts.severity.high' => '高',
			'web.memoryHealth.title' => ({required Object days}) => '记忆健康 — 最近 ${days} 天',
			'web.memoryHealth.subtitle' => '本项目两套记忆子系统的运行情况聚合。',
			'web.memoryHealth.loading' => '正在加载健康快照…',
			'web.memoryHealth.errorLoading' => '加载健康快照失败。',
			'web.memoryHealth.pickCwd' => '选一个项目查看其记忆健康。',
			'web.memoryHealth.newFacts' => '新增 facts',
			'web.memoryHealth.newFactsHint' => ({required Object total}) => '累计 ${total} 条',
			'web.memoryHealth.captureFires' => 'Capture 触发次数',
			'web.memoryHealth.captureFiresHint' => ({required Object stored, required Object deduped}) => '存入 ${stored} · 去重 ${deduped}',
			'web.memoryHealth.newJournal' => '新增日志条目',
			'web.memoryHealth.newJournalHint' => ({required Object total}) => '累计 ${total} 条',
			'web.memoryHealth.planAge' => '计划最近更新',
			'web.memoryHealth.planAgeHint' => ({required Object count}) => '${count} 条 plan-drift 提案待审',
			'web.memoryHealth.planAgeHintNone' => '无 plan-drift 提案待审',
			'web.memoryHealth.goalAge' => '目标最近更新',
			'web.memoryHealth.pending' => '待审提案',
			'web.memoryHealth.pendingHint' => ({required Object days}) => '最久 ${days} 天',
			'web.memoryHealth.topHit' => ({required Object hits}) => '命中最多 · ${hits} 次',
			'web.memoryHealth.zeroHit' => ({required Object count}) => '${count} 条超过 7 天未命中的 fact — 清理候选。',
			'web.memoryHealth.never' => '从未',
			'web.memoryHealth.today' => '今日',
			'web.memoryHealth.daysAgo_one' => ({required Object count}) => '${count} 天前',
			'web.memoryHealth.daysAgo_other' => ({required Object count}) => '${count} 天前',
			'web.memoryWorkers.title' => 'Memory workers',
			'web.memoryWorkers.loading' => '正在加载 worker 配置…',
			'web.memoryWorkers.errorTitle' => '无法访问该接口。',
			'web.memoryWorkers.errorDescription' => '/api/v1/memory/workers 路由在 M25 中新增 — opendray 二进制可能需要重启以挂载它们并执行 migration 0029。',
			'web.memoryWorkers.intro' => 'Memory 子系统的每个 LLM 接入点都可由本地 <1>summarizer</1>（LM Studio / OpenAI 兼容）端点服务，或通过以 <5>--print</5> 模式启动的无头 <3>Claude / Gemini agent</3> 服务。叙事性任务（gitactivity、transcript）从 agent worker 中获益；高频任务（gatekeeper）按设计仍走本地端点。',
			'web.memoryWorkers.enabledBadge' => '已启用',
			'web.memoryWorkers.disabledBadge' => '已禁用',
			'web.memoryWorkers.summarizerOnlyBadge' => '仅 summarizer',
			'web.memoryWorkers.callsCount' => ({required Object count}) => '${count} 次调用 · 24 小时',
			'web.memoryWorkers.avgMs' => ({required Object ms}) => '平均 ${ms}ms',
			'web.memoryWorkers.errorsCount' => ({required Object count}) => '${count} 次错误',
			'web.memoryWorkers.workerLabel' => 'Worker',
			'web.memoryWorkers.summarizerHttp' => 'Summarizer (HTTP)',
			'web.memoryWorkers.agentCliPrint' => 'Agent (CLI --print)',
			'web.memoryWorkers.summarizerProviderLabel' => 'Summarizer Provider',
			'web.memoryWorkers.registryDefault' => '注册表默认值',
			'web.memoryWorkers.cliLabel' => 'CLI',
			'web.memoryWorkers.selectPlaceholder' => '选择',
			'web.memoryWorkers.cliClaude' => 'Claude',
			'web.memoryWorkers.cliGemini' => 'Gemini',
			'web.memoryWorkers.claudeAccountLabel' => 'Claude 账号',
			'web.memoryWorkers.claudeAccountDefault' => '默认',
			'web.memoryWorkers.agentWarning' => 'Agent 模式每次调用都会启动一个无头 CLI。延迟从 <1>~1s</1>（summarizer）上升到 <3>~5-15s</3>；成本从 CPU 转移到 Claude/Gemini 配额。',
			'web.memoryWorkers.enabledCheckbox' => '启用',
			'web.memoryWorkers.testButton' => '测试',
			'web.memoryWorkers.saveButton' => '保存',
			'web.memoryWorkers.recentCalls' => ({required Object count}) => '最近调用 (${count})',
			'web.memoryWorkers.tableWhen' => '时间',
			'web.memoryWorkers.tableWorker' => 'worker',
			'web.memoryWorkers.tableMs' => 'ms',
			'web.memoryWorkers.tableOk' => 'ok',
			'web.memoryWorkers.savedToast' => ({required Object label}) => '${label} 已更新',
			'web.memoryWorkers.saveFailedToast' => '保存失败',
			'web.memoryWorkers.testOkToast' => ({required Object label, required Object ms}) => '${label} OK — ${ms}ms',
			'web.memoryWorkers.testFailedToast' => ({required Object label}) => '${label} 失败',
			'web.memoryWorkers.testCallFailedToast' => '测试调用失败',
			'web.memoryWorkers.unknownError' => '未知错误',
			'web.memoryWorkers.tasks.gatekeeper.label' => 'Gatekeeper',
			'web.memoryWorkers.tasks.gatekeeper.description' => '每次 memory_store 前的预写过滤器。高频（目标 <500ms） — 仅 summarizer。',
			'web.memoryWorkers.tasks.cleaner.label' => 'Cleaner librarian',
			'web.memoryWorkers.tasks.cleaner.description' => '周期性 LLM 整理。对旧记忆判定为保留 / 过期 / 重复。',
			'web.memoryWorkers.tasks.gitactivity.label' => 'Git 活动总结器',
			'web.memoryWorkers.tasks.gitactivity.description' => 'git log → 每 24 小时生成 2-3 段叙事。天然适合 agent worker。',
			'web.memoryWorkers.tasks.transcript.label' => '会话记录总结器',
			'web.memoryWorkers.tasks.transcript.description' => '会话结束时的“agent 都做了什么”总结。天然适合 agent worker。',
			'web.cleanupInbox.loading' => '加载中…',
			'web.cleanupInbox.emptyTitle' => '清理收件箱为空',
			'web.cleanupInbox.emptyDescription' => '目前所有项目均无待处理的清理决策。LLM librarian 要么尚未跑过这些记忆，要么判断全部仍是关键内容。',
			'web.cleanupInbox.title' => '清理收件箱',
			'web.cleanupInbox.subtitle' => 'LLM memory librarian 的跨项目待处理决策。批准 stale → 删除，批准 duplicate → 合并，批准 keep → 一段时间内冻结该条目不再被重新判断。',
			'web.cleanupInbox.globalScope' => '(全局)',
			'web.cleanupInbox.orphanBadge' => '孤立',
			'web.cleanupInbox.orphanTitle' => 'scope_key 被截断（老旧的镜像导入数据）。不是可导航的项目。',
			'web.cleanupInbox.openProject' => '打开项目',
			'web.cleanupInbox.mergeIntoPrefix' => '→ 合并到',
			'web.cleanupInbox.reasonPrefix' => '原因：',
			'web.cleanupInbox.executeButton' => '执行',
			'web.cleanupInbox.confirmKeepButton' => '确认保留',
			'web.cleanupInbox.rejectButton' => '驳回',
			'web.cleanupInbox.approvedKeptToast' => '已保留',
			'web.cleanupInbox.approvedExecutedToast' => ({required Object verdict}) => '已执行 ${verdict}',
			'web.cleanupInbox.approveFailedToast' => '批准失败',
			'web.cleanupInbox.rejectedToast' => '已驳回 — 记忆保留',
			'web.cleanupInbox.rejectFailedToast' => '驳回失败',
			'web.project.picker.title' => '选择项目',
			'web.project.picker.subtitle' => '项目记忆按工作目录划分。选择一个以管理它的目标、计划、日志和清理队列。',
			'web.project.picker.pathPlaceholder' => '/path/to/your/project',
			'web.project.picker.browse' => '浏览',
			'web.project.picker.browseTooltip' => '浏览网关主机的文件系统',
			'web.project.picker.open' => '打开',
			'web.project.picker.recentLabel' => '最近的项目（来自已存记忆）：',
			'web.project.picker.orphanTooltip' => '看上去是被截断的 scope_key（老旧镜像导入 bug）。可能没有项目文档。',
			'web.project.picker.orphanBadge' => '孤立',
			'web.project.noCwd' => '选择一个项目以管理其记忆。',
			'web.project.header.docsCount_one' => ({required Object count}) => '${count} 份文档',
			'web.project.header.docsCount_other' => ({required Object count}) => '${count} 份文档',
			'web.project.header.journalEntries_one' => ({required Object count}) => '${count} 条日志',
			'web.project.header.journalEntries_other' => ({required Object count}) => '${count} 条日志',
			'web.project.header.pendingProposals_one' => ({required Object count}) => '${count} 条待处理提案',
			'web.project.header.pendingProposals_other' => ({required Object count}) => '${count} 条待处理提案',
			'web.project.header.cleanupPending' => ({required Object count}) => '${count} 条待清理',
			'web.project.tabs.health' => '健康',
			'web.project.tabs.goal' => '目标',
			'web.project.tabs.plan' => '计划',
			'web.project.tabs.tech' => '技术栈',
			'web.project.tabs.activity' => '活动',
			'web.project.tabs.journal' => '日志',
			'web.project.tabs.inbox' => '收件箱',
			'web.project.tabs.conflicts' => '冲突',
			'web.project.tabs.cleanup' => '清理',
			'web.project.docLabel.goal' => '目标',
			'web.project.docLabel.plan' => '计划',
			'web.project.docLabel.tech_stack' => '技术栈',
			'web.project.docLabel.recent_activity' => '最近活动',
			'web.project.verdictLabel.stale' => '删除',
			'web.project.verdictLabel.duplicate' => '合并',
			'web.project.verdictLabel.keep' => '保留',
			'web.project.editor.updatedBy' => '更新者',
			'web.project.editor.noDocSet' => ({required Object label}) => '尚未设置${label}。',
			'web.project.editor.save' => '保存',
			'web.project.editor.saveFailedToast' => '保存失败',
			'web.project.editor.savedToast' => ({required Object label}) => '${label}已保存',
			'web.project.editor.goalPlaceholder' => '我们在做什么？一段文字。每个 agent 在 spawn 时都会读取。',
			'web.project.editor.planPlaceholder' => '当前计划 — 现在在做什么、下一步是什么。随进度更新。',
			'web.project.readonly.tech_stack.label' => '技术栈与结构',
			'web.project.readonly.tech_stack.empty' => '在该项目运行一次 Claude 会话 — scanner 会在每次 spawn 时刷新。',
			'web.project.readonly.recent_activity.label' => '最近活动 (git → LLM)',
			'web.project.readonly.recent_activity.empty' => 'Git 活动总结每 24 小时运行；下次调度后再来看看。',
			'web.project.readonly.noneCaptured' => ({required Object label}) => '尚未捕获${label}。',
			'web.project.readonly.generatedBy' => '生成者',
			'web.project.readonly.lastRefresh' => '最近刷新',
			'web.project.journal.loading' => '加载中…',
			'web.project.journal.empty' => '暂无日志条目。每次会话结束都会自动追加一条。',
			'web.project.inbox.loading' => '加载中…',
			'web.project.inbox.emptyTitle' => '收件箱为空。',
			'web.project.inbox.emptyHint' => 'Agent 通过 `project_goal_set` / `project_plan_set` MCP 工具在这里提交提案。',
			'web.project.inbox.approvedToast' => ({required Object label}) => '${label}已更新',
			'web.project.inbox.approveFailedToast' => '批准失败',
			'web.project.inbox.rejectedToast' => '已驳回',
			'web.project.inbox.rejectFailedToast' => '驳回失败',
			'web.project.inbox.sessionPrefix' => 'ses',
			'web.project.inbox.warning' => ({required Object label}) => '批准将完全替换当前${label}。',
			'web.project.inbox.warningSuffix' => '请检查下方 diff；这不是合并。',
			'web.project.inbox.current' => '当前',
			'web.project.inbox.proposed' => '提议',
			'web.project.inbox.emptyBody' => '(空)',
			'web.project.inbox.approve' => '批准',
			'web.project.inbox.reject' => '驳回',
			'web.project.inbox.confirmDialogTitle' => ({required Object label}) => '替换${label}?',
			'web.project.inbox.confirmDialogDescription' => ({required Object label}) => '当前${label}将被提议内容覆盖。无法通过本界面撤销（可手动改回）。',
			'web.project.inbox.confirmCancel' => '取消',
			'web.project.inbox.confirmReplace' => '确认替换',
			'web.project.cleanup.hint' => 'LLM librarian 为该项目的记忆提出保留 / 过期 / 重复 的判定。删除前需你批准。',
			'web.project.cleanup.runNow' => '立即运行清理',
			'web.project.cleanup.runSucceededToast' => ({required Object decided, required Object scanned}) => '清理已运行：${decided} 条决策入队（扫描了 ${scanned} 条）',
			'web.project.cleanup.runFailedToast' => '清理运行失败',
			'web.project.cleanup.empty' => '暂无待处理决策。要么没有记忆达到判定年龄，要么上次运行判断全部仍是关键内容。',
			'web.project.cleanup.mergeIntoPrefix' => '→ 合并到',
			'web.project.cleanup.reasonPrefix' => '原因：',
			'web.project.cleanup.executeButton' => '执行',
			'web.project.cleanup.confirmKeepButton' => '确认保留',
			'web.project.cleanup.rejectButton' => '驳回',
			'web.project.cleanup.approvedExecutedToast' => ({required Object label}) => '已执行${label}',
			'web.project.cleanup.approveFailedToast' => '批准失败',
			'web.project.cleanup.rejectedToast' => '已驳回 — 记忆保留',
			'web.project.cleanup.rejectFailedToast' => '驳回失败',
			'web.project.reset.button' => '重置',
			'web.project.reset.dialogTitle' => '重置项目记忆?',
			'web.project.reset.dialogDescription' => '删除该 cwd 下存储的所有项目上下文。不可撤销。',
			'web.project.reset.alwaysDeleted' => '始终删除：目标、计划、提案、日志、清理决策。',
			'web.project.reset.alsoDeleteScannerLabel' => '同时删除 scanner 文档',
			'web.project.reset.alsoDeleteScannerSuffix' => '(tech_stack + recent_activity)。',
			'web.project.reset.alsoDeleteScannerHint' => '下次 spawn 会自动重建 — 通常不勾选即可。',
			'web.project.reset.alsoDeleteMemoriesLabel' => '同时删除 pgvector 记忆',
			'web.project.reset.alsoDeleteMemoriesSuffix' => '（该 scope_key 下的）。',
			'web.project.reset.alsoDeleteMemoriesHint' => 'Agent 存储的长期事实（用户偏好、项目事实）。无法恢复。',
			'web.project.reset.cancel' => '取消',
			'web.project.reset.deleteForever' => '永久删除',
			'web.project.reset.successToast' => ({required Object summary}) => '重置：已删除 ${summary}',
			'web.project.reset.summary.docs_one' => ({required Object count}) => '${count} 份文档',
			'web.project.reset.summary.docs_other' => ({required Object count}) => '${count} 份文档',
			'web.project.reset.summary.journal' => ({required Object count}) => '${count} 条日志',
			'web.project.reset.summary.proposals_one' => ({required Object count}) => '${count} 条提案',
			'web.project.reset.summary.proposals_other' => ({required Object count}) => '${count} 条提案',
			'web.project.reset.summary.cleanup' => ({required Object count}) => '${count} 条清理',
			'web.project.reset.summary.memories' => ({required Object count}) => '${count} 条记忆',
			'web.project.reset.failedToast' => '重置失败',
			'web.memoryInspector.status.label' => '当前 embedder',
			'web.memoryInspector.status.unavailable' => '不可用',
			'web.memoryInspector.status.probing' => '探测中…',
			'web.memoryInspector.status.dimensions' => ({required Object dim, required Object state}) => '${dim} 维 · ${state}',
			'web.memoryInspector.status.enabled' => '已启用',
			'web.memoryInspector.status.disabled' => '已禁用',
			'web.memoryInspector.status.testButton' => '测试 embedder',
			'web.memoryInspector.statusBody' => '这是网关当前用于每次 <1>memory_search</1> / <3>memory_store</3> 调用的 embedder。如果与上方配置不一致，说明有未保存的更改 — 点击 Save 后重启服务即可生效。',
			'web.memoryInspector.scope.label' => 'Scope',
			'web.memoryInspector.scope.scopeKey' => 'Scope key',
			'web.memoryInspector.scope.scopeKeyIgnored' => '(global 时忽略)',
			'web.memoryInspector.scope.scopeKeyCwd' => '(项目的 cwd)',
			'web.memoryInspector.scope.scopeKeySession' => '(session id)',
			'web.memoryInspector.scope.placeholderProject' => '/path/to/project (cwd)',
			'web.memoryInspector.scope.placeholderSession' => 'session id',
			'web.memoryInspector.scope.syncMd' => '同步 .md',
			'web.memoryInspector.scope.syncTooltip' => '把 Claude 的 <cwd>/.claude/memory/*.md 重新摄取到 pgvector',
			'web.memoryInspector.scope.values.project' => 'project',
			'web.memoryInspector.scope.values.session' => 'session',
			'web.memoryInspector.scope.values.global' => 'global',
			'web.memoryInspector.search.placeholder' => '语义搜索查询（Enter 运行；为空则浏览）',
			'web.memoryInspector.search.run' => '搜索',
			'web.memoryInspector.search.clear' => '清空',
			'web.memoryInspector.search.failedToast' => '搜索失败',
			'web.memoryInspector.records.noMemories' => '暂无记忆',
			'web.memoryInspector.records.matches_one' => ({required Object count}) => '${count} 条匹配',
			'web.memoryInspector.records.matches_other' => ({required Object count}) => '${count} 条匹配',
			'web.memoryInspector.records.memories_one' => ({required Object count}) => '${count} 条记忆',
			'web.memoryInspector.records.memories_other' => ({required Object count}) => '${count} 条记忆',
			'web.memoryInspector.records.scopeGlobalSuffix' => '（全局）',
			'web.memoryInspector.records.scopeInSuffix' => ({required Object scope}) => '（${scope}：',
			'web.memoryInspector.records.addButton' => '添加记忆',
			'web.memoryInspector.records.addTooltip' => '手动在此 scope 创建一条记忆',
			'web.memoryInspector.records.deleteAll' => '全部删除',
			'web.memoryInspector.records.deleteAllTooltip' => '删除该 scope/scope_key 下的全部记忆',
			'web.memoryInspector.records.loading' => '加载中…',
			'web.memoryInspector.records.enterScopeKeyHint' => '输入 scope key 以浏览记忆。',
			'web.memoryInspector.records.noMatchesForQuery' => ({required Object query}) => '未找到匹配 "${query}"',
			'web.memoryInspector.records.noMemoriesInScope' => '此 scope 暂无记忆。',
			'web.memoryInspector.row.simBadge' => ({required Object value}) => '相似度 ${value}',
			'web.memoryInspector.row.hits_one' => ({required Object count}) => '命中 ${count} 次',
			'web.memoryInspector.row.hits_other' => ({required Object count}) => '命中 ${count} 次',
			'web.memoryInspector.row.lastHitTooltip' => ({required Object relative}) => '最近命中 ${relative}',
			'web.memoryInspector.row.editPlaceholder' => '记忆文本 — Cmd/Ctrl+Enter 保存 · Esc 取消',
			'web.memoryInspector.row.saveTooltip' => '保存 (Cmd/Ctrl+Enter)',
			'web.memoryInspector.row.cancelTooltip' => '取消 (Esc)',
			'web.memoryInspector.row.editTooltip' => '编辑该记忆',
			'web.memoryInspector.row.deleteTooltip' => '删除该记忆',
			'web.memoryInspector.row.emptyError' => '记忆文本不能为空',
			'web.memoryInspector.row.deleteConfirm' => ({required Object id}) => '删除记忆 ${id}? 不可恢复。',
			'web.memoryInspector.toasts.deleted' => '记忆已删除',
			'web.memoryInspector.toasts.deleteFailed' => '删除失败',
			'web.memoryInspector.toasts.bulkDeleted_one' => ({required Object count}) => '已从此 scope 删除 ${count} 条记忆',
			'web.memoryInspector.toasts.bulkDeleted_other' => ({required Object count}) => '已从此 scope 删除 ${count} 条记忆',
			'web.memoryInspector.toasts.bulkDeleteFailed' => '批量删除失败',
			'web.memoryInspector.toasts.created' => '记忆已创建',
			'web.memoryInspector.toasts.createFailed' => '创建失败',
			'web.memoryInspector.toasts.updated' => '记忆已更新',
			'web.memoryInspector.toasts.updateFailed' => '更新失败',
			'web.memoryInspector.toasts.migrated' => ({required Object reembed, required Object examined, required Object to}) => '已迁移 ${reembed}/${examined} 条记忆到 ${to}',
			'web.memoryInspector.toasts.migrationFailed' => '迁移失败',
			'web.memoryInspector.toasts.syncIngested_one' => ({required Object count}) => '已摄取 ${count} 个新记忆文件',
			'web.memoryInspector.toasts.syncIngested_other' => ({required Object count}) => '已摄取 ${count} 个新记忆文件',
			'web.memoryInspector.toasts.syncEmpty' => '没有需要同步的新 .md 文件',
			'web.memoryInspector.toasts.syncEmptyDescription' => '已是最新，或该 cwd 没有 Claude memory 目录。',
			'web.memoryInspector.toasts.syncFailed' => '同步失败',
			'web.memoryInspector.toasts.testOk' => ({required Object embedder, required Object dim}) => 'Embedder OK：${embedder} · ${dim} 维',
			'web.memoryInspector.toasts.testOkDescription' => ({required Object preview}) => 'vector_preview = [${preview}…]',
			'web.memoryInspector.toasts.testFailed' => 'Embedder 探测失败',
			'web.memoryInspector.bulkDelete.title' => '删除此 scope 的全部记忆?',
			'web.memoryInspector.bulkDelete.description' => '这是一次 SQL 操作 — 该 scope 下全部记忆将被原子性删除。通过 Claude 镜像摄取的记忆会在下次 <1>同步 .md</1> 时重新出现；其余内容永久消失。',
			'web.memoryInspector.bulkDelete.scope' => 'Scope',
			'web.memoryInspector.bulkDelete.scopeKey' => 'Scope key',
			'web.memoryInspector.bulkDelete.currentlyVisible' => '当前可见',
			'web.memoryInspector.bulkDelete.items_one' => ({required Object count}) => '${count} 条记忆',
			'web.memoryInspector.bulkDelete.items_other' => ({required Object count}) => '${count} 条记忆',
			'web.memoryInspector.bulkDelete.cancel' => '取消',
			'web.memoryInspector.bulkDelete.deleteAll' => '全部删除',
			'web.memoryInspector.addMem.title' => '添加记忆',
			'web.memoryInspector.addMem.description' => '手动创建一条记忆。Agent 会通过 <1>memory_store</1> MCP 工具自动创建；此表单用于运维想跳过 agent 直接录入事实的场景。',
			'web.memoryInspector.addMem.textLabel' => '文本',
			'web.memoryInspector.addMem.textPlaceholder' => '纯文本。Embedder 在存储时把它转成向量；agent 通过 memory_search 取回。',
			'web.memoryInspector.addMem.cancel' => '取消',
			'web.memoryInspector.addMem.create' => '创建',
			'web.memoryInspector.picker.button' => '选择',
			'web.memoryInspector.picker.buttonTooltip' => '从已保存的 scope key 或活跃会话中选择',
			'web.memoryInspector.picker.loading' => '加载中…',
			'web.memoryInspector.picker.empty' => ({required Object scope}) => '${scope} 暂无已保存的 key 或活跃会话。',
			'web.memoryInspector.picker.savedHeader' => '已保存的记忆',
			'web.memoryInspector.picker.activeHeader' => '活跃会话',
			'web.memoryInspector.migrationBanner.headline_one' => ({required Object count}) => '${count} 条记忆不会出现在搜索结果中',
			'web.memoryInspector.migrationBanner.headline_other' => ({required Object count}) => '${count} 条记忆不会出现在搜索结果中',
			'web.memoryInspector.migrationBanner.subtitle' => ({required Object summary, required Object current}) => '${summary} — 当前 embedder 为 <1>${current}</1>。pgvector 按 embedder 分区其相似度索引，旧条目在重嵌入前不会被检索到。',
			'web.memoryInspector.migrationBanner.summaryItem' => ({required Object count, required Object name}) => '${count} 条在 ${name}',
			'web.memoryInspector.migrationBanner.migrateButton' => '迁移',
			'web.memoryInspector.reembed.title' => '重新嵌入记忆',
			'web.memoryInspector.reembed.description' => '对存储在其他 embedder 下的记忆重新计算向量，使它们再次可被搜索到。',
			'web.memoryInspector.reembed.targetEmbedder' => '目标 embedder',
			'web.memoryInspector.reembed.onName' => '在',
			'web.memoryInspector.reembed.totalToReembed' => '待重嵌入总数',
			'web.memoryInspector.reembed.explainer' => '每条记忆的文本会重新发送到当前 embedder；新向量原地替换旧向量。ID、scope、scope_key、metadata 与时间戳保持不变。搜索结果立即生效 — 无需重启。',
			'web.memoryInspector.reembed.reportExamined' => '已检查',
			'web.memoryInspector.reembed.reportReembedded' => '已重嵌入',
			'web.memoryInspector.reembed.reportFailed' => '失败',
			'web.memoryInspector.reembed.reportFrom' => '来源',
			'web.memoryInspector.reembed.errors_one' => ({required Object count}) => '${count} 个错误',
			'web.memoryInspector.reembed.errors_other' => ({required Object count}) => '${count} 个错误',
			'web.memoryInspector.reembed.done' => '完成',
			'web.memoryInspector.reembed.cancel' => '取消',
			'web.memoryInspector.reembed.reembedding' => '重嵌入中…',
			'web.memoryInspector.reembed.reembedTotal' => ({required Object total}) => '重嵌入 ${total} 条',
			'web.notes.title' => '笔记',
			'web.notes.header.outline' => '大纲',
			'web.notes.header.showOutline' => '显示大纲',
			'web.notes.header.hideOutline' => '隐藏大纲',
			'web.notes.header.today' => '今天',
			'web.notes.header.todayTooltip' => '打开或创建今天的日志笔记',
			'web.notes.header.kNew' => '新建',
			'web.notes.left.tree' => '目录树',
			'web.notes.left.tags' => '标签',
			'web.notes.left.filterNotes' => '过滤笔记…',
			'web.notes.left.filterTags' => '过滤标签…',
			'web.notes.left.filteredBy' => '已筛选',
			'web.notes.left.clearTagTooltip' => '清除标签筛选',
			'web.notes.left.expandAll' => '全部展开',
			'web.notes.left.expandAllTooltip' => '展开全部文件夹',
			_ => null,
		} ?? switch (path) {
			'web.notes.left.collapseAll' => '全部收起',
			'web.notes.left.collapseAllTooltip' => '收起全部文件夹',
			'web.notes.left.loading' => '加载中…',
			'web.notes.left.footer' => ({required Object visible, required Object total}) => '${visible} / ${total} 条笔记',
			'web.notes.tags.emptyVault' => 'vault 中暂无标签。',
			'web.notes.tags.noMatches' => ({required Object query}) => '未找到匹配 "${query}"。',
			'web.notes.tree.empty' => 'vault 为空。',
			'web.notes.outline.label' => '大纲',
			'web.notes.outline.empty' => '此笔记没有标题。添加 <1>## 标题</1> 行以填充大纲。',
			'web.notes.newNote.prompt' => '新笔记路径（相对 vault，必须以 .md 结尾）',
			'web.notes.newNote.defaultPath' => ({required Object date}) => 'library/notes-${date}.md',
			'web.notes.newNote.errorMustEndMd' => '路径必须以 .md 结尾',
			'web.notes.newNote.createdToast' => '笔记已创建',
			'web.notes.newNote.createFailedToast' => '创建失败',
			'web.notes.empty.title' => '未选择笔记',
			'web.notes.empty.hint' => '从左侧目录树挑选一条笔记，直接跳到今天的日志，或新建一条。AI 生成的项目文档位于 <1>projects/</1>；个人草稿位于 <3>personal/</3>。',
			'web.notes.empty.today' => '今天的日志笔记',
			'web.notes.empty.kNew' => '新建笔记',
			'web.notes.picker.browseAria' => '浏览文件夹',
			'web.notes.picker.matches_one' => ({required Object count}) => '${count} 个匹配',
			'web.notes.picker.matches_other' => ({required Object count}) => '${count} 个匹配',
			'web.notes.picker.foldersInVault' => ({required Object count}) => 'vault 中 ${count} 个文件夹',
			'web.notes.picker.noMatch' => ({required Object value}) => '未找到匹配的文件夹。直接保存即可使用 <1>${value}</1>（首次写入时懒创建）。',
			'web.notes.vaultSync.title' => 'Vault 同步',
			'web.notes.vaultSync.description' => '把 notes vault 作为 git 仓库进行 commit、pull 与 push。认证使用网关主机的 git 凭据（SSH agent / credential helper）。',
			'web.notes.vaultSync.reading' => '正在读取 vault 状态…',
			'web.notes.vaultSync.init.title' => 'vault 尚未初始化为 git 仓库',
			'web.notes.vaultSync.init.body' => '初始化会在 vault 根目录运行 <1>git init -b main</1> 并加入一份合理的 <3>.gitignore</3>。之后你就可以提交笔记并配置 remote（GitHub / Gitea / GitLab）进行跨机同步。',
			'web.notes.vaultSync.init.button' => '把 vault 初始化为 git 仓库',
			'web.notes.vaultSync.init.initToast' => 'vault 已初始化为 git 仓库',
			'web.notes.vaultSync.init.initFailedToast' => '初始化失败',
			'web.notes.vaultSync.branch.clean' => '干净',
			'web.notes.vaultSync.branch.staged' => ({required Object count}) => '${count} 已暂存',
			'web.notes.vaultSync.branch.modified' => ({required Object count}) => '${count} 已修改',
			'web.notes.vaultSync.branch.untracked' => ({required Object count}) => '${count} 未跟踪',
			'web.notes.vaultSync.action.pull' => '拉取',
			'web.notes.vaultSync.action.push' => '推送',
			'web.notes.vaultSync.action.pullTitleNoRemote' => '请先配置 remote',
			'web.notes.vaultSync.action.pullTitleHasUpstream' => 'git pull --rebase --autostash',
			'web.notes.vaultSync.action.pullTitleNoUpstream' => '拉取 origin 的 HEAD；隐式建立 tracking',
			'web.notes.vaultSync.action.pushTitleNoRemote' => '请先配置 remote',
			'web.notes.vaultSync.action.pushTitleHasUpstream' => 'git push -u origin HEAD',
			'web.notes.vaultSync.action.pushTitleNoUpstream' => '首次推送 — 会将 upstream 设为 origin/HEAD',
			'web.notes.vaultSync.action.noRemote' => '尚未配置 remote — pull/push 已禁用',
			'web.notes.vaultSync.action.noUpstream' => '尚无 upstream tracking — 首次推送会自动建立。',
			'web.notes.vaultSync.action.pulledToast' => '已拉取',
			'web.notes.vaultSync.action.pullFailedToast' => '拉取失败',
			'web.notes.vaultSync.action.pushedToast' => '已推送',
			'web.notes.vaultSync.action.pushFailedToast' => '推送失败',
			'web.notes.vaultSync.commit.title' => '提交',
			'web.notes.vaultSync.commit.placeholder' => ({required Object date}) => 'Notes: ${date}（默认）',
			'web.notes.vaultSync.commit.commitAll' => '提交全部',
			'web.notes.vaultSync.commit.hint' => '暂存所有变更（<1>git add .</1>）然后用此 message 提交。message 为空则使用带时间戳的默认值。',
			'web.notes.vaultSync.commit.committedToast' => ({required Object hash}) => '已提交 ${hash}',
			'web.notes.vaultSync.commit.commitFailedToast' => '提交失败',
			'web.notes.vaultSync.fileList.title' => ({required Object count}) => '工作树 · ${count}',
			'web.notes.vaultSync.fileList.moreSuffix' => ({required Object count}) => '+${count} 更多',
			'web.notes.vaultSync.remote.title' => 'Remote（origin）',
			'web.notes.vaultSync.remote.cancel' => '取消',
			'web.notes.vaultSync.remote.change' => '更换',
			'web.notes.vaultSync.remote.configure' => '配置',
			'web.notes.vaultSync.remote.empty' => '尚未设置 remote。添加一个 HTTPS 或 SSH URL（例如 <1>git@github.com:you/notes.git</1> 或 <3>https://tea.linivek.online/you/notes.git</3>）以启用 push / pull。',
			'web.notes.vaultSync.remote.urlLabel' => 'URL（HTTPS 或 SSH）',
			'web.notes.vaultSync.remote.urlPlaceholder' => 'git@host:owner/notes.git',
			'web.notes.vaultSync.remote.save' => '保存',
			'web.notes.vaultSync.remote.savedToast' => 'Remote 已保存',
			'web.notes.vaultSync.remote.saveFailedToast' => '设置 remote 失败',
			'web.notes.vaultSync.history.title' => '最近提交',
			'web.notes.vaultSync.history.loading' => '加载中…',
			'web.notes.vaultSync.history.empty' => '暂无提交。',
			'web.notes.vaultSync.conflict.kinds.rebase' => 'rebase',
			'web.notes.vaultSync.conflict.kinds.merge' => 'merge',
			'web.notes.vaultSync.conflict.kinds.cherryPick' => 'cherry-pick',
			'web.notes.vaultSync.conflict.kinds.operation' => 'operation',
			'web.notes.vaultSync.conflict.headline' => ({required Object kind}) => 'vault 存在暂停的 ${kind} 且有未解决的冲突',
			'web.notes.vaultSync.conflict.explainer' => ({required Object kind}) => '在 ${kind} 完成之前，pull、push 与 commit 都被阻塞。你可以选择 <1>中止</1>（把工作树恢复到 ${kind} 之前的状态 — 保留本地提交，丢弃远端提交），或 <3>强制重置到 remote</3>（丢弃所有本地提交 + 未提交修改；vault 变成 origin 的精确镜像）。',
			'web.notes.vaultSync.conflict.conflictedHeader' => ({required Object count}) => '冲突文件 · ${count}',
			'web.notes.vaultSync.conflict.abort' => ({required Object kind}) => '中止 ${kind}',
			'web.notes.vaultSync.conflict.abortTitle' => ({required Object kind}) => 'git ${kind} --abort',
			'web.notes.vaultSync.conflict.forceReset' => '强制重置到 remote',
			'web.notes.vaultSync.conflict.forceResetTitle' => 'git fetch && git reset --hard origin/<branch> && git clean -fd',
			'web.notes.vaultSync.conflict.forceResetConfirm' => ({required Object kind}) => '破坏性操作：将\n  • 中止进行中的 ${kind}\n  • 运行 git fetch origin\n  • reset --hard 到 origin/<branch>\n  • clean -fd（删除未跟踪文件）\n\n任何尚未推送到 origin 的本地提交以及任何未提交修改都将永久丢失。\n\n继续？',
			'web.notes.vaultSync.conflict.abortedToast' => ({required Object kind}) => '已中止 ${kind}',
			'web.notes.vaultSync.conflict.abortedDescription' => '工作树已恢复到操作前状态。',
			'web.notes.vaultSync.conflict.abortFailedToast' => '中止失败',
			'web.notes.vaultSync.conflict.resetToast' => ({required Object branch}) => '已重置到 ${branch}',
			'web.notes.vaultSync.conflict.resetDescription' => '本地更改已丢弃；vault 与 remote 一致。',
			'web.notes.vaultSync.conflict.resetFailedToast' => '重置失败',
			'web.notes.vaultSync.auth.title' => '认证',
			'web.notes.vaultSync.auth.httpsTokenOk' => ({required Object host}) => '将使用 Plugins → Git hosts 中为 <1>${host}</1> 存的 token。✓',
			'web.notes.vaultSync.auth.httpsTokenMissing' => ({required Object host}) => '<1>${host}</1> 上的 HTTPS remote，opendray 中没有配置 token。在你为其添加 token 之前，私有仓库的 push / pull 很可能失败。',
			'web.notes.vaultSync.auth.ssh' => ({required Object host}) => '<1>${host}</1> 上的 SSH remote。认证使用网关主机的 <3>~/.ssh/</3>（ssh-agent、identity 文件、host config）。可在主机 shell 用 <5>ssh -T git@${host}</5> 验证。',
			'web.notes.vaultSync.auth.configureTokenLink' => '→ 配置 git host token',
			'web.notes.vaultSync.autoSync.loading' => '加载自动同步设置…',
			'web.notes.vaultSync.autoSync.title' => '自动同步',
			'web.notes.vaultSync.autoSync.on' => '开',
			'web.notes.vaultSync.autoSync.runNow' => '立即运行',
			'web.notes.vaultSync.autoSync.runNowTooltip' => '立即唤醒同步循环（跳过等待，然后运行所有到期的步骤）',
			'web.notes.vaultSync.autoSync.configure' => '配置',
			'web.notes.vaultSync.autoSync.hide' => '隐藏',
			'web.notes.vaultSync.autoSync.enabled' => '启用',
			'web.notes.vaultSync.autoSync.enabledTooltipNoRemote' => '请先配置 remote 才能启用自动同步',
			'web.notes.vaultSync.autoSync.noRemoteHint' => '尚无 remote — push/pull 将被跳过。',
			'web.notes.vaultSync.autoSync.commitEvery' => '提交间隔',
			'web.notes.vaultSync.autoSync.commitEveryExamples' => '示例：<1>30s</1>、<3>10m</3>、<5>2h</5>。最小 30s。',
			'web.notes.vaultSync.autoSync.pullEvery' => '拉取间隔',
			'web.notes.vaultSync.autoSync.pullEveryHint' => '仅在启用 Pull 时使用。',
			'web.notes.vaultSync.autoSync.pushAfterCommit' => '提交后 push',
			'web.notes.vaultSync.autoSync.pullPeriodically' => '周期性 pull',
			'web.notes.vaultSync.autoSync.commitTemplateLabel' => '提交 message 模板',
			'web.notes.vaultSync.autoSync.commitTemplatePlaceholder' => ({required Object date}) => 'Auto-sync: ${date}（留空则使用默认）',
			'web.notes.vaultSync.autoSync.saveSettings' => '保存设置',
			'web.notes.vaultSync.autoSync.discard' => '丢弃',
			'web.notes.vaultSync.autoSync.lastCommit' => '最近 commit',
			'web.notes.vaultSync.autoSync.lastPush' => '最近 push',
			'web.notes.vaultSync.autoSync.lastPull' => '最近 pull',
			'web.notes.vaultSync.autoSync.never' => '从未',
			'web.notes.vaultSync.autoSync.savedToast' => '自动同步设置已保存',
			'web.notes.vaultSync.autoSync.saveFailedToast' => '保存失败',
			'web.notes.vaultSync.autoSync.triggeredToast' => '已触发自动同步',
			'web.notes.vaultSync.autoSync.runFailedToast' => '运行失败',
			'web.notes.syncBadge.loading' => '加载中…',
			'web.notes.syncBadge.syncLabel' => '同步',
			'web.notes.syncBadge.initLabel' => '初始化',
			'web.notes.syncBadge.initTooltip' => 'vault 尚未初始化为 git 仓库',
			'web.notes.syncBadge.conflictLabel' => '冲突',
			'web.notes.syncBadge.conflictTooltip' => 'vault 存在未解决的冲突 — 点击进入恢复',
			'web.notes.syncBadge.syncFallback' => 'sync',
			'web.notes.syncBadge.tooltip' => ({required Object branch, required Object files, required Object ahead, required Object behind}) => '分支 ${branch} · ${files} 处改动 · 领先 ${ahead} · 落后 ${behind}',
			'web.notes.syncBadge.tooltipAutoOn' => ' · 自动同步已开启',
			'web.notes.syncBadge.tooltipLastError' => ({required Object error}) => ' · 上次错误：${error}',
			'web.notes.syncBadge.branchPlaceholder' => '—',
			'web.activity.title' => '活动',
			'web.activity.subtitle' => '按调用维度审计每个由注册集成发起的 API 请求。包括入站调用（第三方应用以集成 API key 调用 opendray）和出站代理调用（admin → opendray 代理 → 集成）。本管理端 UI 直接发起的调用不会被记录。',
			'web.activity.refresh' => '刷新',
			'web.activity.refreshTooltip' => '刷新',
			'web.activity.filters.integration' => '集成',
			'web.activity.filters.direction' => '方向',
			'web.activity.filters.status' => '状态',
			'web.activity.filters.allIntegrations' => '所有集成',
			'web.activity.filters.all' => '全部',
			'web.activity.filters.inbound' => '入站',
			'web.activity.filters.outbound' => '出站',
			'web.activity.filters.allStatuses' => '所有状态',
			'web.activity.filters.status2' => '2xx 成功',
			'web.activity.filters.status3' => '3xx 重定向',
			'web.activity.filters.status4' => '4xx 客户端错误',
			'web.activity.filters.status5' => '5xx 服务端错误',
			'web.activity.callsCount_one' => ({required Object count}) => '${count} 次调用',
			'web.activity.callsCount_other' => ({required Object count}) => '${count} 次调用',
			'web.activity.loading' => '加载中…',
			'web.activity.table.time' => '时间',
			'web.activity.table.integration' => '集成',
			'web.activity.table.directionTitle' => '方向',
			'web.activity.table.method' => '方法',
			'web.activity.table.path' => '路径',
			'web.activity.table.status' => '状态',
			'web.activity.table.duration' => '耗时',
			'web.activity.table.inboundAria' => '入站',
			'web.activity.table.outboundAria' => '出站',
			'web.activity.empty.filtered' => '没有调用符合当前筛选条件。',
			'web.activity.empty.title' => '尚未记录任何 API 调用',
			'web.activity.empty.description' => '当第三方应用以其集成 API key 调用 opendray 时，每次请求都会被记录在这里。',
			'web.activity.empty.stepWithIntegrations' => '在你的第三方应用中使用已有集成的 API key',
			'web.activity.empty.stepRegister' => '在 集成 → 新建 中注册一个集成',
			'web.activity.empty.stepCallEndpoint' => '调用任意接口，例如 <1>POST /api/v1/sessions</1>',
			'web.activity.empty.stepAppears' => '调用会在几秒内出现在这里',
			'web.activity.empty.footnote' => '你在本管理端 UI 中发起的调用不会被记录 — 仅集成归属的流量会被记录。',
			'web.activity.events.loading' => '正在加载事件…',
			'web.activity.events.empty' => '尚无事件。',
			'web.activity.events.emptyFiltered' => '没有匹配的事件。',
			'web.activity.events.loadOlder' => '加载更早的事件',
			'web.activity.events.today' => '今天',
			'web.activity.events.yesterday' => '昨天',
			'web.providers.list.title' => 'Providers',
			'web.providers.list.loading' => '加载中…',
			'web.providers.list.disabledBadge' => '已禁用',
			'web.providers.list.noneSelected' => '未选择任何 Provider。',
			'web.providers.detail.enabled' => '已启用',
			'web.providers.detail.disabled' => '已禁用',
			'web.providers.detail.toggleAria' => ({required Object name}) => '切换 ${name}',
			'web.providers.detail.configuration' => '配置',
			'web.providers.detail.noConfig' => '此 Provider 没有可配置字段。',
			'web.providers.detail.executable' => 'executable:',
			'web.providers.detail.manifestHash' => 'manifest_hash:',
			'web.providers.detail.reset' => '重置',
			'web.providers.detail.save' => '保存更改',
			'web.providers.detail.saving' => '保存中…',
			'web.providers.detail.savedToast' => 'Provider 配置已保存',
			'web.providers.detail.saveFailedToast' => '保存失败',
			'web.providers.detail.toggleFailedToast' => '切换失败',
			'web.providers.detail.caps.resume' => 'resume',
			'web.providers.detail.caps.stream' => 'stream',
			'web.providers.detail.caps.images' => 'images',
			'web.providers.detail.caps.mcp' => 'mcp',
			'web.providers.configForm.selectPlaceholder' => '选择…',
			'web.providers.configForm.defaultOption' => '(默认)',
			'web.providers.configForm.switchOn' => '开',
			'web.providers.configForm.switchOff' => '关',
			'web.providers.configForm.showSecret' => '显示密钥',
			'web.providers.configForm.hideSecret' => '隐藏密钥',
			'web.providers.claudeAccounts.title' => 'Claude 账号',
			'web.providers.claudeAccounts.tutorialTooltip' => '打开多账号教程章节',
			'web.providers.claudeAccounts.importLocal' => '导入本地',
			'web.providers.claudeAccounts.importLocalTooltip' => '扫描网关主机上的 ~/.claude-accounts/ 目录并注册新的目录。该按钮仅在网关主机环境下工作 — 详见教程。',
			'web.providers.claudeAccounts.importedNothingToast' => '无需导入 — 账号已同步。',
			'web.providers.claudeAccounts.importedToast_one' => ({required Object count}) => '已从 ~/.claude-accounts 导入 ${count} 个账号',
			'web.providers.claudeAccounts.importedToast_other' => ({required Object count}) => '已从 ~/.claude-accounts 导入 ${count} 个账号',
			'web.providers.claudeAccounts.importFailedToast' => '导入失败',
			'web.providers.claudeAccounts.addingTitle' => '添加新账号。',
			'web.providers.claudeAccounts.addingBodyPrefix' => '在网关主机执行：',
			'web.providers.claudeAccounts.addingBodySuffix' => 'opendray 的文件系统监听会自动注册新目录，或点击 <1>导入本地</1> 立即扫描。',
			'web.providers.claudeAccounts.architectureLink' => '架构与完整指南 →',
			'web.providers.claudeAccounts.loading' => '加载中…',
			'web.providers.claudeAccounts.empty' => '尚无 Claude 账号。在网关主机执行上面的 shell 命令，然后点击 <1>导入本地</1> 扫描。',
			'web.providers.claudeAccounts.noTokenYet' => '尚无 token',
			'web.providers.claudeAccounts.configDir' => 'config_dir:',
			'web.providers.claudeAccounts.tokenPath' => 'token_path:',
			'web.providers.claudeAccounts.toggleFailedToast' => '切换失败',
			'web.providers.claudeAccounts.removeConfirm' => ({required Object name}) => '移除账号 "${name}"?',
			'web.providers.claudeAccounts.removedToast' => '账号已移除',
			'web.providers.claudeAccounts.removeFailedToast' => '移除失败',
			'web.providers.claudeAccounts.toggleAria' => ({required Object name}) => '切换 ${name}',
			'web.providers.claudeAccounts.removeAria' => ({required Object name}) => '移除 ${name}',
			'web.channels.title' => '频道',
			'web.channels.subtitle' => '双向消息集成。每个频道的出站通知按其 <1>notify_on</1> 过滤。',
			'web.channels.newButton' => '新建频道',
			'web.channels.loading' => '加载中…',
			'web.channels.empty.title' => '暂无频道',
			'web.channels.empty.description' => '内置类型：Telegram · Slack · Discord · 飞书 · 钉钉 · 企业微信。挑选一个并填入凭证，或使用 <1>bridge</1> 通过 WebSocket 接入自定义平台。',
			'web.channels.card.running' => '运行中',
			'web.channels.card.starting' => '启动中…',
			'web.channels.card.disabled' => '已禁用',
			'web.channels.card.muted' => '已静音',
			'web.channels.card.tokenLabel' => 'token:',
			'web.channels.card.chatIdLabel' => 'chat_id:',
			'web.channels.card.channelIdLabel' => 'channel_id:',
			'web.channels.card.notifyOnLabel' => 'notify_on:',
			'web.channels.card.webhookLabel' => 'webhook:',
			'web.channels.card.copyWebhookTooltip' => '复制 webhook URL',
			'web.channels.card.webhookCopiedToast' => '已复制 webhook URL',
			'web.channels.card.setup' => '配置',
			'web.channels.card.setupTooltip' => '查看适配器连接信息和示例代码',
			'web.channels.card.test' => '测试',
			'web.channels.card.testNotRunningTooltip' => '频道必须处于运行状态',
			'web.channels.card.testBridgeTooltip' => 'Bridge 频道无法从管理端测试 — 请先连接一个适配器',
			'web.channels.card.editAria' => '编辑频道',
			'web.channels.card.editTooltip' => '编辑频道配置',
			'web.channels.card.deleteAria' => '删除频道',
			'web.channels.card.bridgeSuffix' => '(bridge)',
			'web.channels.toasts.testSent' => '测试消息已发送',
			'web.channels.toasts.testFailed' => '测试失败',
			'web.channels.toasts.deleteConfirm' => ({required Object id}) => '删除频道 ${id}?',
			'web.channels.toasts.deleted' => '频道已删除',
			'web.channels.toasts.created' => '频道已创建',
			'web.channels.toasts.updated' => '频道已更新',
			'web.channels.dialog.editTitle' => '编辑频道',
			'web.channels.dialog.createTitle' => '注册频道',
			'web.channels.dialog.descriptionBridge' => '外部适配器（Python/Node/...）通过 WebSocket 连接并出示此 token。',
			'web.channels.dialog.descriptionDefault' => '配置消息集成。',
			'web.channels.dialog.kindLabel' => '类型',
			'web.channels.dialog.kindImmutable' => '（不可更改 — 如需更换类型请删除后重建）',
			'web.channels.dialog.enabledLabel' => '启用',
			'web.channels.dialog.enabledBridgeHint' => '（立即开始接受适配器连接）',
			'web.channels.dialog.enabledWebhookHint' => '（立即开始接收 webhook）',
			'web.channels.dialog.enabledDefaultHint' => '（立即启动）',
			'web.channels.dialog.cancel' => '取消',
			'web.channels.dialog.save' => '保存',
			'web.channels.dialog.saving' => '保存中…',
			'web.channels.dialog.create' => '创建',
			'web.channels.dialog.creating' => '创建中…',
			'web.channels.dialog.unknownKind' => ({required Object kind}) => '未知类型：${kind}',
			'web.channels.dialog.nameRequired' => 'name 不能为空',
			'web.channels.dialog.tokenRequired' => 'token 不能为空',
			'web.channels.dialog.topicIdsNumeric' => ({required Object value}) => 'Topic ID 必须是数字（收到 ${value}）',
			'web.channels.dialog.fieldRequired' => ({required Object label}) => '${label} 不能为空',
			'web.channels.dialog.cooldownInvalid' => 'Cooldown 必须是非负整数秒',
			'web.channels.dialog.snippetCapInvalid' => 'Snippet cap 必须是非负数字',
			'web.channels.notifications.sectionTitle' => '会话通知',
			'web.channels.notifications.notifyOnLabel' => '通知触发条件',
			'web.channels.notifications.hintAll' => '接收全部会话事件。点击标签可退订。',
			'web.channels.notifications.hintNone' => '未选择任何事件 — 出站通知已静音。',
			'web.channels.notifications.hintSome' => ({required Object total, required Object selected}) => '只选择了 ${total} 中的 ${selected} 个 topic。',
			'web.channels.notifications.repeatPolicyLabel' => '重复策略',
			'web.channels.notifications.cooldownLabel' => '冷却时长',
			'web.channels.notifications.onceReplyHint' => '在该聊天中以非命令文本回复会重置抑制 — opendray 会把你的回复转发到会话的 stdin 并重新启用通知。',
			'web.channels.notifications.terminalSnippetLabel' => '终端片段',
			'web.channels.notifications.embedSnippetLabel' => '在 idle 通知中嵌入最近的终端画面',
			'web.channels.notifications.snippetExplainer' => '开启后，idle 卡片会包含一段代码块片段，呈现用户在网页终端中会看到的内容 — Claude TUI 自身的装饰（状态 spinner、"bypass permissions" 提示、分隔线）会被自动过滤。',
			'web.channels.notifications.modes.onceLabel' => '每个会话仅一次（推荐）',
			'web.channels.notifications.modes.onceHint' => '当会话变为 idle 时通知一次，然后保持静默，直到会话结束或你通过该频道回复。',
			'web.channels.notifications.modes.cooldownLabel' => '时间窗口冷却',
			'web.channels.notifications.modes.cooldownHint' => '对同一 (会话, 事件) 在所选时间窗口内抑制重复通知。',
			'web.channels.notifications.modes.everyLabel' => '每次事件都通知（嘈杂）',
			'web.channels.notifications.modes.everyHint' => '不做抑制。仅用于低频频道。',
			'web.channels.notifications.cooldowns.k60' => '1 分钟',
			'web.channels.notifications.cooldowns.k300' => '5 分钟',
			'web.channels.notifications.cooldowns.k900' => '15 分钟',
			'web.channels.notifications.cooldowns.k1800' => '30 分钟',
			'web.channels.notifications.cooldowns.k3600' => '1 小时',
			'web.channels.notifications.snippetCaps.k0' => '不限制 — 拆分到多条消息（默认）',
			'web.channels.notifications.snippetCaps.k1000' => '1000 字符（精简）',
			'web.channels.notifications.snippetCaps.k3000' => '3000 字符',
			'web.channels.notifications.snippetCaps.k6000' => '6000 字符',
			'web.channels.notifications.snippetCaps.k12000' => '12000 字符',
			'web.channels.bridge.nameLabel' => 'Bridge 名称',
			'web.channels.bridge.namePlaceholder' => 'wechat / discord-custom / whatsapp...',
			'web.channels.bridge.nameHint' => '适配器的人类可读标签。会显示在频道列表中。',
			'web.channels.bridge.tokenLabel' => '适配器 token',
			'web.channels.bridge.regenerateTooltip' => '重新生成',
			'web.channels.bridge.copyTooltip' => '复制',
			'web.channels.bridge.tokenCopiedToast' => '已复制 token',
			'web.channels.bridge.tokenHint' => '适配器通过 WS register 帧发送此 token（也可作为 <1>X-Bridge-Token</1> header）。',
			'web.channels.bridge.capsLabel' => '接受的能力（可选白名单）',
			'web.channels.bridge.capsHint' => '为空 = 接受适配器声明的任意能力。已选 = 即使适配器提供更多能力，也只允许这些。',
			'web.channels.bridge.afterCreate' => '点击 <1>创建</1> 后，适配器设置对话框会自动打开，里面包含 WebSocket URL 和可直接复制的 Python / Node / wscat 起步代码。',
			'web.channels.setup.title' => ({required Object name}) => '适配器设置 — ${name}',
			'web.channels.setup.description' => '运行一个任意语言的适配器，使用这些凭证通过 WebSocket 连接到 opendray。opendray 会通过它路由会话通知和 slash 命令。',
			'web.channels.setup.wsUrlLabel' => 'WebSocket URL',
			'web.channels.setup.tokenLabel' => '适配器 token',
			'web.channels.setup.authInfo' => ({required Object frame}) => '<1>认证：</1>通过 <3>X-Bridge-Token</3> header、<5>?token=</5> query 参数或 <7>Authorization: Bearer …</7> 之一发送 token。首个 WS 帧必须是 <9>${frame}</9>。完整协议见仓库中的 <11>docs/bridge-protocol.md</11>。',
			'web.channels.setup.pythonInstall' => '安装：<1>pip install websockets</1>。运行：<3>python adapter.py</3>。',
			'web.channels.setup.nodeInstall' => '安装：<1>npm i ws</1>。运行：<3>node adapter.mjs</3>。',
			'web.channels.setup.wscatInstall' => '安装：<1>npm i -g wscat</1>。连接后，粘贴上方的 JSON 注册帧，然后手动发送后续帧。',
			'web.channels.setup.close' => '关闭',
			'web.channels.setup.copyHide' => '隐藏',
			'web.channels.setup.copyShow' => '显示',
			'web.channels.setup.copyLabelToast' => ({required Object label}) => '已复制 ${label}',
			'web.channels.setup.copyCode' => '复制',
			'web.channels.setup.copied' => '已复制',
			'web.channels.setup.codeCopiedToast' => '已复制代码',
			'web.integrations.title' => '集成',
			'web.integrations.subtitle' => '调用 opendray 的外部应用。通过 <1>/api/v1/proxy/&lt;prefix&gt;/…</1> 反向代理，并通过 WS 端点订阅事件。',
			'web.integrations.register' => '注册',
			'web.integrations.loading' => '加载中…',
			'web.integrations.tabs.registered' => '已注册',
			'web.integrations.tabs.console' => '反向代理',
			'web.integrations.empty.title' => '暂无集成',
			'web.integrations.empty.description' => '注册一个外部应用，给它一个受限的 API key。它的代码不需要进入本仓库。',
			'web.integrations.empty.register' => '注册集成',
			'web.integrations.groupSystem' => '系统（由 opendray 管理）',
			'web.integrations.groupOperator' => '用户注册',
			'web.integrations.card.managedBadge' => '受管',
			'web.integrations.card.managedTooltip' => 'opendray 管理这个集成。编辑或轮换它的 key 会导致正在运行的会话（mcp.json 中持有旧 bearer）失效。',
			'web.integrations.card.consumerBadge' => 'consumer',
			'web.integrations.card.consumerTooltip' => '仅消费型集成 — 没有可供探测的 HTTP 服务',
			'web.integrations.card.disabledBadge' => '已禁用',
			'web.integrations.card.consumerOnlyHint' => '消费 opendray 的 API。未挂载反向代理。',
			'web.integrations.card.lastProbed' => ({required Object relative}) => '${relative} 探测过',
			'web.integrations.card.rotated' => ({required Object relative}) => '${relative} 轮换过',
			'web.integrations.card.managedReadOnly' => '只读 — opendray 把它的 key 烤进每次 spawn 的 mcp.json',
			'web.integrations.card.managedReadOnlyTooltip' => 'opendray 管理此行。如需重置：删除 ~/.opendray/memory.key 后重启，或直接通过 SQL 删除此行 — 下次启动时会重新引导。',
			'web.integrations.card.editAria' => '编辑集成',
			'web.integrations.card.editTooltip' => '编辑 scopes / base URL / version',
			'web.integrations.card.rotateKey' => '轮换 key',
			'web.integrations.card.deleteAria' => '删除集成',
			'web.integrations.card.rotateConfirm' => ({required Object name}) => '轮换 "${name}" 的 API key? 当前 key 将立即失效。',
			'web.integrations.card.deleteConfirm' => ({required Object name}) => '删除集成 ${name}?',
			'web.integrations.card.removedToast' => '集成已移除',
			'web.integrations.register_dialog.title' => '注册集成',
			'web.integrations.register_dialog.description' => '签发一次性 API key。关闭前请复制它 — opendray 不会再显示明文。',
			'web.integrations.register_dialog.nameLabel' => '名称',
			'web.integrations.register_dialog.namePlaceholder' => 'PetTracker',
			'web.integrations.register_dialog.modeHint' => '如果是 <1>仅消费型</1> 集成（第三方应用调用 opendray API 但不暴露自身服务），保留下面两个字段为空。两个都填则是 <3>反向代理型</3> 集成。',
			'web.integrations.register_dialog.baseUrlLabel' => 'Base URL',
			'web.integrations.register_dialog.optionalSuffix' => '(可选)',
			'web.integrations.register_dialog.baseUrlPlaceholder' => 'http://192.168.3.42:8080',
			'web.integrations.register_dialog.routePrefixLabel' => 'Route prefix',
			'web.integrations.register_dialog.routePrefixPlaceholder' => 'pet-tracker',
			'web.integrations.register_dialog.routePrefixHint' => ({required Object prefix}) => '可通过 <1>/api/v1/proxy/${prefix}/*</1> 访问。',
			'web.integrations.register_dialog.routePrefixPlaceholderToken' => '<prefix>',
			'web.integrations.register_dialog.versionLabel' => 'Version（可选）',
			'web.integrations.register_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.register_dialog.scopesLabel' => 'Scopes',
			'web.integrations.register_dialog.scopesIntro' => '选择该集成允许调用的 API 范围。每个开关映射到一个 Bearer token 声明 — opendray 会拒绝任何越权请求。',
			'web.integrations.register_dialog.errorNameRequired' => 'Name 不能为空。',
			'web.integrations.register_dialog.errorBothOrNeither' => 'base_url 和 route_prefix 必须成对设置。同时填入 = 反向代理集成；同时留空 = 仅消费型集成。',
			'web.integrations.register_dialog.cancel' => '取消',
			'web.integrations.register_dialog.submit' => '注册',
			'web.integrations.register_dialog.submitting' => '注册中…',
			'web.integrations.reveal.titleIssued' => 'API key 已签发',
			'web.integrations.reveal.titleRotated' => 'API key 已轮换',
			'web.integrations.reveal.description' => '这是明文 key 唯一一次显示的机会。立即复制并更新所有消费端 — 旧 key（如有）将不再有效。',
			'web.integrations.reveal.discardAria' => '丢弃新 key',
			'web.integrations.reveal.discardTooltip' => '丢弃新 key（轮换已经发生 — 旧 key 同样失效）',
			'web.integrations.reveal.discardConfirm' => '确定丢弃新 key? 轮换已经使旧 key 失效 — 丢弃意味着此集成将没有任何可用 key，直到再次轮换。',
			'web.integrations.reveal.copy' => '复制',
			'web.integrations.reveal.copied' => '已复制',
			'web.integrations.reveal.updateHint' => '<1>请用此新 key 更新每个消费端应用。</1> 旧 key 已在服务端失效，下次请求会返回 <3>401 unauthorized</3>。',
			'web.integrations.reveal.acknowledge' => '我已复制 key 并会更新消费端应用。我了解 opendray 不会再显示它。',
			'web.integrations.reveal.discard' => '丢弃',
			'web.integrations.reveal.done' => '完成',
			'web.integrations.edit_dialog.title' => ({required Object name}) => '编辑集成 · ${name}',
			'web.integrations.edit_dialog.description' => '修改 scopes、version 或 base URL。Name 和 route prefix 不可更改 — 如需修改请删除并重新注册。',
			'web.integrations.edit_dialog.nameLabel' => 'Name',
			'web.integrations.edit_dialog.routePrefixLabel' => 'Route prefix',
			'web.integrations.edit_dialog.consumerOnlyLabel' => '(仅消费型)',
			'web.integrations.edit_dialog.baseUrlLabel' => 'Base URL',
			'web.integrations.edit_dialog.baseUrlConsumerSuffix' => '(仅消费型 — 留空)',
			'web.integrations.edit_dialog.baseUrlProxySuffix' => '(反向代理目标)',
			'web.integrations.edit_dialog.baseUrlConsumerPlaceholder' => '(留空 — 此集成消费 opendray API)',
			'web.integrations.edit_dialog.baseUrlProxyPlaceholder' => 'http://127.0.0.1:8080',
			'web.integrations.edit_dialog.consumerHint' => '这是一个仅消费型集成。在此修改 base URL 还需要 route prefix；请删除后重新注册。',
			'web.integrations.edit_dialog.versionLabel' => 'Version',
			'web.integrations.edit_dialog.versionPlaceholder' => '0.1.0',
			'web.integrations.edit_dialog.scopesLabel' => 'Scopes',
			'web.integrations.edit_dialog.scopesIntro' => '收窄或放宽此集成 API key 授权的 API 范围。已颁发的 token 不受影响 — 新的 scope 集在下次请求时生效。',
			'web.integrations.edit_dialog.errorModeSwitch' => '在仅消费型与反向代理型之间切换需要删除并重新注册 — name 和 route_prefix 无法原地修改。',
			'web.integrations.edit_dialog.updatedToast' => '集成已更新',
			'web.integrations.edit_dialog.cancel' => '取消',
			'web.integrations.edit_dialog.save' => '保存更改',
			'web.integrations.proxy.emptyTitle' => '尚未注册集成',
			'web.integrations.proxy.emptyDescription' => ({required Object prefix}) => '请先注册一个集成；控制台通过 /api/v1/proxy/${prefix}/* 以 admin token 代理请求。',
			'web.integrations.proxy.targetLabel' => '目标',
			'web.integrations.proxy.selectPlaceholder' => '选择集成…',
			'web.integrations.proxy.baseLabel' => 'base:',
			'web.integrations.proxy.history' => '历史',
			'web.integrations.proxy.historyEmpty' => '此集成尚无历史请求',
			'web.integrations.proxy.send' => '发送',
			'web.integrations.proxy.sending' => '发送中…',
			'web.integrations.proxy.extraHeadersLabel' => '额外 header（每行一条，Name: Value）',
			'web.integrations.proxy.bodyLabel' => 'Body',
			'web.integrations.proxy.headers' => 'Headers',
			'web.integrations.proxy.body' => 'Body',
			'web.integrations.proxy.emptyBody' => '(空)',
			'web.integrations.proxy.requestFailed' => '请求失败',
			'web.integrations.proxy.stubText' => '发送一个请求即可查看上游响应。',
			'web.integrations.proxy.stubInjects' => 'opendray 会注入 <1>X-Integration-ID</1>，并剥离你的 <3>Authorization</3> header。',
			'web.integrations.proxy.prefixPlaceholder' => '<prefix>',
			'web.plugins.title' => '检查器插件',
			'web.plugins.subtitle' => '配置在会话打开时右侧检查器面板呈现的数据源。每个插件都是管理员级别且在所有会话间共享。点击章节标题可折叠。',
			'web.plugins.common.loading' => '加载中…',
			'web.plugins.common.cancel' => '取消',
			'web.plugins.common.edit' => '编辑',
			'web.plugins.common.add' => '添加',
			'web.plugins.common.save' => '保存',
			'web.plugins.common.create' => '创建',
			'web.plugins.mcp.title' => 'MCP 服务器',
			'web.plugins.mcp.description' => ({required Object KEY}) => '注入到每次 spawn（claude / codex）的 Model Context Protocol 服务器。Vault 条目位于 <1>~/.opendray/vault/mcp/&lt;id&gt;/mcp.json</1>；env / headers 中以 <3>\$${KEY}</3> 引用的密钥来自下方 <5>MCP secrets</5>。',
			'web.plugins.mcp.newServer' => '新建服务器',
			'web.plugins.mcp.empty' => '尚无 MCP 服务器。添加一个以为 agent 会话暴露额外工具。',
			'web.plugins.mcp.columns.name' => '名称',
			'web.plugins.mcp.columns.transport' => 'Transport',
			'web.plugins.mcp.columns.spec' => '规范',
			'web.plugins.mcp.columns.enabled' => '启用',
			'web.plugins.mcp.noUrl' => '无 URL',
			'web.plugins.mcp.noCommand' => '无 command',
			'web.plugins.mcp.deleteConfirm' => ({required Object id}) => '删除 MCP 服务器 "${id}"?',
			'web.plugins.mcp.removedToast' => 'MCP 服务器已移除',
			'web.plugins.mcp.deleteFailedToast' => '删除失败',
			'web.plugins.mcp.toggleFailedToast' => '切换失败',
			'web.plugins.mcp.editor.createTitle' => '新建 MCP 服务器',
			'web.plugins.mcp.editor.editTitle' => ({required Object id}) => '编辑 MCP: ${id}',
			'web.plugins.mcp.editor.description' => ({required Object API_KEY}) => 'JSON 结构：stdio（默认）使用 <1>command</1>+<3>args</3>+<5>env</5>；sse / http 使用 <7>transport</7> +<9> url</9>+<11>headers</11>。以 <13>\$${API_KEY}</13> 引用密钥 — spawn 时会从密钥文件替换。',
			'web.plugins.mcp.editor.idLabel' => 'ID',
			'web.plugins.mcp.editor.idPlaceholder' => 'filesystem',
			'web.plugins.mcp.editor.idHint' => '小写字母 / 数字 / 短横 / 下划线。同时作为目录名与默认 <1>name</1>。',
			'web.plugins.mcp.editor.bodyLabel' => 'mcp.json',
			'web.plugins.mcp.editor.invalidJson' => ({required Object error}) => '无效的 JSON：${error}',
			'web.plugins.mcp.editor.createdToast' => 'MCP 服务器已创建',
			'web.plugins.mcp.editor.savedToast' => 'MCP 服务器已保存',
			'web.plugins.mcp.editor.createFailedToast' => '创建失败',
			'web.plugins.mcp.editor.saveFailedToast' => '保存失败',
			'web.plugins.mcpSecrets.title' => 'MCP 密钥',
			'web.plugins.mcpSecrets.encryptedBadge' => '已加密',
			'web.plugins.mcpSecrets.plaintextBadge' => '明文',
			'web.plugins.mcpSecrets.encryptedTooltip' => 'AES-GCM 在磁盘上加密；密钥位于操作系统 keychain',
			'web.plugins.mcpSecrets.plaintextTooltip' => '操作系统 keychain 不可用 — 文件以明文存储。请检查网关日志。',
			'web.plugins.mcpSecrets.description' => ({required Object KEY}) => '在任意 <3>mcp.json</3> 中以 <1>\$${KEY}</1> 占位符引用的值在 spawn 时会被替换。<5>已保存的值不会通过 API 返回</5> — 可以覆盖或删除，但无法读回。',
			'web.plugins.mcpSecrets.descriptionStored' => ({required Object path}) => ' 存储于 <1>${path}</1>。',
			'web.plugins.mcpSecrets.addSecret' => '添加密钥',
			'web.plugins.mcpSecrets.empty' => ({required Object KEY}) => '暂无已存密钥。添加后即可在 MCP 服务器配置中以 <1>\$${KEY}</1> 引用。',
			'web.plugins.mcpSecrets.columns.key' => 'Key',
			'web.plugins.mcpSecrets.columns.value' => 'Value',
			'web.plugins.mcpSecrets.editTooltip' => '覆盖已存的值',
			'web.plugins.mcpSecrets.deleteConfirm' => ({required Object key}) => '删除密钥 "${key}"? 任何引用 \$${key} 的 mcp.json 在你重新设置之前都会回退到字面占位符。',
			'web.plugins.mcpSecrets.removedToast' => '密钥已移除',
			'web.plugins.mcpSecrets.deleteFailedToast' => '删除失败',
			'web.plugins.mcpSecrets.editor.addTitle' => '添加密钥',
			'web.plugins.mcpSecrets.editor.updateTitle' => ({required Object key}) => '更新 ${key}',
			'web.plugins.mcpSecrets.editor.addDescription' => ({required Object KEY}) => '若操作系统 keychain 可用，则在磁盘上加密存储。可在任意 mcp.json 的 env / headers / args / url 中以 \$${KEY} 引用。',
			'web.plugins.mcpSecrets.editor.editDescription' => '输入新值以覆盖。旧值无法恢复。',
			'web.plugins.mcpSecrets.editor.keyLabel' => 'Key',
			'web.plugins.mcpSecrets.editor.keyPlaceholder' => 'BRAVE_API_KEY',
			'web.plugins.mcpSecrets.editor.keyPattern' => '必须匹配 <1>[A-Za-z_][A-Za-z0-9_]*</1>',
			'web.plugins.mcpSecrets.editor.keyCollision' => '已存在 — 请使用编辑，或选择另一个名称。',
			'web.plugins.mcpSecrets.editor.valueLabel' => 'Value',
			'web.plugins.mcpSecrets.editor.valueHint' => '输入时隐藏。已保存的值不会通过 API 返回。',
			'web.plugins.mcpSecrets.editor.addedToast' => '密钥已添加',
			'web.plugins.mcpSecrets.editor.updatedToast' => '密钥已更新',
			'web.plugins.mcpSecrets.editor.saveFailedToast' => '保存失败',
			'web.plugins.skills.title' => 'Agent skills',
			'web.plugins.skills.description' => '作为 Tier 1 索引注入到 Claude 会话的可复用能力 — agent 通过 <1>opendray skill describe &lt;id&gt;</1> 按需加载完整 SKILL.md。内置 skill 随二进制发布但可被 <3>自定义</3> — 你的修改保存到 <5>~/.opendray/vault/skills/&lt;id&gt;/SKILL.md</5> 并覆盖内置版本。点击重置可还原。',
			'web.plugins.skills.newSkill' => '新建 skill',
			'web.plugins.skills.empty' => '未找到任何 skill。',
			'web.plugins.skills.columns.id' => 'ID',
			'web.plugins.skills.columns.description' => '描述',
			'web.plugins.skills.columns.source' => '来源',
			'web.plugins.skills.noDescription' => '无描述',
			'web.plugins.skills.builtinBadge' => '内置',
			'web.plugins.skills.builtinTooltip' => '嵌入 opendray 二进制 — 点击自定义可在 vault 中覆盖',
			'web.plugins.skills.vaultBadge' => 'vault',
			'web.plugins.skills.overridesBuiltin' => '覆盖内置',
			'web.plugins.skills.overridesBuiltinTooltip' => '此 vault skill 覆盖了同 id 的内置版本',
			'web.plugins.skills.customize' => '自定义',
			'web.plugins.skills.customizeTooltip' => '打开 SKILL.md 并保存为 vault 覆盖',
			'web.plugins.skills.editTooltip' => '编辑此 vault skill',
			'web.plugins.skills.resetTooltip' => '删除 vault 覆盖并回退到内置版本',
			'web.plugins.skills.reset' => '重置',
			'web.plugins.skills.resetConfirm' => ({required Object id}) => '将 "${id}" 重置为内置版本? 这会删除你的 vault SKILL.md 并回退到嵌入副本。',
			'web.plugins.skills.deleteConfirm' => ({required Object id}) => '从 vault 删除 skill "${id}"? 这会移除该 SKILL.md 文件。',
			'web.plugins.skills.removedToast' => 'Skill 已移除',
			_ => null,
		} ?? switch (path) {
			'web.plugins.skills.deleteFailedToast' => '删除失败',
			'web.plugins.skills.editor.createTitle' => '新建 skill',
			'web.plugins.skills.editor.customizeTitle' => ({required Object id}) => '自定义内置：${id}',
			'web.plugins.skills.editor.editTitle' => ({required Object id}) => '编辑 skill：${id}',
			'web.plugins.skills.editor.customizeDescription' => '你正在查看一个嵌入到 opendray 的内置 skill。保存会在相同 id 上创建一个 vault 覆盖 — 你的修改保存在 ~/.opendray/vault/skills/<id>/SKILL.md 中并遮盖内置版本，直到你点击重置。',
			'web.plugins.skills.editor.editDescription' => 'SKILL.md 格式 — frontmatter（name + description），然后是 markdown 指令。description 会出现在 agent 的 Tier 1 索引中。',
			'web.plugins.skills.editor.idLabel' => 'ID',
			'web.plugins.skills.editor.idPlaceholder' => 'my-helper',
			'web.plugins.skills.editor.idHint' => '小写字母 / 数字 / 短横 / 下划线。作为 <1>~/.opendray/vault/skills/&lt;id&gt;/</1> 下的目录名。',
			'web.plugins.skills.editor.bodyLabel' => 'SKILL.md',
			'web.plugins.skills.editor.createdToast' => 'Skill 已创建',
			'web.plugins.skills.editor.savedToast' => 'Skill 已保存',
			'web.plugins.skills.editor.savedOverrideToast' => '已保存为 vault 覆盖',
			'web.plugins.skills.editor.createFailedToast' => '创建失败',
			'web.plugins.skills.editor.saveFailedToast' => '保存失败',
			'web.plugins.skills.editor.saveAsOverride' => '保存为 vault 覆盖',
			'web.plugins.customTasks.title' => '自定义任务',
			'web.plugins.customTasks.description' => '在 Tasks 选项卡中以点选即运行的方式呈现的快捷方式。留空 cwd 即为所有会话可见的全局任务，或填写绝对路径以限定 scope。',
			'web.plugins.customTasks.addTask' => '添加任务',
			'web.plugins.customTasks.empty' => '尚无自定义任务。',
			'web.plugins.customTasks.columns.name' => '名称',
			'web.plugins.customTasks.columns.command' => '命令',
			'web.plugins.customTasks.columns.scope' => 'Scope',
			'web.plugins.customTasks.globalScope' => '全局',
			'web.plugins.customTasks.deleteConfirm' => ({required Object name}) => '删除自定义任务 "${name}"?',
			'web.plugins.customTasks.removedToast' => '任务已移除',
			'web.plugins.customTasks.deleteFailedToast' => '删除失败',
			'web.plugins.customTasks.dialog.addTitle' => '添加自定义任务',
			'web.plugins.customTasks.dialog.editTitle' => ({required Object name}) => '编辑 ${name}',
			'web.plugins.customTasks.dialog.description' => '命令会原样发送到会话的终端中，等同于在 prompt 处键入并回车。',
			'web.plugins.customTasks.dialog.nameLabel' => '名称',
			'web.plugins.customTasks.dialog.namePlaceholder' => 'dev',
			'web.plugins.customTasks.dialog.commandLabel' => '命令',
			'web.plugins.customTasks.dialog.commandPlaceholder' => 'docker compose up --build',
			'web.plugins.customTasks.dialog.descLabel' => '描述（可选）',
			'web.plugins.customTasks.dialog.descPlaceholder' => '启动开发环境并跟踪日志',
			'web.plugins.customTasks.dialog.cwdLabel' => 'cwd scope（可选）',
			'web.plugins.customTasks.dialog.cwdPlaceholder' => '/Users/me/projects/foo（留空 = 全局）',
			'web.plugins.customTasks.dialog.cwdHint' => '留空 = 在每个会话中都可见。否则只有当会话的 cwd 与此绝对路径匹配时才显示。',
			'web.plugins.customTasks.dialog.addedToast' => '任务已添加',
			'web.plugins.customTasks.dialog.updatedToast' => '任务已更新',
			'web.plugins.customTasks.dialog.addFailedToast' => '添加失败',
			'web.plugins.customTasks.dialog.updateFailedToast' => '更新失败',
			'web.plugins.gitHosts.title' => 'Git 主机',
			'web.plugins.gitHosts.description' => '每个主机一个 token — 被 Git 选项卡用来拉取 pull request，<1>也被 Notes vault sync</1> 使用（当其 remote 通过 HTTPS 指向同一主机上的私有仓库时）。支持 GitHub.com、自托管 GitHub Enterprise、Gitea 与 GitLab。',
			'web.plugins.gitHosts.addHost' => '添加主机',
			'web.plugins.gitHosts.empty' => '尚未配置任何 git 主机。\n添加一个以在检查器 Git 选项卡中启用 PR 列表。',
			'web.plugins.gitHosts.columns.host' => '主机',
			'web.plugins.gitHosts.columns.kind' => '类型',
			'web.plugins.gitHosts.columns.token' => 'Token',
			'web.plugins.gitHosts.columns.enabled' => '启用',
			'web.plugins.gitHosts.statusEnabled' => '已启用',
			'web.plugins.gitHosts.statusDisabled' => '已禁用',
			'web.plugins.gitHosts.deleteConfirm' => ({required Object host}) => '移除 git 主机 ${host}? 对该主机的 PR 查询将停止工作。',
			'web.plugins.gitHosts.removedToast' => 'Git 主机已移除',
			'web.plugins.gitHosts.deleteFailedToast' => '删除失败',
			'web.plugins.gitHosts.dialog.addTitle' => '添加 git 主机',
			'web.plugins.gitHosts.dialog.editTitle' => ({required Object host}) => '编辑 ${host}',
			'web.plugins.gitHosts.dialog.description' => 'Token 存储在网关上。仅用于只读 API 调用（列出 PR 等）。',
			'web.plugins.gitHosts.dialog.kindLabel' => '类型',
			'web.plugins.gitHosts.dialog.kindGitHub' => 'GitHub',
			'web.plugins.gitHosts.dialog.kindGitea' => 'Gitea',
			'web.plugins.gitHosts.dialog.kindGitLab' => 'GitLab',
			'web.plugins.gitHosts.dialog.hostLabel' => '主机',
			'web.plugins.gitHosts.dialog.hostPlaceholder' => 'github.com',
			'web.plugins.gitHosts.dialog.displayNameLabel' => '显示名称（可选）',
			'web.plugins.gitHosts.dialog.displayNamePlaceholder' => 'Personal',
			'web.plugins.gitHosts.dialog.tokenLabel' => 'Token',
			'web.plugins.gitHosts.dialog.newTokenLabel' => '新 token（留空表示保留）',
			'web.plugins.gitHosts.dialog.tokenPlaceholder' => 'ghp_… / gho_… / glpat-…',
			'web.plugins.gitHosts.dialog.tokenPlaceholderEdit' => '…',
			'web.plugins.gitHosts.dialog.tokenHint' => 'GitHub：带 <1>repo</1> scope 的 PAT。Gitea：带 <3>read:repository</3> 的 token。GitLab：带 <5>read_api</5> 的 PAT。',
			'web.plugins.gitHosts.dialog.enabledLabel' => '启用',
			'web.plugins.gitHosts.dialog.addedToast' => 'Git 主机已添加',
			'web.plugins.gitHosts.dialog.updatedToast' => 'Git 主机已更新',
			'web.plugins.gitHosts.dialog.addFailedToast' => '添加失败',
			'web.plugins.gitHosts.dialog.updateFailedToast' => '更新失败',
			'web.backups.title' => '备份',
			'web.backups.subtitle' => '写入到可插拔目标的加密 PostgreSQL dump。配置计划与保留策略，或触发一次性备份作为快速安全网。',
			'web.backups.exportData' => '导出数据',
			'web.backups.loading' => '加载中…',
			'web.backups.loadStatusFailedToast' => '加载备份状态失败',
			'web.backups.tabs.backups' => '备份',
			'web.backups.tabs.schedules' => '计划',
			'web.backups.tabs.targets' => '目标',
			'web.backups.inventory.title' => '备份里包含什么？',
			'web.backups.inventory.summary' => ({required Object tables, required Object rows}) => '${tables} 张表共 ${rows} 行',
			'web.backups.inventory.description' => '每次备份是一个对下方所有表执行 <1>pg_dump --format=custom</1> 的产物，外加 <3>manifest.json</3> 和（可选的）<5>config.toml</5>。计数是实时的；bundle 捕获的是备份发生那一刻的数据。',
			'web.backups.inventory.loadFailedToast' => '加载清单失败',
			'web.backups.inventory.rowsLabel' => '行',
			'web.backups.restart.title' => '请重启 opendray 以激活备份',
			'web.backups.restart.description' => '你的口令已保存。网关仅在启动时加载它，因此该功能在进程重启之前保持关闭。',
			'web.backups.restart.keyFile' => '密钥文件：',
			'web.backups.restart.configuredVia' => '配置方式：',
			'web.backups.restart.envVar' => 'OPENDRAY_BACKUP_KEY 环境变量',
			'web.backups.restart.checkAgain' => '再次检查',
			'web.backups.setup.title' => '设置备份',
			'web.backups.setup.description' => '选择一个主口令。opendray 会用它加密每个备份 blob。<1>丢失它意味着你的备份无法恢复</1>，请在继续之前把它保存到密码管理器（Vaultwarden、1Password 等）。',
			'web.backups.setup.generate' => '生成',
			'web.backups.setup.pasteOwn' => '粘贴自定义',
			'web.backups.setup.generateTitle' => '256 位随机密钥',
			'web.backups.setup.generateHint' => '服务端生成一个加密随机的口令并仅显示一次。你必须在继续前复制它 — 没有恢复路径。',
			'web.backups.setup.pasteLabel' => '你的口令',
			'web.backups.setup.pastePlaceholder' => '至少 20 个字符',
			'web.backups.setup.pasteHint' => '建议：使用密码管理器生成 40 个以上字符。',
			'web.backups.setup.savesTo' => '保存到：',
			'web.backups.setup.saving' => '保存中…',
			'web.backups.setup.generateAndSave' => '生成并保存',
			'web.backups.setup.save' => '保存',
			'web.backups.generated.title' => '立即保存此口令',
			'web.backups.generated.description' => '这是 <1>唯一一次</1> 显示。opendray 与任何其他地方都将无法取回。请在继续前复制到密码管理器。',
			'web.backups.generated.copy' => '复制',
			'web.backups.generated.copiedToast' => '口令已复制到剪贴板',
			'web.backups.generated.copyFailedToast' => '复制失败 — 请手动选中并复制',
			'web.backups.generated.savedTo' => '已保存到：',
			'web.backups.generated.ack' => '我已将此口令保存到密码管理器',
			'web.backups.generated.kContinue' => '继续',
			'web.backups.status.keyFingerprint' => '密钥指纹：',
			'web.backups.status.pgDump' => 'pg_dump：',
			'web.backups.status.pgDumpUnavailable' => '不可用',
			'web.backups.status.pgDumpHint' => '备份在 pg_dump 进入 PATH 之前无法运行（也可通过 <1>backup.pg_dump_path</1> 设置绝对路径）。请安装与你的服务器主版本匹配的 <3>postgresql-client</3> 并重启。',
			'web.backups.backupsTab.backupNow' => '立即备份',
			'web.backups.backupsTab.triggering' => '触发中…',
			'web.backups.backupsTab.includeConfig' => '包含 config.toml',
			'web.backups.backupsTab.restoreFromFile' => '从文件恢复',
			'web.backups.backupsTab.refresh' => '刷新',
			'web.backups.backupsTab.queuedToast' => '备份已排队',
			'web.backups.backupsTab.triggerFailedToast' => '触发失败',
			'web.backups.backupsTab.listFailedToast' => '加载备份列表失败',
			'web.backups.backupsTab.deleteConfirm' => ({required Object id}) => '删除备份 ${id}? 该 blob 将从目标中移除。',
			'web.backups.backupsTab.deletedToast' => '备份已删除',
			'web.backups.backupsTab.deleteFailedToast' => '删除失败',
			'web.backups.backupsTab.empty' => '暂无备份。点击上方 "立即备份" 进行第一次。',
			'web.backups.backupsTab.columns.id' => 'ID',
			'web.backups.backupsTab.columns.target' => '目标',
			'web.backups.backupsTab.columns.status' => '状态',
			'web.backups.backupsTab.columns.started' => '开始',
			'web.backups.backupsTab.columns.size' => '大小',
			'web.backups.backupsTab.columns.actions' => '操作',
			'web.backups.backupsTab.downloadTooltip' => '下载',
			'web.backups.backupsTab.deleteTooltip' => '删除',
			'web.backups.restore.title' => '从备份 bundle 恢复',
			'web.backups.restore.bundleLabel' => '加密 bundle（.tar.gz.enc）',
			'web.backups.restore.targetDsnLabel' => '目标数据库 DSN',
			'web.backups.restore.targetDsnHint' => '（留空 = opendray 自己的 DB — 危险）',
			'web.backups.restore.targetDsnPlaceholder' => 'postgres://user:pass@host:5432/dbname',
			'web.backups.restore.cleanLabel' => '--clean --if-exists（先 drop 现有 schema；恢复到已填充的 DB 上时必需）',
			'web.backups.restore.auditNoteLabel' => '审计备注（可选）',
			'web.backups.restore.auditNotePlaceholder' => '恢复原因 — 会出现在 slog 中',
			'web.backups.restore.ownDbWarning' => '你正在恢复到 <1>opendray 自己的数据库</1>。启用 "--clean" 时会 drop 每张表并按字面回放备份 — 不可逆。请输入 <3>I understand</3> 以继续。',
			'web.backups.restore.confirmPlaceholder' => 'I understand',
			'web.backups.restore.confirmSentinel' => 'I understand',
			'web.backups.restore.pgRestoreOutput' => 'pg_restore 输出（最后 8 KiB）',
			'web.backups.restore.noPgRestoreOutput' => '（无 pg_restore 输出）',
			'web.backups.restore.pickFileToast' => '请先选择一个 bundle 文件',
			'web.backups.restore.succeededToast' => '恢复成功',
			'web.backups.restore.replayedDescription' => ({required Object bytes, required Object id}) => '已回放 ${bytes}，来自 manifest ${id}',
			'web.backups.restore.failedToast' => '恢复失败',
			'web.backups.restore.restoring' => '恢复中…',
			'web.backups.restore.restore' => '恢复',
			'web.backups.schedulesTab.description' => '周期备份。调度器每 30 秒轮询一次，并运行最旧的到期计划。',
			'web.backups.schedulesTab.newSchedule' => '新建计划',
			'web.backups.schedulesTab.loadFailedToast' => '加载计划失败',
			'web.backups.schedulesTab.deleteConfirm' => ({required Object id}) => '删除计划 ${id}?',
			'web.backups.schedulesTab.deletedToast' => '计划已删除',
			'web.backups.schedulesTab.deleteFailedToast' => '删除失败',
			'web.backups.schedulesTab.toggleFailedToast' => '切换失败',
			'web.backups.schedulesTab.empty' => '暂无计划。添加一个以进行周期性自动备份。',
			'web.backups.schedulesTab.columns.id' => 'ID',
			'web.backups.schedulesTab.columns.target' => '目标',
			'web.backups.schedulesTab.columns.interval' => '间隔',
			'web.backups.schedulesTab.columns.keep' => '保留',
			'web.backups.schedulesTab.columns.nextRun' => '下次运行',
			'web.backups.schedulesTab.columns.enabled' => '启用',
			'web.backups.schedulesTab.columns.actions' => '操作',
			'web.backups.schedulesTab.keepCount' => ({required Object count}) => '${count} 个备份',
			'web.backups.schedulesTab.deleteTooltip' => '删除',
			'web.backups.newSchedule.title' => '新建备份计划',
			'web.backups.newSchedule.targetLabel' => '目标',
			'web.backups.newSchedule.everyHoursLabel' => '每隔（小时）',
			'web.backups.newSchedule.keepLastNLabel' => '保留最近 N 个',
			'web.backups.newSchedule.enableImmediately' => '立即启用',
			'web.backups.newSchedule.createdToast' => '计划已创建',
			'web.backups.newSchedule.createFailedToast' => '创建失败',
			'web.backups.newSchedule.creating' => '创建中…',
			'web.backups.newSchedule.create' => '创建',
			'web.backups.targetsTab.description' => '存储目标。v1 支持 <1>local</1>（opendray 主机磁盘）与 <3>smb</3>（任意 SMB / CIFS 共享，如 UNAS 或群晖）。',
			'web.backups.targetsTab.newTarget' => '新建目标',
			'web.backups.targetsTab.listFailedToast' => '加载目标列表失败',
			'web.backups.targetsTab.deleteConfirm' => ({required Object id}) => '删除目标 ${id}? 引用它的计划会阻止删除。',
			'web.backups.targetsTab.deletedToast' => '目标已删除',
			'web.backups.targetsTab.deleteFailedToast' => '删除失败',
			'web.backups.targetsTab.connectionOkToast' => '连接成功',
			'web.backups.targetsTab.connectionFailedToast' => '连接失败',
			'web.backups.targetsTab.testFailedToast' => '测试失败',
			'web.backups.targetsTab.columns.id' => 'ID',
			'web.backups.targetsTab.columns.kind' => '类型',
			'web.backups.targetsTab.columns.config' => '配置',
			'web.backups.targetsTab.columns.enabled' => '启用',
			'web.backups.targetsTab.columns.actions' => '操作',
			'web.backups.targetsTab.on' => '开',
			'web.backups.targetsTab.off' => '关',
			'web.backups.targetsTab.test' => '测试',
			'web.backups.targetsTab.testing' => '测试中…',
			'web.backups.targetsTab.deleteTooltip' => '删除',
			'web.backups.targetEditor.title' => '新建备份目标',
			'web.backups.targetEditor.kindPicker' => '你想备份到哪里？',
			'web.backups.targetEditor.idLabel' => 'ID（可选）',
			'web.backups.targetEditor.idPlaceholder' => '留空则自动生成，例如 tgt_xxx',
			'web.backups.targetEditor.createdToast' => '目标已创建',
			'web.backups.targetEditor.createFailedToast' => '创建失败',
			'web.backups.targetEditor.creating' => '创建中…',
			'web.backups.targetEditor.create' => '创建目标',
			'web.backups.targetEditor.enableImmediately' => '立即启用（否则保存为禁用 — 适合 "先配置好，稍后开启"）',
			'web.backups.targetEditor.local.rootLabel' => '根目录',
			'web.backups.targetEditor.local.rootHint' => '留空 = cfg.backup.local_dir (~/.opendray/backups)',
			'web.backups.targetEditor.local.rootPlaceholder' => '~/backups/opendray  或  /mnt/external-hdd/opendray',
			'web.backups.targetEditor.smb.hostLabel' => '主机',
			'web.backups.targetEditor.smb.hostPlaceholder' => '192.168.9.8',
			'web.backups.targetEditor.smb.portLabel' => '端口',
			'web.backups.targetEditor.smb.shareLabel' => 'Share',
			'web.backups.targetEditor.smb.shareHint' => 'SMB 服务器上的顶层共享名',
			'web.backups.targetEditor.smb.sharePlaceholder' => 'Claude_Workspace',
			'web.backups.targetEditor.smb.userLabel' => '用户',
			'web.backups.targetEditor.smb.passwordLabel' => '密码',
			'web.backups.targetEditor.smb.pathPrefixLabel' => '路径前缀',
			'web.backups.targetEditor.smb.pathPrefixHint' => '共享根下的子文件夹（可选）',
			'web.backups.targetEditor.smb.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.endpointLabel' => 'Endpoint',
			'web.backups.targetEditor.s3.endpointHint' => '主机（不要带协议）。AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000',
			'web.backups.targetEditor.s3.endpointPlaceholder' => 's3.amazonaws.com',
			'web.backups.targetEditor.s3.regionLabel' => 'Region',
			'web.backups.targetEditor.s3.regionHint' => '仅 AWS；R2 用 \'auto\'',
			'web.backups.targetEditor.s3.regionPlaceholder' => 'us-east-1 / auto',
			'web.backups.targetEditor.s3.bucketLabel' => 'Bucket',
			'web.backups.targetEditor.s3.bucketPlaceholder' => 'opendray-backups',
			'web.backups.targetEditor.s3.accessKeyLabel' => 'Access key',
			'web.backups.targetEditor.s3.secretKeyLabel' => 'Secret key',
			'web.backups.targetEditor.s3.secretKeyHint' => 'AES-256-GCM 加密存储；不会被回显',
			'web.backups.targetEditor.s3.pathPrefixLabel' => 'Path prefix',
			'web.backups.targetEditor.s3.pathPrefixHint' => 'Object-key 前缀（可选）',
			'web.backups.targetEditor.s3.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.s3.useHttps' => '使用 HTTPS',
			'web.backups.targetEditor.s3.pathStyle' => 'Path-style 寻址（legacy / MinIO）',
			'web.backups.targetEditor.webdav.baseUrlLabel' => 'Base URL',
			'web.backups.targetEditor.webdav.baseUrlHint' => '包含任意路径的完整 URL。示例：https://cloud.example.com/remote.php/dav/files/me/（Nextcloud）、https://nas.local:5006/（群晖）、https://dav.jianguoyun.com/dav/（坚果云）',
			'web.backups.targetEditor.webdav.baseUrlPlaceholder' => 'https://cloud.example.com/remote.php/dav/files/<user>/',
			'web.backups.targetEditor.webdav.userLabel' => '用户',
			'web.backups.targetEditor.webdav.passwordLabel' => '密码',
			'web.backups.targetEditor.webdav.pathPrefixLabel' => '路径前缀',
			'web.backups.targetEditor.webdav.pathPrefixHint' => 'Base URL 下的子文件夹（可选）',
			'web.backups.targetEditor.webdav.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.sftp.hostLabel' => '主机',
			'web.backups.targetEditor.sftp.hostPlaceholder' => 'vps.example.com',
			'web.backups.targetEditor.sftp.portLabel' => '端口',
			'web.backups.targetEditor.sftp.userLabel' => '用户',
			'web.backups.targetEditor.sftp.passwordLabel' => '密码',
			'web.backups.targetEditor.sftp.passwordHint' => '密码或私钥二选一。若两者都填，密码会被作为私钥口令使用。',
			'web.backups.targetEditor.sftp.privateKeyLabel' => '私钥（PEM）',
			'web.backups.targetEditor.sftp.privateKeyHint' => '粘贴 OpenSSH/PEM 格式的私钥内容（例如 ~/.ssh/id_ed25519）。留空则仅用密码认证。',
			'web.backups.targetEditor.sftp.privateKeyPlaceholder' => '-----BEGIN OPENSSH PRIVATE KEY-----...',
			'web.backups.targetEditor.sftp.hostKeyLabel' => 'Host key（pinning）',
			'web.backups.targetEditor.sftp.hostKeyHint' => 'OpenSSH 风格的服务器公钥（运行 `ssh-keyscan host` 获取）。留空则禁用 pinning（LAN 之外不推荐）。',
			'web.backups.targetEditor.sftp.hostKeyPlaceholder' => 'ssh-ed25519 AAAA...',
			'web.backups.targetEditor.sftp.pathPrefixLabel' => '路径前缀',
			'web.backups.targetEditor.sftp.pathPrefixHint' => '绝对路径或相对家目录（可选）',
			'web.backups.targetEditor.sftp.pathPrefixPlaceholder' => '/var/backups/opendray  或  opendray-backups',
			'web.backups.targetEditor.rclone.rcloneHint' => '要求 opendray 主机上已安装 <1>rclone</1> CLI。请先通过 <3>rclone config</3> 配置 remote，然后在下面填写 remote 名。opendray 在内部调用 <5>rclone rcat / cat / lsd</5>。',
			'web.backups.targetEditor.rclone.remoteLabel' => 'Remote name',
			'web.backups.targetEditor.rclone.remoteHint' => '来自 `rclone config` 的名字（不带冒号）。例如：gdrive、onedrive、dropbox-personal、baidu-pan',
			'web.backups.targetEditor.rclone.remotePlaceholder' => 'gdrive',
			'web.backups.targetEditor.rclone.pathPrefixLabel' => '路径前缀',
			'web.backups.targetEditor.rclone.pathPrefixHint' => 'Remote 根下的子文件夹（可选）',
			'web.backups.targetEditor.rclone.pathPrefixPlaceholder' => 'opendray/backups',
			'web.backups.targetEditor.rclone.binaryPathLabel' => '二进制路径',
			'web.backups.targetEditor.rclone.binaryPathHint' => '覆盖 `which rclone`。留空则走 PATH 查找。',
			'web.backups.targetEditor.rclone.binaryPathPlaceholder' => '/opt/homebrew/bin/rclone',
			'web.backups.targetEditor.rclone.configPathLabel' => 'Config 路径',
			'web.backups.targetEditor.rclone.configPathHint' => '覆盖 --config（默认 ~/.config/rclone/rclone.conf 或 ~/.rclone.conf）',
			'web.backups.targetEditor.rclone.configPathPlaceholder' => '留空则使用 rclone 默认',
			'web.serverSettings.sections.general.title' => '通用',
			'web.serverSettings.sections.general.desc' => '监听地址、操作员账号、令牌 TTL。',
			'web.serverSettings.sections.logging.title' => '日志',
			'web.serverSettings.sections.logging.desc' => '日志级别、格式与实时跟踪。',
			'web.serverSettings.sections.sessions.title' => '会话',
			'web.serverSettings.sections.sessions.desc' => '空闲检测阈值。',
			'web.serverSettings.sections.vault.title' => 'Vault',
			'web.serverSettings.sections.vault.desc' => '笔记、技能与 git 版本化根目录。',
			'web.serverSettings.sections.mcp.title' => 'MCP 注册表',
			'web.serverSettings.sections.mcp.desc' => '服务器注册表与密钥。',
			'web.serverSettings.sections.memory.title' => '记忆',
			'web.serverSettings.sections.memory.desc' => '跨 CLI 的持久化记忆子系统。',
			'web.serverSettings.sections.memoryAmbient.title' => '记忆 · 环境感知',
			'web.serverSettings.sections.memoryAmbient.desc' => '自动捕获对话进入记忆，并在 spawn 时注入。',
			'web.serverSettings.sections.backup.title' => '备份',
			'web.serverSettings.sections.backup.desc' => '加密数据库备份、恢复，以及管理员数据导出。',
			'web.serverSettings.sections.claude.title' => '存储 · Claude',
			'web.serverSettings.sections.claude.desc' => 'Claude 会话记录在磁盘上的位置。',
			'web.serverSettings.sections.codex.title' => '存储 · Codex',
			'web.serverSettings.sections.codex.desc' => 'Codex 会话根目录。',
			'web.serverSettings.sections.gemini.title' => '存储 · Gemini',
			'web.serverSettings.sections.gemini.desc' => 'Gemini 每项目 tmp 与 projects.json。',
			'web.serverSettings.loading' => '正在加载服务器设置…',
			'web.serverSettings.loadFailed' => ({required Object message}) => '加载失败：${message}',
			'web.serverSettings.noConfigFlag' => 'opendray 启动时未指定 -config，设置仅从环境变量加载，无法在此编辑。',
			'web.serverSettings.resetButton' => '重置',
			'web.serverSettings.resetButtonTitle' => '丢弃此分区中未保存的修改',
			'web.serverSettings.resetConfirm' => ({required Object section}) => '将"${section}"重置为上次保存的值？',
			'web.serverSettings.badgeRestartRequired' => '需要重启',
			'web.serverSettings.badgeUnsaved' => '未保存',
			'web.serverSettings.saveButton' => '保存修改',
			'web.serverSettings.saveToastTitle' => '设置已保存',
			'web.serverSettings.saveToastDesc' => '点击「重启」以应用。',
			'web.serverSettings.saveErrorTitle' => '保存失败',
			'web.serverSettings.dangerousConfirm' => '您更改了监听地址 / 管理员账号 / 管理员密码。重启后可能需要重新登录或使用新地址。是否继续？',
			'web.serverSettings.unsavedHint' => '有未保存的修改',
			'web.serverSettings.savedHint' => '所有修改已保存',
			'web.serverSettings.searchPlaceholder' => '筛选字段…',
			'web.serverSettings.restart.button' => '重启服务器',
			'web.serverSettings.restart.buttonTitle' => '对网关进程执行 self-exec',
			'web.serverSettings.restart.dirtyConfirm' => '您有未保存的修改。重启将使用「上次保存」的配置，是否继续？',
			'web.serverSettings.restart.confirm' => '重启 opendray 网关？所有打开的终端会话将自动重新连接。',
			'web.serverSettings.restart.overlay' => '正在重启服务器…',
			'web.serverSettings.restart.waiting' => ({required Object tick}) => '等待 /health · ${tick}s',
			'web.serverSettings.restart.timedOutTitle' => '重启超时',
			'web.serverSettings.restart.timedOutDesc' => 'Health 接口未恢复。请查看服务器日志。',
			'web.serverSettings.restart.successToast' => '服务器已重启',
			'web.serverSettings.formGroups.network' => '网络',
			'web.serverSettings.formGroups.operatorAccount' => '操作员账号',
			'web.serverSettings.formGroups.memoryConfiguration' => '配置',
			'web.serverSettings.formGroups.memoryHttp' => 'HTTP 后端（当 backend=http 时使用）',
			'web.serverSettings.formGroups.memoryLocal' => '本地 ONNX（当 backend=local 时使用）',
			'web.serverSettings.formGroups.backupStatus' => '状态',
			'web.serverSettings.formGroups.backupWhere' => '备份目标位置',
			'web.serverSettings.formGroups.backupSchedules' => '计划任务',
			'web.serverSettings.formGroups.backupWhatsInside' => '备份里有什么？',
			'web.serverSettings.fields.listenAddress.label' => '监听地址',
			'web.serverSettings.fields.listenAddress.hint' => 'HTTP 服务绑定的 host:port，例如：0.0.0.0:8770。',
			'web.serverSettings.fields.username.label' => '用户名',
			'web.serverSettings.fields.username.hint' => '登录表单使用的账号。修改后下次请求会强制重新登录。',
			'web.serverSettings.fields.password.label' => '密码',
			'web.serverSettings.fields.password.hint' => '留空保持当前密码不变；填值则会覆盖。',
			'web.serverSettings.fields.password.hideTitle' => '隐藏',
			'web.serverSettings.fields.password.revealTitle' => '显示',
			'web.serverSettings.fields.tokenTTL.label' => '令牌 TTL',
			'web.serverSettings.fields.tokenTTL.hint' => 'Bearer 令牌生命周期，使用 Go duration，如 "24h"、"30m"。留空 = 永不过期。',
			'web.serverSettings.fields.logLevel.label' => '日志级别',
			'web.serverSettings.fields.logLevel.hint' => '低于此级别的日志将被丢弃。',
			'web.serverSettings.fields.logFormat.label' => '格式',
			'web.serverSettings.fields.logFormat.hint' => '"text" 适合人读；"json" 便于机器解析。',
			'web.serverSettings.fields.logFile.label' => '日志文件',
			'web.serverSettings.fields.logFile.hint' => '可选的文件路径。10MB 自动轮转，保留 5 个备份。留空 = 仅输出到 stderr。',
			'web.serverSettings.fields.idleThreshold.label' => '空闲阈值',
			'web.serverSettings.fields.idleThreshold.hint' => '会话静默这么久后触发 session.idle。留空 = 30s。',
			'web.serverSettings.fields.idlePollInterval.label' => '空闲轮询间隔',
			'web.serverSettings.fields.idlePollInterval.hint' => '空闲检测器的唤醒频率。越低 = 延迟越低、唤醒越多。留空 = 5s。',
			'web.serverSettings.fields.vaultRoot.label' => 'Vault 根目录',
			'web.serverSettings.fields.vaultRoot.hint' => '笔记、技能和 MCP 注册表的顶层目录。',
			'web.serverSettings.fields.notesDirectory.label' => '笔记目录',
			'web.serverSettings.fields.notesDirectory.hint' => '覆盖笔记位置。默认为 <vault root>/notes。',
			'web.serverSettings.fields.skillsDirectory.label' => '技能目录',
			'web.serverSettings.fields.skillsDirectory.hint' => '覆盖技能位置。默认为 <vault root>/skills。',
			'web.serverSettings.fields.gitRoot.label' => 'Git 根目录',
			'web.serverSettings.fields.gitRoot.hint' => 'Vault Sync 功能提交到的工作树。',
			'web.serverSettings.fields.personalPrefix.label' => '个人前缀',
			'web.serverSettings.fields.personalPrefix.hint' => '自动派生路径时用于个人笔记的文件夹名。默认 "personal"。',
			'web.serverSettings.fields.projectsPrefix.label' => '项目前缀',
			'web.serverSettings.fields.projectsPrefix.hint' => '项目笔记的文件夹名。默认 "projects"。',
			'web.serverSettings.fields.registryRoot.label' => '注册表根目录',
			'web.serverSettings.fields.registryRoot.hint' => '存放 MCP server JSON 定义的目录。默认为 <vault>/mcp。',
			'web.serverSettings.fields.secretsFile.label' => '密钥文件',
			'web.serverSettings.fields.secretsFile.hint' => 'spawn 时替换进 MCP server 命令的 key=value 文件。',
			'web.serverSettings.fields.memoryBackend.label' => '嵌入器后端',
			'web.serverSettings.fields.memoryBackend.hint' => '"auto" / "bm25" 使用纯 Go 关键词路径（无需 cgo）；"http" 调用任何兼容 OpenAI 的 /v1/embeddings（ollama / OpenAI / LocalAI）；"local" 进程内运行 ONNX sentence-transformer — 需要用 `-tags local_onnx` 编译的二进制。',
			'web.serverSettings.fields.memoryStore.label' => '存储',
			'web.serverSettings.fields.memoryStore.hint' => '"pgvector" 复用 opendray 已有的 PG + vector 扩展；v1 唯一选项。',
			'web.serverSettings.fields.memoryTopK.label' => '默认 top-K',
			'web.serverSettings.fields.memoryTopK.hint' => 'agent 未指定时 memory_search 返回多少条命中。留空 = 5。',
			'web.serverSettings.fields.memoryThreshold.label' => '相似度阈值',
			'web.serverSettings.fields.memoryThreshold.hint' => '分数低于此值的命中将被丢弃。留空 = 0.1（宽松 — BM25 稀疏向量很少超过 0.5）。',
			'web.serverSettings.fields.memoryScope.label' => '默认作用域',
			'web.serverSettings.fields.memoryScope.hint' => 'agent 未指定时 memory_store 使用的作用域。"project"（推荐）按 cwd 分组；"session" 按会话隔离；"global" 跨 cwd 共享。',
			'web.serverSettings.fields.memoryBaseUrl.label' => 'Base URL',
			'web.serverSettings.fields.memoryBaseUrl.hint' => '如 ollama 用 "http://localhost:11434/v1"，OpenAI 用 "https://api.openai.com/v1"。',
			'web.serverSettings.fields.memoryModel.label' => '模型',
			'web.serverSettings.fields.memoryModel.hint' => '如 ollama 用 "nomic-embed-text"，OpenAI 用 "text-embedding-3-small"。',
			'web.serverSettings.fields.memoryApiKey.label' => 'API 密钥',
			'web.serverSettings.fields.memoryApiKey.hint' => 'ollama / 本地服务可留空；OpenAI / Voyage / 托管服务必填。',
			'web.serverSettings.fields.memoryLocalModel.label' => '模型名',
			'web.serverSettings.fields.memoryLocalModel.hint' => '仅作显示用 — 出现在日志 / Inspector 中。如 "bge-m3"、"bge-small-en-v1.5"。',
			'web.serverSettings.fields.memoryLibraryPath.label' => '库路径',
			'web.serverSettings.fields.memoryLibraryPath.hint' => '存放 libonnxruntime.dylib (macOS) / libonnxruntime.so (Linux) 的目录。`brew install onnxruntime` 后即 /opt/homebrew/opt/onnxruntime/lib。',
			'web.serverSettings.fields.memoryModelPath.label' => '模型路径',
			'web.serverSettings.fields.memoryModelPath.hint' => '.onnx 权重的绝对路径。从 HuggingFace 下载，如 Xenova/bge-m3 或 Xenova/bge-small-en-v1.5。',
			'web.serverSettings.fields.memoryTokenizerPath.label' => 'Tokenizer 路径',
			'web.serverSettings.fields.memoryTokenizerPath.hint' => 'tokenizer.json（HuggingFace 标准格式）的绝对路径 — 通常和模型放在一起。',
			'web.serverSettings.fields.memoryMaxSeqLen.label' => '最大序列长度',
			'web.serverSettings.fields.memoryMaxSeqLen.hint' => '超过这个 token 数会被截断。bge-m3 默认 512。留空 = 512。',
			'web.serverSettings.fields.claudeHistoryRoots.label' => '历史根目录',
			'web.serverSettings.fields.claudeHistoryRoots.hint' => '扫描 Claude 每项目 JSONL 记录的目录。留空 = 扫描 ~/.claude/projects + 所有 ~/.claude-accounts/*/projects。',
			'web.serverSettings.fields.claudeAccountsDir.label' => '账号目录',
			'web.serverSettings.fields.claudeAccountsDir.hint' => 'opendray 管理的 Claude 账号 ConfigDir 根目录。默认 ~/.claude-accounts。',
			'web.serverSettings.fields.codexSessionsRoot.label' => '会话根目录',
			'web.serverSettings.fields.codexSessionsRoot.hint' => '遍历 Codex rollout JSONL 文件的目录。默认 ~/.codex/sessions。',
			'web.serverSettings.fields.geminiTmpRoot.label' => 'tmp 目录',
			'web.serverSettings.fields.geminiTmpRoot.hint' => '存放 Gemini 每项目 tmp 文件夹的根目录。默认 ~/.gemini/tmp。',
			'web.serverSettings.fields.geminiProjectsFile.label' => 'projects.json',
			'web.serverSettings.fields.geminiProjectsFile.hint' => 'Gemini cwd→短名映射文件的路径。默认 ~/.gemini/projects.json。',
			'web.serverSettings.fields.backupLocalDir.label' => '本地备份目录',
			'web.serverSettings.fields.backupLocalDir.hint' => '自动创建的 `local` 目标的默认根目录。留空 = ~/.opendray/backups。需要重启。',
			'web.serverSettings.fields.backupExportDir.label' => '导出目录',
			'web.serverSettings.fields.backupExportDir.hint' => '一次性导出 zip 在磁盘上的暂存位置。留空 = ~/.opendray/exports。包将在 24 小时后自动过期。需要重启。',
			'web.serverSettings.fields.backupPgDumpPath.label' => 'pg_dump 路径',
			'web.serverSettings.fields.backupPgDumpPath.hint' => 'pg_dump 的绝对路径。主版本号必须 ≥ 服务器的。留空 = PATH 上的第一个 pg_dump。',
			'web.serverSettings.fields.backupPgRestorePath.label' => 'pg_restore 路径',
			'web.serverSettings.fields.backupPgRestorePath.hint' => '/backups/restore 流程使用的 pg_restore 绝对路径。同样的主版本号规则。',
			'web.serverSettings.liveTail.heading' => '实时日志',
			'web.serverSettings.liveTail.description' => '内存中的环形缓冲区（最近约 2,000 条）。重启后清空。',
			'web.serverSettings.memoryInspectorCard.heading' => '检查器',
			'web.serverSettings.memoryInspectorCard.description' => '在专门页面浏览、搜索、编辑已存储的记忆。',
			'web.serverSettings.memoryInspectorCard.openButton' => '打开记忆 →',
			'web.serverSettings.localOnnxBanner' => '需要使用 <1>-tags local_onnx</1> 编译二进制。标准构建在选择此后端时会返回明确的 stub 错误。设置步骤参见 <3>记忆 → 本地 ONNX</3> 教程。',
			'web.serverSettings.stringList.noneDefault' => '（无 — 使用内置默认值）',
			'web.serverSettings.stringList.addPath' => '添加路径',
			'web.serverSettings.stringList.removeTitle' => '移除',
			'web.serverSettings.httpHelpers.autoDetected' => '启动时自动检测到',
			'web.serverSettings.httpHelpers.modelCount' => ({required Object count}) => '${count} 个模型 — 点击使用',
			'web.serverSettings.httpHelpers.presets' => '预设：',
			'web.serverSettings.httpHelpers.testConnection' => '测试连接',
			'web.serverSettings.httpHelpers.presetTip.ollama' => '本地 ollama 守护进程',
			'web.serverSettings.httpHelpers.presetTip.lmStudio' => 'LM Studio 本地服务',
			'web.serverSettings.httpHelpers.presetTip.openai' => 'OpenAI 云端（需要 API key）',
			'web.serverSettings.probe.unreachable' => ({required Object error}) => '✗ 不可达：${error}',
			'web.serverSettings.probe.connectionFailed' => '连接失败',
			'web.serverSettings.probe.reachable' => ({required Object detected, required Object total, required Object embedding}) => '✓ 可达 ${detected}· 共 ${total} 个模型 · ${embedding} 个嵌入',
			'web.serverSettings.probe.modelMissing' => ({required Object model}) => '⚠ 配置的模型 ${model} 不在列表中。从下方嵌入模型中选一个，或修正名称。',
			'web.serverSettings.probe.embeddingModelsLabel' => '嵌入模型：',
			'web.serverSettings.probe.moreModels' => ({required Object count}) => '还有 ${count} 个',
			'web.serverSettings.probe.noEmbeddingFound' => '⚠ 没有模型名包含 "embed"。该端点可能未加载嵌入模型 — 请检查本地服务。',
			'web.serverSettings.probe.configuredTitle' => '当前已配置',
			'web.serverSettings.probe.applyTitle' => '点击应用',
			'web.serverSettings.backup.featureDisabledTitle' => '功能已禁用',
			'web.serverSettings.backup.featureDisabledHint' => '在 opendray 的环境变量中设置 <1>OPENDRAY_BACKUP_ENABLED=1</1> + <3>OPENDRAY_BACKUP_KEY=&lt;passphrase&gt;</3>，然后重启。主密码仅来自环境变量 — 永不写入 config.toml。',
			'web.serverSettings.backup.statusRowLabel' => '状态',
			'web.serverSettings.backup.enabledHealthy' => '已启用 · 健康',
			'web.serverSettings.backup.enabledDegraded' => '已启用 · 异常',
			'web.serverSettings.backup.keyFingerprintLabel' => '密钥指纹',
			'web.serverSettings.backup.keyFingerprintHint' => '记录到 Vaultwarden — 丢失会锁死所有先前的备份',
			'web.serverSettings.backup.pgDumpLabel' => 'pg_dump',
			'web.serverSettings.backup.pgDumpUnavailable' => '不可用',
			'web.serverSettings.backup.pgRestoreLabel' => 'pg_restore',
			'web.serverSettings.backup.pgRestoreNotResolved' => '（未解析）',
			'web.serverSettings.backup.openBackups' => '打开备份页 →',
			'web.serverSettings.backup.openExport' => '打开导出 / 导入 →',
			'web.serverSettings.backup.whereDesc' => '每个目标都是备份块可写入的一个地方。opendray 支持 <1>本地磁盘</1>、<3>SMB/CIFS</3>（Windows / NAS）、<5>S3 兼容</5>（AWS、R2、B2、MinIO、阿里云 OSS、腾讯云 COS …）、<7>WebDAV</7>（Nextcloud、群晖、坚果云）、<9>SFTP</9>，外加 <11>rclone</11> 透传，接入另外 70+ 个后端（Google Drive、OneDrive、Dropbox、百度云、阿里云盘 …）。',
			'web.serverSettings.backup.loading' => '加载中…',
			'web.serverSettings.backup.noTargets' => '尚未添加目标。添加一个开始备份。',
			'web.serverSettings.backup.addTarget' => '添加目标',
			'web.serverSettings.backup.noSchedulesHint' => '没有循环计划。在 <1>/backups → 计划任务</1> 添加一个以自动执行备份。',
			'web.serverSettings.backup.scheduleHeaders.schedule' => '计划',
			'web.serverSettings.backup.scheduleHeaders.target' => '目标',
			'web.serverSettings.backup.scheduleHeaders.cadence' => '频率',
			'web.serverSettings.backup.scheduleHeaders.keep' => '保留',
			'web.serverSettings.backup.scheduleHeaders.state' => '状态',
			'web.serverSettings.backup.every' => ({required Object interval}) => '每 ${interval}',
			'web.serverSettings.backup.backupsKeep' => ({required Object count}) => '${count} 份备份',
			'web.serverSettings.backup.stateEnabled' => '已启用',
			'web.serverSettings.backup.statePaused' => '已暂停',
			'web.serverSettings.backup.manageSchedules' => '在 /backups → 计划任务 管理 →',
			'web.serverSettings.backup.whatsInsideDesc' => '每份备份都是所有 opendray 表（sessions、integrations、memories、audit_log 等）的 <1>pg_dump --format=custom</1>，加上一个 <3>manifest.json</3>，以及（可选）当前的 <5>config.toml</5>。在 <7>备份页</7> 打开「备份里有什么？」面板可以看到带行数的实时清单。',
			'web.serverSettings.backup.advancedToggle' => '高级选项（路径与客户端二进制）— 需要重启',
			'web.serverSettings.targetRow.on' => '开',
			'web.serverSettings.targetRow.off' => '关',
			'web.serverSettings.targetRow.test' => '测试',
			'web.serverSettings.targetRow.testing' => '测试中…',
			'web.serverSettings.targetRow.delete' => '删除',
			'web.serverSettings.targetRow.connectionOk' => ({required Object id}) => '${id}：连接正常',
			'web.serverSettings.targetRow.connectionFailedTitle' => '连接失败',
			'web.serverSettings.targetRow.testFailedTitle' => '测试失败',
			'web.serverSettings.targetRow.deleteConfirm' => ({required Object id}) => '删除目标 "${id}"？引用它的计划任务将阻止删除。',
			'web.serverSettings.targetRow.deleteSuccess' => '目标已删除',
			'web.serverSettings.targetRow.deleteFailedTitle' => '删除失败',
			'web.serverSettings.targetRow.unknownError' => '未知错误',
			'web.settings.title' => '设置',
			'web.settings.subtitle' => '工作区、账号与网关配置。',
			'web.settings.groups.workspace' => '工作区',
			'web.settings.groups.server' => '服务器',
			'web.settings.groups.system' => '系统',
			'web.settings.items.appearance' => '外观',
			'web.settings.items.font' => '字号',
			'web.settings.items.account' => '账号',
			'web.settings.items.status' => '状态',
			'web.settings.items.about' => '关于',
			'web.settings.health.connecting' => '连接中…',
			'web.settings.health.dbOk' => 'db 正常',
			'web.settings.health.dbDown' => 'db 异常',
			'web.settings.breadcrumb.server' => '服务器',
			'web.settings.appearance.title' => '外观',
			'web.settings.appearance.description' => '选择 opendray 的外观风格。',
			'web.settings.appearance.options.light' => '浅色',
			'web.settings.appearance.options.lightDesc' => '始终浅色',
			'web.settings.appearance.options.dark' => '深色',
			'web.settings.appearance.options.darkDesc' => '始终深色',
			'web.settings.appearance.options.system' => '跟随系统',
			'web.settings.appearance.options.systemDesc' => '跟随操作系统设置',
			'web.settings.font.title' => '字号',
			'web.settings.font.description' => '缩放整个界面。按浏览器保存。',
			'web.settings.font.options.compact' => '紧凑',
			'web.settings.font.options.kDefault' => '默认',
			'web.settings.font.options.comfy' => '舒适',
			'web.settings.font.options.large' => '大',
			'web.settings.account.title' => '账号',
			_ => null,
		} ?? switch (path) {
			'web.settings.account.description' => '运维与当前 bearer token。',
			'web.settings.account.username' => '用户名',
			'web.settings.account.tokenExpires' => 'Token 过期',
			'web.settings.account.changeCredentials' => '修改凭证',
			'web.settings.changeCredentials.title' => '修改凭证',
			'web.settings.changeCredentials.description' => '先验证当前密码，再选择新凭证。所有其它已登录会话都将被吊销。',
			'web.settings.changeCredentials.currentPassword' => '当前密码',
			'web.settings.changeCredentials.newUsername' => '新用户名',
			'web.settings.changeCredentials.newPassword' => '新密码',
			'web.settings.changeCredentials.newPasswordHint' => '至少 8 个字符。',
			'web.settings.changeCredentials.confirm' => '确认新密码',
			'web.settings.changeCredentials.errorTooShort' => '新密码至少 8 个字符。',
			'web.settings.changeCredentials.errorMismatch' => '新密码和确认不一致。',
			'web.settings.changeCredentials.errorWrongPassword' => '当前密码不正确。',
			'web.settings.changeCredentials.cancel' => '取消',
			'web.settings.changeCredentials.update' => '更新',
			'web.settings.changeCredentials.saving' => '保存中…',
			'web.settings.system.title' => '系统状态',
			'web.settings.system.description' => '来自网关 /health 接口的实时状态。',
			'web.settings.system.status' => '状态',
			'web.settings.system.version' => '版本',
			'web.settings.system.uptime' => '运行时长',
			'web.settings.system.database' => '数据库',
			'web.settings.system.reachable' => '可达',
			'web.settings.system.unreachable' => '不可达',
			'web.settings.about.title' => '关于',
			'web.settings.about.description' => 'opendray v2 — 面向 AI agent CLI 的多路复用 + 集成网关。源码采用 Apache 2.0 协议。',
			'web.logViewer.filterPlaceholder' => '过滤…',
			'web.logViewer.debugTooltip' => 'Debug 计数',
			'web.logViewer.infoTooltip' => 'Info 计数',
			'web.logViewer.warnTooltip' => 'Warn 计数',
			'web.logViewer.errorTooltip' => 'Error 计数',
			'web.logViewer.streaming' => '正在流式传输',
			'web.logViewer.disconnected' => '已断开',
			'web.logViewer.live' => '实时',
			'web.logViewer.offline' => '离线',
			'web.logViewer.pauseTooltip' => '暂停自动滚动',
			'web.logViewer.resumeTooltip' => '恢复自动滚动',
			'web.logViewer.clearTooltip' => '清空本地视图（服务端 ring 不受影响）',
			'web.logViewer.downloadTooltip' => '下载完整 ring 为 .log 文件',
			'web.logViewer.emptyWaiting' => '等待日志记录…',
			'web.logViewer.emptyFiltered' => ({required Object query}) => '没有匹配 "${query}" 的记录',
			'web.pathInput.testButton' => '测试',
			'web.pathInput.testTooltip' => '解析并检查此路径',
			'web.pathInput.notFound' => '未找到 ·',
			'web.pathInput.childrenSuffix' => '项',
			'web.pathInput.expectedDirectory' => '· 期望是目录',
			'web.memoryAmbient.header.title' => '环境记忆 — 自动捕获与注入',
			'web.memoryAmbient.header.body' => 'opendray 每 10 秒轮询所有运行中的 agent 会话，通过可配置的 LLM 提取持久事实，去重后存入共享记忆池。配置由哪个 LLM 做提取（Provider）、何时触发提取（Capture rule）、以及在 spawn 时把什么内容（如果有）拼接到 agent 的 system prompt（Injection profile）。',
			'web.memoryAmbient.loading' => '加载中…',
			'web.memoryAmbient.providers.title' => 'Summarizer Providers',
			'web.memoryAmbient.providers.addButton' => '添加 provider',
			'web.memoryAmbient.providers.intro' => '至少需要一个已启用的 provider 才能真正触发捕获。本地选项（Ollama、LM Studio、Integration）让你的会话内容不出外网。',
			'web.memoryAmbient.providers.empty' => '尚未配置 provider。',
			'web.memoryAmbient.providers.row.defaultBadge' => '★ 默认',
			'web.memoryAmbient.providers.row.makeDefault' => '设为默认',
			'web.memoryAmbient.providers.row.test' => '测试',
			'web.memoryAmbient.providers.row.testing' => '测试中…',
			'web.memoryAmbient.providers.row.delete' => '删除',
			'web.memoryAmbient.providers.row.testOk' => ({required Object name}) => '${name}：连接成功',
			'web.memoryAmbient.providers.row.testFailedToast' => '测试失败',
			'web.memoryAmbient.providers.row.deleteConfirm' => ({required Object name}) => '删除 provider "${name}"?',
			'web.memoryAmbient.providers.row.deletedToast' => 'Provider 已删除',
			'web.memoryAmbient.providers.row.deleteFailedToast' => '删除失败',
			'web.memoryAmbient.providers.row.updateFailedToast' => '更新失败',
			'web.memoryAmbient.providers.row.madeDefaultToast' => ({required Object name}) => '${name} 已设为默认',
			'web.memoryAmbient.providers.dialog.title' => '添加 summarizer provider',
			'web.memoryAmbient.providers.dialog.kindLabel' => '类型',
			'web.memoryAmbient.providers.dialog.nameLabel' => '名称',
			'web.memoryAmbient.providers.dialog.namePlaceholder' => '例如 lmstudio-qwen',
			'web.memoryAmbient.providers.dialog.modelLabel' => '模型',
			'web.memoryAmbient.providers.dialog.baseUrlLabel' => 'Base URL',
			'web.memoryAmbient.providers.dialog.integrationNote' => 'Integration 类型 provider 通过一个已注册的集成解析 base URL。请先在 Integrations 中配置；更高级的 wiring（extra_config）在本版本中仅 DB 配置。',
			'web.memoryAmbient.providers.dialog.apiKeyLabel' => 'API key',
			'web.memoryAmbient.providers.dialog.apiKeyHint' => 'AES-GCM 加密存储（使用 backup 主口令）。不会被回显；保存后只显示指纹。',
			'web.memoryAmbient.providers.dialog.makeDefaultLabel' => '将此设为默认 provider',
			'web.memoryAmbient.providers.dialog.create' => '创建',
			'web.memoryAmbient.providers.dialog.nameRequiredToast' => '名称不能为空',
			'web.memoryAmbient.providers.dialog.createdToast' => ({required Object name}) => '已创建 Provider ${name}',
			'web.memoryAmbient.providers.dialog.createFailedToast' => '创建失败',
			'web.memoryAmbient.rules.title' => '捕获规则',
			'web.memoryAmbient.rules.addButton' => '添加规则',
			'web.memoryAmbient.rules.intro' => '每条规则表示 "当此 trigger 触发时，对新的会话消息做总结并存储持久事实。" 单会话规则覆盖全局默认。v1 内置 4 种 trigger 类型。',
			'web.memoryAmbient.rules.empty' => '尚无捕获规则。添加一条以启用自动捕获。',
			'web.memoryAmbient.rules.row.globalDefault' => '全局默认',
			'web.memoryAmbient.rules.row.scopeLabel' => 'scope:',
			'web.memoryAmbient.rules.row.dedupLabel' => 'dedup:',
			'web.memoryAmbient.rules.row.runNow' => '立即运行',
			'web.memoryAmbient.rules.row.running' => '运行中…',
			'web.memoryAmbient.rules.row.delete' => '删除',
			'web.memoryAmbient.rules.row.firedToast' => ({required Object sessions}) => '规则已对 ${sessions} 个会话触发',
			'web.memoryAmbient.rules.row.runNowFailedToast' => '立即运行失败',
			'web.memoryAmbient.rules.row.deleteConfirm' => ({required Object name}) => '删除规则 "${name}"?',
			'web.memoryAmbient.rules.row.deletedToast' => '规则已删除',
			'web.memoryAmbient.rules.row.deleteFailedToast' => '删除失败',
			'web.memoryAmbient.rules.row.summary.afterMessages' => ({required Object n}) => '每 ${n} 条消息',
			'web.memoryAmbient.rules.row.summary.onIdle' => ({required Object seconds}) => 'idle ≥ ${seconds}s',
			'web.memoryAmbient.rules.row.summary.kChars' => ({required Object k}) => '≥ ${k} 字符',
			'web.memoryAmbient.rules.row.summary.manual' => '仅手动',
			'web.memoryAmbient.rules.dialog.title' => '添加捕获规则',
			'web.memoryAmbient.rules.dialog.nameLabel' => '名称',
			'web.memoryAmbient.rules.dialog.triggerLabel' => 'Trigger',
			'web.memoryAmbient.rules.dialog.nLabel' => 'N（消息条数）',
			'web.memoryAmbient.rules.dialog.idleLabel' => 'Idle 秒数',
			'web.memoryAmbient.rules.dialog.kLabel' => 'K（字符数）',
			'web.memoryAmbient.rules.dialog.scopeLabel' => '目标 scope',
			'web.memoryAmbient.rules.dialog.scopeSession' => 'session',
			'web.memoryAmbient.rules.dialog.scopeProject' => 'project（推荐）',
			'web.memoryAmbient.rules.dialog.scopeGlobal' => 'global',
			'web.memoryAmbient.rules.dialog.dedupLabel' => '去重阈值（0.0 – 1.0）',
			'web.memoryAmbient.rules.dialog.dedupHint' => '越高 = 去重越严格。0.85 是推荐的平衡点。',
			'web.memoryAmbient.rules.dialog.create' => '创建',
			'web.memoryAmbient.rules.dialog.nameRequiredToast' => '名称不能为空',
			'web.memoryAmbient.rules.dialog.createdToast' => ({required Object name}) => '已创建规则 ${name}',
			'web.memoryAmbient.rules.dialog.createFailedToast' => '创建失败',
			'web.memoryAmbient.profiles.title' => 'Injection Profiles',
			'web.memoryAmbient.profiles.addButton' => '添加 profile',
			'web.memoryAmbient.profiles.intro' => 'spawn 时，opendray 会把最近的项目记忆作为一段 markdown banner 拼接到 agent 的 system prompt — 前提是配置了 profile。没有 profile 时，模型仍可按需调用 memory_search。',
			'web.memoryAmbient.profiles.empty' => '尚无 injection profile。spawn 时不会自动注入记忆 — 模型仍可使用 memory_search。',
			'web.memoryAmbient.profiles.row.globalDefault' => '全局默认',
			'web.memoryAmbient.profiles.row.delete' => '删除',
			'web.memoryAmbient.profiles.row.deleteConfirm' => '删除该 injection profile?',
			'web.memoryAmbient.profiles.row.deletedToast' => 'Profile 已删除',
			'web.memoryAmbient.profiles.row.deleteFailedToast' => '删除失败',
			'web.memoryAmbient.profiles.dialog.title' => '添加 injection profile',
			'web.memoryAmbient.profiles.dialog.strategyLabel' => '策略',
			'web.memoryAmbient.profiles.dialog.kLabel' => 'K（注入的 top memory 数）',
			'web.memoryAmbient.profiles.dialog.hint' => '每个 session_id 一个 profile（或全局默认）。单会话 profile 暂只能通过 API 添加；UI 当前只管理全局默认。',
			'web.memoryAmbient.profiles.dialog.create' => '创建',
			'web.memoryAmbient.profiles.dialog.createdToast' => 'Profile 已创建',
			'web.memoryAmbient.profiles.dialog.createFailedToast' => '创建失败',
			'web.memoryAmbient.cost.title' => 'Token 成本（总计）',
			'web.memoryAmbient.cost.intro' => '按 provider 聚合自 <1>memory_summarizer_calls</1>。本地 provider（Ollama、LM Studio、Integration）按 \$0 计价 — 硬件成本由运维承担。',
			'web.memoryAmbient.cost.empty' => '暂无已启用的 provider — 没有成本数据。',
			'web.memoryAmbient.cost.columns.provider' => 'Provider',
			'web.memoryAmbient.cost.columns.calls' => '调用',
			'web.memoryAmbient.cost.columns.inTokens' => '输入 token',
			'web.memoryAmbient.cost.columns.outTokens' => '输出 token',
			'web.memoryAmbient.cost.columns.usdEst' => 'USD 估算',
			'web.noteEditor.loading' => '加载中…',
			'web.noteEditor.source' => '源码',
			'web.noteEditor.preview' => '预览',
			'web.noteEditor.tagTitle' => ({required Object tag}) => '标签 #${tag}',
			'web.noteEditor.emptyNote' => '空白笔记。切换到源码标签开始书写。',
			'web.noteEditor.saveFailedToast' => '保存失败',
			'web.noteEditor.status.saveFailed' => '保存失败',
			'web.noteEditor.status.saving' => '保存中…',
			'web.noteEditor.status.unsaved' => '未保存',
			'web.noteEditor.status.newNote' => '新笔记',
			'web.noteEditor.status.saved' => '已保存',
			'web.export.title' => '导出数据',
			'web.export.subtitle' => '把选中的逻辑实体打成一份一次性 zip 包。服务器上保留 24 小时后自动回收。',
			'web.export.backToBackups' => '← 备份',
			'web.export.sections.export' => '导出',
			'web.export.sections.import' => '导入',
			'web.export.form.scope' => '范围',
			'web.export.form.memories' => '记忆',
			'web.export.form.memoriesHint' => '跨 CLI 持久化的记忆行（text + scope + metadata）。向量被省略；导入端重嵌入。',
			'web.export.form.integrations' => '集成',
			'web.export.form.customTasks' => '自定义任务',
			'web.export.form.customTasksHint' => '在 Inspector 的 Tasks 标签里展示的运维自定义任务。',
			'web.export.form.integrationOptions.none' => '无',
			'web.export.form.integrationOptions.noneHint' => '完全跳过 integrations 表。',
			'web.export.form.integrationOptions.metadata' => '仅元数据（推荐）',
			'web.export.form.integrationOptions.metadataHint' => 'ID、name、route prefix、scopes — 不包含任何 API key 凭证。',
			'web.export.form.integrationOptions.plaintext' => '包含明文 API key',
			'web.export.form.integrationOptions.plaintextHint' => 'v1 仅 bcrypt：不存在可恢复的明文。Manifest 会记录此事实；不会泄露任何内容。',
			'web.export.form.confirmWarning' => '输入 <1>I understand</1> 以确认。opendray 当前只存 bcrypt 哈希 — 选择明文也不会导出任何明文（该选项为将来保留明文缓存的版本而预留）。',
			'web.export.form.confirmPlaceholder' => 'I understand',
			'web.export.form.confirmSentinel' => 'i understand',
			'web.export.form.footnote' => '审计日志与会话记录不在范围内 — 由 /backups（运维 dump）覆盖。',
			'web.export.form.building' => '构建中…',
			'web.export.form.create' => '创建导出',
			'web.export.form.readyToast' => '导出就绪',
			'web.export.form.readyDescription' => ({required Object bytes}) => '${bytes} 字节',
			'web.export.form.failedToast' => '导出失败',
			'web.export.history.loading' => '加载中…',
			'web.export.history.empty' => '暂无导出。请使用上面的表单创建一个。',
			'web.export.history.title' => '历史',
			'web.export.history.columns.id' => 'ID',
			'web.export.history.columns.status' => '状态',
			'web.export.history.columns.scope' => '范围',
			'web.export.history.columns.size' => '大小',
			'web.export.history.columns.expires' => '过期',
			'web.export.history.columns.actions' => '操作',
			'web.export.history.download' => '下载',
			'web.export.history.deleteTooltip' => '删除',
			'web.export.history.listFailedToast' => '加载导出列表失败',
			'web.export.history.downloadFailedToast' => '下载失败',
			'web.export.history.noTokenToast' => '没有下载 token（已过期？）',
			'web.export.history.deleteConfirm' => ({required Object id}) => '删除导出 ${id}?',
			'web.export.history.deletedToast' => '导出已删除',
			'web.export.history.deleteFailedToast' => '删除失败',
			'web.export.history.scopeEmpty' => '(空)',
			'web.export.import.intro' => '将一个导出 bundle（zip）回放到当前数据库。冲突项（id 一致，或 integrations 的 route_prefix 唯一冲突）默认 <1>跳过</1>。记忆条目会被标记为 <3>embedder=imported_v1</3>，需要做一次重嵌入后搜索才能返回；可在 <5>Memory → 维护</5> 触发。集成以 <7>enabled=false</7> 导入并使用非 bcrypt 的占位 key — 启用前请轮换。',
			'web.export.import.memoryLink' => 'Memory → 维护',
			'web.export.import.bundleLabel' => 'Bundle (.zip)',
			'web.export.import.memoriesLabel' => '记忆',
			'web.export.import.integrationsLabel' => '集成（仅元数据 — 不会导入 key）',
			'web.export.import.customTasksLabel' => '自定义任务',
			'web.export.import.importing' => '导入中…',
			'web.export.import.importBundle' => '导入 bundle',
			'web.export.import.pickFileToast' => '请先选择一个 bundle 文件',
			'web.export.import.doneToast' => '导入完成',
			'web.export.import.finishedWithErrors' => '导入完成但有错误',
			'web.export.import.failedToast' => '导入失败',
			'web.export.import.summaryCard.memories' => '记忆',
			'web.export.import.summaryCard.integrations' => '集成',
			'web.export.import.summaryCard.customTasks' => '自定义任务',
			'web.export.import.summaryCard.created' => '已创建',
			'web.export.import.summaryCard.skipped' => '已跳过',
			'web.export.import.summaryCard.failed' => '失败',
			'web.export.imports.loading' => '加载中…',
			'web.export.imports.empty' => '暂无导入。',
			'web.export.imports.title' => '历史',
			'web.export.imports.columns.id' => 'ID',
			'web.export.imports.columns.status' => '状态',
			'web.export.imports.columns.source' => '来源',
			'web.export.imports.columns.counts' => '计数',
			'web.export.imports.columns.when' => '时间',
			'web.export.imports.noneCounts' => '(无)',
			'web.export.imports.listFailedToast' => '加载导入列表失败',
			'more.title' => '更多',
			'more.identity.signedInAs' => '登录账号',
			'more.identity.server' => '服务器',
			'more.identity.tokenExpires' => '令牌到期',
			'more.sections.gateway' => '网关',
			'more.sections.memory' => '记忆',
			'more.sections.system' => '系统',
			'more.items.integrations.title' => '集成',
			'more.items.integrations.subtitle' => 'API 调用方 — 近期活动与错误率',
			'more.items.channels.title' => '通道',
			'more.items.channels.subtitle' => '通知目的地',
			'more.items.providers.title' => '提供商',
			'more.items.providers.subtitle' => 'Claude / Codex / Gemini CLI 状态',
			'more.items.mcp.title' => 'MCP',
			'more.items.mcp.subtitle' => 'Model Context Protocol 服务与密钥',
			'more.items.skills.title' => '技能',
			'more.items.skills.subtitle' => 'Agent SKILL.md 库（内置 + 库）',
			'more.items.gitHosts.title' => 'Git 主机',
			'more.items.gitHosts.subtitle' => 'GitHub / GitLab 等的 PAT 凭据',
			'more.items.customTasks.title' => '自定义任务',
			'more.items.customTasks.subtitle' => '会话任务选择器中显示的斜杠命令',
			'more.items.projectMemory.title' => '项目目标 / 计划 / 日志',
			'more.items.projectMemory.subtitle' => '按 cwd 的记忆层 2-4 + 代理提案',
			'more.items.cleanupInbox.title' => '清理收件箱',
			'more.items.cleanupInbox.subtitle' => '跨项目的 LLM 提议删除 / 合并',
			'more.items.backups.title' => '备份',
			'more.items.backups.subtitle' => '最新备份状态与立即运行',
			'more.items.settings.title' => '设置',
			'more.items.settings.subtitle' => '语言、外观、账户',
			'more.items.about.title' => '关于',
			'more.items.about.subtitle' => '构建版本与服务器信息',
			'more.signOut' => '退出登录',
			'sessions.title' => '会话',
			'sessions.refresh' => '刷新',
			'sessions.actions' => '操作',
			'sessions.spawn' => '创建',
			'sessions.filters.all' => '全部',
			'sessions.filters.running' => '运行中',
			'sessions.filters.idle' => '空闲',
			'sessions.filters.ended' => '已结束',
			'sessions.card.startedRelative' => ({required Object provider, required Object when}) => '${provider} · ${when}启动',
			'sessions.empty.titleAll' => '暂无会话',
			'sessions.empty.titleFiltered' => ({required Object filter}) => '没有匹配「${filter}」筛选的会话。',
			'sessions.empty.subtitleAll' => '点击「创建」按钮新建一个。',
			'sessions.empty.subtitleFiltered' => '试试其他筛选条件或下拉刷新。',
			'sessions.errorTitle' => '加载会话失败',
			'sessions.relative.secondsAgo' => ({required Object n}) => '${n} 秒前',
			'sessions.relative.minutesAgo' => ({required Object n}) => '${n} 分钟前',
			'sessions.relative.hoursAgo' => ({required Object n}) => '${n} 小时前',
			'sessions.relative.daysAgo' => ({required Object n}) => '${n} 天前',
			'sessions.detail.fallbackTitle' => '会话',
			'sessions.detail.refreshMetadata' => '刷新元数据',
			'sessions.detail.inspector' => '检查器（文件 / Git / 任务 / 历史 / 笔记）',
			'sessions.detail.projectMemory' => '项目记忆（目标 / 计划 / 日志 / 收件箱）',
			'sessions.detail.actions' => '操作',
			'sessions.detail.started' => ({required Object when}) => '${when} 启动',
			'sessions.detail.startedEnded' => ({required Object started, required Object ended}) => '${started} 启动  ·  ${ended} 结束',
			'sessions.detail.idPrefix' => ({required Object id}) => 'id: ${id}',
			'sessions.detail.errorTitle' => '加载会话失败',
			'sessions.terminal.snackbar.imagePickerFailed' => ({required Object error}) => '图片选择失败：${error}',
			'sessions.terminal.snackbar.uploadingImage' => '正在上传图片…',
			'sessions.terminal.snackbar.imageAttached' => ({required Object path}) => '已附加图片：${path}',
			'sessions.terminal.snackbar.uploadFailed' => ({required Object status, required Object message}) => '上传失败（${status}）：${message}',
			'sessions.terminal.snackbar.uploadFailedGeneric' => ({required Object error}) => '上传失败：${error}',
			'sessions.terminal.imageSource.photoLibrary' => '相册',
			'sessions.terminal.imageSource.takePhoto' => '拍照',
			'sessions.terminal.keyboard.copyBuffer' => '复制缓冲',
			'sessions.terminal.keyboard.paste' => '粘贴',
			'sessions.terminal.keyboard.attachImage' => '附加图片',
			'sessions.terminal.keyboard.enter' => '回车',
			'sessions.terminal.connection.connecting' => '连接中…',
			'sessions.terminal.connection.connected' => '已连接',
			'sessions.terminal.connection.reconnecting' => '重连中…',
			'sessions.terminal.connection.reconnectingWithError' => ({required Object error}) => '重连中（${error}）…',
			'sessions.terminal.connection.disconnected' => '已断开',
			'sessions.terminal.connection.disconnectedWithError' => ({required Object error}) => '已断开（${error}）',
			'sessions.terminal.connection.ended' => '会话已结束',
			'sessions.action.stop' => '停止',
			'sessions.action.stopping' => '停止中…',
			'sessions.action.stopDescription' => '发送 SIGTERM，保留历史',
			'sessions.action.restart' => '重启',
			'sessions.action.restarting' => '重启中…',
			'sessions.action.restartDescription' => '重新启动 CLI 进程',
			'sessions.action.delete' => '删除',
			'sessions.action.deleteDescription' => '移除会话及其历史',
			'sessions.action.deleteConfirm' => '确定永久删除此会话吗？其环形缓冲区和历史将全部丢失。',
			'sessions.action.errors.stop' => ({required Object error}) => '停止失败：${error}',
			'sessions.action.errors.start' => ({required Object error}) => '重启失败：${error}',
			'sessions.action.errors.delete' => ({required Object error}) => '删除失败：${error}',
			'sessions.dirPicker.parent' => '上级',
			'sessions.dirPicker.newFolder' => '新建文件夹',
			'sessions.dirPicker.useThisFolder' => '使用此文件夹',
			'sessions.dirPicker.loading' => '加载中…',
			'sessions.dirPicker.empty' => '此处没有子文件夹。\n选择此文件夹，或新建一个。',
			'sessions.dirPicker.createdSnack' => ({required Object path}) => '已创建 ${path}',
			'sessions.dirPicker.mkdirFailedSnack' => ({required Object error}) => '创建文件夹失败：${error}',
			'sessions.dirPicker.dialog.title' => '新建文件夹',
			'sessions.dirPicker.dialog.hint' => '文件夹名',
			'sessions.dirPicker.dialog.create' => '创建',
			'sessions.inspector.shell.title' => '检查器',
			'sessions.inspector.shell.loadError' => ({required Object error}) => '加载会话失败：${error}',
			'sessions.inspector.shell.tabs.files' => '文件',
			'sessions.inspector.shell.tabs.git' => 'Git',
			'sessions.inspector.shell.tabs.tasks' => '任务',
			'sessions.inspector.shell.tabs.history' => '历史',
			'sessions.inspector.shell.tabs.notes' => '笔记',
			'sessions.inspector.shared.refresh' => '刷新',
			'sessions.inspector.shared.inserted' => ({required Object text}) => '已插入：${text}',
			'sessions.inspector.shared.insertFailedApi' => ({required Object status, required Object message}) => '插入失败（${status}）：${message}',
			'sessions.inspector.shared.insertFailedGeneric' => ({required Object error}) => '插入失败：${error}',
			'sessions.inspector.shared.insertFailedShort' => ({required Object error}) => '插入失败：${error}',
			'sessions.inspector.history.insertIntoTerminal' => '插入到终端',
			'sessions.inspector.history.searchHint' => '搜索提示…',
			'sessions.inspector.files.insertAtRef' => '作为 @引用 插入',
			'sessions.inspector.files.insertPath' => '插入路径',
			'sessions.inspector.files.insertPathSubtitle' => '原样粘贴绝对路径',
			'sessions.inspector.files.readContent' => '读取内容',
			'sessions.inspector.files.readContentSubtitle' => '最多 256 KiB 纯文本',
			'sessions.inspector.files.readFailedApi' => ({required Object status, required Object message}) => '读取失败（${status}）：${message}',
			'sessions.inspector.files.readFailedGeneric' => ({required Object error}) => '读取失败：${error}',
			'sessions.inspector.files.parent' => '上级',
			'sessions.inspector.files.backToCwd' => '返回会话目录',
			'sessions.inspector.git.insertAtRef' => '作为 @引用 插入',
			'sessions.inspector.git.insertPath' => '插入路径',
			'sessions.inspector.git.showDiff' => '查看 diff',
			'sessions.inspector.git.diffFailedApi' => ({required Object status, required Object message}) => 'Diff 失败（${status}）：${message}',
			'sessions.inspector.git.diffFailedGeneric' => ({required Object error}) => 'Diff 失败：${error}',
			'sessions.inspector.git.insertHash' => '插入哈希',
			'sessions.inspector.git.showFullPatch' => '查看完整 patch',
			'sessions.inspector.git.showFailedApi' => ({required Object status, required Object message}) => '查看失败（${status}）：${message}',
			'sessions.inspector.git.showFailedGeneric' => ({required Object error}) => '查看失败：${error}',
			'sessions.inspector.git.tabStatus' => '状态',
			'sessions.inspector.git.tabLog' => '日志',
			'sessions.inspector.tasks.runCommand' => '运行命令',
			'sessions.inspector.tasks.insertCommand' => '插入命令',
			'sessions.inspector.tasks.insertCommandSubtitle' => '粘贴但不回车，方便编辑',
			'sessions.inspector.notes.insertedAt' => ({required Object path}) => '已插入：@${path}',
			'sessions.inspector.notes.myNotes' => '我的笔记',
			'sessions.inspector.notes.projectDocs' => '项目文档',
			'sessions.inspector.notes.insertAtRefTooltip' => '作为 @引用 插入',
			'sessions.inspector.notes.insertAtRefShort' => '插入 @引用',
			'sessions.inspector.notes.draftHint' => ({required Object project}) => '# ${project}\n\n想法、待办、为 agent 提供的上下文…',
			'sessions.inspector.notes.createFailed' => ({required Object error}) => '创建失败：${error}',
			'sessions.inspector.notes.saveFailed' => ({required Object error}) => '保存失败：${error}',
			'sessions.inspector.notes.changeLocationTooltip' => '更改项目文档位置',
			'sessions.inspector.notes.filenameHint' => '文件名（例如：spec 或 design.md）',
			'sessions.inspector.notes.create' => '创建',
			'sessions.inspector.notes.filterHint' => '筛选…',
			'sessions.inspector.notes.locationDialogTitle' => '项目文档位置',
			'sessions.inspector.notes.loadFailedApi' => ({required Object error}) => '加载失败：${error}',
			'sessions.inspector.notes.loadFailedGeneric' => ({required Object error}) => '加载失败：${error}',
			'sessions.inspector.notes.saveFailedApi' => ({required Object error}) => '保存失败：${error}',
			'sessions.inspector.notes.saveFailedGeneric' => ({required Object error}) => '保存失败：${error}',
			'sessions.inspector.notes.insertFailedApi' => ({required Object error}) => '插入失败：${error}',
			'sessions.inspector.notes.insertFailedGeneric' => ({required Object error}) => '插入失败：${error}',
			'sessions.inspector.notes.createFailedApi' => ({required Object error}) => '创建失败：${error}',
			'sessions.inspector.notes.createFailedGeneric' => ({required Object error}) => '创建失败：${error}',
			'sessions.inspector.notes.personalHint' => '个人草稿 — 随输入自动保存。AI agent 不会写入这里。',
			'sessions.inspector.notes.projectDocsHint' => '架构 / 规范 / 决策 / 计划 / 回顾 — 通常由 agent 撰写或维护。',
			'sessions.inspector.notes.mappingCleared' => '映射已清除 — 使用默认值',
			'sessions.inspector.notes.mappedTo' => ({required Object path}) => '已映射到 ${path}',
			'sessions.inspector.notes.cancelTooltip' => '取消',
			'sessions.inspector.notes.newDocTooltip' => '新建文档',
			'sessions.inspector.notes.noProjectMapping' => '无法为此会话解析项目映射。检查网关是否配置了笔记库，以及会话的 cwd 是否已设置。',
			'sessions.inspector.notes.emptyProjectDocs' => '暂无项目文档。点击 + 创建一个，或让 AI agent 根据提示生成。',
			'sessions.inspector.notes.emptyFilterMatch' => ({required Object query}) => '未找到匹配「${query}」的内容。',
			'sessions.inspector.notes.locationDialogHelp' => '将此会话的 cwd 固定到笔记库下的某个文件夹。留空 = 重置。',
			'sessions.inspector.notes.sessionCwd' => '会话 cwd',
			'sessions.inspector.notes.projectDocsPath' => '相对笔记库的项目文档路径',
			'sessions.inspector.notes.locationStoredHint' => '存储于 <vault>/.opendray-projects.json — 与笔记库其余部分一起 git 同步。',
			'sessions.inspector.notes.pinnedHint' => ({required Object path, required Object defaultPath}) => '已固定到 ${path}/（覆盖 ${defaultPath}）。AI agent 也会在此撰写文档。',
			'sessions.inspector.notes.noProjectMapping2' => '（无项目映射）',
			'sessions.inspector.notes.clearOverride' => '清除覆盖',
			'sessions.inspector.notes.save' => '保存',
			'sessions.spawnSheet.title' => '新建会话',
			'sessions.spawnSheet.errorRequired' => '需要指定提供商和工作目录',
			'sessions.spawnSheet.errorGeneric' => ({required Object error}) => '创建会话失败：${error}',
			'sessions.spawnSheet.cancel' => '取消',
			'sessions.spawnSheet.spawn' => '创建',
			'sessions.spawnSheet.providerLabel' => '提供商',
			'sessions.spawnSheet.disabledSuffix' => '（已停用）',
			'sessions.spawnSheet.cwdLabel' => '工作目录',
			'sessions.spawnSheet.cwdHint' => '/Users/you/projects/foo',
			'sessions.spawnSheet.cwdHelper' => '网关主机上的绝对路径。',
			'sessions.spawnSheet.browse' => '浏览',
			'sessions.spawnSheet.nameLabel' => '名称（可选）',
			'sessions.spawnSheet.nameHint' => '例如：backend-refactor',
			'sessions.spawnSheet.argsLabel' => '额外参数（可选）',
			'sessions.spawnSheet.argsHint' => '--continue --verbose',
			'sessions.spawnSheet.argsHelper' => '以空格分隔；留空使用提供商默认值。',
			'sessions.spawnSheet.bypass.labelClaude' => '绕过权限',
			'sessions.spawnSheet.bypass.labelCodex' => '跳过批准与沙盒',
			'sessions.spawnSheet.bypass.labelGemini' => 'YOLO 模式',
			'sessions.spawnSheet.bypass.subtitleOn' => '此会话将以提升的自主权运行。',
			'sessions.spawnSheet.bypass.subtitleOff' => '关闭 — 确认和提示按正常方式处理。',
			'sessions.spawnSheet.noProviders.title' => '未配置任何提供商',
			'sessions.spawnSheet.noProviders.message' => '网关没有启用任何 CLI 提供商。在 Web 管理端的「提供商」中配置，或编辑 config.toml 的 [providers] 段，然后点击重新加载。',
			'sessions.spawnSheet.noProviders.reload' => '重新加载',
			'sessions.spawnSheet.providerLoadError.title' => '无法加载提供商',
			'sessions.spawnSheet.providerLoadError.networkError' => '网络错误',
			'sessions.spawnSheet.providerLoadError.serverPrefix' => ({required Object code}) => '服务器 ${code}',
			'sessions.spawnSheet.providerLoadError.format' => ({required Object prefix, required Object message}) => '${prefix}：${message}',
			'sessions.spawnSheet.claudeAccount.label' => 'Claude 账号',
			'sessions.spawnSheet.claudeAccount.helperMulti' => '已配置多个账号 — 为此会话选择一个。',
			'sessions.spawnSheet.claudeAccount.helperSingle' => '选择已配置的账号，或使用默认（环境变量 / 系统）。',
			'sessions.spawnSheet.claudeAccount.kDefault' => '默认（环境变量 / 系统）',
			'sessions.spawnSheet.claudeAccount.disabledSuffix' => '（已停用）',
			'sessions.spawnSheet.claudeAccount.noTokenSuffix' => '（无令牌）',
			'sessions.spawnSheet.claudeAccount.noneHint' => '未配置 Claude 账号 — 网关将使用系统的 ANTHROPIC_API_KEY。在 Web 管理端的「设置 → 账号」中添加账号。',
			'sessions.spawnSheet.claudeAccount.errorHint' => ({required Object error}) => '无法加载 Claude 账号（${error}）。会话将以网关默认配置启动。',
			'mcp.title' => 'MCP',
			'mcp.newServer' => '新建服务器',
			'mcp.addSecret' => '添加密钥',
			'mcp.editConfig' => '编辑配置',
			'mcp.viewRawConfig' => '查看原始配置',
			'mcp.copyId' => '复制 ID',
			'mcp.copiedSnack' => ({required Object id}) => '已复制 ${id}',
			'mcp.deleteServerTitle' => '删除 MCP 服务器？',
			'mcp.deleteSecretTitle' => '删除密钥？',
			'mcp.errorPrefix.delete' => '删除失败',
			'mcp.errorPrefix.add' => '添加失败',
			'mcp.errorPrefix.update' => '更新失败',
			'mcp.errorPrefix.toggle' => '切换失败',
			'mcp.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'mcp.editor.nameHint' => 'my-mcp-server',
			'mcp.editor.jsonHint' => 'JSON 配置 — name、transport: stdio、command、args…',
			'mcp.editor.descriptionPlaceholder' => '可选的一行说明',
			'mcp.editor.validateJsonObject' => '正文必须是 JSON 对象',
			'mcp.editor.validateJsonInvalid' => ({required Object error}) => '无效的 JSON：${error}',
			'mcp.editor.appBarEdit' => '编辑 MCP 服务器',
			'mcp.editor.appBarNew' => '新建 MCP 服务器',
			'mcp.editor.idLockedHint' => '编辑模式下锁定 — 需删除后重建以更改。',
			'mcp.editor.jsonLabel' => '服务器 JSON',
			'mcp.editor.jsonSchemaHelp' => 'Schema：transport 必须是 stdio、http 或 sse。stdio 需要 command + args。http/sse 需要 url + headers。用 \$secret:KEY 引用密钥库的密钥。',
			'mcp.editor.idLabel' => 'id（URL 片段，小写字母数字 / 横线 / 下划线）',
			'mcp.editor.idRequired' => 'id 必填',
			'mcp.editor.saving' => '保存中…',
			'mcp.editor.save' => '保存',
			'mcp.editor.create' => '创建',
			'mcp.secret.keyLabel' => '键',
			'mcp.secret.keyHint' => 'GITHUB_TOKEN、OPENAI_KEY、…',
			'mcp.secret.valueLabel' => '值',
			'mcp.secret.keyRequired' => '必须填写键。',
			'mcp.secret.keyInvalid' => '键必须匹配 [A-Za-z_][A-Za-z0-9_]* — 与 shell 环境变量规则相同。',
			'mcp.secret.valueRequired' => '必须填写值。',
			'mcp.secret.replaceTitle' => '替换密钥值',
			'mcp.secret.addTitle' => '添加密钥',
			'mcp.secret.saveButton' => '保存',
			'mcp.secret.addButton' => '添加',
			'mcp.secret.helpRules' => 'shell 环境变量规则：字母或 _ 开头，仅含字母 / 数字 / _。',
			'mcp.secret.replaceHint' => '粘贴新值（旧值被擦除）',
			'mcp.secret.addHint' => '粘贴密钥值',
			'mcp.secret.addedSnack' => ({required Object key}) => '已添加密钥 ${key}。',
			'mcp.secret.updatedSnack' => ({required Object key}) => '已更新密钥 ${key}。',
			'mcp.secret.deletedSnack' => ({required Object key}) => '已删除 ${key}。',
			'mcp.secret.deleteBody' => '从加密的密钥库中移除该值。引用此密钥的 MCP 服务器在恢复前将无法启动。',
			'mcp.popup.editConfigSubtitle' => '完整 JSON 编辑器 — 仅限密钥库支持的服务器',
			'mcp.popup.viewRawSubtitle' => '服务器 JSON 的只读查看器',
			'mcp.popup.deleteLabel' => '删除',
			'mcp.kv.transport' => '传输',
			'mcp.kv.description' => '描述',
			'mcp.kv.command' => '命令',
			'mcp.kv.args' => '参数',
			'mcp.kv.headers' => 'Headers',
			'mcp.deleteServerBody' => ({required Object id}) => '移除 ${id} 的密钥库目录。引用此服务器的会话将无法启动。',
			'mcp.deleteServerSnack' => ({required Object id}) => '已删除 ${id}。',
			'mcp.serversCount' => ({required Object count}) => '服务器（${count}）',
			'mcp.secretsCount' => ({required Object count}) => '密钥（${count}）',
			'mcp.emptyServers' => '未注册任何 MCP 服务器。点击「新建服务器」添加一个。',
			'mcp.emptySecrets' => '暂无密钥。添加一个，将敏感的 env / headers 注入 MCP 服务器，无需放在 JSON 里。',
			'mcp.noVaultFileYet' => '尚无密钥库文件 — 添加密钥时会创建。',
			'mcp.tapToReplaceHint' => '点击替换 · 长按 / 垃圾桶 删除',
			'mcp.failedToLoad' => '加载 MCP 状态失败',
			'mcp.serverCreatedSnack' => 'MCP 服务器已创建。',
			'mcp.serverUpdatedSnack' => 'MCP 服务器已更新。',
			'mcp.envHeading' => '环境变量',
			'mcp.encryptionAes' => 'AES-GCM 加密（密钥存于 OS keychain）',
			'mcp.encryptionPlaintext' => '明文 — keychain 不可用',
			'mcp.toggleEnabledSnack' => ({required Object name}) => '${name} 已启用。',
			'mcp.toggleDisabledSnack' => ({required Object name}) => '${name} 已停用。',
			'providers.title' => '提供商',
			'providers.configSaved' => '提供商配置已更新。',
			'providers.saveFailedApi' => ({required Object error}) => '保存失败：${error}',
			'providers.saveFailedGeneric' => ({required Object error}) => '保存失败：${error}',
			'providers.reload' => '重新加载',
			'providers.errorPrefix.toggle' => '切换失败',
			'providers.errorPrefix.rename' => '重命名失败',
			'providers.errorPrefix.delete' => '删除失败',
			'providers.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'providers.accounts.rename' => '重命名',
			_ => null,
		} ?? switch (path) {
			'providers.accounts.renameTitle' => ({required Object name}) => '重命名 ${name}',
			'providers.accounts.displayNameLabel' => '显示名',
			'providers.accounts.displayNameHint' => '工作账号',
			'providers.accounts.deleteTitle' => '删除账号？',
			'providers.accounts.importFailedApi' => ({required Object error}) => '导入失败：${error}',
			'providers.accounts.importFailedGeneric' => ({required Object error}) => '导入失败：${error}',
			'providers.accounts.enable' => '启用',
			'providers.accounts.disable' => '停用',
			'providers.accounts.deleteLabel' => '删除',
			'providers.accounts.deleteBody' => '移除该账号及其存储的 OAuth token。已使用此账号的会话保持运行，但重新认证会失败。',
			'providers.accounts.deletedSnack' => ({required Object name}) => '已删除 ${name}。',
			'providers.accounts.importSyncedSnack' => '已同步 — 网关没有新账号。',
			'providers.accounts.importedSnackOne' => ({required Object n}) => '已导入 ${n} 个账号。',
			'providers.accounts.importedSnackOther' => ({required Object n}) => '已导入 ${n} 个账号。',
			'providers.accounts.importing' => '同步中…',
			'providers.accounts.importLocal' => '导入本地',
			'providers.accounts.addHint' => '添加新账号仅可在网关主机上操作。',
			'providers.accounts.addBody' => '新目录会自动出现在这里。OAuth 流程步骤参见文档。',
			'providers.accounts.loadFailed' => ({required Object error}) => '加载账号失败：${error}',
			'providers.accounts.intro' => '以 Claude 提供商启动的会话会从这些账号中选择（或回退到环境变量）。',
			'providers.accounts.enabledSnack' => ({required Object name}) => '${name} 已启用。',
			'providers.accounts.disabledSnack' => ({required Object name}) => '${name} 已停用。',
			'providers.accounts.renamedSnack' => ({required Object name}) => '已重命名为 ${name}。',
			'providers.configFallbackTitle' => '提供商配置',
			'providers.saving' => '保存中…',
			'providers.save' => '保存',
			'providers.configLoadFailed' => '加载提供商失败',
			'providers.argsHelper' => '以空格分隔的 CLI 参数。',
			'providers.listEmptyHeadline' => '未加载任何提供商。',
			'providers.listEmptyBody' => '网关在启动时从插件目录解析提供商。如果有遗漏，请检查日志。',
			'providers.listLoadFailed' => '加载提供商失败',
			'providers.cliSectionHeader' => 'CLI 提供商',
			'providers.enabledSnack' => ({required Object name}) => '${name} 已启用。',
			'providers.disabledSnack' => ({required Object name}) => '${name} 已停用。',
			'integrations.title' => '集成',
			'integrations.register' => '注册',
			'integrations.registerDialogTitle' => '注册集成',
			'integrations.edit' => '编辑',
			'integrations.editTitle' => ({required Object name}) => '编辑 ${name}',
			'integrations.enabledLabel' => '已启用',
			'integrations.iSavedIt' => '我已保存',
			'integrations.apiKeyForName' => ({required Object name}) => '${name} 的 API key',
			'integrations.apiKeySubtitleRegister' => ({required Object routePrefix}) => '将其交给集成方，使其能够通过 /api/v1/${routePrefix}/… 进行认证。',
			'integrations.copiedRequestId' => ({required Object id}) => '已复制 request_id ${id}',
			'integrations.updateOk' => '集成已更新。',
			'integrations.registerFailedApi' => ({required Object error}) => '注册失败：${error}',
			'integrations.registerFailedGeneric' => ({required Object error}) => '注册失败：${error}',
			'integrations.updateFailedApi' => ({required Object error}) => '更新失败：${error}',
			'integrations.updateFailedGeneric' => ({required Object error}) => '更新失败：${error}',
			'integrations.deleteTitle' => '删除集成？',
			'integrations.deletedSnack' => ({required Object name}) => '已删除 ${name}。',
			'integrations.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'integrations.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'integrations.rotateKey' => '轮换密钥',
			'integrations.rotateConfirmTitle' => '轮换 API key？',
			'integrations.rotate' => '轮换',
			'integrations.newApiKeyTitle' => ({required Object name}) => '${name} 的新 API key',
			'integrations.newApiKeySubtitle' => '将其交给集成方。旧密钥已失效。',
			'integrations.rotateFailedApi' => ({required Object error}) => '轮换失败：${error}',
			'integrations.rotateFailedGeneric' => ({required Object error}) => '轮换失败：${error}',
			'integrations.deleteBody' => '移除该注册并吊销 API key。使用旧 key 的进行中请求将开始失败。',
			'integrations.rotateBody' => ({required Object name}) => '为 ${name} 生成新 API key 并立即让旧 key 失效。',
			'integrations.appBarFallback' => '集成',
			'integrations.tooltipMore' => '更多',
			'integrations.tooltipReadOnly' => '系统集成 — 只读',
			'integrations.kvRoutePrefix' => '路由前缀',
			'integrations.kvBaseUrl' => 'Base URL',
			'integrations.kvScopes' => '范围',
			'integrations.kvVersion' => '版本',
			'integrations.kvLastHealthPing' => '最近健康检查',
			'integrations.kvCreated' => '创建于',
			'integrations.kvKeyRotated' => 'Key 轮换于',
			'integrations.detailLoadFailed' => ({required Object error}) => '加载集成失败：${error}',
			'integrations.callsLoadFailed' => '加载调用失败',
			'integrations.noMatchingCalls' => '日志中暂无匹配的调用。',
			'integrations.directionAll' => '全部',
			'integrations.directionInbound' => '入站',
			'integrations.directionOutbound' => '出站',
			'integrations.form.validateRequired' => '名称、Base URL、路由前缀必填。',
			'integrations.form.fieldName' => '名称',
			'integrations.form.fieldNameHint' => 'My Bot',
			'integrations.form.fieldBaseUrl' => 'Base URL',
			'integrations.form.fieldRoutePrefix' => '路由前缀',
			'integrations.form.routePrefixHelper' => '可通过 /api/v1/<前缀>/... 访问',
			'integrations.form.fieldScopes' => '范围（可选）',
			'integrations.form.scopesHelper' => '逗号分隔。留空 = 服务器默认。',
			'integrations.form.fieldVersion' => '版本（可选）',
			'integrations.form.validateBaseUrl' => '必须填写 Base URL。',
			'integrations.form.editFieldScopes' => '范围',
			'integrations.form.editScopesHelper' => '逗号分隔。',
			'integrations.form.editFieldVersion' => '版本',
			'integrations.form.apiKeyWarn' => '此 key 只显示这一次。',
			'integrations.form.copyCopied' => '已复制',
			'integrations.form.copyCopy' => '复制',
			'integrations.emptyState' => '在 Web 管理端注册：集成 → 新建。',
			'integrations.sectionRegistered' => '已注册',
			'integrations.sectionSystem' => '系统',
			'integrations.listLoadFailed' => '加载集成失败',
			'memoryWorkers.title' => '记忆工作器',
			'memoryWorkers.savedSnack' => ({required Object label}) => '${label} 已保存',
			'memoryWorkers.saveFailed' => ({required Object error}) => '保存失败：${error}',
			'memoryWorkers.testFailed' => ({required Object error}) => '测试调用失败：${error}',
			'memoryWorkers.workerLabel' => '工作器',
			'memoryWorkers.summarizerHttp' => '摘要器（HTTP）',
			'memoryWorkers.agentCliPrint' => 'Agent（CLI --print）',
			'memoryWorkers.cliLabel' => 'CLI',
			'memoryWorkers.cliClaude' => 'Claude',
			'memoryWorkers.cliGemini' => 'Gemini',
			'memoryWorkers.claudeAccountLabel' => 'Claude 账号',
			'memoryWorkers.claudeAccountDefault' => '默认',
			'memoryWorkers.test' => '测试',
			'memoryWorkers.intro' => '每个记忆系统的 LLM 触点都可以独立服务 — 由本地 summarizer 端点（LM Studio / OpenAI 兼容）或在 --print 模式下生成无头 Claude / Gemini agent 来处理。高质量叙事任务（gitactivity、transcript）适合 agent 工作器；高频任务（gatekeeper）按设计保留在本地端点上。',
			'memoryWorkers.errorTitle' => '端点不可达',
			'memoryWorkers.errorDetail' => '/api/v1/memory/workers 路由在 M25 中是新增的 — opendray 二进制可能需要重启以挂载这些路由并运行迁移 0029。',
			'memoryWorkers.summarizerOnlyBadge' => '仅 summarizer',
			'memoryWorkers.summarizerInfo' => '使用注册表默认 summarizer 提供商。在 Web 管理端选择具体行。',
			'memoryWorkers.agentWarning' => 'Agent 模式每次调用都会生成无头 CLI。延迟约 5-15 秒（相比 summarizer 约 1 秒）；成本从 CPU 转移到你的 Claude / Gemini 配额。',
			'memoryWorkers.noCalls24h' => '过去 24 小时没有调用。',
			'memoryWorkers.testOkSnack' => ({required Object label, required Object duration}) => '${label} OK — ${duration}ms',
			'memoryWorkers.testFailedReturnedSnack' => ({required Object label, required Object error}) => '${label} 失败：${error}',
			'memoryWorkers.unknownError' => '未知',
			'memoryWorkers.tasks.gatekeeper.label' => '守门员',
			'memoryWorkers.tasks.gatekeeper.description' => '每次 memory_store 写入前的过滤器。高频（目标 <500ms） — 仅 summarizer。',
			'memoryWorkers.tasks.cleaner.label' => '清理馆员',
			'memoryWorkers.tasks.cleaner.description' => '定期 LLM 馆员。判断旧记忆为保留 / 过期 / 重复。',
			'memoryWorkers.tasks.gitactivity.label' => 'Git 活动摘要器',
			'memoryWorkers.tasks.gitactivity.description' => 'git log → 每 24 小时的 2-3 段叙事。天然适合 agent 工作器。',
			'memoryWorkers.tasks.transcript.label' => '会话记录摘要器',
			'memoryWorkers.tasks.transcript.description' => '会话结束时的「agent 做了什么」摘要。天然适合 agent 工作器。',
			'memoryCleanup.title' => '记忆清理',
			'memoryCleanup.approveFailed' => ({required Object error}) => '批准失败：${error}',
			'memoryCleanup.rejectFailed' => ({required Object error}) => '拒绝失败：${error}',
			'memoryCleanup.loadFailed' => ({required Object error}) => '加载失败：${error}',
			'memoryCleanup.reject' => '拒绝',
			'project.title' => '项目',
			'project.pickFirst' => '请先选择一个项目。',
			'project.loadFailed' => ({required Object error}) => '加载失败：${error}',
			'project.projectsLoadFailed' => ({required Object error}) => '加载项目列表失败：${error}',
			'project.projectLabel' => '项目',
			'project.resetTooltip' => '重置项目记忆',
			'project.append' => '追加',
			'project.appendDialogTitle' => '追加日志条目',
			'project.titleFieldLabel' => '标题（可选）',
			'project.contentFieldLabel' => '内容（Markdown）',
			'project.appendFailed' => ({required Object error}) => '失败：${error}',
			'project.approveFailed' => ({required Object error}) => '批准失败：${error}',
			'project.rejectFailed' => ({required Object error}) => '拒绝失败：${error}',
			'project.cleanupFailed' => ({required Object error}) => '清理失败：${error}',
			'project.resetConfirmTitle' => '重置项目记忆？',
			'project.alsoDeleteScanner' => '同时删除扫描器文档',
			'project.alsoDeletePgvector' => '同时删除 pgvector 记忆',
			'project.deleteForever' => '永久删除',
			'project.resetDoneSnack' => ({required Object parts}) => '已重置：${parts}',
			'project.resetFailed' => ({required Object error}) => '重置失败：${error}',
			'project.docSavedSnack' => ({required Object kind}) => '${kind} 已保存',
			'project.docSaveFailed' => ({required Object error}) => '保存失败：${error}',
			'project.docHintTemplate' => ({required Object kind}) => '以 Markdown 编写 ${kind}…',
			'project.deleteEntryTooltip' => '删除条目',
			'project.agentReason' => 'Agent 原因',
			'project.reject' => '拒绝',
			'project.approve' => '批准',
			'project.replaceConfirmTitle' => ({required Object kind}) => '替换当前 ${kind}？',
			'project.replaceKind' => ({required Object kind}) => '替换 ${kind}',
			'project.reason' => '原因',
			'project.willMergeInto' => '将合并到',
			'backups.title' => '备份',
			'backups.runConfirmTitle' => '立即运行备份？',
			'backups.runConfirmBody' => '向本地目标触发一次新的转储。任务在服务端运行；此列表会随进度刷新。',
			'backups.run' => '运行',
			'backups.runNow' => '立即运行',
			'backups.queueing' => '入队中…',
			'backups.queuedSnack' => ({required Object id}) => '备份已入队（${id}）。监控进度中…',
			'backups.runFailedApi' => ({required Object error}) => '运行失败：${error}',
			'backups.runFailedGeneric' => ({required Object error}) => '运行失败：${error}',
			'backups.rowSucceededSnack' => ({required Object bytes}) => '备份成功（${bytes}）。',
			'backups.rowFailedSnack' => ({required Object error}) => '备份失败：${error}',
			'backups.unknownError' => '未知错误',
			'backups.detailTitle' => '备份详情',
			'backups.deleteTitle' => '删除备份？',
			'backups.deleteBody' => ({required Object target}) => '从 ${target} 移除二进制文件，并在索引中标记该行为已删除。',
			'backups.deletedSnack' => ({required Object id}) => '已删除 ${id}。',
			'backups.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'backups.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'backups.menuSchedules' => '计划',
			'backups.menuTargets' => '目标',
			'backups.kv.status' => '状态',
			'backups.kv.target' => '目标',
			'backups.kv.triggeredBy' => '触发者',
			'backups.kv.started' => '开始',
			'backups.kv.finished' => '完成',
			'backups.kv.size' => '大小',
			'backups.kv.encrypted' => '已加密',
			'backups.kv.targetPath' => '目标路径',
			'backups.kv.error' => '错误',
			'backups.kv.yes' => '是',
			'backups.kv.no' => '否',
			'backups.emptyMissingDeps.headline' => '备份暂时无法运行',
			'backups.emptyMissingDeps.body' => '安装 postgresql-client 并重启 opendray。',
			'backups.emptyNoTargets.headline' => '未配置任何备份目标',
			'backups.emptyNoTargets.body' => '打开「更多」菜单 → 目标，添加一个目的地（本地 / S3 / SMB / SFTP / WebDAV / rclone）。然后返回并点击「立即运行」。',
			'backups.emptyNoBackups.headline' => '暂无备份',
			'backups.emptyNoBackups.body' => '点击「立即运行」生成一次新快照，或打开「计划」设置定期运行。',
			'backups.restartToActivate' => '重启 opendray 以激活备份',
			'backups.passphraseSaved' => '你的密语已保存。网关仅在启动时加载，因此更改需重启后才生效。',
			'backups.keyFileLabel' => '密钥文件',
			'backups.configuredViaLabel' => '配置方式',
			'backups.wizard.title' => '设置备份',
			'backups.wizard.intro' => '选择一个主密语。opendray 用它通过 AES-256-GCM 加密每一份备份。丢失密语就丢失数据 — 无法恢复。',
			'backups.wizard.saving' => '保存中…',
			'backups.wizard.generateAndSave' => '生成并保存',
			'backups.wizard.savePassphrase' => '保存密语',
			'backups.wizard.generateHint' => '服务器生成密码学级别随机密语，你复制到密码管理器，然后确认。',
			'backups.wizard.helperRecommended' => '建议：从密码管理器生成 40+ 字符',
			'backups.wizard.saveNowHeader' => '立即保存这个密语',
			'backups.wizard.saveNowBody' => '此处只显示一次。之后无法从 opendray 取回。',
			'backups.statusReady' => '备份就绪',
			'backups.statusCannot' => '备份无法运行',
			'backups.overviewTargets' => '目标',
			'backups.overviewSchedules' => '计划',
			'backups.overviewBackups' => '备份',
			'backups.failedToLoad' => '加载备份失败',
			'backups.envVarConfigured' => 'OPENDRAY_BACKUP_KEY 环境变量',
			'backups.savedConfirmCheckbox' => '我已将密语保存到密码管理器',
			'backups.pgDumpMissing' => 'pg_dump 不在 PATH 中。请安装 postgresql-client 并重启 opendray。',
			'backups.encryption.checkAgain' => '重新检查',
			'backups.encryption.generate' => '生成',
			'backups.encryption.paste' => '粘贴',
			'backups.encryption.random256bit' => '256 位随机密钥',
			'backups.encryption.passphraseLabel' => '你的密语',
			'backups.encryption.passphraseHint' => '至少 20 个字符',
			'backups.encryption.passphraseCopied' => '密语已复制到剪贴板',
			'backupTargets.title' => '备份目标',
			'backupTargets.newTarget' => '新建目标',
			'backupTargets.testConnection' => '测试连接',
			'backupTargets.editConfig' => '编辑配置',
			'backupTargets.viewRawConfig' => '查看原始配置',
			'backupTargets.configDialogTitle' => ({required Object kind}) => '${kind} 配置',
			'backupTargets.deleteTitle' => '删除目标？',
			'backupTargets.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'backupSchedules.title' => '备份计划',
			'backupSchedules.newButton' => '新建',
			'backupSchedules.deleteTitle' => '删除计划？',
			'backupSchedules.targetLabel' => '目标',
			'backupSchedules.intervalLabel' => '间隔',
			'backupSchedules.retentionLabel' => '保留（最近 N 个）',
			'backupSchedules.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'backupSchedules.noTargets' => '未配置任何备份目标。请从 Web 管理端或「目标」屏添加。',
			'backupSchedules.okMsgCreate' => '计划已创建。',
			'backupSchedules.okMsgUpdate' => '计划已更新。',
			'backupSchedules.okMsgDelete' => '计划已删除。',
			'backupSchedules.errorPrefixCreate' => '创建失败',
			'backupSchedules.errorPrefixUpdate' => '更新失败',
			'backupSchedules.errorPrefixDelete' => '删除失败',
			'backupSchedules.deleteBody' => ({required Object targetId}) => '移除目标 ${targetId} 的定期规格。已存在的备份不受影响。',
			'backupSchedules.emptyList' => '暂无计划。\n点击「新建」创建一个。',
			'backupSchedules.validatePickTarget' => '请选择一个目标。',
			'backupSchedules.validateInterval' => '间隔必须大于 0。',
			'backupSchedules.formTitleEdit' => '编辑计划',
			'backupSchedules.formTitleNew' => '新建计划',
			'backupSchedules.saveButtonEdit' => '保存',
			'backupSchedules.saveButtonNew' => '创建',
			'backupSchedules.targetFixedHint' => '目标一旦创建即不可改。',
			'backupSchedules.enabledOn' => '调度器将按周期运行。',
			'backupSchedules.enabledOff' => '已暂停 — 重新启用前不会自动运行。',
			'backupSchedules.loadFailedTitle' => '加载计划失败',
			'backupSchedules.pausedBadge' => '已暂停',
			'backupSchedules.everyInterval' => ({required Object interval}) => '每 ${interval}',
			'backupSchedules.keepRetention' => ({required Object n}) => '· 保留 ${n}',
			'backupSchedules.nextRun' => ({required Object when}) => '· 下次 ${when}',
			'backupSchedules.lastRun' => ({required Object when}) => '· 上次 ${when}',
			'backupTargetEditor.useHttps' => '使用 HTTPS',
			'backupTargetEditor.pathStyle' => '路径风格寻址',
			'backupTargetEditor.pathStyleSubtitle' => '旧版 / MinIO',
			'backupTargetEditor.kinds.local.label' => '本地磁盘',
			'backupTargetEditor.kinds.local.description' => '运行 opendray 的机器上的文件夹',
			'backupTargetEditor.kinds.smb.label' => 'SMB 共享',
			'backupTargetEditor.kinds.smb.description' => 'Windows 共享 + 多数家用 NAS 设备',
			'backupTargetEditor.kinds.webdav.label' => 'WebDAV',
			'backupTargetEditor.kinds.webdav.description' => '自托管云盘 + 文件共享服务',
			'backupTargetEditor.kinds.sftp.label' => 'SFTP',
			'backupTargetEditor.kinds.sftp.description' => '任何可 SSH 访问的服务器',
			'backupTargetEditor.kinds.s3.label' => 'S3 / 兼容',
			'backupTargetEditor.kinds.s3.description' => 'Amazon S3 + S3 兼容存储桶（MinIO、R2、B2）',
			'backupTargetEditor.kinds.rclone.label' => 'rclone（任意）',
			'backupTargetEditor.kinds.rclone.description' => '通过 rclone CLI 访问 OneDrive、Google Drive、Dropbox',
			'backupTargetEditor.formTitleEdit' => '编辑目标',
			'backupTargetEditor.formTitleNew' => '新建备份目标',
			'backupTargetEditor.idHintAuto' => ({required Object prefix}) => '自动：${prefix}-1',
			'backupTargetEditor.idHelper' => '小写字母、数字、连字符。默认为下一个可用槽。',
			'backupTargetEditor.enabledOn' => '定期和临时备份可使用此目标。',
			'backupTargetEditor.enabledOff' => '服务器将拒绝向此处写入备份。',
			'backupTargetEditor.saving' => '保存中…',
			'backupTargetEditor.create' => '创建',
			'backupTargetEditor.rootDirLabel' => '根目录',
			'backupTargetEditor.rootDirHint' => '留空 = cfg.backup.local_dir (~/.opendray/backups)',
			'backupTargetEditor.hostLabel' => '主机',
			'backupTargetEditor.portLabel' => '端口',
			'backupTargetEditor.shareLabel' => '共享',
			'backupTargetEditor.shareHint' => '顶层共享名',
			'backupTargetEditor.shareSampleHint' => 'Claude_Workspace',
			'backupTargetEditor.userLabel' => '用户',
			'backupTargetEditor.passwordLabel' => '密码',
			'backupTargetEditor.passwordHintKeepCurrent' => '留空 = 保留当前值',
			'backupTargetEditor.passwordHintKeep' => '留空 = 保留',
			'backupTargetEditor.pathPrefixLabel' => '路径前缀',
			'backupTargetEditor.pathPrefixHintShareRoot' => '共享根下的子文件夹（可选）',
			'backupTargetEditor.pathPrefixHintBaseUrl' => 'Base URL 下的子文件夹（可选）',
			'backupTargetEditor.pathPrefixHintObjectKey' => '对象键前缀（可选）',
			'backupTargetEditor.pathPrefixHintSshFolder' => '绝对路径或相对用户主目录（可选）',
			'backupTargetEditor.pathPrefixHintRemoteRoot' => '远端根下的子文件夹（可选）',
			'backupTargetEditor.endpointLabel' => '端点',
			'backupTargetEditor.regionLabel' => '区域',
			'backupTargetEditor.bucketLabel' => '存储桶',
			'backupTargetEditor.accessKeyLabel' => 'Access Key',
			'backupTargetEditor.secretKeyLabel' => 'Secret Key',
			'backupTargetEditor.secretKeyHintEdit' => '留空 = 保留当前值。已 AES-256-GCM 加密存储。',
			'backupTargetEditor.secretKeyHintNew' => '已 AES-256-GCM 加密存储；不会回显。',
			'backupTargetEditor.baseUrlLabel' => 'Base URL',
			'backupTargetEditor.baseUrlHint' => '完整 URL 包含路径。Nextcloud：https://cloud.example/remote.php/dav/files/<user>',
			'backupTargetEditor.sftpPasswordHintEdit' => '留空 = 保留。如果密码 + 私钥同时存在，私钥优先。',
			'backupTargetEditor.sftpPasswordHintNew' => '密码或私钥二选一。两者同时存在时，密码仅作回退。',
			'backupTargetEditor.privateKeyLabel' => '私钥（PEM）',
			'backupTargetEditor.privateKeyHintEdit' => '留空 = 保留。粘贴 OpenSSH/PEM 内容。',
			'backupTargetEditor.privateKeyHintNew' => '粘贴 OpenSSH/PEM 私钥内容。多行输入 — 保留 BEGIN/END 标记。',
			'backupTargetEditor.hostKeyLabel' => 'Host key（pinning）',
			'backupTargetEditor.hostKeyHint' => 'OpenSSH 格式的服务器公钥。`ssh-keyscan <host>` 获取。留空 = 不 pinning（局域网外不推荐）。',
			'backupTargetEditor.rcloneNote' => '需要 opendray 主机上安装 rclone CLI。首次需运行 `rclone config` 交互式认证云账户。',
			'backupTargetEditor.rcloneRemoteLabel' => '远端名',
			'backupTargetEditor.rcloneRemoteHint' => '来自 `rclone config` 的名字（不带冒号）。',
			'backupTargetEditor.rcloneBinaryLabel' => '二进制路径',
			'backupTargetEditor.rcloneBinaryHint' => '覆盖 `which rclone`。留空 = PATH 查找。',
			'backupTargetEditor.rcloneConfigLabel' => '配置路径',
			'backupTargetEditor.rcloneConfigHint' => '覆盖 --config。留空 = rclone 默认。',
			'githosts.title' => 'Git 主机',
			'githosts.addHost' => '添加主机',
			'githosts.deleteTitle' => '删除 Git 主机？',
			'githosts.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'githosts.errorPrefix.toggle' => '切换失败',
			'githosts.errorPrefix.delete' => '删除失败',
			'githosts.form.kindLabel' => '类型',
			'githosts.form.hostLabel' => '主机',
			'githosts.form.nameLabel' => '名称',
			'githosts.form.nameHint' => 'work-github、personal-gitlab、…',
			'githosts.form.kinds.github' => 'GitHub',
			'githosts.form.kinds.gitlab' => 'GitLab',
			'githosts.form.kinds.bitbucket' => 'Bitbucket',
			'githosts.form.kinds.gitea' => 'Gitea',
			'githosts.form.kinds.custom' => '自定义',
			'githosts.form.validateHost' => '必须填写主机。',
			'githosts.form.validateName' => '必须填写名称。',
			'githosts.form.snackAdded' => '主机已添加。',
			'githosts.form.snackUpdated' => '主机已更新。',
			'githosts.form.saveFailedApi' => ({required Object error}) => '保存失败：${error}',
			'githosts.form.saveFailedGeneric' => ({required Object error}) => '保存失败：${error}',
			'githosts.form.saving' => '保存中…',
			'githosts.form.save' => '保存',
			'githosts.form.add' => '添加',
			'githosts.form.nameHelper' => '在 PR 列表中显示的名字。',
			'githosts.form.tokenLabelKeep' => 'Token（留空 = 保留现有）',
			'githosts.form.tokenLabel' => 'Token',
			'githosts.form.tokenHintKeep' => '留空 = 保留现有。',
			'githosts.form.tokenHintNew' => '粘贴个人访问令牌。',
			'githosts.form.enabledHelper' => '可供会话用于 PR / 远端查找。',
			'githosts.form.validateTokenRequired' => '添加主机时必须填写 Token。',
			'githosts.form.appBarEdit' => ({required Object name}) => '编辑 ${name}',
			'githosts.form.appBarNew' => '添加 Git 主机',
			'githosts.form.tokenPreviewHint' => ({required Object preview}) => '当前预览：${preview}',
			'githosts.form.tokenPreviewNone' => '（无）',
			'githosts.form.pausedSubtitle' => '已暂停 — 会话跳过此主机。',
			'githosts.deleteBody' => ({required Object host}) => '移除该凭据。试图列出 ${host} 的 PR 的会话将回退到未认证 API。',
			'githosts.deletedSnack' => ({required Object name}) => '已删除 ${name}。',
			'githosts.enabledSnack' => ({required Object name}) => '${name} 已启用。',
			'githosts.disabledSnack' => ({required Object name}) => '${name} 已停用。',
			'githosts.emptyList' => '未配置任何 Git 主机。\n\n添加一个凭据，让网关可以列出你仓库的 pull request。',
			'githosts.failedToLoad' => '加载 Git 主机失败',
			'channels.title' => '通道',
			'channels.kNew' => '新建',
			'channels.sendTest' => '发送测试消息',
			'channels.editConfig' => '编辑配置',
			'channels.editNotifications' => '编辑通知',
			'channels.viewRawConfig' => '查看原始配置',
			'channels.copyChannelId' => '复制通道 ID',
			'channels.copiedSnack' => ({required Object id}) => '已复制 ${id}',
			'channels.createdSnack' => ({required Object kind}) => '已创建 ${kind} 通道。',
			'channels.createFailedApi' => ({required Object error}) => '创建失败：${error}',
			'channels.createFailedGeneric' => ({required Object error}) => '创建失败：${error}',
			'channels.deleteTitle' => '删除通道？',
			'channels.configDialog.title' => ({required Object kind}) => '${kind} 配置',
			'channels.webhookDialog.title' => ({required Object kind}) => '${kind} Webhook URL',
			'channels.webhookDialog.copiedSnack' => '已复制 Webhook URL。',
			'channels.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'channels.notifications.title' => '通知偏好',
			'channels.notifications.notifyOn' => '通知时机',
			'channels.notifications.repeatPolicy' => '重复策略',
			'channels.notifications.cooldownWindow' => '冷却时间',
			'channels.notifications.includeSnippet' => '包含终端片段',
			'channels.notifications.snippetLengthCap' => '片段长度上限',
			'channels.notifications.notifyOnAll' => '所有会话事件。',
			'channels.notifications.notifyOnEmpty' => '未选择事件 — 已静音外发通知。',
			'channels.notifications.snippetHelper' => '在每条通知中嵌入终端最近的内容。',
			'channels.notifications.snippetNoCap' => '无上限',
			'channels.notifications.snippetChars' => ({required Object n}) => '${n} 字符',
			'channels.notifications.updatedSnack' => '通知偏好已更新。',
			'channels.notifications.modes.onceLabel' => '每会话一次',
			'channels.notifications.modes.onceDescription' => '空闲时触发一次，回复或结束前不再触发。',
			'channels.notifications.modes.cooldownLabel' => '时间窗冷却',
			'channels.notifications.modes.cooldownDescription' => '在所选时间窗内抑制重复。',
			'channels.notifications.modes.everyLabel' => '每次事件（嘈杂）',
			'channels.notifications.modes.everyDescription' => '不抑制 — 仅适合低频通道。',
			'channels.popup.enable' => '启用',
			'channels.popup.disable' => '停用',
			'channels.popup.mute' => '静音',
			'channels.popup.unmute' => '取消静音',
			'channels.popup.deleteLabel' => '删除',
			'channels.badges.running' => '运行中',
			'channels.badges.starting' => '启动中…',
			'channels.badges.disabled' => '已停用',
			'channels.badges.muted' => '已静音',
			'channels.capsLabel' => ({required Object list}) => '· 能力：${list}',
			'channels.bridgeWebOnly' => 'Bridge 通道仅 Web 端',
			'channels.bridgeEmptyAdd' => '在 Web 管理端添加：通道 → 新建。',
			'channels.deleteBody' => '停止该通道并移除其配置。仍在传输中的通知会被静默丢弃。',
			'channels.snacks.testDispatched' => '测试消息已发送。',
			'channels.snacks.channelEnabled' => '通道已启用。',
			'channels.snacks.channelDisabled' => '通道已停用。',
			'channels.snacks.channelMuted' => '通道已静音。',
			'channels.snacks.channelUnmuted' => '通道已取消静音。',
			'channels.snacks.configUpdated' => '通道配置已更新。',
			'channels.snacks.channelDeleted' => '通道已删除。',
			'channels.errorPrefix.test' => '测试失败',
			'channels.errorPrefix.toggle' => '切换失败',
			'channels.errorPrefix.muteToggle' => '静音切换失败',
			'channels.errorPrefix.update' => '更新失败',
			'channels.errorPrefix.delete' => '删除失败',
			'channels.failedToLoad' => '加载通道失败',
			'channels.kinds.telegram.description' => '通过 @BotFather 创建机器人。opendray 长轮询 getUpdates 并通过 REST 发送。原生支持按钮和 reply_to_message。',
			'channels.kinds.telegram.botTokenLabel' => '机器人 Token',
			'channels.kinds.telegram.botTokenHint' => '从 @BotFather 获取。存储于通道配置；仅管理员 API 可见。',
			'channels.kinds.telegram.chatIdLabel' => '默认 chat ID',
			'channels.kinds.telegram.chatIdPlaceholder' => '42（可选 — 没有 ReplyCtx 时使用）',
			'channels.kinds.slack.description' => 'Socket Mode — 无需公网 webhook。需要 bot OAuth token（xoxb-）和带 connections:write 的 app-level token（xapp-）。',
			'channels.kinds.slack.botTokenLabel' => 'Bot token（xoxb-…）',
			'channels.kinds.slack.botTokenHint' => 'OAuth & Permissions → Bot User OAuth Token。需要 chat:write。',
			'channels.kinds.slack.appTokenLabel' => 'App-level token（xapp-…）',
			'channels.kinds.slack.appTokenHint' => 'Settings → Basic Information → App-Level Tokens。范围：connections:write。',
			'channels.kinds.slack.channelIdLabel' => '默认 channel ID',
			'channels.kinds.slack.channelIdPlaceholder' => 'C0123ABC456（可选）',
			'channels.kinds.discord.description' => '通过 Discord Developer Portal 创建机器人，启用 MESSAGE CONTENT INTENT。连接 Gateway WS — 无需公网 URL。',
			'channels.kinds.discord.botTokenLabel' => 'Bot token',
			'channels.kinds.discord.botTokenPlaceholder' => '来自 Discord Developer Portal 的 Bot token',
			'channels.kinds.discord.botTokenHint' => 'Application → Bot → Reset Token。邀请机器人时勾选 send_messages + embed_links。',
			'channels.kinds.discord.channelIdLabel' => '默认 channel ID',
			'channels.kinds.discord.channelIdPlaceholder' => '123456789012345678（可选）',
			'channels.kinds.feishu.description' => '应用级凭据。入站走事件订阅 webhook。下方生成公网 webhook URL — 粘贴到飞书开放平台控制台。',
			'channels.kinds.feishu.afterCreateHint' => '在通道卡上打开 webhook URL，粘贴到飞书开放平台 → 事件订阅 → Request URL。',
			'channels.kinds.feishu.appIdLabel' => 'App ID',
			'channels.kinds.feishu.appSecretLabel' => 'App secret',
			'channels.kinds.feishu.appSecretPlaceholder' => '应用凭据 secret',
			'channels.kinds.feishu.verificationTokenLabel' => 'Verification token',
			'channels.kinds.feishu.verificationTokenHint' => '来自 事件订阅 → Verification Token。设置后，opendray 拒绝 token 不匹配的 webhook。',
			'channels.kinds.feishu.chatIdLabel' => '默认 chat ID（oc_…）',
			'channels.kinds.feishu.chatIdPlaceholder' => 'oc_xxxxxxxxxx（可选）',
			'channels.kinds.dingtalk.description' => '自定义群机器人。仅外发。群聊 → 机器人 → 添加 → 加签模式 → 复制 webhook + secret。',
			'channels.kinds.dingtalk.webhookUrlLabel' => 'Webhook URL',
			'channels.kinds.dingtalk.secretLabel' => '加签 secret',
			'channels.kinds.dingtalk.secretHint' => '当机器人为「加签」安全模式时，将 secret 复制到这里。opendray 自动添加 timestamp + sign 参数。',
			'channels.kinds.wecom.description' => '群机器人 webhook。仅外发（文本 + markdown）。群设置 → 群机器人 → 添加 → 复制 webhook URL。',
			'channels.kinds.wecom.webhookKeyLabel' => 'Webhook key',
			'channels.kinds.wecom.webhookKeyPlaceholder' => '「key=」查询参数值',
			'channels.kinds.wecom.webhookKeyHint' => '或将整个 webhook URL 粘贴到下方字段 — 任一即可。',
			'channels.kinds.wecom.webhookUrlLabel' => '或完整 webhook URL',
			'channels.kinds.wechat.description' => '通过 WxPusher 推送到个人微信。仅外发 — 推送服务不转发用户回复。每个接收方需通过二维码订阅一次。',
			'channels.kinds.wechat.appTokenLabel' => 'App token（AT_…）',
			'channels.kinds.wechat.appTokenHint' => 'WxPusher → 应用管理 → 创建应用 → 复制 App Token。',
			'channels.kinds.wechat.uidsLabel' => '接收方 UID（每行一个）',
			'channels.kinds.wechat.uidsHint' => 'UID 或 topic ID 至少需要一个。',
			'channels.kinds.wechat.topicIdsLabel' => 'Topic ID（每行一个）',
			'channels.kinds.wechat.urlLabel' => '点击跳转 URL',
			'channels.kinds.wechat.urlHint' => '设置后，点击微信通知会打开此页面。',
			'onboarding.gatewayLabel' => '网关 URL',
			'onboarding.gatewayHint' => 'https://opendray.example.com',
			'onboarding.kContinue' => '继续',
			'skills.title' => '技能',
			'skills.newSkill' => '新建技能',
			'skills.customizingBuiltin' => ({required Object id}) => '自定义内置 ${id}',
			'skills.idLabel' => 'Id（slug）',
			'skills.idHint' => '例如：tdd-guide',
			'skills.bodyLabel' => '正文（Markdown）',
			'skills.loadFailedApi' => ({required Object error}) => '加载失败：${error}',
			'skills.loadFailedGeneric' => ({required Object error}) => '加载失败：${error}',
			'skills.idRequired' => '必须填写 Id。',
			'skills.bodyRequired' => '正文不能为空。',
			'skills.snackCreated' => '技能已创建。',
			'skills.snackOverride' => '已保存为库覆盖。',
			'skills.snackUpdated' => '技能已更新。',
			'skills.saveFailedApi' => ({required Object error}) => '保存失败：${error}',
			'skills.saveFailedGeneric' => ({required Object error}) => '保存失败：${error}',
			'skills.resetTitle' => '重置为内置？',
			'skills.deleteTitle' => '删除技能？',
			'skills.resetBody' => ({required Object id}) => '移除 ${id} 的库覆盖。会话将回退到内置正文。',
			'skills.resetButton' => '重置',
			'skills.resetSnack' => ({required Object id}) => '已将 ${id} 重置为内置。',
			'skills.deletedSnack' => ({required Object id}) => '已删除 ${id}。',
			'skills.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'skills.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'skills.deleteBody' => ({required Object id}) => '从库中移除 ${id}。引用它的会话在恢复前会失败。',
			'skills.newSkillTitle' => '新建技能',
			'skills.customizeTitle' => ({required Object id}) => '自定义 ${id}',
			'skills.editTitle' => ({required Object id}) => '编辑 ${id}',
			'skills.resetTooltip' => '重置为内置',
			'skills.deleteTooltip' => '删除',
			'skills.saving' => '保存中…',
			_ => null,
		} ?? switch (path) {
			'skills.saveOverride' => '保存覆盖',
			'skills.overrideBanner' => '保存会以相同 id 创建一个库覆盖。会话将使用此正文而非内置版本，直到你重置。',
			'skills.idHelper' => '小写字母 / 数字 / 横线。创建后锁定。',
			'skills.emptyList' => '未配置任何技能。网关附带内置技能（planner、code-reviewer 等）。',
			'skills.failedToLoad' => '加载技能失败',
			'customTasks.title' => '自定义任务',
			'customTasks.newTask' => '新建任务',
			'customTasks.deleteTitle' => '删除任务？',
			'customTasks.deletedSnack' => ({required Object name}) => '已删除 ${name}。',
			'customTasks.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'customTasks.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'customTasks.popupEdit' => '编辑',
			'customTasks.popupDelete' => '删除',
			'customTasks.nameHint' => '例如：backend-tests',
			'customTasks.commandHint' => '/run pnpm test --filter backend',
			'customTasks.descriptionHint' => '在任务名下方显示的一行说明。',
			'customTasks.scopeGlobal' => '全局',
			'customTasks.scopeProject' => '项目',
			'customTasks.cwdHint' => '/Users/you/projects/backend',
			'customTasks.snackCreated' => '任务已创建。',
			'customTasks.snackUpdated' => '任务已更新。',
			'customTasks.deleteBody' => '从目录中移除任务。已插入到会话中的实例不受影响。',
			'customTasks.introBanner' => '定义自己的斜杠命令。它们会与内置任务一起出现在会话任务选择器中。',
			'customTasks.validateNameRequired' => '必须填写名称',
			'customTasks.validateCommandRequired' => '必须填写命令',
			'customTasks.validateProjectCwd' => '项目范围任务需要绝对 cwd 路径',
			'customTasks.appBarEdit' => '编辑自定义任务',
			'customTasks.appBarNew' => '新建自定义任务',
			'customTasks.fieldName' => '名称',
			'customTasks.nameHelper' => '在检查器的任务选择器中显示。',
			'customTasks.fieldCommand' => '命令',
			'customTasks.commandHelper' => '选择时插入到会话的文本。可以是 CLI 命令或 Claude 斜杠命令。',
			'customTasks.fieldDescription' => '描述（可选）',
			'customTasks.fieldScope' => '范围',
			'customTasks.globalScopeHint' => '从任何会话可见，不论 cwd。',
			'customTasks.projectScopeHint' => '仅当会话的 cwd 匹配以下路径时可见。',
			'customTasks.fieldProjectCwd' => '项目 cwd',
			'customTasks.cwdHelper' => '绝对路径。以此 cwd 启动的会话将看到该任务。',
			'customTasks.saving' => '保存中…',
			'customTasks.save' => '保存',
			'customTasks.create' => '创建',
			'customTasks.failedToLoad' => '加载自定义任务失败',
			'notesPage.title' => '笔记',
			'notesPage.newButton' => '新建',
			'notesPage.newNoteDialogTitle' => '新建笔记',
			'notesPage.searchHint' => '搜索整个仓库…',
			'notesPage.up' => '上级',
			'notesPage.copyPath' => '复制路径',
			'notesPage.open' => '打开',
			'notesPage.copiedSnack' => ({required Object path}) => '已复制 ${path}',
			'notesPage.deleteTitle' => '删除笔记？',
			'notesPage.deletedSnack' => ({required Object path}) => '已删除 ${path}',
			'notesPage.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'notesPage.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'notesPage.createFailedApi' => ({required Object error}) => '创建失败：${error}',
			'notesPage.createFailedGeneric' => ({required Object error}) => '创建失败：${error}',
			'notesPage.pathLabel' => '相对仓库的路径',
			'notesPage.pathHint' => 'personal/scratch.md',
			'notesPage.create' => '创建',
			'notesPage.popupDelete' => '删除',
			'notesPage.deleteBody' => '此操作不可撤销。仓库的 git 同步会同时移除网关主机上的文件。',
			'notesPage.emptyFilterMatch' => ({required Object query}) => '未找到匹配「${query}」的笔记。',
			'notesPage.emptyVault' => '仓库为空。点击 + 创建第一条笔记。',
			'notesPage.emptyFolder' => ({required Object path}) => '文件夹「${path}」为空。',
			'notesPage.validatePath' => '必须填写路径',
			'notesPage.validatePathDots' => '路径不能包含「..」',
			'notesPage.pathHelper' => '缺失时自动追加 .md。',
			'notesPage.editor.markdownHint' => 'Markdown…',
			'notesPage.editor.saving' => '保存中…',
			'notesPage.editor.autosave' => '随输入自动保存',
			'notesPage.editor.loadFailedApi' => ({required Object error}) => '加载失败：${error}',
			'notesPage.editor.loadFailedGeneric' => ({required Object error}) => '加载失败：${error}',
			'notesPage.editor.saveFailedApi' => ({required Object error}) => '保存失败：${error}',
			'notesPage.editor.saveFailedGeneric' => ({required Object error}) => '保存失败：${error}',
			'notesPage.editor.savedAt' => ({required Object time}) => '${time} 已保存',
			'memory.title' => '记忆',
			'memory.more' => '更多',
			'memory.workers' => '记忆工作器',
			'memory.kNew' => '新建',
			'memory.searchHint' => '搜索…',
			'memory.projectLabel' => '项目',
			'memory.filterHint' => '按名称或路径筛选…',
			'memory.copied' => '已复制',
			'memory.copyTooltip' => '复制文本',
			'memory.deleteAllConfirm.title' => '删除此范围内所有记忆？',
			'memory.deleteAllConfirm.deleteAll' => '全部删除',
			'memory.deletedSnackOne' => ({required Object n}) => '已删除 ${n} 条记忆',
			'memory.deletedSnackOther' => ({required Object n}) => '已删除 ${n} 条记忆',
			'memory.bulkDeleteFailedApi' => ({required Object error}) => '批量删除失败：${error}',
			'memory.bulkDeleteFailedGeneric' => ({required Object error}) => '批量删除失败：${error}',
			'memory.deleteOne.title' => '删除该记忆？',
			'memory.deleteOne.body' => '此操作不可撤销。',
			'memory.scope.project' => '项目',
			'memory.scope.global' => '全局',
			'memory.create.textLabel' => '文本',
			'memory.create.scopeKeyLabel' => '范围键（项目 cwd）',
			'memory.create.scopeKeyHint' => '/Users/you/projects/foo',
			'memory.create.submit' => '创建',
			'about.title' => '关于',
			'about.loading' => '加载中…',
			'about.sections.app' => '应用',
			'about.sections.server' => '服务器',
			'about.fields.app' => '应用',
			'about.fields.version' => '版本',
			'about.fields.versionFormat' => ({required Object version, required Object build}) => '${version}（build ${build}）',
			'about.fields.package' => '包名',
			'about.fields.url' => 'URL',
			'about.fields.signedInAs' => '登录账号',
			'about.fields.tokenExpires' => '令牌到期',
			'about.copied' => ({required Object label}) => '已复制 ${label}',
			'about.copyTooltip' => '复制',
			'about.copyLabels.version' => '版本',
			'about.copyLabels.serverUrl' => '服务器 URL',
			'about.tagline' => 'opendray mobile — 多 CLI 网关控制。\n源码：github.com/Opendray/opendray_v2',
			'settings.title' => '设置',
			'settings.language.section' => '语言',
			'settings.language.system' => '跟随系统',
			'settings.language.systemSubtitle' => '跟随手机的语言设置',
			'settings.language.english' => 'English',
			'settings.language.chinese' => '中文',
			'settings.appearance.section' => '外观',
			'settings.appearance.system' => '跟随系统',
			'settings.appearance.systemSubtitle' => '跟随手机的外观设置',
			'settings.appearance.light' => '浅色',
			'settings.appearance.lightSubtitle' => '始终使用浅色主题',
			'settings.appearance.dark' => '深色',
			'settings.appearance.darkSubtitle' => '始终使用深色主题',
			'settings.account.section' => '账户',
			'settings.account.changeCredentials' => '修改凭据',
			'settings.account.changeCredentialsSubtitle' => '用户名和密码',
			'settings.gateway.section' => '网关',
			'settings.gateway.serverSettings' => '服务器设置',
			'settings.gateway.serverSettingsSubtitle' => '监听地址、日志、凭据库、内存、存储路径…',
			'settings.gateway.liveLogs' => '实时日志',
			'settings.gateway.liveLogsSubtitle' => '查看网关实时日志 — 与 Web 管理端同源',
			'settings.changeCredentials.title' => '修改凭据',
			'settings.changeCredentials.explanation' => '验证当前密码，然后选择新凭据。其他已登录会话将全部失效。',
			'settings.changeCredentials.currentPassword' => '当前密码',
			'settings.changeCredentials.newUsername' => '新用户名',
			'settings.changeCredentials.newPassword' => '新密码',
			'settings.changeCredentials.confirmPassword' => '确认新密码',
			'settings.changeCredentials.validatorRequired' => '必填',
			'settings.changeCredentials.passwordHelper' => '至少 8 个字符',
			'settings.changeCredentials.passwordTooShort' => '至少需要 8 个字符',
			'settings.changeCredentials.passwordMismatch' => '与新密码不一致',
			'settings.changeCredentials.updatedSnack' => '凭据已更新。',
			'settings.changeCredentials.wrongCurrent' => '当前密码不正确。',
			'settings.changeCredentials.saving' => '保存中…',
			'settings.changeCredentials.update' => '更新',
			'settings.logViewer.title' => '实时日志',
			'settings.logViewer.reconnect' => '重新连接',
			'settings.logViewer.copyBuffer' => '复制缓冲',
			'settings.logViewer.clearLocal' => '清除本地视图',
			'settings.logViewer.copiedSnack' => '已将缓冲复制到剪贴板',
			'settings.logViewer.filterHint' => '筛选子串…',
			'settings.logViewer.levels.all' => '全部',
			'settings.logViewer.levels.debug' => '调试',
			'settings.logViewer.levels.info' => '信息',
			'settings.logViewer.levels.warn' => '警告',
			'settings.logViewer.levels.error' => '错误',
			'settings.serverSettings.title' => '服务器设置',
			'settings.serverSettings.reloadTooltip' => '从服务器重新加载',
			'settings.serverSettings.restartTooltip' => '重启网关',
			'settings.serverSettings.restartConfirmTitle' => '重启 opendray？',
			'settings.serverSettings.restartConfirmBody' => '网关将自我 exec。手机应用可能短暂断开连接。',
			'settings.serverSettings.restart' => '重启',
			'settings.serverSettings.restartQueuedSnack' => '已请求重启。稍后下拉刷新。',
			'settings.serverSettings.restartFailedApi' => ({required Object error}) => '重启失败：${error}',
			'settings.serverSettings.restartFailedGeneric' => ({required Object error}) => '重启失败：${error}',
			'settings.serverSettings.loadedFrom' => ({required Object path}) => '加载自：${path}',
			'settings.serverSettings.restartHint' => '大部分配置需要重启网关后生效。重启按钮在 AppBar 中。',
			'settings.serverSettings.savedNeedsRestart' => '已保存。重启网关以生效。',
			'settings.serverSettings.savedSimple' => '已保存。',
			'settings.serverSettings.changesNeedRestart' => '此配置的修改需重启网关。',
			'settings.serverSettings.loadFailed' => '加载服务器设置失败',
			'settings.serverSettings.sections.general' => '通用',
			'settings.serverSettings.sections.logging' => '日志',
			'settings.serverSettings.sections.sessions' => '会话',
			'settings.serverSettings.sections.vault' => '凭据库',
			'settings.serverSettings.sections.mcpRegistry' => 'MCP 注册表',
			'settings.serverSettings.sections.memory' => '记忆',
			'settings.serverSettings.sections.backup' => '备份',
			'settings.serverSettings.sections.storageClaude' => '存储 · Claude',
			'settings.serverSettings.sections.storageCodex' => '存储 · Codex',
			'settings.serverSettings.sections.storageGemini' => '存储 · Gemini',
			'settings.serverSettings.sectionDescriptions.general' => '监听地址、管理员账号、令牌 TTL。',
			'settings.serverSettings.sectionDescriptions.logging' => '详细程度、格式、磁盘日志路径。',
			'settings.serverSettings.sectionDescriptions.sessions' => '空闲检测阈值。',
			'settings.serverSettings.sectionDescriptions.vault' => '笔记、技能、git 版本化的根目录。',
			'settings.serverSettings.sectionDescriptions.mcpRegistry' => 'MCP 服务器 + 密钥文件的凭据库路径。',
			'settings.serverSettings.sectionDescriptions.memory' => '跨 CLI 的持久记忆子系统。',
			'settings.serverSettings.sectionDescriptions.backup' => '加密的数据库备份 + 管理数据导出。密语保存在密钥文件（设置 → 备份）。',
			'settings.serverSettings.sectionDescriptions.storageClaude' => 'Claude 会话记录在磁盘的存放位置。',
			'settings.serverSettings.sectionDescriptions.storageCodex' => 'Codex 会话根目录。',
			'settings.serverSettings.sectionDescriptions.storageGemini' => '按项目的临时目录 + projects.json 路径。',
			'settings.serverSettings.fields.listenAddress' => '监听地址',
			'settings.serverSettings.fields.adminUser' => '管理员用户',
			'settings.serverSettings.fields.adminUserHelper' => '当未设置密钥文件或环境变量时生效。否则参见 设置 → 账户。',
			'settings.serverSettings.fields.adminPassword' => '管理员密码',
			'settings.serverSettings.fields.adminPasswordHelper' => '留空 = 保留。日常轮换请用 设置 → 账户（密钥文件支持，无需重启）。',
			'settings.serverSettings.fields.tokenTtlWeb' => '令牌 TTL（Web）',
			'settings.serverSettings.fields.tokenTtlHelper' => 'Go duration 字符串，如 24h、30m。',
			'settings.serverSettings.fields.level' => '级别',
			'settings.serverSettings.fields.format' => '格式',
			'settings.serverSettings.fields.filePath' => '文件路径',
			'settings.serverSettings.fields.filePathHelper' => '留空 = 仅 stdout。',
			'settings.serverSettings.fields.idleThreshold' => '空闲阈值',
			'settings.serverSettings.fields.idleThresholdHelper' => '会话被标记为空闲前的安静时长。Go duration。',
			'settings.serverSettings.fields.idleCheckInterval' => '空闲检查间隔',
			'settings.serverSettings.fields.idleCheckHelper' => '空闲回收器运行的频率。',
			'settings.serverSettings.fields.root' => '根目录',
			'settings.serverSettings.fields.rootHelper' => 'notes / skills / git_root 子路径的父目录。',
			'settings.serverSettings.fields.notesPath' => '笔记路径',
			'settings.serverSettings.fields.skillsPath' => '技能路径',
			'settings.serverSettings.fields.gitRoot' => 'Git 根',
			'settings.serverSettings.fields.personalPrefix' => '个人前缀',
			'settings.serverSettings.fields.projectsPrefix' => '项目前缀',
			'settings.serverSettings.fields.registryRoot' => '注册表根',
			'settings.serverSettings.fields.secretsFile' => '密钥文件',
			'settings.serverSettings.fields.backend' => '后端',
			'settings.serverSettings.fields.store' => '存储',
			'settings.serverSettings.fields.defaultTopK' => '默认 top-k',
			'settings.serverSettings.fields.similarityThreshold' => '相似度阈值',
			'settings.serverSettings.fields.defaultScope' => '默认范围',
			'settings.serverSettings.fields.chromemHelper' => '当 store=chromem 时。',
			'settings.serverSettings.fields.preserveHelper' => '留空 = 保留当前值。',
			'settings.serverSettings.fields.localModelName' => '本地模型名',
			'settings.serverSettings.fields.localLibraryPath' => '本地库路径',
			'settings.serverSettings.fields.localModelPath' => '本地模型路径',
			'settings.serverSettings.fields.localTokenizerPath' => '本地分词器路径',
			'settings.serverSettings.fields.localMaxSeqLen' => '本地最大序列长度',
			'settings.serverSettings.fields.backupEnabled' => '已启用',
			'settings.serverSettings.fields.backupEnabledHelper' => '即使打开此项，备份子系统仍需配置 OPENDRAY_BACKUP_KEY 或密钥文件才会运行。',
			'settings.serverSettings.fields.backupLocalDir' => '本地目录',
			'settings.serverSettings.fields.backupExportDir' => '导出目录',
			'settings.serverSettings.fields.pathHelper' => '留空 = 启动时从 PATH 解析。',
			'settings.serverSettings.fields.accountsDir' => '账号目录',
			'settings.serverSettings.fields.accountsHelper' => '各账号 .claude/ 子目录的父目录。留空 = ~/.claude-accounts。',
			'settings.serverSettings.fields.sessionsRoot' => '会话根目录',
			'settings.serverSettings.fields.sessionsRootHelper' => '留空 = ~/.codex/sessions。',
			'settings.serverSettings.fields.listenHelper' => '网关绑定的 host:port。需重启。',
			'settings.serverSettings.fields.secretsHelper' => 'AES-256-GCM 加密的密钥库。',
			'settings.serverSettings.fields.backendHelper' => 'auto 选择最佳可用；local 需要 ONNX。',
			'settings.serverSettings.fields.similarityHelper' => '0.0–1.0；低于此值的结果会被过滤。',
			'settings.serverSettings.fields.chromemPath' => 'chromem 路径',
			'settings.serverSettings.fields.httpBaseUrl' => 'HTTP base URL',
			'settings.serverSettings.fields.httpModel' => 'HTTP model',
			'settings.serverSettings.fields.httpApiKey' => 'HTTP api key',
			'settings.serverSettings.fields.httpDimensions' => 'HTTP dimensions',
			'settings.serverSettings.fields.pgDumpPath' => 'pg_dump 路径',
			'settings.serverSettings.fields.pgRestorePath' => 'pg_restore 路径',
			'settings.serverSettings.fields.tmpRoot' => '临时根目录',
			'settings.serverSettings.fields.projectsJson' => 'projects.json',
			'settings.serverSettings.validateInteger' => ({required Object field}) => '「${field}」必须是整数',
			'settings.serverSettings.validateNumber' => ({required Object field}) => '「${field}」必须是数字',
			_ => null,
		};
	}
}
