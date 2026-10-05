# NSSM Plus

一个基于 Go + Wails (WebView2) 的 Windows 服务管理工具，是 NSSM (Non-Sucking Service Manager) 的现代化替代方案。提供原生 GUI 界面，无需命令行操作，支持配置导入/导出和一站式服务管理。

# Author
```
-------------------------------------
- 🚀 Powered by Moshow郑锴
- 🌟 Might the holy code be with you!
-------------------------------------
🔍 公众号 👉 软件开发大百科
💻 CSDN 👉 https://zhengkai.blog.csdn.net
📂 GitHub 👉 https://github.com/moshowgame
```

# Introduction

## 界面预览

![NSSM Plus Screenshot](screencap1.png)

## 功能特性

- **原生 GUI** - 直接双击打开，无需命令行启动
- **命令行模式** - 完整 CLI 支持，`nssm-plus install/remove/start/stop/restart/status/show/edit/list/log/export/import`
- **单页面操作** - 左侧服务列表 + 右侧配置表单，无 Tab 切换
- **完整服务管理** - 安装、修改、启动、停止、重启、删除服务
- **多服务配置文件** - 一份 JSON 文件统一管理多个服务配置，方便批量导入/导出
- **服务状态监控** - 实时显示 Running / Stopped 等状态，10 秒自动刷新
- **环境变量编辑** - 可视化 Key-Value 编辑器，支持增删改
- **服务依赖管理** - 可视化添加/移除服务依赖项
- **DPAPI 密码加密** - 使用 Windows DPAPI 机器级加密保护服务账户密码
- **日志轮转** - 按文件大小自动轮转，最多保留 5 份历史日志
- **自定义日志路径** - 支持独立配置 stdout/stderr 日志路径
- **崩溃自动重启** - 可配置重启延迟，进程异常退出后自动恢复
- **优雅停止** - 可配置超时时间，先优雅终止再强制 kill
- **安全密码输入** - CLI 支持 `--password-stdin` 和 `--password-file`，避免明文暴露
- **暗色主题** - 现代化深色 UI
- **组件化架构** - 前端拆分为 ServiceList / ConfigForm / ActionBar 子组件

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 桌面框架 | [Wails](https://wails.io/) (WebView2) | v2.12 |
| 后端 | Go | 1.22+ |
| 前端 | Vue 3 + Vite 5 | ^3.4 / ^5.4 |
| Windows API | `golang.org/x/sys/windows/svc/mgr` | - |

## 项目结构

```
nssm-plus/
├── main.go                       # 入口：检测 service 模式 / CLI / 启动 Wails GUI
├── app.go                        # 前后端桥接层，暴露给前端的 Go 方法
├── go.mod / go.sum               # Go 模块依赖
├── wails.json                    # Wails 项目配置
├── Install.ps1                   # 一键安装依赖脚本
├── Run.ps1                       # 开发模式运行脚本
├── Build.ps1                     # 生产构建脚本
│
├── internal/                     # 后端核心逻辑（不直接暴露给前端）
│   ├── cli/
│   │   └── cli.go                # 命令行界面（install/remove/start/stop/restart/status/show/edit/list/log/export/import）
│   ├── service/
│   │   └── manager.go            # Windows SCM 服务管理（安装/删除/启停/查询/修改）
│   ├── wrapper/
│   │   ├── config.go             # Wrapper 配置文件管理（ProgramData 持久化）+ 参数解析
│   │   └── wrapper.go            # Windows 服务包装器（进程托管/日志/轮转/崩溃重启/优雅停止）
│   ├── config/
│   │   └── config.go             # 配置文件序列化（JSON 导入/导出 + DPAPI 密码加密）
│   ├── common/
│   │   └── path.go               # 共享路径处理函数（Wrapper BinaryPath 构建/解析）
│   └── dpapi/
│       └── dpapi.go              # Windows DPAPI 加密/解密（机器级作用域）
│
├── frontend/                     # 前端源码
│   ├── index.html                # HTML 入口
│   ├── package.json              # npm 依赖
│   ├── vite.config.js            # Vite 构建配置
│   ├── tsconfig.json             # TypeScript 配置
│   └── src/
│       ├── main.js               # Vue 应用挂载
│       ├── style.css             # 全局样式 + CSS 变量（暗色主题）
│       ├── App.vue               # 主组件（组合子组件）
│       ├── components/
│       │   ├── ServiceList.vue   # 服务列表侧边栏组件
│       │   ├── ConfigForm.vue    # 服务配置表单组件（含环境变量/依赖编辑器）
│       │   └── ActionBar.vue     # 操作按钮栏组件
│       └── locales/
│           ├── zh.json           # 中文语言包
│           └── en.json           # 英文语言包
│
├── build/
│   └── appicon.png               # 应用图标
│
├── configs/
│   ├── example.json              # 旧版多服务格式示例（用于 Import Config）
│   └── examples/                 # 新版单服务格式示例（每服务一个文件）
│       ├── MyAppService.json
│       └── AnotherService.json
│
└── .gitignore
```

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     WebView2 窗口                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Vue 3 前端                                  │ │
│  │  ┌──────────┐  ┌────────────┐  ┌──────────────┐        │ │
│  │  │ServiceList│  │ ConfigForm │  │  ActionBar   │        │ │
│  │  └────┬─────┘  └─────┬──────┘  └──────┬───────┘        │ │
│  └───────┼──────────────┼───────────────┼───────────────────┘ │
│          │  window.go.main.App  (Wails 自动生成桥接)          │
├──────────┼──────────────────────────────────────────────────┤
│  Go 后端  │                                                   │
│  ┌───────┴──────────┐  ┌────────────────────────┐            │
│  │    app.go         │  │  main.go               │            │
│  │  前后端绑定方法    │  │  ┌─ service 模式检测    │            │
│  │  InstallService() │  │  │  nssm-plus.exe        │            │
│  │  StartService()   │  │  │    service MySvc     │            │
│  │  ...              │  │  │  → wrapper.Run()     │            │
│  └────┬─────────┬───┘  │  ├─ CLI 模式             │            │
│       │         │       │  │  nssm-plus.exe install│            │
│  ┌────┴────┐ ┌─┴───┐   │  │  → cli.Run()         │            │
│  │ service │ │config│   │  ├─ GUI 模式（默认）     │            │
│  │ manager │ │      │   │  │  → wails.Run()       │            │
│  │ (SCM)   │ │(JSON)│   │  └──────────────────────┘            │
│  └────┬────┘ └──┬───┘   └────────────────────────┘             │
│       │         │                                             │
│  ┌────┴─────────┴──┐  ┌──────────┐  ┌──────────┐              │
│  │    wrapper       │  │  common   │  │  dpapi   │              │
│  │  config.go       │  │  path.go  │  │  dpapi.go│              │
│  │  wrapper.go      │  │ (共享路径) │  │ (加密)   │              │
│  └─────────────────┘  └──────────┘  └──────────┘              │
├───────────────────────────────────────────────────────────────┤
│                   Windows Service Control Manager              │
└───────────────────────────────────────────────────────────────┘
```

### 服务包装器（Wrapper）机制

NSSM Plus 的核心设计借鉴了 NSSM 的 Wrapper 模式。由于 `java.exe`、`node.exe` 等程序本身不是 Windows 服务，无法直接被 SCM 管理，因此 NSSM Plus 采用**自托管包装器**方案：

```
启动流程:
  SCM → nssm-plus.exe service MyService → 读取配置 → 启动子进程 → 监控

停止流程:
  SCM → STOP 信号 → taskkill /T /PID (优雅停止) → 5s 超时 → taskkill /F /T /PID (强制)
```

**工作原理**：
1. 安装服务时，SCM 的 `BinaryPathName` 指向 `nssm-plus.exe` 自身，格式为：`"<exePath>" service <ServiceName>`
2. 服务启动时，`main.go` 检测到 `service` 参数，调用 `wrapper.Run()` 进入服务模式
3. Wrapper 实现 `svc.Handler` 接口，读取 `ProgramData/NSSM-Plus/services/<name>.json` 中的配置
4. 使用 `exec.Command` 启动目标子进程（如 `java -jar app.jar`），将 stdout/stderr 重定向到日志文件
5. 进入事件循环，响应 SCM 的 Stop/Shutdown 信号，通过 `taskkill /T /PID` 终止子进程树

**数据存储路径**：
```
%ProgramData%\NSSM-Plus\
├── services\           # 每个服务的独立配置文件
│   ├── MyService.json
│   └── AnotherService.json
└── logs\               # 子进程输出日志
    ├── MyService.log
    └── AnotherService.log
```

**Wrapper 配置结构**（`internal/wrapper/config.go`）：
```go
type WrapperConfig struct {
    AppPath        string            `json:"appPath"`        // 应用程序路径
    Arguments      string            `json:"arguments"`      // 启动参数
    WorkDir        string            `json:"workDir"`        // 工作目录
    Env            map[string]string `json:"env"`            // 环境变量
    RestartDelay   int               `json:"restartDelay"`   // 崩溃后重启延迟(秒)，0=不重启
    LogStdout      string            `json:"logStdout"`      // 标准输出日志路径（空=默认路径）
    LogStderr      string            `json:"logStderr"`      // 标准错误日志路径（空=跟随stdout）
    RotateLog      bool              `json:"rotateLog"`      // 是否启用日志轮转
    RestartTimeout int               `json:"restartTimeout"` // 优雅停止超时(秒)，默认5
}
```

### 前后端通信

Wails 框架在编译时自动生成 Go → JS 绑定代码。前端通过 `window.go.main.App.xxx()` 调用后端方法：

```javascript
// frontend/src/App.vue 中的调用方式
window.go.main.App.InstallService(config)   // 安装服务
window.go.main.App.StartService(name)       // 启动服务
window.go.main.App.GetInstalledServices()   // 获取服务列表
window.go.main.App.ShowOpenDialog(title)    // 打开文件选择对话框
window.go.main.App.ShowSaveDialog(title)    // 打开文件保存对话框
```

所有在 `app.go` 中定义的 `App` 结构体的公开方法，只要参数和返回值是可序列化类型，都会自动暴露给前端。

### 服务识别机制

NSSM Plus 通过两个维度识别自己管理的服务：

1. **Description 标记**：服务的 Description 字段中添加 `[NSSM-Plus]` 前缀
   ```
   Description: "[NSSM-Plus] My web application service"
   ```
2. **Wrapper BinaryPathName**：BinaryPathName 包含 ` service ` 关键字，指向 `nssm-plus.exe` 自身
   ```
   BinaryPathName: "C:\path\to\nssm-plus.exe" service MyService
   ```

`ListServices()` 会枚举系统所有服务，通过 Description 标记筛选。对于 Wrapper 服务，`GetServiceConfig()` 和 `ListServices()` 会从 `ProgramData/NSSM-Plus/services/` 读取真实的应用路径和参数，而非显示 wrapper 的 BinaryPathName。

### 配置数据结构

服务配置以 `ServiceConfig` 结构体为核心，定义在 `internal/service/manager.go` 中：

```go
type ServiceConfig struct {
    ServiceName    string            `json:"serviceName"`    // 服务内部名称
    DisplayName    string            `json:"displayName"`    // 服务显示名称
    Description    string            `json:"description"`    // 服务描述
    AppPath        string            `json:"appPath"`        // 应用程序路径
    Arguments      string            `json:"arguments"`      // 启动参数
    StartType      string            `json:"startType"`      // auto / demand / disabled
    Account        string            `json:"account"`        // 运行账户
    Password       string            `json:"password"`       // 账户密码（DPAPI 加密存储）
    Environment    map[string]string `json:"environment"`    // 环境变量
    LogStdout      string            `json:"logStdout"`      // 标准输出日志路径
    LogStderr      string            `json:"logStderr"`      // 标准错误日志路径
    RotateLog      bool              `json:"rotateLog"`      // 日志轮转
    RestartDelay   int               `json:"restartDelay"`   // 崩溃后重启延迟(秒)
    RestartTimeout int               `json:"restartTimeout"` // 优雅停止超时(秒)
    Dependencies   []string          `json:"dependencies"`   // 依赖服务
}
```

该结构体同时用于 JSON 配置文件存储和前后端数据传输。

### 配置文件模型：单服务单文件 + 配置目录集中管理

自 v0.4 起，配置改为**每个服务一个独立 JSON 文件**，统一存放在配置目录 `%ProgramData%\NSSM-Plus\configs\`，文件名为 `<serviceName>.json`：

```
%ProgramData%\NSSM-Plus\configs\
├── MyAppService.json      # MyAppService 的完整 ServiceConfig
└── AnotherService.json    # AnotherService 的完整 ServiceConfig
```

```json
{
  "serviceName": "MyAppService",
  "appPath": "C:\\path\\to\\app.exe",
  "arguments": "--port 8080",
  ...
}
```

这一模型的特点：

- **互相隔离**：单个配置文件损坏、被删或改名不影响其他服务的加载；无法解析的文件会在界面上以 ⚠ 徽章提示，其余服务照常工作。
- **集中管理**：应用启动时自动扫描整个配置目录，侧栏合并显示已安装服务（带状态）和配置文件中的未安装服务（标记为 "Not Installed" + "File" 标签）。
- **自动落盘**：通过 GUI 或 CLI 安装/修改服务时，配置会自动写入对应的服务配置文件，配置目录始终是配置的权威来源。
- **改名迁移**：服务改名后旧配置文件会被自动删除，避免残留。
- **旧格式兼容**：多服务合并 JSON（`{"services": [...]}`、裸数组、旧单服务对象）仍可通过 **Import Config** / `nssm-plus import` 导入——导入时自动拆分为每服务一个文件。多服务格式也用于 **Export Config** / `nssm-plus export` 的聚合导出。示例参见 [`configs/example.json`](configs/example.json)（旧格式）与 [`configs/examples/`](configs/examples/)（新格式）。

## 环境要求

- **操作系统**: Windows 10 / 11 (需要 WebView2 Runtime)
- **Go**: 1.22+
- **Node.js**: 18+ (用于前端构建)
- **权限**: 管理员权限 (服务管理操作需要)

> Windows 11 和 Windows 10 (21H2+) 通常已内置 WebView2 Runtime。旧版本系统需手动安装：https://developer.microsoft.com/en-us/microsoft-edge/webview2/

## 快速开始

### 方式零：使用一键脚本（最简单）

```powershell
# 1. 安装依赖（首次运行）
.\Install.ps1

# 2. 开发模式运行（热重载，需管理员终端）
.\Run.ps1

# 3. 生产构建
.\Build.ps1
# 产出: build/bin/nssm-plus.exe
```

### 方式一：使用 Wails CLI（推荐，支持热重载）

```bash
# 1. 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. 克隆项目
git clone <repo-url> nssm-plus
cd nssm-plus

# 3. 开发模式运行（前端和后端热重载，需管理员终端）
wails dev

# 4. 生产构建
wails build
# 产出: build/bin/nssm-plus.exe
```

### 方式二：手动构建

```bash
# 1. 克隆项目
git clone <repo-url> nssm-plus
cd nssm-plus

# 2. 安装前端依赖
cd frontend
npm install

# 3. 构建前端
npm run build
# 产出: frontend/dist/

# 4. 回到项目根目录，编译 Go 程序
cd ..
go build -o nssm-plus.exe .

# 5. 运行（需管理员权限）
.\nssm-plus.exe
```

### 方式三：仅预览前端

如果只想修改 UI 而不涉及后端，可以独立启动前端开发服务器：

```bash
cd frontend
npm install
npm run dev
# 浏览器打开 http://localhost:34115 预览（后端调用会报错，仅用于 UI 开发）
```

## 使用方法

### GUI 模式

1. **以管理员身份运行** `nssm-plus.exe`，应用自动扫描配置目录 `%ProgramData%\NSSM-Plus\configs\`（每服务一个 `<serviceName>.json`）
2. 点击 **New Config** 直接填写表单创建新服务，或点击 **Import Config** 导入旧版多服务 JSON（自动拆分为单服务文件）
3. 侧栏会显示已安装服务和配置目录中的未安装服务（带 "File" 标签）
4. 点击侧栏服务查看/编辑配置，点击 **Install** 安装（配置自动落盘到配置目录）
5. 已安装服务可通过 **Reconfigure** 重新配置（自动停止→修改→启动）
6. 使用 **Start / Stop / Restart** 控制服务运行
7. **Uninstall** 卸载服务（保留配置文件，便于重新安装），**Delete** 删除配置文件本身
8. 编辑后点击 **Save Service** 将当前服务保存到 `<serviceName>.json`（改名会自动删除旧文件）
9. 点击 **Export Config** 将配置目录中的所有服务聚合导出为一份多服务 JSON（可再导入）
10. 点击 **Debug** 输出调试信息到控制台（按 F12 查看）

### CLI 模式

```bash
# 安装服务
nssm-plus install MyService --app "C:\java\bin\java.exe" --args "-jar app.jar"
nssm-plus install MyService --app "node.exe" --dir "C:\myapp" --start auto --env "PORT=3000;NODE_ENV=prod"

# 安全密码输入
nssm-plus install MyService --app "app.exe" --account "DOMAIN\User" --password-stdin
nssm-plus install MyService --app "app.exe" --account "DOMAIN\User" --password-file "C:\secrets\pw.txt"

# 服务控制
nssm-plus start MyService
nssm-plus stop MyService
nssm-plus restart MyService
nssm-plus status MyService

# 查看和编辑配置
nssm-plus show MyService
nssm-plus show MyService --json
nssm-plus edit MyService --args "-jar app.jar --server.port=8081"
nssm-plus edit MyService --restart-delay 5 --rotate-log

# 列出服务
nssm-plus list
nssm-plus list --json

# 查看日志
nssm-plus log MyService
nssm-plus log MyService --lines 100

# 导入/导出（导入会同时写入配置目录的单服务文件）
nssm-plus export --output services.json          # 聚合导出为多服务 JSON
nssm-plus export MyService --output myservice.json
nssm-plus export --dir "C:\backup\configs"       # 拆分导出为每服务一个文件
nssm-plus import services.json                   # 导入多服务 JSON（自动拆分落盘）
nssm-plus import "C:\backup\configs"             # 导入整个配置文件目录

# 删除服务
nssm-plus remove MyService
```

配置文件示例参见 [`configs/example.json`](configs/example.json)。

## 开发指南：如何基于本项目修改

### 1. 修改服务管理逻辑

**文件**: `internal/service/manager.go`

所有 Windows 服务操作集中在此文件。核心 API 来自 `golang.org/x/sys/windows/svc/mgr` 包：

| 方法 | 用途 | 底层 API |
|------|------|---------|
| `Install()` | 安装服务 | `mgr.CreateService()` |
| `Remove()` | 删除服务 | `s.Delete()` |
| `Start()` | 启动服务 | `s.Start()` |
| `Stop()` | 停止服务 | `s.Control(svc.Stop)` |
| `Modify()` | 修改配置 | `s.UpdateConfig()` |
| `ListServices()` | 列出服务 | `scMgr.ListServices()` |
| `GetServiceConfig()` | 读取配置 | `s.Config()` |

**常见修改场景**：

- **增加服务配置字段**：在 `ServiceConfig` 结构体中添加字段，然后在 `Install()` 和 `Modify()` 中将新字段写入 `mgr.Config`
- **自定义日志路径**：通过 `WrapperConfig.LogStdout` / `LogStderr` 设置自定义日志路径，空值时使用默认 `%ProgramData%\NSSM-Plus\logs\` 路径
- **实现崩溃重启**：通过 `WrapperConfig.RestartDelay` 配置重启延迟（秒），进程异常退出后自动恢复
- **修改停止策略**：通过 `WrapperConfig.RestartTimeout` 配置优雅停止超时时间（默认 5 秒），超时后强制终止

### 2. 添加新的前后端桥接方法

**文件**: `app.go`

在 `App` 结构体上添加新的公开方法即可自动暴露给前端：

```go
// app.go - 添加新方法
func (a *App) GetServiceLogs(serviceName string) (string, error) {
    // 实现读取服务日志的逻辑
    return logContent, nil
}
```

前端调用方式：

```javascript
// frontend/src/App.vue
const logs = await call('GetServiceLogs', serviceName)
```

### 3. 修改 GUI 界面

**文件**: `frontend/src/App.vue`（主组件）+ `frontend/src/components/`（子组件）
**全局样式**: `frontend/src/style.css`（CSS 变量定义）

前端已拆分为三个子组件：

| 组件 | 文件 | 职责 |
|------|------|------|
| ServiceList | `components/ServiceList.vue` | 服务列表侧边栏，显示已安装和文件中的服务 |
| ConfigForm | `components/ConfigForm.vue` | 配置表单，含环境变量 KV 编辑器和依赖管理 |
| ActionBar | `components/ActionBar.vue` | 操作按钮栏，安装/启动/停止/重启等 |

界面布局分三层：

```
┌──────────────────────────────────────────┐
│ Header: 标题 + 当前配置文件名             │
├──────────┬───────────────────────────────┤
│ Sidebar  │  Main Content                 │
│ 服务列表  │  配置表单（分 5 个 Section）    │
│ (已安装+  │                               │
│  已加载)  │                               │
├──────────┴───────────────────────────────┤
│ Action Bar: New / Open / Save / 操作按钮  │
└──────────────────────────────────────────┘
```

**常见修改场景**：

- **添加新子组件**：在 `frontend/src/components/` 目录创建新的 `.vue` 文件，在 `App.vue` 中导入并使用
- **更换 UI 框架**：安装 Element Plus / Ant Design Vue 等，替换原生 HTML 表单控件
- **添加 Tab 页**：如需要"日志查看"等功能页，在 `app-body` 中用 `v-if/v-show` 切换视图
- **改用 TypeScript**：将 `App.vue` 的 `<script>` 改为 `<script setup lang="ts">`，并创建 `.d.ts` 类型声明
- **调整配色**：修改 `frontend/src/style.css` 中的 CSS 变量（`--bg-primary`, `--accent` 等）

### 4. 修改配置文件格式

**文件**: `internal/config/config.go`

当前使用 JSON 格式，核心入口：`SaveServiceTo` / `LoadDir`（单服务文件）与 `SaveToFile` / `LoadFromFile`（多服务聚合导入导出）。如需改用 YAML/TOML：

1. 安装对应库：`go get gopkg.in/yaml.v3`
2. 替换 `json.MarshalIndent` / `json.Unmarshal` 为 YAML/TOML 的序列化方法
3. 更新 `configs/example.json` 与 `configs/examples/` 的格式和扩展名

### 5. 窗口配置

**文件**: `main.go`

修改窗口标题、尺寸、图标等：

```go
err := wails.Run(&options.App{
    Title:     "你的应用名称",
    Width:     1100,           // 窗口宽度
    Height:    720,            // 窗口高度
    MinWidth:  900,
    MinHeight: 600,
    // ...
})
```

### 6. 多语言支持 (i18n)

项目已集成 `vue-i18n`，语言包位于 `frontend/src/locales/`：

- `zh.json` - 中文
- `en.json` - 英文

切换语言：前端界面右上角的语言切换按钮，或修改 `App.vue` 中的 `locale.value`。

添加新语言：在 `locales/` 目录创建新的 JSON 文件，并在 `main.js` 中注册。

### 7. 关键注意事项

- **管理员权限**：所有 SCM 操作需要管理员权限。开发时以管理员身份运行终端/IDE
- **Go 代理**：国内网络建议设置 `GOPROXY=https://goproxy.cn,direct`
- **WebView2**：目标机器必须有 WebView2 Runtime（Win11 已内置）
- **服务标记**：`[NSSM-Plus]` 前缀是识别已管理服务的唯一标记，修改 `nssmPlusMarker` 常量会影响已有服务的识别
- **Wails 绑定**：`app.go` 中的方法参数/返回值必须是可 JSON 序列化的类型，不支持 `chan`、`func`、`unsafe.Pointer` 等
- **服务删除状态**：卸载服务时，如果遇到 "marked for deletion" 提示，说明该服务已被系统标记为待删除状态（可能由上一次卸载操作残留）。此时服务状态会变为 `Stopped` + `Disabled`，**重启计算机后该服务将被自动彻底删除**，无需手动干预

## 与 NSSM 的对比

| 特性 | NSSM | NSSM Plus |
|------|------|-----------|
| 操作方式 | 命令行 `nssm.exe install` | GUI 界面 + 完整 CLI |
| 配置界面 | Tab 页切换 (5+ 个 Tab) | 单页面，无需切换 |
| 配置迁移 | 无内置支持 | 多服务 JSON 文件统一管理 |
| 界面语言 | 英文 | 多语言 (i18n) |
| 服务包装 | Wrapper 二进制托管进程 | 自托管 Wrapper 模式（同一可执行文件） |
| 日志重定向 | 支持 stdout/stderr 捕获 | 支持，可自定义路径，输出到 `%ProgramData%\NSSM-Plus\logs\` |
| 日志轮转 | 不支持 | 支持，按大小自动轮转，保留 5 份历史 |
| 崩溃重启 | 内置 | 支持，可配置重启延迟 |
| 进程停止 | 直接终止 | 优雅停止 + 可配置超时 + 强制终止 |
| 环境变量 | 支持 | 支持，GUI 可视化编辑 |
| 服务依赖 | 支持 | 支持，GUI 可视化编辑 |
| 密码安全 | 明文存储 | DPAPI 机器级加密存储 |
| CLI 密码安全 | 明文参数 | 支持 --password-stdin / --password-file |
| 工作目录 | 支持 | 支持 |
| 命令行模式 | 仅命令行 | GUI + CLI 双模式，CLI 支持 show/edit/log/export/import |
| 服务状态监控 | 无 | 10 秒自动刷新 |
| BinaryPathName | `nssm.exe` 指向目标程序 | 自身 exe 作为 Wrapper，避免 `syscall.EscapeArg` 问题 |
| 跨平台 | 仅 Windows | 仅 Windows |

## 已完成功能

以下是服务包装器核心功能的实现状态：

- [x] **服务包装器架构** - `internal/wrapper/` 模块，实现 `svc.Handler` 接口
- [x] **日志重定向** - 子进程 stdout/stderr 输出到 `%ProgramData%\NSSM-Plus\logs\<name>.log`
- [x] **自定义日志路径** - `LogStdout` / `LogStderr` 支持自定义路径，空值时使用默认路径
- [x] **日志轮转** - `RotateLog` 按文件大小自动轮转，保留 5 份历史日志
- [x] **环境变量注入** - 通过 `WrapperConfig.Env` 在启动子进程时注入自定义环境变量
- [x] **环境变量编辑器** - 前端可视化 Key-Value 编辑器，支持增删改
- [x] **服务依赖管理** - 前端可视化添加/移除服务依赖项
- [x] **工作目录设置** - 通过 `WrapperConfig.WorkDir` 设置子进程的 `cwd`
- [x] **崩溃自动重启** - `RestartDelay` 配置重启延迟，进程异常退出后自动恢复
- [x] **优雅停止** - 先 `taskkill /T /PID`（可配置超时），再 `taskkill /F /T /PID` 强制终止
- [x] **Wrapper 配置持久化** - `ProgramData/NSSM-Plus/services/<name>.json` 独立存储
- [x] **进程树终止** - 通过 `/T` 参数终止子进程及其所有衍生进程（如 java.exe → 子线程）
- [x] **控制台调试模式** - 非服务环境下可直接运行 wrapper 进行调试，支持 Ctrl+C 优雅退出
- [x] **原生文件对话框** - 使用 Wails 的 `runtime.SaveFileDialog` / `runtime.OpenFileDialog`
- [x] **DisplayName/Description 自动填充** - 输入 ServiceName 后点击对应字段自动填充
- [x] **DPAPI 密码加密** - 使用 Windows DPAPI 机器级加密（`CRYPTPROTECT_LOCAL_MACHINE`），服务账户可正常解密
- [x] **安全密码输入** - CLI 支持 `--password-stdin` 和 `--password-file`，避免密码明文暴露在进程列表
- [x] **命令行模式** - 完整 CLI 支持 install/remove/start/stop/restart/status/show/edit/list/log/export/import
- [x] **CLI --json 输出** - `list` 和 `show` 命令支持 `--json` 格式化输出
- [x] **服务状态自动刷新** - 前端 10 秒定时刷新已安装服务状态
- [x] **前端组件化** - App.vue 拆分为 ServiceList / ConfigForm / ActionBar 子组件
- [x] **共享代码提取** - `internal/common/path.go` 提取 Wrapper BinaryPath 构建和解析函数
- [x] **详细日志记录** - 所有核心分支添加 `log.Printf` 日志，统一 `[模块名]` 前缀
- [x] **配置文件安全** - `SaveToFile` 创建副本加密，不修改输入参数；`LoadFromFile` 兼容三种格式
- [x] **参数转义支持** - `splitArgs` 支持 `\"` 和 `\\` 转义字符
- [x] **服务重启等待** - `Restart()` 等待服务完全停止后再启动（最多 15 秒）

## 待完成功能

- [ ] **服务重命名** - 当前 `Modify` 不支持更改服务名称
- [ ] **系统托盘** - 最小化到系统托盘，后台运行
- [ ] **日志查看页面** - GUI 内直接查看服务日志，支持实时滚动

## License

MIT
