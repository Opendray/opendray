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
}

// Path: mcp.secret
class _TranslationsMcpSecretZh extends TranslationsMcpSecretEn {
	_TranslationsMcpSecretZh._(TranslationsZh root) : this._root = root, super.internal(root);

	final TranslationsZh _root; // ignore: unused_field

	// Translations
	@override String get keyLabel => '键';
	@override String get keyHint => 'GITHUB_TOKEN、OPENAI_KEY、…';
	@override String get valueLabel => '值';
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
			'mcp.secret.keyLabel' => '键',
			'mcp.secret.keyHint' => 'GITHUB_TOKEN、OPENAI_KEY、…',
			'mcp.secret.valueLabel' => '值',
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
			'skills.title' => '技能',
			'skills.newSkill' => '新建技能',
			'skills.customizingBuiltin' => ({required Object id}) => '自定义内置 ${id}',
			'skills.idLabel' => 'Id（slug）',
			'skills.idHint' => '例如：tdd-guide',
			'skills.bodyLabel' => '正文（Markdown）',
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
			_ => null,
		};
	}
}
