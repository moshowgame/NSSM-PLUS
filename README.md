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
- **单页面操作** - 左侧服务列表 + 右侧配置表单，无 Tab 切换
- **完整服务管理** - 安装、修改、启动、停止、重启、删除服务
- **多服务配置文件** - 一份 JSON 文件统一管理多个服务配置，方便批量导入/导出
- **服务状态监控** - 实时显示 Running / Stopped 等状态
- **暗色主题** - 现代化深色 UI

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
├── main.go                       # 入口：检测 service 模式 / 启动 Wails GUI
├── app.go                        # 前后端桥接层，暴露给前端的 Go 方法
├── go.mod / go.sum               # Go 模块依赖
├── wails.json                    # Wails 项目配置
│
├── internal/                     # 后端核心逻辑（不直接暴露给前端）
│   ├── service/
│   │   └── manager.go            # Windows SCM 服务管理（安装/删除/启停/查询/修改）
│   ├── wrapper/
│   │   ├── config.go             # Wrapper 配置文件管理（ProgramData 持久化）
│   │   └── wrapper.go            # Windows 服务包装器（进程托管/日志/停止）
│   └── config/
│       └── config.go             # 配置文件序列化（JSON 导入/导出）
│
├── frontend/                     # 前端源码
│   ├── index.html                # HTML 入口
│   ├── package.json              # npm 依赖
│   ├── vite.config.js            # Vite 构建配置
│   ├── tsconfig.json             # TypeScript 配置
│   └── src/
│       ├── main.js               # Vue 应用挂载
│       ├── style.css             # 全局样式 + CSS 变量（暗色主题）
│       └── App.vue               # 唯一的 Vue 组件（全部 UI 逻辑）
│
├── build/
│   └── appicon.png               # 应用图标
│
├── configs/
│   └── example.json              # 示例服务配置文件
│
└── .gitignore
```

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     WebView2 窗口                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Vue 3 前端 (App.vue)                       │ │
│  │  ┌──────────┐  ┌────────────┐  ┌──────────────┐        │ │
│  │  │ 服务列表  │  │ 配置表单    │  │  操作按钮栏   │        │ │
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
│  └────┬─────────┬───┘  │  ├─ GUI 模式（默认）     │            │
│       │         │       │  │  → wails.Run()       │            │
│  ┌────┴────┐ ┌─┴───┐   │  └──────────────────────┘            │
│  │ service │ │config│   └────────────────────────┘             │
│  │ manager │ │      │                                         │
│  │ (SCM)   │ │(JSON)│                                         │
│  └────┬────┘ └──────┘                                         │
│       │                                                        │
│  ┌────┴──────────┐                                             │
│  │    wrapper     │  ← 服务包装器核心                          │
│  │  config.go     │  WrapperConfig → ProgramData/NSSM-Plus/   │
│  │  wrapper.go    │  svc.Handler → 启动/监控/停止子进程         │
│  └───────────────┘                                             │
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
    AppPath   string            `json:"appPath"`   // 应用程序路径
    Arguments string            `json:"arguments"` // 启动参数
    WorkDir   string            `json:"workDir"`   // 工作目录
    Env       map[string]string `json:"env"`       // 环境变量
}
```

### 前后端通信

Wails 框架在编译时自动生成 Go → JS 绑定代码。前端通过 `window.go.main.App.xxx()` 调用后端方法：

```javascript
// frontend/src/App.vue 中的调用方式
window.go.main.App.InstallService(config)   // 安装服务
window.go.main.App.StartService(name)       // 启动服务
window.go.main.App.GetInstalledServices()   // 获取服务列表
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
    Password       string            `json:"password"`       // 账户密码
    Environment    map[string]string `json:"environment"`    // 环境变量
    LogStdout      string            `json:"logStdout"`      // 标准输出日志路径
    LogStderr      string            `json:"logStderr"`      // 标准错误日志路径
    RotateLog      bool              `json:"rotateLog"`      // 日志轮转
    RestartDelay   int               `json:"restartDelay"`   // 崩溃后重启延迟(秒)
    Dependencies   []string          `json:"dependencies"`   // 依赖服务
}
```

该结构体同时用于 JSON 配置文件存储和前后端数据传输。

配置文件以多服务格式存储，一个 JSON 文件包含所有服务定义：

```json
{
  "services": [
    {
      "serviceName": "MyAppService",
      "appPath": "C:\\path\\to\\app.exe",
      ...
    },
    {
      "serviceName": "AnotherService",
      "appPath": "C:\\path\\to\\another.exe",
      ...
    }
  ]
}
```

加载配置文件后，侧栏会合并显示已安装服务（带状态）和文件中的未安装服务（标记为 "Not Installed" + "File" 标签），可逐个点击查看并安装。向后兼容旧的单服务格式和裸数组格式。

## 环境要求

- **操作系统**: Windows 10 / 11 (需要 WebView2 Runtime)
- **Go**: 1.22+
- **Node.js**: 18+ (用于前端构建)
- **权限**: 管理员权限 (服务管理操作需要)

> Windows 11 和 Windows 10 (21H2+) 通常已内置 WebView2 Runtime。旧版本系统需手动安装：https://developer.microsoft.com/en-us/microsoft-edge/webview2/

## 快速开始

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

1. **以管理员身份运行** `nssm-plus.exe`
2. 点击 **Open Config** 加载多服务配置文件（或点击 **New Config** 直接填写表单创建新服务）
3. 侧栏会显示已安装服务和文件中未安装的服务（带 "File" 标签）
4. 点击侧栏服务查看/编辑配置，点击 **Install** 安装
5. 已安装服务可通过 **Reconfigure** 重新配置（自动停止→修改→启动）
6. 使用 **Start / Stop / Restart** 控制服务运行
7. **Uninstall** 卸载服务，**Delete** 仅清空当前表单
8. 点击 **Save Config** 将所有已管理服务保存到一份 JSON 文件
9. 点击 **Debug** 输出调试信息到控制台（按 F12 查看）

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
- **自定义日志路径**：当前日志固定输出到 `%ProgramData%\NSSM-Plus\logs\`，可在 `WrapperConfig` 中增加 `LogPath` 字段，并修改 `wrapper.go` 的日志重定向逻辑
- **实现崩溃重启**：在 `wrapper.go` 的 `Execute()` 方法中，当子进程非正常退出时，按 `RestartDelay` 延迟后重新启动
- **修改停止策略**：当前使用 `taskkill /T /PID`（优雅 5s + 强制），可在 `stopProcess()` 函数中调整超时时间和终止策略

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

**文件**: `frontend/src/App.vue`（模板 + 脚本 + 样式）
**全局样式**: `frontend/src/style.css`（CSS 变量定义）

当前所有 UI 逻辑集中在一个 `App.vue` 组件中。界面布局分三层：

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

- **拆分组件**：将 `ServiceList`、`ConfigForm`、`ActionBar` 拆分为独立的 `.vue` 文件，放入 `frontend/src/components/` 目录
- **更换 UI 框架**：安装 Element Plus / Ant Design Vue 等，替换原生 HTML 表单控件
- **添加 Tab 页**：如需要"日志查看"等功能页，在 `app-body` 中用 `v-if/v-show` 切换视图
- **改用 TypeScript**：将 `App.vue` 的 `<script>` 改为 `<script setup lang="ts">`，并创建 `.d.ts` 类型声明
- **调整配色**：修改 `frontend/src/style.css` 中的 CSS 变量（`--bg-primary`, `--accent` 等）

### 4. 修改配置文件格式

**文件**: `internal/config/config.go`

当前使用 JSON 格式。如需改用 YAML/TOML：

1. 安装对应库：`go get gopkg.in/yaml.v3`
2. 替换 `json.MarshalIndent` / `json.Unmarshal` 为 YAML/TOML 的序列化方法
3. 更新 `configs/example.json` 的格式和扩展名

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

### 6. 添加多语言支持 (i18n)

1. 安装 `vue-i18n`：`npm install vue-i18n`
2. 在 `frontend/src/` 下创建 `locales/zh.json` 和 `locales/en.json`
3. 在 `main.js` 中配置 i18n 插件
4. 在 `App.vue` 中将硬编码文本替换为 `$t('key')`

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
| 操作方式 | 命令行 `nssm.exe install` | GUI 界面直接操作 |
| 配置界面 | Tab 页切换 (5+ 个 Tab) | 单页面，无需切换 |
| 配置迁移 | 无内置支持 | 多服务 JSON 文件统一管理 |
| 界面语言 | 英文 | 可扩展多语言 |
| 服务包装 | Wrapper 二进制托管进程 | 自托管 Wrapper 模式（同一可执行文件） |
| 日志重定向 | 支持 stdout/stderr 捕获 | 已实现，输出到 `%ProgramData%\NSSM-Plus\logs\` |
| 崩溃重启 | 内置 | 字段已预留（待实现） |
| 进程停止 | 直接终止 | 优雅停止（taskkill /T）+ 强制终止（5s 超时后 taskkill /F） |
| 环境变量 | 支持 | 已实现，通过 WrapperConfig.Env 注入 |
| 工作目录 | 支持 | 已实现，通过 WrapperConfig.WorkDir 设置 |
| BinaryPathName | `nssm.exe` 指向目标程序 | 自身 exe 作为 Wrapper，避免 `syscall.EscapeArg` 问题 |
| 跨平台 | 仅 Windows | 仅 Windows |

## 已完成功能

以下是服务包装器核心功能的实现状态：

- [x] **服务包装器架构** - `internal/wrapper/` 模块，实现 `svc.Handler` 接口
- [x] **日志重定向** - 子进程 stdout/stderr 输出到 `%ProgramData%\NSSM-Plus\logs\<name>.log`
- [x] **环境变量注入** - 通过 `WrapperConfig.Env` 在启动子进程时注入自定义环境变量
- [x] **工作目录设置** - 通过 `WrapperConfig.WorkDir` 设置子进程的 `cwd`
- [x] **优雅停止** - 先 `taskkill /T /PID`（5s 超时），再 `taskkill /F /T /PID` 强制终止
- [x] **Wrapper 配置持久化** - `ProgramData/NSSM-Plus/services/<name>.json` 独立存储
- [x] **进程树终止** - 通过 `/T` 参数终止子进程及其所有衍生进程（如 java.exe → 子线程）
- [x] **控制台调试模式** - 非服务环境下可直接运行 wrapper 进行调试
- [x] **原生文件对话框** - 使用 Wails 的 `runtime.SaveFileDialog` / `runtime.OpenFileDialog`
- [x] **DisplayName/Description 自动填充** - 输入 ServiceName 后点击对应字段自动填充

## 待完成功能

- [ ] **日志轮转** - `RotateLog` 字段已定义，需实现按大小/日期切割日志文件
- [ ] **崩溃自动重启** - `RestartDelay` / `RestartTimeout` 字段已定义，需实现进程监控和自动重启
- [ ] **自定义日志路径** - `LogStdout` / `LogStderr` 字段已定义，当前固定输出到 ProgramData，需支持自定义路径
- [ ] **服务重命名** - 当前 `Modify` 不支持更改服务名称
- [ ] **多语言支持 (i18n)** - UI 文本硬编码为中文/英文混合
- [ ] **系统托盘** - 最小化到系统托盘，后台运行

## License

MIT
