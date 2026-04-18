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
  String t(String key) {
    if (_code == 'en') return key;
    final catalog = _catalogs[_code];
    if (catalog == null) return key;
    return catalog[key] ?? key;
  }

  // ── Catalogs ────────────────────────────────────────────────────

  static final Map<String, Map<String, String>> _catalogs = {
    'zh': _zh,
  };

  static const Map<String, String> _zh = {
    // Bottom navigation
    'Sessions': '会话',
    'Browser':  '浏览器',
    'Settings': '设置',

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

    // Session page
    'Switch session':    '切换会话',
    'Start Session':     '启动会话',
    'Start':             '启动',
    'Stop':              '停止',
    'Session':           '会话',
    'Idle':              '空闲',
    'Loading history...': '加载历史...',
    'Attach image':      '附加图片',
    'Voice input':       '语音输入',
    'Voice / Dictation': '语音 / 听写',
    'Tap the mic on your keyboard and speak…':
        '点击键盘上的麦克风图标,然后开始说话…',
    'Dictation uses your phone\'s built-in speech recognition. Review the text before sending.':
        '使用手机内置的语音识别。发送前请确认文本无误。',
    'Append Enter':      '附加回车',
    'Sends as a command — a newline is added after the text.':
        '作为命令发送 —— 在文本末尾附加换行符。',
    'Toggle quick keys': '切换快捷键',

    // Browser
    'Docs':                           '文档',
    'Files':                          '文件',
    'Database':                       '数据库',
    'Tasks':                          '任务',
    'Preview':                        '预览',
    'MCP Servers':                    'MCP 服务器',
    'Simulator':                      '模拟器',
    'In-app browser with multi-tab URL preview':
        '内置浏览器,支持多标签页 URL 预览',
    'Live iOS / Android device screen with touch & key input':
        '实时 iOS / Android 设备屏幕,支持触控与按键输入',
    'No web preview plugin configured':
        '未配置网页预览插件',
    'No simulator plugin configured':
        '未配置模拟器插件',
    'No preview plugins configured':
        '未配置预览插件',
    'No browser panels enabled':      '未启用任何浏览器面板',
    'Enable a File Browser, Obsidian Reader, Database, Tasks, or Preview plugin in Settings → Plugins.':
        '请在 设置 → 插件 中启用文件浏览器、Obsidian Reader、数据库、任务或预览插件。',
    'Enable a File Browser, Database, Tasks, Preview or other panel plugin in Settings → Plugins.':
        '请在 设置 → 插件 中启用文件浏览器、数据库、任务、预览或其他面板插件。',
    // Launcher card descriptions
    'Read markdown and Git-forge sources':        '阅读 Markdown 与 Git 仓库文档',
    'Browse & edit server files':                 '浏览并编辑服务器文件',
    'Read-only Postgres browsing & SQL':          '只读浏览 Postgres 与执行 SQL',
    'Run Makefile / npm / shell tasks':           '运行 Makefile / npm / shell 任务',
    'Tail and grep log files live':               '实时 tail 与 grep 日志文件',
    'Telegram bridge & session links':            'Telegram 桥与会话链接',
    'Manage MCP servers injected into agents':    '管理注入到 agent 的 MCP 服务器',
    'Web preview & device simulator':             '网页预览与设备模拟器',
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

    // ── Telegram Bridge — config field labels ────────────────
    'Bot Token':                            '机器人 Token',
    'Allowed Chat IDs':                     '允许的 Chat ID',
    'Notifications Chat ID':                '通知的 Chat ID',
    'Notify on Idle':                       '空闲时通知',
    'Notify on Exit':                       '退出时通知',
    'Notification Tail Lines':              '通知中包含的末尾行数',
    'Poll Interval (seconds)':              '轮询间隔(秒)',
    'Token from @BotFather. Send /newbot in Telegram, follow the prompts, copy the token here. Or leave blank and set the OPENDRAY_TELEGRAM_BOT_TOKEN env var on the server.':
        '来自 @BotFather 的 Token。在 Telegram 中发送 /newbot, 按提示操作并将 Token 复制到这里。也可以留空并在服务器上设置 OPENDRAY_TELEGRAM_BOT_TOKEN 环境变量。',
    'Comma-separated Telegram chat / user IDs allowed to talk to the bot. Anything from a chat NOT in this list is silently dropped. Use @userinfobot to find your own ID.':
        '允许与机器人对话的 Telegram chat / user ID, 用逗号分隔。不在列表中的聊天会被静默丢弃。可用 @userinfobot 查看自己的 ID。',
    'Default chat that receives idle / exit / task-finish notifications. Defaults to the first allowed chat ID.':
        '接收空闲 / 退出 / 任务完成通知的默认聊天。留空则使用第一个允许的 Chat ID。',
    'Send a Telegram message when a session goes idle (CLI waiting for input).':
        '当会话进入空闲(CLI 等待输入)时发送 Telegram 消息。',
    'Send a Telegram message when a session exits.':
        '当会话退出时发送 Telegram 消息。',
    'How many lines from the terminal buffer to include in idle / exit notification messages.':
        '空闲 / 退出通知消息中包含的终端缓冲区末尾行数。',
    'Telegram getUpdates long-poll timeout in seconds. The bot holds a connection open for this duration — if a message arrives during this window it is delivered instantly (not delayed). 25 is the recommended value. Values below 5 increase API call frequency without improving responsiveness.':
        'Telegram getUpdates 长轮询超时(秒)。机器人会保持连接到此时长 — 如有消息在此期间到达会立即下发(无延迟)。建议 25 秒。低于 5 秒只会增加 API 调用频率,不会提升响应速度。',

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

    // ── Plugin config — group headers ────────────────────────
    'Connection':                 '连接',
    'Authentication':             '认证',
    'Runtime':                    '运行时',
    'Advanced':                   '高级',

    // ── Plugin displayName + description ─────────────────────
    'Claude Code':                'Claude Code',
    'Anthropic Claude Code CLI — agentic coding with tool use, MCP, and session resume':
        'Anthropic Claude Code CLI — 支持工具调用、MCP 以及会话恢复的智能编码助手',
    'Codex CLI':                  'Codex CLI',
    'OpenAI Codex CLI — agentic coding with sandboxed execution':
        'OpenAI Codex CLI — 在沙箱中执行的智能编码助手',
    'Gemini CLI':                 'Gemini CLI',
    'Google Gemini CLI — agentic coding with sandbox, multimodal, and Google Search':
        'Google Gemini CLI — 支持沙箱、多模态与 Google 搜索的智能编码助手',
    'Qwen Code':                  '通义千问 Qwen Code',
    'Alibaba Qwen Code CLI — open-source agentic coding based on Qwen3-Coder, forked from Gemini CLI':
        '阿里巴巴 Qwen Code CLI —— 基于 Qwen3-Coder 的开源智能编码助手(由 Gemini CLI 派生)',
    'Ollama':                     'Ollama',
    'Ollama — run open-source LLMs locally (Llama, Mistral, DeepSeek, Qwen)':
        'Ollama — 在本地运行开源大模型(Llama、Mistral、DeepSeek、Qwen 等)',
    'LM Studio':                  'LM Studio',
    'LM Studio — run local models with hardware acceleration (MLX, CUDA, Metal)':
        'LM Studio — 使用硬件加速运行本地模型(MLX、CUDA、Metal)',
    'Terminal':                   '终端',
    'System login shell — zsh, bash, or configured shell':
        '系统登录 shell —— zsh、bash 或已配置的 shell',
    'File Browser':               '文件浏览器',
    'Browse and view project files on the server with syntax highlighting':
        '浏览并查看服务器上的项目文件,支持语法高亮',
    'Obsidian Reader':            'Obsidian 阅读器',
    'Browse and read Obsidian vault documents from a Git repository (Gitea, GitHub, etc.)':
        '从 Git 仓库(Gitea、GitHub 等)浏览和阅读 Obsidian 知识库文档',
    'PostgreSQL Browser':         'PostgreSQL 浏览器',
    'Browse PostgreSQL schemas, tables, and columns. Run read-only SELECT queries with row/time limits.':
        '浏览 PostgreSQL 的模式、表和列。可执行只读 SELECT 查询并支持行数/耗时限制。',
    'Task Runner':                '任务运行器',
    'Discover Makefile targets, package.json scripts, and shell scripts in a project, then run them with live streaming output.':
        '发现项目中的 Makefile 目标、package.json 脚本和 shell 脚本,并实时流式输出运行结果。',
    'Web Preview':                '网页预览',
    'In-app browser panel. Works with any web framework — React, Vue, Next.js, FastAPI, Go, Rails, etc.':
        '内置浏览器面板,兼容任意 Web 框架 — React、Vue、Next.js、FastAPI、Go、Rails 等。',
    'Simulator Preview':          '模拟器预览',
    'Real-time WebSocket stream of iOS Simulator or Android Emulator. Adaptive FPS (8 fps during interaction, 1 fps idle). JPEG compression for fast mobile delivery. Touch, swipe, and key input forwarded over the same WebSocket.':
        'iOS 模拟器或 Android 模拟器的实时 WebSocket 流。自适应帧率(交互时 8fps, 空闲 1fps)。JPEG 压缩加速移动端传输。触摸、滑动和按键输入通过同一 WebSocket 转发。',
    'JPEG Quality':               'JPEG 质量',
    'JPEG compression quality (10–95). Lower = smaller frames + faster, higher = sharper. 50 is a good balance for mobile.':
        'JPEG 压缩质量(10–95)。越低 = 帧越小越快, 越高 = 越清晰。50 是移动端的良好平衡点。',
    'Max Width (px)':             '最大宽度(px)',
    'Scale screenshots down to this width before streaming. Reduces bandwidth. Set to 0 to send at native resolution.':
        '流式传输前将截图缩放到此宽度。减少带宽。设为 0 则按原始分辨率发送。',
    'Active FPS':                 '交互帧率',
    'Frames per second during active interaction (touch/key). Max 15.':
        '交互时(触摸/按键)的每秒帧数。最大 15。',
    'Idle FPS':                   '空闲帧率',
    'Frames per second when no interaction for 5 seconds. Set to 0 to stop streaming when idle.':
        '5 秒无交互后的每秒帧数。设为 0 则空闲时停止流式传输。',
    'Android Emulator (adb) or iOS Simulator (xcrun simctl). Android supports full touch input; iOS supports key events only.':
        'Android 模拟器(adb)或 iOS 模拟器(xcrun simctl)。Android 支持完整触摸输入; iOS 仅支持按键事件。',

    // ── Plugin config — common field labels ──────────────────
    'Command Path':               '命令路径',
    'API Key':                    'API 密钥',
    'Base URL':                   '基础 URL',
    'Host':                       '主机',
    'Username':                   '用户名',
    'Password':                   '密码',
    'Branch':                     '分支',
    'Repository':                 '仓库',
    'Git Forge':                  'Git 平台',
    'Default Model':              '默认模型',
    'Context Window':             '上下文窗口',
    'Extra Args':                 '额外参数',
    'Extra CLI Args':             '额外 CLI 参数',
    'Approval Mode':              '审批模式',
    'Bypass Permissions':         '跳过权限检查',
    'YOLO Mode':                  'YOLO 模式',
    'Sandbox':                    '沙箱',
    'Max Turns':                  '最大轮次',
    'Max Rows':                   '最大行数',
    'Max Concurrent Runs':        '最大并发运行',
    'Max File Size (KB)':         '最大文件大小 (KB)',
    'File Extensions':            '文件扩展名',
    'Show Hidden Files':          '显示隐藏文件',
    'Default Path':               '默认路径',
    'Default Schema':             '默认模式',
    'Bootstrap Database':         '引导数据库',
    'SSL Mode':                   'SSL 模式',
    'Query Timeout (seconds)':    '查询超时(秒)',
    'Task Timeout (seconds)':     '任务超时(秒)',
    'Output Buffer (KB)':         '输出缓冲区 (KB)',
    'Include Makefile Targets':   '包含 Makefile 目标',
    'Include package.json Scripts':'包含 package.json 脚本',
    'Include Shell Scripts (*.sh)':'包含 shell 脚本 (*.sh)',
    'Allowed Directories':        '允许的目录',
    'Base Paths':                 '根路径',
    'Platform':                   '平台',
    'Device ID':                  '设备 ID',
    'Auto-refresh (seconds)':     '自动刷新(秒)',
    'GPU Offload Layers':         'GPU 卸载层数',
    'Ollama Host':                'Ollama 主机',
    'LMS Binary Path':            'LMS 可执行路径',
    'Shell':                      'Shell',
    'Shell Args':                 'Shell 参数',
    'DashScope API Key':          'DashScope API 密钥',

    // ── Plugin config — common descriptions ──────────────────
    'Type of Git hosting service':'Git 托管服务类型',
    'Git forge base URL (no trailing slash)':
        'Git 平台基础 URL(不含末尾斜杠)',
    'owner/repo format':          'owner/repo 格式',
    'API token for private repositories. Leave empty for public repos.':
        '用于访问私有仓库的 API 令牌。公开仓库可留空。',
    'Branch to read from':        '要读取的分支',
    'Directories to show (comma-separated for multiple, e.g. \'Projects/,Infrastructure/\'). Leave empty to show entire repo.':
        '要显示的目录(多个用逗号分隔,如 \'Projects/,Infrastructure/\')。留空表示显示整个仓库。',
    'Comma-separated file extensions to show (e.g. .md,.txt)':
        '要显示的文件扩展名,用逗号分隔(如 .md,.txt)',
    'Comma-separated root directories that can be browsed. All paths must be under these roots.':
        '允许浏览的根目录,用逗号分隔。所有路径必须位于这些根目录之下。',
    'Comma-separated root directories. Tasks can only be discovered and executed inside these roots.':
        '根目录,用逗号分隔。任务只能在这些根目录内发现和执行。',
    'Initial directory shown when the panel opens.':
        '打开面板时显示的初始目录。',
    'Starting directory when opening the file browser':
        '打开文件浏览器时的起始目录',
    'Show files and directories starting with .':
        '显示以 . 开头的文件和目录',
    'Maximum file size to read. Larger files will show metadata only.':
        '读取的最大文件大小。超过此大小的文件仅显示元数据。',
    'Leave blank to read from the env var OPENDRAY_DB_PASSWORD_<PLUGIN_NAME>.':
        '留空则从环境变量 OPENDRAY_DB_PASSWORD_<插件名> 中读取。',
    'Use a role that only has SELECT on the objects you want to browse.':
        '建议使用只对要浏览的对象具有 SELECT 权限的角色。',
    'The database to connect to for listing all other databases. Usually \'postgres\'. The UI will let you switch databases at runtime.':
        '用于列出所有其他数据库的连接目标数据库,通常为 \'postgres\'。界面允许在运行时切换数据库。',
    'Hard cap on rows returned per query. Results past this cap are marked truncated.':
        '每次查询返回的行数上限,超出上限的结果将被标记为已截断。',
    'Hard kill after this many seconds. Set 0 to disable.':
        '达到此秒数后强制终止。设为 0 表示不限制。',
    'Reject new runs once this many tasks are already running.':
        '当正在运行的任务达到此数量时,拒绝新的运行请求。',
    'Per-run rolling output snapshot kept in memory for late subscribers.':
        '为后续订阅者保留的每次运行的滚动输出快照(内存中)。',
    'Discover *.sh files in the project root and ./scripts/.':
        '发现项目根目录和 ./scripts/ 下的 *.sh 文件。',
    'iOS Simulator (xcrun simctl) or Android Emulator (adb)':
        'iOS 模拟器(xcrun simctl)或 Android 模拟器(adb)',
    'Android: adb device ID (e.g. emulator-5554). iOS: leave empty to use the booted simulator.':
        'Android: adb 设备 ID(如 emulator-5554)。iOS: 留空则使用已启动的模拟器。',
    'Screenshot refresh interval. Set to 0 to disable auto-refresh.':
        '截图刷新间隔。设为 0 可禁用自动刷新。',
    'Full URL to open (any framework, any host). Takes priority over the Port field.':
        '要打开的完整 URL(任意框架、任意主机)。优先于端口字段。',
    'Port on the OpenDray server — host is auto-filled from your connection. Used only when URL above is empty.':
        'OpenDray 服务器上的端口 —— 主机自动从连接中填充。仅当 URL 为空时使用。',
    'Appended to \'ollama run <model>\'. Detect available models in Providers page.':
        '附加到 \'ollama run <模型>\' 后。可在 Providers 页面检测可用模型。',
    'Model to load. Use Providers page to detect available models.':
        '要加载的模型。可在 Providers 页面检测可用模型。',
    'Remote Ollama server address':'远程 Ollama 服务器地址',
    'Context window size in tokens':'上下文窗口大小(tokens)',
    'Limit autonomous agentic turns (0 = unlimited)':
        '限制自主智能体的轮次(0 = 不限制)',
    'Auto-approve all tool calls — no confirmation prompts':
        '自动批准所有工具调用 —— 无确认提示',
    'Skip all permission prompts — full autonomous mode':
        '跳过所有权限提示 —— 完全自主模式',
    'Empty = default sandbox; none = no sandbox':
        '留空 = 默认沙箱; none = 不使用沙箱',
    'default = confirm each tool call; auto-edit = auto-apply edits; yolo = no confirmations':
        'default = 每次工具调用都确认; auto-edit = 自动应用编辑; yolo = 不确认',
    'suggest = ask before changes; auto-edit = auto-apply file edits; full-auto = YOLO mode':
        'suggest = 变更前询问; auto-edit = 自动应用文件编辑; full-auto = YOLO 模式',
    'env = read ANTHROPIC_API_KEY from system; custom = use key below; oauth = browser login':
        'env = 从系统读取 ANTHROPIC_API_KEY; custom = 使用下方密钥; oauth = 浏览器登录',
    'env = read OPENAI_API_KEY from system; custom = use key below':
        'env = 从系统读取 OPENAI_API_KEY; custom = 使用下方密钥',
    'env = read GOOGLE_API_KEY from system; custom = use key below; oauth = gcloud auth':
        'env = 从系统读取 GOOGLE_API_KEY; custom = 使用下方密钥; oauth = gcloud 授权',
    'qwen-oauth = browser login (2000 req/day free); dashscope = Alibaba DashScope API; openai-compatible = ModelScope / OpenRouter / custom':
        'qwen-oauth = 浏览器登录(每日 2000 次免费); dashscope = 阿里 DashScope API; openai-compatible = ModelScope / OpenRouter / 自定义',
    'e.g. ModelScope: https://api-inference.modelscope.cn/v1 · OpenRouter: https://openrouter.ai/api/v1':
        '例如 ModelScope: https://api-inference.modelscope.cn/v1 · OpenRouter: https://openrouter.ai/api/v1',
    '-1 = all layers to GPU':     '-1 = 全部层放到 GPU',

    // ── Git panel ────────────────────────────────────────────
    'Git':                                        'Git',
    'Track and commit per-session changes':       '追踪并提交本次会话的变更',
    'Git panel not enabled':                      '未启用 Git 面板',
    'Enable the "git" panel plugin in Settings → Plugins first.':
        '请先在 设置 → 插件 中启用 "git" 面板插件。',
    'Pick a repository':                          '选择仓库',
    'Choose a directory containing a .git folder.':
        '请选择包含 .git 文件夹的目录。',
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

    // ── Git plugin — manifest displayName + description ──────
    'Track and manage Git changes per working directory. Shows status, unified diff, log, branches, and a per-session baseline snapshot so you can see only what changed during the current session.':
        '按工作目录追踪和管理 Git 变更。展示状态、统一差异、日志、分支,并支持每次会话的基线快照,让你只看到本次会话产生的变更。',

    // ── Git plugin — config field labels ─────────────────────
    'Default Repository':                         '默认仓库',
    'Git Binary':                                 'Git 可执行文件',
    'Log Entries':                                '日志条数',
    'Diff Context Lines':                         '差异上下文行数',
    'Command Timeout (seconds)':                  '命令超时(秒)',
    'Allow Commit':                               '允许提交',

    // ── Git plugin — config descriptions ─────────────────────
    'Comma-separated root directories. Git operations can only run inside these roots.':
        '根目录,用逗号分隔。Git 操作仅能在这些根目录内执行。',
    'Repository path opened when the panel first loads.':
        '面板首次加载时打开的仓库路径。',
    'Path to the git executable. Leave as \'git\' to use PATH.':
        'git 可执行文件的路径。保留为 \'git\' 则使用 PATH 中的版本。',
    'Maximum commits returned by the log endpoint.':
        '日志接口返回的提交数量上限。',
    'Number of context lines around each hunk in the unified diff.':
        '统一差异中每个 hunk 周围的上下文行数。',
    'Hard kill any git subprocess that runs longer than this.':
        '运行超过此时长的 git 子进程会被强制终止。',
    'Disable to make the panel strictly read-only (no stage, commit, or discard).':
        '关闭后面板严格只读(禁用暂存、提交、放弃)。',
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
}
