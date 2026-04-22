import 'package:flutter/widgets.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Lightweight, notifier-based i18n for the mobile app.
///
/// Design choices:
///   • String keys ARE the English source text. Cheap to read in code; if a
///     key is missing from a language catalog we fall back to the key itself
///     (i.e. English), so a typo or new untranslated string is never blank.
///   • Catalogs compiled into the binary — no asset load step, no flicker.
///   • ChangeNotifier → `context.watch<L10n>()` in a widget rebuilds it on
///     language switch; no hot-restart needed.
///
/// Supported languages:
///   • `en` — English (the canonical source; no catalog lookup needed)
///   • `zh` — 简体中文
class L10n extends ChangeNotifier {
  static const _prefsKey = 'opendray.lang';

  String _code;
  L10n(this._code);

  /// Current language code (`en` / `zh`).
  String get code => _code;

  /// Human label for a language code — used in the settings picker.
  static const List<({String code, String name, String flag})> languages = [
    (code: 'en', name: 'English',  flag: 'EN'),
    (code: 'zh', name: '简体中文', flag: '中'),
  ];

  /// Loads persisted choice. Falls back to English.
  static Future<L10n> load() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString(_prefsKey);
    final initial = (saved != null && _catalogs.containsKey(saved)) ? saved : 'en';
    return L10n(initial);
  }

  Future<void> setLanguage(String code) async {
    if (code == _code) return;
    if (code != 'en' && !_catalogs.containsKey(code)) return;
    _code = code;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsKey, code);
    notifyListeners();
  }

  /// Translate [key] to the current language. Returns the key itself (the
  /// English source) when no translation is available — so any untranslated
  /// string degrades to readable English instead of a blank or error token.
  ///
  /// In debug builds, a missing key is logged once via [debugPrint] so we
  /// notice a typo or a forgotten translation when adding new UI. The
  /// release build stays silent — degrading to English is intentional,
  /// not an error.
  String t(String key) {
    if (_code == 'en') return key;
    final catalog = _catalogs[_code];
    if (catalog == null) return key;
    final v = catalog[key];
    if (v == null) {
      assert(() {
        if (_reportedMissing.add('$_code::$key')) {
          debugPrint('[l10n] missing $_code translation: "$key"');
        }
        return true;
      }());
      return key;
    }
    return v;
  }

  static final Set<String> _reportedMissing = <String>{};

  /// Picks a per-field i18n overlay shipped by a plugin manifest.
  ///
  /// Companion to [t]: `t` looks up keys in the app's own catalogs
  /// (for shell chrome we translate ourselves). `pick` consumes
  /// overlays the plugin author wrote into its manifest
  /// (`displayName_zh`, `label_zh`, etc.). The two can't share a
  /// single entry point — manifest strings aren't guessable keys;
  /// they're free-form text owned by the plugin.
  ///
  /// Fallback: when [zh] is null/empty, or the current locale isn't
  /// zh, the caller gets [en] unchanged.
  String pick(String en, String? zh) {
    if (_code == 'zh' && zh != null && zh.isNotEmpty) return zh;
    return en;
  }

  // ── Catalogs ────────────────────────────────────────────────────

  static final Map<String, Map<String, String>> _catalogs = {
    'zh': _zh,
  };

  // ── Catalog contents ────────────────────────────────────────────
  //
  // Keys are English source strings rendered by the Flutter app itself
  // (nav chrome, dashboard, built-in panel pages). Plugin-owned text —
  // displayName, description, configSchema labels/descriptions — no
  // longer lives here: every plugin manifest ships optional `*_zh`
  // overlays which the UI resolves via `context.pickL10n`. See
  // `plugin/manifest.go` Provider.DisplayNameZh and ConfigField.LabelZh.
  //
  // If a plugin-facing string shows up only in English, fix it in
  // THAT plugin's manifest.json — not here.
  static const Map<String, String> _zh = {
    // Bottom navigation
    'Sessions': '会话',
    'Settings': '设置',
    'Running': '运行中',
    'No running plugins': '暂无运行中插件',
    'Open a plugin to see it here.': '打开一个插件后会显示在这里。',

    // Dashboard / sessions
    'New':              '新建',
    'sessions':         '个会话',
    'No sessions':      '暂无会话',
    'Create a session to start': '创建会话以开始',
    'Create Session':   '创建会话',
    'Cannot connect to server': '无法连接到服务器',
    'Check Settings → Server URL': '检查 设置 → 服务器 URL',
    'Retry':            '重试',

    // New session dialog
    'New Session':       '新建会话',
    'Provider':          '提供者',
    'Working Directory': '工作目录',
    '/path/to/project':  '/路径/到/项目',
    'Session Name':      '会话名称',
    'Optional':          '可选',
    'Model':             '模型',
    'Default':           '默认',
    'Create & Start':    '创建并启动',
    'Browse':            '浏览',

    // Directory picker
    'Select Directory':   '选择目录',
    'Select or Create Directory': '选择或新建目录',
    'New Folder':         '新建文件夹',
    'Folder name':        '文件夹名称',
    'my-project':         '我的项目',
    'Create':             '新建',
    'Cancel':             '取消',
    'Use':                '使用',
    'Use this folder':    '使用此文件夹',
    'Up':                 '上一级',
    '(root)':             '（根目录）',
    'No sub-folders here':'此处没有子文件夹',
    'Browse from':        '浏览于',
    'Recent':             '最近',
    'Starred':            '已收藏',
    'Star':               '收藏',
    'Unstar':             '取消收藏',
    'New folder & use':   '新建并使用',

    // Session page
    'Switch session':    '切换会话',
    'Start Session':     '启动会话',
    'Start':             '启动',
    'Stop':              '停止',
    'Session':           '会话',
    'Idle':              '空闲',
    'Loading history...': '加载历史...',
    'Attach image':      '附加图片',
    'Toggle quick keys': '切换快捷键',

    // Browser
    'Docs':                           '文档',
    'Files':                          '文件',
    'Database':                       '数据库',
    'Tasks':                          '任务',
    'Preview':                        '预览',
    'MCP Servers':                    'MCP 服务器',
    'Simulator':                      '模拟器',
    'Launch this agent from the New Session dialog on the dashboard.':
        '请在首页顶部点击 New 按钮,通过会话弹框启动这个 agent。',
    'LLM Endpoints':                  'LLM 端点',
    'Address book of OpenAI-compatible model endpoints (Ollama, LM Studio, Groq, Gemini, custom). Shared by every agent.':
        'OpenAI 兼容模型端点地址簿(Ollama / LM Studio / Groq / Gemini / 自定义),所有 agent 共享。',
    'Pull Requests':                  'Pull Request 列表',
    'PostgreSQL':                     'PostgreSQL',
    'SQL editor + schema browser with read-only safety':
        'SQL 编辑器 + 结构浏览(读写分离 / 只读保护)',
    'pg-browser plugin not enabled':  'pg-browser 插件未启用',
    'Install pg-browser from the Hub and configure host/user/password in Plugins → Configure.':
        '在 Hub 里安装 pg-browser,并在 Plugins → Configure 里填 host / user / password。',
    'Run':                            '运行',
    'rows':                           '行',
    'rows affected':                  '行受影响',
    'Executed':                       '执行完成',
    'No tables':                      '无表',
    'No rows':                        '无数据',
    'truncated':                      '已截断',
    'Run a query to see results here.':
        '运行一条查询以在此显示结果。',
    'Review PRs from Gitea, GitHub, or GitLab':
        '浏览 Gitea / GitHub / GitLab 上的 Pull Request',
    'No comments':                    '暂无评论',
    'Open in browser':                '在浏览器打开',
    'Explain this PR':                '让 Claude 总结这个 PR',
    'Review this diff':               '让 Claude Review 这份 diff',
    'Diff copied — start a Claude session on the dashboard and paste.':
        'Diff 已复制到剪贴板,请回首页创建 Claude session 后粘贴。',
    'approved':                       '已批准',
    'changes':                        '待改',
    'commented':                      '已评论',
    'Context comment':                '上下文评论',
    'Diff':                           'Diff',
    'Comments':                       '评论',
    'comments':                       '条评论',
    'State':                          '状态',
    'open':                           'open',
    'closed':                         'closed',
    'all':                            'all',
    'In-app browser with multi-tab URL preview':
        '内置浏览器,支持多标签页 URL 预览',
    'Live iOS / Android device screen with touch & key input':
        '实时 iOS / Android 设备屏幕,支持触控与按键输入',
    'No web browser plugin configured':
        '未配置网页浏览器插件',
    'No simulator plugin configured':
        '未配置模拟器插件',
    'No preview plugins configured':
        '未配置预览插件',
    'No browser panels enabled':      '未启用任何浏览器面板',
    'Enable a File Browser, Obsidian Reader, Database, Tasks, or Preview plugin in Settings → Plugins.':
        '请在 设置 → 插件 中启用文件浏览器、Obsidian Reader、数据库、任务或预览插件。',
    'Enable a File Browser, Database, Tasks, Preview or other panel plugin in Settings → Plugins.':
        '请在 设置 → 插件 中启用文件浏览器、数据库、任务、预览或其他面板插件。',
    'No file browser configured':     '未配置文件浏览器',
    'Enable File Browser in Settings → Plugins and configure the allowed directories.':
        '请在 设置 → 插件 中启用 File Browser 并配置允许访问的目录。',
    'No docs browser configured':     '未配置文档浏览器',
    'Enable a docs-type plugin (e.g. Obsidian Reader) in Settings → Plugins and configure the connection.':
        '请在 设置 → 插件 中启用 docs 类型的插件（例如 Obsidian Reader）并配置连接。',
    'No PostgreSQL browser configured': '未配置 PostgreSQL 浏览器',
    'Enable PostgreSQL Browser in Settings → Plugins and configure the connection.':
        '请在 设置 → 插件 中启用 PostgreSQL Browser 并配置连接。',
    'Search files...':                '搜索文件...',
    'Empty directory':                '空目录',
    'Create session here':            '在此创建会话',
    'Copy path':                      '复制路径',
    'Open':                           '打开',
    'Path copied':                    '路径已复制',
    'Back':                           '返回',
    'Copied':                         '已复制',
    'Type a URL in the bar above and press Go.': '在上方地址栏输入 URL 并按回车。',
    'New tab':                        '新建标签页',
    'Type a URL above and press Go':  '在上方输入 URL 并按回车',
    'Open multiple tabs with the + button.': '通过 + 按钮打开多个标签页。',

    // Settings page
    'Plugins':            '插件',
    'Plugins / 插件':     '插件 / Plugins',
    'Server':             '服务器',
    'Server URL':         '服务器 URL',
    'Change':             '更改',
    'About':              '关于',
    'Language':           '语言',
    'Version':            '版本',
    'Build Date':         '构建日期',
    'GitHub':             'GitHub 仓库',

    // Image attach
    'Send to session':          '发送到会话',
    'Image uploaded':           '图片已上传',
    'Insert into terminal':     '插入终端',
    'Close':                    '关闭',
    'Photo Library':            '相册',
    'Take Photo':               '拍照',
    'No running sessions — start one first': '没有正在运行的会话 — 请先启动一个',

    // Misc buttons
    'Save':   '保存',
    'Edit':   '编辑',
    'Delete': '删除',
    'Add':    '添加',
    'Enable': '启用',
    'Disable':'禁用',
    'Dismiss':'知道了',

    // Composer / quick keys
    'Keys':                  '按键',
    'Ctrl':                  '控制',
    'Commands':              '命令',
    'Custom Commands':       '自定义命令',
    'Type text':             '输入文字',
    'Type into simulator':   '向模拟器输入',
    'Text to send...':       '要发送的文字...',
    'Send':                  '发送',

    // ── Telegram Bridge ──────────────────────────────────────
    'Messaging':                            '消息',
    'Telegram Bridge':                      'Telegram 桥',
    'Telegram Bridge not enabled':          'Telegram 桥未启用',
    'Two-way bridge between OpenDray and a Telegram bot — control sessions, receive idle/exit notifications, send commands when you\'re away from the app.':
        'OpenDray 与 Telegram 机器人的双向桥 —— 远程控制会话、接收空闲/退出通知,在远离应用时也能发送命令。',
    'Enable the Telegram plugin in Settings → Plugins, then add a Bot Token (from @BotFather) and Allowed Chat IDs.':
        '请在 设置 → 插件 中启用 Telegram 插件, 然后填入 Bot Token(来自 @BotFather)和 Allowed Chat IDs。',
    'Connected as':                         '已连接:',
    'Bot offline':                          '机器人离线',
    'Allowed chats':                        '允许的聊天数',
    'Messages sent':                        '已发送消息',
    'Updates received':                     '收到的更新',
    'Last poll':                            '上次轮询',
    'Send Test':                            '发送测试',
    'Send a test message to the configured notifications chat — verifies the bot can reach Telegram and that your chat ID is correct.':
        '向配置的通知聊天发送一条测试消息 —— 验证机器人能访问 Telegram 且 chat ID 正确。',
    'Send test message':                    '发送测试消息',
    'Test message sent to chat':            '测试消息已发送到聊天',
    'Available Commands':                   '可用命令',
    'Show command list':                    '显示命令列表',
    'List running sessions':                '列出运行中的会话',
    'Alias for /status':                    '/status 的别名',
    'Last N lines of a session':            '会话最后 N 行输出',
    'Stop a running session':               '停止运行中的会话',
    'Show your chat id':                    '显示你的 chat id',
    'Send any of these to the bot in Telegram. Anything else (M1) gets a hint to use /help. M2 will add /link to forward plain messages into a session.':
        '在 Telegram 中向机器人发送以上任意命令。其他消息(M1 阶段)会提示使用 /help。M2 阶段会增加 /link 把普通消息转发到指定会话。',

    // ── Telegram Bridge — M2 (linking) ───────────────────────
    'Active Links':                         '活动链接',
    'Each linked Telegram chat sends plain messages to its bound session, and receives the session\'s output (coalesced into 2-second chunks).':
        '每个已绑定的 Telegram 聊天发送的普通消息会作为会话输入,并接收会话的输出(以 2 秒窗口合并)。',
    'No active links yet. In Telegram, send the bot:  /link <session_id>':
        '尚无活动链接。在 Telegram 中向机器人发送: /link <会话 id>',
    'Unlink':                               '解除链接',
    'Linked-chat commands':                 '链接聊天命令',
    'Bind chat → session (two-way)':        '绑定聊天 → 会话(双向)',
    'Remove this chat\'s binding':          '解除此聊天的绑定',
    'List all active links':                '列出全部活动链接',
    'One-shot send without /link':          '一次性发送(不绑定)',
    'Quick keys':                           '快捷键',
    'Answer yes/no + Enter':                '回答 yes/no + 回车',
    'Plain text in a linked chat → session input. Output is polled every 5 s (only sent when content changes). Reply directly to any idle notification → routed automatically.':
        '已链接聊天中的普通文字 → 会话输入。输出每 5 秒轮询一次(仅当内容变化时发送)。直接回复任何空闲通知 → 自动路由到该会话。',

    // ── Claude Multi-Account ─────────────────────────────────
    'Claude Accounts':                         'Claude 账号',
    'Claude account':                          'Claude 账号',
    'Manage multiple Claude subscriptions (OAuth tokens). Each session picks an account at creation time.':
        '管理多个 Claude 订阅(OAuth token)。每个会话在创建时选择账号。',
    'Add Claude account':                      '添加 Claude 账号',
    'No Claude accounts yet':                  '尚未添加 Claude 账号',
    'Register a Claude subscription token so this server can launch sessions as that account. Tokens stay chmod 600 on the host — this UI only tracks metadata.':
        '注册 Claude 订阅 token,以便本服务器可用该账号启动会话。token 在服务器上保持 chmod 600 — 此 UI 仅记录元数据。',
    'Each account maps to one OAuth token managed by the claude-acc host tool. Sessions pick an account at creation time; tokens never enter Postgres or this UI.':
        '每个账号对应由 claude-acc 主机工具管理的一个 OAuth token。会话在创建时选择账号; token 不会进入 Postgres 或此 UI。',
    'Import from ~/.claude-accounts':          '从 ~/.claude-accounts 导入',
    'Failed to load Claude accounts':          '加载 Claude 账号失败',
    'New Claude account':                      '新建 Claude 账号',
    'Edit Claude account':                     '编辑 Claude 账号',
    'Delete Claude account?':                  '删除此 Claude 账号?',
    'This removes "@name" from OpenDray. The on-disk token file and config directory are left intact.':
        '将从 OpenDray 删除 "@name"。服务器上的 token 文件与 config 目录保持不变。',
    'Name must be 1-32 chars of [a-z0-9_-]':   '名称必须为 1-32 个 [a-z0-9_-] 字符',
    'Lowercase alphanumeric, dash or underscore. Matches claude-<name> shortcut.':
        '小写字母/数字/短横线/下划线。对应 claude-<name> 快捷命令。',
    'Display Name':                            '显示名称',
    'Config dir':                              'Config 目录',
    'Token path':                              'Token 路径',
    'Leave empty to use the claude-acc default.':
        '留空则使用 claude-acc 默认路径。',
    'OAuth token (optional)':                  'OAuth token(可选)',
    'OAuth token (leave empty to keep)':       'OAuth token(留空则保留)',
    'Generated by `claude setup-token`. Written chmod 600 at the token path.':
        '由 `claude setup-token` 生成。将以 chmod 600 写入到 token 路径。',
    'Rotate OAuth token':                      '更换 OAuth token',
    'Set OAuth token':                         '设置 OAuth token',
    'Rotate token':                            '更换 token',
    'Set token':                               '设置 token',
    'Paste the token for @name. Generated by `claude setup-token` on the host.':
        '粘贴 @name 的 token。在主机上由 `claude setup-token` 生成。',
    'token set':                               '已配置',
    'no token':                                '未配置',
    'Imported':                                '已导入',
    'Skipped':                                 '已跳过',
    'No ~/.claude-accounts/tokens — run `claude-acc init` on the host first':
        '未找到 ~/.claude-accounts/tokens — 请先在主机上运行 `claude-acc init`',
    'System (keychain / env)':                 '系统默认(keychain / env)',
    'No env override':                         '不覆盖环境变量',
    'Manage':                                  '管理',
    'No accounts registered — tap to add':     '尚未注册账号 — 点击添加',
    'ready':                                   '就绪',
    'need token':                              '缺 token',
    'Each session picks one account. Add, delete, or rotate tokens here.':
        '每个会话选择一个账号。在此添加、删除或更换 token。',

    // ── MCP Servers ──────────────────────────────────────────
    'Add MCP server':              '添加 MCP 服务器',
    'No MCP servers yet':          '还没有 MCP 服务器',
    'Add a server entry — it will be injected into Claude / Codex sessions as a temporary config file at spawn time. Your user home configs stay untouched.':
        '添加一个服务器条目 —— 会话启动时会作为临时配置文件注入到 Claude / Codex,不会修改用户主目录里的任何配置。',
    'Failed to load MCP servers':  '加载 MCP 服务器失败',
    'New MCP server':              '新建 MCP 服务器',
    'Edit MCP server':             '编辑 MCP 服务器',
    'Delete MCP server?':          '删除此 MCP 服务器?',
    'This removes "@name" from OpenDray. Sessions already running keep their injected config.':
        '将从 OpenDray 删除 "@name"。已在运行的会话仍保留已注入的配置。',
    'Name is required':            '名称必填',
    'Transport':                   '传输',
    'Command':                     '命令',
    'Args (space-separated)':      '参数 (空格分隔)',
    'Environment':                 '环境变量',
    'KEY':                         'KEY',
    'value':                       'value',
    'URL':                         'URL',
    'Applies to':                  '应用到',
    'all agents':                  '所有 agent',
    'Enabled':                     '启用',

    // ── Log Viewer ───────────────────────────────────────────
    'Logs':                       '日志',
    'Log Viewer':                 '日志查看器',
    'Tail and filter log files on the server in real time — follow, grep, highlight levels.':
        '实时跟踪并过滤服务器上的日志文件 —— 滚动、grep 过滤、按级别高亮。',
    'No log viewer configured':   '未配置日志查看器',
    'Enable Log Viewer in Settings → Plugins and set allowed directories.':
        '请在 设置 → 插件 中启用日志查看器并设置允许访问的目录。',
    'No allowed roots configured':'未配置允许访问的目录',
    'No log files here':          '此处没有日志文件',
    'Log File Extensions':        '日志文件扩展名',
    'Initial Backlog (KB)':       '初始回溯长度 (KB)',
    'Comma-separated root directories that can be browsed for log files. All tailed paths must live under one of these roots.':
        '允许浏览日志文件的根目录,用逗号分隔。所有 tail 的路径必须位于这些根目录之下。',
    'Comma-separated file extensions to list. Empty shows every file.':
        '要列出的文件扩展名,用逗号分隔。留空则显示所有文件。',
    'How many KB from the tail of the file to send before following. Large files with a huge backlog take longer to render.':
        '开始跟踪前发送的文件末尾字节数(KB)。回溯过大会延长初次加载时间。',
    'Filter lines (regex) — e.g. ERROR|WARN':
        '过滤行(正则)—— 例如 ERROR|WARN',
    'Waiting for log lines…':     '等待日志输出…',
    'Pause':                      '暂停',
    'Resume stream':              '继续',
    'Auto-scroll on':             '自动滚动:开',
    'Auto-scroll off':            '自动滚动:关',
    'Clear':                      '清空',
    'Copy all':                   '复制全部',
    'lines':                      '行',
    'paused':                     '已暂停',

    // ── Plugin config — UI chrome ────────────────────────────
    'registered':                 '个已注册',
    'Reload':                     '重新加载',
    'No plugins registered':      '尚未注册任何插件',
    'Failed to load plugins':     '加载插件失败',
    '(default)':                  '（默认）',
    'MODELS':                     '模型',
    'Not found':                  '未找到',
    'Active':                     '已启用',
    'Resume':                     '会话恢复',
    'Images':                     '图片支持',
    'MCP':                        'MCP',
    'Dynamic Models':             '动态模型',

    // ── Plugin configure page / form ─────────────────────────
    'Configure':                                '配置',
    'Failed to load config':                    '加载配置失败',
    'This plugin has no configurable fields.':  '此插件没有可配置项。',
    'Saved':                                    '已保存',
    'Save failed':                              '保存失败',
    'General':                                  '通用',
    'is required':                              '为必填项',
    'must be numeric':                          '必须为数字',
    '(stored — leave blank to keep)':           '（已保存 — 留空以保留原值）',

    // ── Settings → Built-in plugins ──────────────────────────
    'Built-in plugins':                         '内置插件',
    'Browse the plugins bundled with OpenDray and restore anything you previously uninstalled.':
        '浏览随 OpenDray 附带的插件，恢复任何之前卸载过的。',
    'These plugins ship with OpenDray. Uninstalling one removes it from the Plugins page; restore it here.':
        '这些插件随 OpenDray 一同发布。卸载后会从插件页面消失；在此处恢复。',
    'All built-in plugins are currently installed.':
        '所有内置插件都已安装。',
    'built-in plugin(s) uninstalled — tap Restore to bring them back.':
        '个内置插件已卸载 —— 点击"恢复"把它们装回来。',
    'No built-in plugins':                      '没有内置插件',
    'The server reported zero bundled manifests — this usually means the binary was built without the plugins/builtin tree.':
        '服务端没有返回任何内置清单 —— 通常说明二进制构建时漏了 plugins/builtin 目录。',
    'Failed to load built-in plugins':          '加载内置插件失败',
    'Installed':                                '已安装',
    'Disabled':                                 '已禁用',
    'Uninstalled':                              '已卸载',
    'Restore':                                  '恢复',
    'Restoring…':                               '恢复中…',
    'Restored':                                 '已恢复',
    'Restore failed':                           '恢复失败',
    'Toggle from Plugins page to enable.':      '请到插件页面打开开关启用。',
    'Already active. Manage from Plugins page.':'已激活。在插件页面管理。',

    // ── Source Control panel ─────────────────────────────────
    // Retired git-viewer + git-forge strings went with those plugins;
    // the remaining keys are shared with source-control's Changes /
    // History / PRs / Branches tabs.
    'Pick folder':                                '选择文件夹',
    'Refresh':                                    '刷新',
    'Clean':                                      '干净',
    'Changes':                                    '变更',
    'History':                                    '历史',
    'No changes':                                 '无变更',
    'Working tree is clean.':                     '工作树已干净。',
    'Stage':                                      '暂存',
    'Unstage':                                    '取消暂存',
    'Discard':                                    '放弃',
    'Discard changes?':                           '放弃变更?',
    'This overwrites unstaged changes with the committed version. Cannot be undone.':
        '这会将未暂存的变更覆盖为已提交的版本,无法撤销。',
    'Confirm':                                    '确认',
    'staged diff':                                '已暂存的差异',
    'unstaged diff':                              '未暂存的差异',
    'Select a file to view its diff.':            '选择一个文件以查看其差异。',
    'Commit message':                             '提交信息',
    'Commit':                                     '提交',
    'Commit created':                             '已创建提交',
    'No commits yet':                             '暂无提交',
    'The log will appear here once there are commits.':
        '一旦有提交,日志将显示在此处。',
    'just now':                                   '刚刚',
    'Snapshot':                                   '快照',
    'Snapshot HEAD to track session-only changes.':
        '对 HEAD 建立快照以仅追踪本次会话的变更。',
    'Showing changes since session start @ ':     '显示自会话开始 @ 之后的变更 ',
  };
}

/// Ergonomic helper: `context.tr('Sessions')` inside any widget that's under
/// the `L10n` provider. Using `watch` so the widget rebuilds on language
/// switch.
extension L10nContext on BuildContext {
  String tr(String key) => watch<L10n>().t(key);

  /// Non-watching variant for use in callbacks / builders where you don't
  /// want to subscribe to rebuilds.
  String trOnce(String key) => read<L10n>().t(key);

  /// Ergonomic wrapper over [L10n.pick] for manifest i18n overlays.
  /// Use when rendering plugin-owned text (displayName, config labels,
  /// descriptions) that ships with optional `*_zh` overrides.
  String pickL10n(String en, String? zh) => watch<L10n>().pick(en, zh);

  /// Non-watching variant — safe inside SnackBar / dialog builders.
  String pickL10nOnce(String en, String? zh) => read<L10n>().pick(en, zh);
}
