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
	@override String get errorRequired => '请输入用户名和密码';
	@override String errorGeneric({required Object error}) => '登录失败：${error}';
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
	@override String get run => '运行';
	@override String queuedSnack({required Object id}) => '备份已入队（${id}）。监控进度中…';
	@override String runFailedApi({required Object error}) => '运行失败：${error}';
	@override String runFailedGeneric({required Object error}) => '运行失败：${error}';
	@override String get detailTitle => '备份详情';
	@override String get deleteTitle => '删除备份？';
	@override String deletedSnack({required Object id}) => '已删除 ${id}。';
	@override String deleteFailedApi({required Object error}) => '删除失败：${error}';
	@override String deleteFailedGeneric({required Object error}) => '删除失败：${error}';
	@override String get menuSchedules => '计划';
	@override String get menuTargets => '目标';
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
	@override String get deleteTitle => '删除计划？';
	@override String get targetLabel => '目标';
	@override String get intervalLabel => '间隔';
	@override String get retentionLabel => '保留（最近 N 个）';
	@override String errorWithMessage({required Object prefix, required Object error}) => '${prefix}：${error}';
}

// Path: backupTargetEditor
class _TranslationsBackupTargetEditorZh extends TranslationsBackupTargetEditorEn {
	_TranslationsBackupTargetEditorZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get useHttps => '使用 HTTPS';
	@override String get pathStyle => '路径风格寻址';
	@override String get pathStyleSubtitle => '旧版 / MinIO';
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
	@override String get restart => '重启';
	@override String get restartQueuedSnack => '已请求重启。稍后下拉刷新。';
	@override String restartFailedApi({required Object error}) => '重启失败：${error}';
	@override String restartFailedGeneric({required Object error}) => '重启失败：${error}';
	@override late final _TranslationsSettingsServerSettingsSectionsZh sections = _TranslationsSettingsServerSettingsSectionsZh._(_root);
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
}

// Path: sessions.spawnSheet.bypass
class _TranslationsSessionsSpawnSheetBypassZh extends TranslationsSessionsSpawnSheetBypassEn {
	_TranslationsSessionsSpawnSheetBypassZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get labelClaude => '绕过权限';
	@override String get labelCodex => '自动批准（从不询问）';
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
			'auth.errorRequired' => '请输入用户名和密码',
			'auth.errorGeneric' => ({required Object error}) => '登录失败：${error}',
			'nav.sessions' => '会话',
			'nav.memory' => '记忆',
			'nav.notes' => '笔记',
			'nav.more' => '更多',
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
			'sessions.spawnSheet.bypass.labelCodex' => '自动批准（从不询问）',
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
			'backups.run' => '运行',
			'backups.queuedSnack' => ({required Object id}) => '备份已入队（${id}）。监控进度中…',
			'backups.runFailedApi' => ({required Object error}) => '运行失败：${error}',
			'backups.runFailedGeneric' => ({required Object error}) => '运行失败：${error}',
			'backups.detailTitle' => '备份详情',
			'backups.deleteTitle' => '删除备份？',
			'backups.deletedSnack' => ({required Object id}) => '已删除 ${id}。',
			'backups.deleteFailedApi' => ({required Object error}) => '删除失败：${error}',
			'backups.deleteFailedGeneric' => ({required Object error}) => '删除失败：${error}',
			'backups.menuSchedules' => '计划',
			'backups.menuTargets' => '目标',
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
			'backupSchedules.deleteTitle' => '删除计划？',
			'backupSchedules.targetLabel' => '目标',
			'backupSchedules.intervalLabel' => '间隔',
			'backupSchedules.retentionLabel' => '保留（最近 N 个）',
			'backupSchedules.errorWithMessage' => ({required Object prefix, required Object error}) => '${prefix}：${error}',
			'backupTargetEditor.useHttps' => '使用 HTTPS',
			'backupTargetEditor.pathStyle' => '路径风格寻址',
			'backupTargetEditor.pathStyleSubtitle' => '旧版 / MinIO',
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
			_ => null,
		} ?? switch (path) {
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
			'notesPage.editor.markdownHint' => 'Markdown…',
			'notesPage.editor.saving' => '保存中…',
			'notesPage.editor.autosave' => '随输入自动保存',
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
			'settings.serverSettings.restart' => '重启',
			'settings.serverSettings.restartQueuedSnack' => '已请求重启。稍后下拉刷新。',
			'settings.serverSettings.restartFailedApi' => ({required Object error}) => '重启失败：${error}',
			'settings.serverSettings.restartFailedGeneric' => ({required Object error}) => '重启失败：${error}',
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
			_ => null,
		};
	}
}
