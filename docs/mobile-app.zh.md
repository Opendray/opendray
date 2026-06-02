# 移动 App —— 构建与安装

opendray 移动 App（`app/mobile/`）是一个**控制端客户端**，并不是第二个网关。
它和网页后台 `/admin/` 做的是同一件事：创建并驱动会话、管理 channel 与
integration、浏览 memory、查看 git host。Agent 本身始终运行在你的网关主机上 ——
手机只是“接管”这些会话。

正因如此，App 本身无法独立工作：它需要通过 HTTPS 连接到一个**正在运行的
opendray 网关**。请先把网关跑起来（见 [getting-started](getting-started.md)），
再构建 App 并指向你的网关地址。

> **为什么没有 App Store / Play Store 下载？**
> opendray 是自托管、单租户软件。商店版本必须内置“某个人的”后端，而这恰恰与
> opendray 的设计相悖。所以你自行构建 App、用你自己的身份签名，它只与你的网关
> 通信。下面给出两条受支持的路径：**(A)** 旁加载（sideload）一个 Android APK，
> **(B)** 通过 Xcode 安装 iOS 版本。

---

## 第 0 步 —— 让手机能访问到网关

App 通过网络与网关通信，因此手机必须能访问到网关。

| 场景 | Gateway URL 填什么 |
|---|---|
| 手机与网关在同一局域网 | `http://<网关局域网IP>:8770`（如 `http://192.168.1.50:8770`） |
| 网关在带 TLS 的反向代理后 | `https://opendray.yourdomain.com` |
| 在外网访问（蜂窝、出差） | 一个公网 HTTPS 端点 —— Cloudflare Tunnel、Tailscale，或 nginx/Caddy 反代 |

> **不要把 `:8770` 裸暴露到公网。** 前面要套 TLS 和入口网关。Cloudflare Tunnel
> 是阻力最小的方案（无需端口转发、无需公网 IP）。nginx / Caddy 配置片段 ——
> 包含 Sessions 终端所需的 **WebSocket upgrade 头** —— 见
> [operator-guide §Topology](operator-guide.md#topology)。

构建之前，请先在手机上确认可达性，例如用手机浏览器打开 Gateway URL，应当看到
网页后台的登录页。

---

## 第 1 步 —— 安装 Flutter 工具链

App 用 Flutter 构建。你需要在**执行构建的机器**上安装它（不是手机上）。

```sh
# 按 https://docs.flutter.dev/get-started/install 安装对应系统版本。
flutter --version          # 需要 3.41+（Dart SDK ^3.11）
flutter doctor             # 解决目标平台的所有 ✗
```

`flutter doctor` 是关卡：它会准确告诉你 Android（Android SDK + 设备/模拟器）或
iOS（Xcode + CocoaPods）还缺什么。继续之前，先把目标平台的 ✗ 修干净。

拉取依赖（一次即可）：

```sh
cd app/mobile
flutter pub get
```

---

## 第 2A 步 —— Android：构建 APK 并旁加载

这是最简单的路径 —— 不需要开发者账号，不需要商店。

### 构建 APK

```sh
cd app/mobile

# 单个通用 APK（最便于分享 / 旁加载）：
flutter build apk --release

# —— 或 —— 更小的按架构拆分 APK（挑你手机对应的那个）：
flutter build apk --release --split-per-abi
```

产物位于：

```
app/mobile/build/app/outputs/flutter-apk/app-release.apk
```

> **签名说明。** 开箱即用时，release 包使用 **debug keystore** 签名（见
> `android/app/build.gradle.kts` 里的 `TODO`）。个人旁加载完全够用。若你想要一个
> 正式的 upload key（上架 Play Store 必需，长期迭代也更稳妥），参考
> [Flutter — Sign the app](https://docs.flutter.dev/deployment/android#sign-the-app)
> 为 `release` 增加 `signingConfig`。

### 把 APK 装到手机上

任选其一：

```sh
# 手机已用 USB 连接且开启 USB 调试，直接安装：
flutter install                 # 构建并安装到已连接设备
# 或用已有的 APK：
adb install -r build/app/outputs/flutter-apk/app-release.apk
```

或者把 `.apk` 文件传到手机（隔空投送类工具、文件共享、下载链接、自己发邮件给
自己），点击它。Android 会要求你为“打开该文件的 App”（文件、Chrome 等）授予
**“安装未知应用”** 权限 —— 授予后确认安装即可。

App 会以 **Opendray**（`io.opendray.opendray`）出现。

---

## 第 2B 步 —— iOS：用 Xcode 构建并安装

iOS 没有“旁加载 APK”的等价方式 —— 每次安装都要代码签名。你需要一台**装有
Xcode 的 Mac**和一个 **Apple ID**。免费 Apple ID 可用（App 每 7 天需重新签名，
profile 过期后重装即可）；付费 Apple Developer 账号（99 美元/年）可获得为期一年
的 profile 和 TestFlight。

### 一次性签名配置

```sh
cd app/mobile
flutter pub get
open ios/Runner.xcworkspace        # 打开 WORKSPACE，不是 .xcodeproj
```

在 Xcode 中：

1. 选择 **Runner** target → **Signing & Capabilities** 标签页。
2. 勾选 **Automatically manage signing**。
3. **Team**：选择你的 Apple ID team（若没列出，在
   Xcode → Settings → Accounts 添加你的 Apple ID）。
4. **Bundle Identifier**：默认是 `io.opendray.opendray`。用免费 Apple ID 时，
   这个 ID 在 Apple 侧可能已被占用 —— 若 Xcode 报 provisioning 错误，改成唯一的
   值，如 `io.opendray.opendray.<你的名字>`。

### 构建并安装到 iPhone

1. 用 USB 连接 iPhone；提示时**信任**这台电脑。
2. 在手机上开启 **开发者模式**：
   设置 → 隐私与安全性 → 开发者模式 → 打开 → 重启。
3. 在 Xcode 顶部设备下拉菜单中选择你的 iPhone。
4. 点 **▶ Run**（或 `⌘R`）。Xcode 会构建、签名并安装。

或者用命令行驱动，由 Xcode 处理签名：

```sh
flutter run --release -d <device-id>     # `flutter devices` 可列出 id
```

### 设备上首次启动

在你信任开发者 profile 之前，iOS 不会运行个人 team 签名的 App：

- 手机上：**设置 → 通用 → VPN与设备管理 →** 你的 Apple ID → **信任**。

App 会以 **Opendray** 出现在主屏幕。

> **免费 Apple ID 过期。** 约 7 天后 App 会无法启动（“无法验证 App”）。从 Xcode
> 重新构建即可刷新 profile。付费账号可避免此问题。

---

## 第 3 步 —— 让 App 连接到网关

首次启动会显示引导页：

1. **Gateway URL** —— 填入第 0 步得到的 URL
   （如 `https://opendray.yourdomain.com`）。点 **Continue**。
2. **登录** —— `admin` + 你的 admin 密码（即你在 `[admin].password` 设置的，或
   之后修改的密码）。

完成 —— 你将进入与网页后台相同的界面：Sessions、Channels、Integrations、
Memory、Git、Settings。

之后若要指向另一个网关，在登录页点 **Change**（或 设置 → server）重新填写 URL。

---

## 更新 App

没有自动更新 —— 拉取新代码后重装：

```sh
git pull
cd app/mobile
flutter pub get

# Android：
flutter build apk --release      # 然后旁加载 / `flutter install`

# iOS：
open ios/Runner.xcworkspace      # 再次 ▶ Run，或 `flutter run --release`
```

App 自身的版本号在 `app/mobile/pubspec.yaml`（`version: <semver>+<build>`）。

---

## 排错

| 现象 | 原因 | 解决 |
|---|---|---|
| 引导页 “无法连接” | 手机访问不到 Gateway URL | 用手机浏览器打开该 URL；先修好局域网 IP / tunnel / TLS（第 0 步） |
| 能登录但 Sessions 终端连不上 | 反向代理丢掉了 WebSocket upgrade | 补上 WS 头 —— [operator-guide §Topology](operator-guide.md#topology) |
| Android 拦截安装 | 未授予“安装未知应用” | 为打开 `.apk` 的 App（文件 / Chrome）授予该权限 |
| iOS 启动报 “未受信任的开发者” | 个人 team profile 尚未被信任 | 设置 → 通用 → VPN与设备管理 → 信任 |
| Xcode 里报 “无法安装 / 签名” | 免费 Apple ID 的 Bundle ID 冲突 | 把 Bundle Identifier 改成 `io.opendray.opendray.<你的名字>` |
| iOS App 一周后打不开 | 免费 Apple ID profile 过期（7 天） | 从 Xcode 重新构建，或改用付费账号 |
| `flutter doctor` 目标平台出现 ✗ | 缺 Android SDK / Xcode / CocoaPods | 按 `flutter doctor` 打印的具体行修复 |

---

## 另见

- [getting-started.md](getting-started.md) —— 先把 App 连接的网关搭起来
- [operator-guide.md](operator-guide.md) —— 外网访问的反代 / tunnel 拓扑
- [Flutter — 构建并发布 Android](https://docs.flutter.dev/deployment/android)
- [Flutter — 构建并发布 iOS](https://docs.flutter.dev/deployment/ios)
