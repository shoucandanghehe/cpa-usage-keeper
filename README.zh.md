<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/keeper-logo-dark.svg" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/keeper-logo-light.svg" />
    <img src="./assets/keeper-logo-light.svg" alt="Keeper" width="560" />
  </picture>
</p>

<p align="center">
  <a href="./README.md">English</a> ｜ <a href="./README.zh.md"><strong>简体中文</strong></a>
</p>

<h1 align="center">CPA Usage Keeper</h1>

<p align="center">万千流转，皆有迹可循。</p>

<p align="center">
  <a href="https://github.com/Willxup/cpa-usage-keeper/releases/latest"><img src="https://img.shields.io/github/v/release/Willxup/cpa-usage-keeper?style=flat-square" alt="最新版本" /></a>
  <a href="https://github.com/Willxup/cpa-usage-keeper/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/Willxup/cpa-usage-keeper/ci.yml?branch=main&amp;style=flat-square&amp;label=CI" alt="CI 状态" /></a>
  <a href="https://github.com/Willxup/cpa-usage-keeper/pkgs/container/cpa-usage-keeper"><img src="https://img.shields.io/badge/Docker-GHCR-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white" alt="GHCR Docker 镜像" /></a>
  <a href="https://github.com/Willxup/homebrew-cpa-usage-keeper"><img src="https://img.shields.io/badge/Homebrew-supported-FBB040?style=flat-square&amp;logo=homebrew&amp;logoColor=black" alt="支持 Homebrew" /></a>
  <a href="https://github.com/Willxup/cpa-usage-keeper/releases/latest"><img src="https://img.shields.io/badge/Linux-FCC624?style=flat-square&amp;logo=linux&amp;logoColor=black" alt="支持 Linux" /></a>
  <a href="https://github.com/Willxup/cpa-usage-keeper/releases/latest"><img src="https://img.shields.io/badge/macOS-A2AAAD?style=flat-square&amp;logo=apple&amp;logoColor=black" alt="支持 macOS" /></a>
  <a href="https://github.com/Willxup/cpa-usage-keeper/releases/latest"><img src="https://img.shields.io/badge/Windows-0078D4?style=flat-square&amp;logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0iI2ZmZiIgZD0iTTIgMy41IDExIDJ2OUgyem0xMC0xLjdMMjIgLjNWMTFIMTJ6TTIgMTJoOXY5TDIgMTkuNXptMTAgMGgxMHYxMC43bC0xMC0xLjV6Ii8+PC9zdmc%2B" alt="支持 Windows" /></a>
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/Willxup/cpa-usage-keeper?style=flat-square" alt="MIT License" /></a>
</p>

CPA Usage Keeper 是面向 [CLIProxyAPI（CPA）](https://github.com/router-for-me/CLIProxyAPI) 的独立用量持久化与分析面板。它将 CPA 用量保存到 SQLite，自动拉取 CPA 配置和凭证数据，并提供用量、成本、请求健康、限额及模型/API 统计。

## 界面预览

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/overview-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/overview-light.png" />
    <img src="./assets/screenshots/overview-light.png" alt="CPA Usage Keeper 总览" width="49%" />
  </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/analysis-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/analysis-light.png" />
    <img src="./assets/screenshots/analysis-light.png" alt="CPA Usage Keeper 分析" width="49%" />
  </picture>
</p>
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/auth-files-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/auth-files-light.png" />
    <img src="./assets/screenshots/auth-files-light.png" alt="CPA Usage Keeper Auth Files" width="49%" />
  </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/ai-provider-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/ai-provider-light.png" />
    <img src="./assets/screenshots/ai-provider-light.png" alt="CPA Usage Keeper AI Provider" width="49%" />
  </picture>
</p>
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/ranking-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/ranking-light.png" />
    <img src="./assets/screenshots/ranking-light.png" alt="CPA Usage Keeper 排名" width="49%" />
  </picture>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/screenshots/login-dark.png" />
    <source media="(prefers-color-scheme: light)" srcset="./assets/screenshots/login-light.png" />
    <img src="./assets/screenshots/login-light.png" alt="CPA Usage Keeper 登录页" width="49%" />
  </picture>
</p>

## 功能特性

- 将 CPA 用量持久保存到 SQLite，并支持可选的定时备份
- 统计请求量、Token、成本、缓存、成功率、RPM/TPM 和延迟，并可按时间、模型、API Key、来源及结果筛选
- 查看和导出请求级事件，并自定义表格列
- 分析用量趋势、成本构成、模型/API Key/AI Provider 占比、时段热力图和延迟诊断
- 按 Auth File 账号和独立账期，将人民币订阅费按各 API Key 的折算费用占比分摊，保存账单快照并导出 CSV
- 监控 Auth Files 与 AI Providers 的用量、健康状态和限额，支持健康巡检与限额刷新
- 可选择加入社区排名，按综合得分、Token、请求量、缓存率、平均 TTFT/延迟或峰值 TPM/RPM 对比表现
- 为单个 CPA API Key 提供独立的只读用量视图
- 自动同步 CPA Auth Files、API Keys 和 AI Providers，并维护模型价格用于成本估算
- 支持 Docker/Docker Compose、Homebrew、二进制和 systemd 部署，并可启用密码保护
- 通过 CPA 插件将 Keeper Dashboard 嵌入 CPAMC

### 订阅费用分账

管理员登录后，在页面菜单打开 **订阅费用分账**（`/billing`；子路径部署会自动加上应用前缀）。

1. 选择一个 Auth File 账号。每个账号独立维护账期，适合多个 Pro 20x 账号在不同日期续费的拼车场景。
2. 新建账期，填写周期序号、精确起止时间、实际人民币费用及费用构成备注。首次从账号的订阅时间预填；后续从上一期结束时间预填，并建议一个自然月后的结束时间（月末向当月最后一天收敛）。**建议时间必须核对，也可以手动校正**；编辑器使用浏览器本地时区，保留秒数。
3. 保存后查看各 API Key 的请求数、成功/失败数、Token 明细、缓存读取率、折算费用、占比和应付金额。每人使用一个 API Key；成员按稳定 Key ID 区分，同名或已删除但仍有历史记录的 Key 不会合并。
4. 使用“导出快照 CSV”在 Excel 中查看或分享结果。修改表单不影响已保存结果；只有明确确认“重新计算并保存”后才覆盖对应账号/周期的快照。刷新只更新账单列表，点击账期可重新选择已保存快照。

分摊口径：

- 精确统计 **`开始时间 <= 请求时间 < 结束时间`**，包含成功和失败请求的已记录用量，同时读取在线与归档事件，不依赖仪表盘的整小时汇总。
- 每个成员的折算费用先保留四位美元小数，再按其占比分摊人民币费用；金额按半偶舍入取整到分，尾差归到折算费用最高的成员（同额按 Key ID 排序）。合计严格等于订阅费；无法非负分配的极小金额会拒绝保存。
- 折算费用使用 Keeper **当前配置的模型价格和定价规则**，仅作分摊权重，不是实际美元支出，也不保证等于供应商官方牌价。
- 账单保存为 SQLite 快照，重启或后续价格/账号信息变化不会自动改写历史账单。主动重算使用当前存量用量和当前定价；重算失败保留旧快照。
- 同一账号的不同账期不可重叠，前一期结束与后一期开始可以相同。缺失模型价格、无法匹配成员 API Key、零用量或零费用权重时拒绝出账，不把缺失值悄悄当作零。
- 只纳入匹配该账号 `auth_index` 的已采集事件；无法归属账号或从未采集的请求无法由分账功能补回。API Key 只读用户无权访问管理员账单。

升级前请备份完整数据目录。首次升级会建立账单表和归档事件的账号/时间索引；归档数据量大时，索引创建可能延长启动时间。删除账单只删除快照，不删除请求用量。

#### 与上游 release 互相切换

已使用对应 Git tag 的后端源码，在同一份隔离数据库验证：

**`v1.15.1` 建库 → 分账版 → 上游 `v1.15.3` → 分账版**

原始/归档用量、应用设置和登录会话均保留；上游阶段可以继续写入用量。上游不提供分账页面/API，但不会删除分账表或改写已保存快照；切回后可直接查看原账单，明确重算后可纳入上游阶段新增的用量。各阶段 SQLite 完整性和外键检查均通过。`v1.15.3` 是 2026-09-11 验证时的最新正式 release。

- 切换前停止当前 Keeper 实例，再启动另一个版本，始终挂载同一个完整数据目录（例如 `./keeper:/data`），保留时区和认证配置。不要让两个实例同时采集同一 CPA。
- 事先备份完整数据目录；正常切换**不需要恢复旧备份、删除迁移记录或清空数据库**。恢复旧备份会丢失备份之后新写入的用量和账单。
- 此兼容范围不等于任意旧版本或未来 release 的保证。升级到尚未验证的 release 前，应先用数据副本重复上述切换演练，尤其检查其数据库迁移。

## 赞助与特别感谢

- 感谢 [CLIProxyAPI（CPA）](https://github.com/router-for-me/CLIProxyAPI) 提供本项目所依赖的上游 CPA 基础与数据来源。
- 感谢 [@YouShouldBetOnMe](https://github.com/YouShouldBetOnMe) 对 CPA Usage Keeper 的支持。
- 感谢 CPA 讨论组（QQ群组）的讨论与反馈。

## 快速开始

> 使用前请确认 CPA 配置已开启 usage 统计：`usage-statistics-enabled: true`。
>
> 同一 CPA 接入多个 usage 采集服务时，请确保均使用订阅模式，否则可能导致收数中断或数据不完整。

Docker Compose 是推荐部署方式：首次部署可同时运行 CPA + Keeper，已有 CPA 时则使用 Keeper-only Compose。

| 场景 | 推荐方式 | 架构 |
| --- | --- | --- |
| 首次部署 CPA + Keeper | [Docker Compose：CPA + Keeper](#docker-compose推荐) | `linux/amd64`、`linux/arm64` |
| 已有 CPA | [Docker Compose：仅 Keeper](#docker-compose推荐) | `linux/amd64`、`linux/arm64` |
| 已有 CPA，偏好 Docker CLI | [Docker](#dockercpa-已在宿主机运行) | `linux/amd64`、`linux/arm64` |
| macOS | [Homebrew](#macos-homebrew) | `amd64`、`arm64` |
| Linux 不使用容器 | [Linux 二进制](#linux-二进制) | `amd64`、`arm64` |
| Windows | [Windows Binary](#windows-binary) | `amd64`、`arm64` |

登录保护默认启用。启动 Keeper 前请配置 `LOGIN_PASSWORD`；只有部署环境已可靠隔离访问时，才显式设置 `AUTH_ENABLED=false`。

## Benchmark

`linux/amd64` 生产型容量测试覆盖持续 ingestion、Dashboard 延迟、CPU 利用率和 Keeper cgroup 峰值内存，完整结果见 [容量 Benchmark 报告](./internal/benchmark/REPORT.zh.md)。

## 项目结构

```text
cmd/server/              应用入口
internal/api/            HTTP 路由与处理器
internal/app/            应用装配与启动
internal/auth/           Session 与访问控制
internal/poller/         CPA 用量与配置同步
internal/repository/     SQLite 持久化与聚合
internal/service/        用量、定价与身份服务
internal/quota/          Provider 限额刷新与巡检
internal/ranking/        社区排名聚合与同步
internal/benchmark/      容量套件、报告、manifest 与历史 Go microbenchmark
deploy/                  部署模板
web/                     React + TypeScript 前端
```

## 本地开发

### 前置依赖

- Go 1.26+
- Node.js 24+
- npm
- 一个可用的 [CLIProxyAPI（CPA）](https://github.com/router-for-me/CLIProxyAPI) 实例

### 本地运行

1. 将 `.env.example` 复制为 `.env`，至少设置 `CPA_BASE_URL` 和 `CPA_MANAGEMENT_KEY`。

```bash
cp .env.example .env
vim .env
```

2. 启动后端。

```bash
go run ./cmd/server/main.go
```

3. 在另一个终端安装前端依赖并启动开发服务器。

```bash
npm --prefix ./web ci
npm --prefix ./web run dev -- --host 127.0.0.1
```

打开 `http://127.0.0.1:5173`。前端默认将 `/api` 代理到 `http://127.0.0.1:8080`；后端使用其它端口时可通过 `VITE_API_PROXY_TARGET` 覆盖。

### 测试

运行完整验证：

```bash
make verify
```

也可以分别运行：

```bash
go test ./cmd/... ./internal/...
npm --prefix ./web run test
npm --prefix ./web run lint
npm --prefix ./web run typecheck
npm --prefix ./web run build
```

## 部署方式

### Docker Compose（推荐）

Docker Compose 同时推荐用于 CPA + Keeper 联合部署和 Keeper 单独部署。

#### CPA + Keeper

将下面内容保存为 `docker-compose.yml`，并替换管理密钥和登录密码：

```yaml
services:
  cli-proxy-api:
    image: eceasy/cli-proxy-api:latest
    container_name: cli-proxy-api
    restart: unless-stopped
    ports:
      - "8317:8317"
      - "1455:1455"
    volumes:
      - ./cpa/config.yaml:/CLIProxyAPI/config.yaml
      - ./cpa/auths:/root/.cli-proxy-api
      - ./cpa/logs:/CLIProxyAPI/logs
    networks:
      - cpa-network

  cpa-usage-keeper:
    image: ghcr.io/willxup/cpa-usage-keeper:latest
    container_name: cpa-usage-keeper
    restart: unless-stopped
    depends_on:
      - cli-proxy-api
    ports:
      - "8080:8080"
    environment:
      TZ: Asia/Shanghai # 设置容器时区，日志时间会按该时区显示。
      CPA_BASE_URL: http://cli-proxy-api:8317
      CPA_MANAGEMENT_KEY: replace-with-your-management-key
      REDIS_QUEUE_ADDR: cli-proxy-api:8317
      AUTH_ENABLED: true
      LOGIN_PASSWORD: ${KEEPER_LOGIN_PASSWORD:?set KEEPER_LOGIN_PASSWORD}
    volumes:
      - ./keeper:/data
    networks:
      - cpa-network

networks:
  cpa-network:
    driver: bridge
```

启动前请在 shell 或 Compose `.env` 文件中设置 `KEEPER_LOGIN_PASSWORD`。

运行 `docker compose up -d` 启动，使用 `docker compose down` 停止。

CPA 数据保存在 `./cpa`，Keeper 数据保存在 `./keeper`。

#### Keeper Only

CPA 已经部署好时，直接使用仓库中的 Keeper-only Compose 模板：

```bash
cp deploy/docker-compose.example.yml docker-compose.yml
cp .env.example .env
vim .env
```

CPA 运行在 Docker 宿主机上时，可从以下配置开始：

```env
CPA_BASE_URL=http://host.docker.internal:8317
CPA_MANAGEMENT_KEY=replace-with-your-management-key
AUTH_ENABLED=true
LOGIN_PASSWORD=
```

启动容器前请设置私有的 `LOGIN_PASSWORD`。

其它网络环境请将 `CPA_BASE_URL` 改为容器可访问的 CPA 地址。CPA 使用非默认 Redis/RESP 地址时，再设置 `REDIS_QUEUE_ADDR`。

运行 `docker compose up -d` 启动 Keeper，使用 `docker compose down` 停止。

模板默认将 Keeper 数据保存在 `./data`。

### Docker（CPA 已在宿主机运行）

偏好使用 `docker run` 时，复用上面 Keeper-only Compose 的 `.env` 配置：

```bash
docker run -d \
  --name cpa-usage-keeper \
  --add-host=host.docker.internal:host-gateway \
  -p 8080:8080 \
  -v "$(pwd)/keeper:/data" \
  --env-file .env \
  ghcr.io/willxup/cpa-usage-keeper:latest
```

### macOS Homebrew

Homebrew 是 macOS 推荐安装方式：

```bash
brew tap Willxup/cpa-usage-keeper
brew install cpa-usage-keeper
```

设置 `CPA_BASE_URL`、`CPA_MANAGEMENT_KEY` 和私有的 `LOGIN_PASSWORD`，然后启动服务：

```bash
vim "$(brew --prefix)/etc/cpa-usage-keeper.env"
brew services start cpa-usage-keeper
```

升级和服务管理命令：

```bash
brew services list
brew services restart cpa-usage-keeper
brew update
brew upgrade cpa-usage-keeper
```

数据保存在 `$(brew --prefix)/var/cpa-usage-keeper`，日志写入 `$(brew --prefix)/var/log/`。

### Linux 二进制

从 [Releases](https://github.com/Willxup/cpa-usage-keeper/releases/latest) 下载 `linux_amd64` 或 `linux_arm64` 压缩包，然后解压并运行：

```bash
mkdir -p cpa-usage-keeper
tar -xzf ./cpa-usage-keeper_*_linux_*.tar.gz -C cpa-usage-keeper --strip-components=1
cd cpa-usage-keeper
cp .env.example .env
vim .env
./cpa-usage-keeper
```

#### systemd

Linux 压缩包内置 service 模板。请在解压后的目录中运行：

```bash
sudo cp cpa-usage-keeper.service /etc/systemd/system/cpa-usage-keeper.service
sudo sed -i "s|__CPA_USAGE_KEEPER_DIR__|$(pwd)|g" /etc/systemd/system/cpa-usage-keeper.service
sudo systemctl daemon-reload
sudo systemctl enable --now cpa-usage-keeper
```

```bash
sudo systemctl status cpa-usage-keeper
sudo journalctl -u cpa-usage-keeper -f
sudo systemctl restart cpa-usage-keeper
```

### 命令行参数

二进制支持以下可选启动参数：

```bash
cpa-usage-keeper --host 127.0.0.1 # 仅为当前进程覆盖 APP_HOST。
cpa-usage-keeper -v               # 输出构建版本并退出；也支持 --version。
```

### Windows Binary

从 [Releases](https://github.com/Willxup/cpa-usage-keeper/releases/latest) 下载 `windows_amd64` 或 `windows_arm64` ZIP 并解压。在 PowerShell 中进入解压目录后运行：

```powershell
Copy-Item .env.example .env
notepad .env
.\cpa-usage-keeper.exe
```

启动前请设置 `CPA_BASE_URL`、`CPA_MANAGEMENT_KEY` 和私有的 `LOGIN_PASSWORD`。认证默认启用；只有隔离部署才显式设置 `AUTH_ENABLED=false`。

## 配置

复制配置模板：

```bash
cp .env.example .env
```

新手部署时优先看“最小必填”和“Web 访问与反代”两组，其它配置保持默认即可。

### 最小必填

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `CPA_BASE_URL` | 是 | - | Keeper 服务端访问 CPA 的地址。Docker Compose 内通常是 `http://cli-proxy-api:8317`，可以是内网地址或容器服务名 |
| `CPA_MANAGEMENT_KEY` | 是 | - | CPA management key，用于读取 CPA 管理接口数据 |

### Web 访问与反代

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `APP_HOST` | 否 | 所有接口 | Keeper HTTP 监听主机；原生部署仅允许本机访问时可设为 `127.0.0.1` |
| `APP_PORT` | 否 | `8080` | Keeper HTTP 监听端口 |
| `APP_BASE_PATH` | 否 | 根路径 | Keeper 子路径部署前缀，例如 `/keeper`；留空表示部署在 `/` |
| `CPA_PUBLIC_URL` | 否 | 当前浏览器同源根路径 | 浏览器访问 CPA 的公开地址，用于“返回 CPA”跳转和 CPAMC frame 信任来源 |
| `TRUSTED_PROXY_CIDRS` | 否 | 仅本机 loopback | 允许提供 `X-Forwarded-For` 的额外反向代理 CIDR，多个值用逗号分隔 |

- 启动参数 `--host` 的优先级高于 `APP_HOST`。两者都未设置时，Keeper 保持现有行为，监听所有可用网络接口。
- Docker/Compose 请保持 `APP_HOST` 为空；如需仅允许 Docker 宿主机访问，请将端口发布为 `127.0.0.1:8080:8080`。
- `APP_BASE_PATH` 必须为空或以 `/` 开头；`/cpa/` 会规范为 `/cpa`。
- `CPA_BASE_URL` 是服务端访问 CPA 的地址，可以使用内网地址或 Docker 服务名。
- `CPA_PUBLIC_URL` 控制浏览器跳转和跨域 CPAMC frame 信任。同源且 CPA 位于 `/management.html` 时可留空；域名、端口或路径不同时应设置公开 CPA 地址。
- Keeper 只信任本机 loopback 和 `TRUSTED_PROXY_CIDRS` 提供的 `X-Forwarded-For`；直连客户端不能通过该请求头切换登录限流来源。只配置实际代理地址或网段，全网 CIDR 会被拒绝。

跨域嵌入 CPAMC 时，`CPA_PUBLIC_URL` 必须是带 host 的完整 `http://` 或 `https://` URL；相对路径只影响浏览器跳转。

### 登录保护

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `AUTH_ENABLED` | 否 | `true` | 是否启用登录保护 |
| `LOGIN_PASSWORD` | 鉴权启用时必填 | - | 登录密码 |
| `AUTH_SESSION_TTL` | 否 | `168h` | 登录 session 有效时长 |
| `API_KEY_VIEWER_LOCAL_RANKING_ENABLED` | 否 | `false` | 允许 API Key 登录用户只读查看本地排行；Community 排行始终只读 |

### 时区与请求行为

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `TZ` | 否 | `Asia/Shanghai` | 统计和展示使用的时区；Today、按天统计、页面时间、日志时间和每日清理时间都会按这个时区计算 |
| `REQUEST_TIMEOUT` | 否 | `30s` | 请求 CPA HTTP 接口和 Redis 队列的超时时间 |
| `TLS_SKIP_VERIFY` | 否 | `false` | 跳过 CPA HTTPS 和 Redis 队列 TLS 的证书验证；仅在使用自签名证书时启用 |

### Auth Files 限额刷新

Auth Files 定时限额刷新在 Auth Files 巡检弹窗的小齿轮中配置。设置保存在本地 SQLite，不依赖页面保持打开。

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `QUOTA_REFRESH_WORKER_LIMIT` | 否 | `10` | 手动刷新和定时刷新共用的 Auth Files 限额刷新队列最大并发数，最大 `100` |
| `QUOTA_UPSTREAM_RESPONSES_ENABLED` | 否 | `false` | 缓存每个凭证最近一次限额查询的原始上游响应，并通过 quota task/cache API 返回，供浏览器 Network 面板排障；响应可能包含账号数据 |

### Redis 队列高级配置

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `REDIS_QUEUE_ADDR` | 否 | `CPA_BASE_URL` 主机名 + `8317` | CPA Redis/RESP TCP 地址；一般保持空即可。非默认端口或单独暴露 Redis stream 时填写 `host:port` |
| `REDIS_QUEUE_TLS` | 否 | `false` | 是否使用 TLS 连接 Redis 队列；显式设置 `REDIS_QUEUE_ADDR` 且需要 TLS 时设为 `true` |
| `REDIS_QUEUE_BATCH_SIZE` | 否 | `10000` | 每次最多拉取的队列记录数 |
| `REDIS_QUEUE_IDLE_INTERVAL` | 否 | `1s` | 队列为空时的检查间隔 |

### 存储、日志与备份

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `WORK_DIR` | 否 | `./data` | 应用工作目录；数据库、日志和备份默认分别写入 `app.db`、`logs/`、`backups/` |
| `LOG_LEVEL` | 否 | `info` | 日志级别 |
| `LOG_FILE_ENABLED` | 否 | `true` | 是否写入持久化日志文件 |
| `LOG_RETENTION_DAYS` | 否 | `7` | 综合日志保留历史天数，并额外保留当天；`0` 表示不自动清理。仅错误日志固定保留历史 30 天及当天 |
| `BACKUP_ENABLED` | 否 | `true` | 是否启用 SQLite 数据库备份 |
| `BACKUP_INTERVAL` | 否 | `24h` | 数据库备份间隔 |
| `BACKUP_RETENTION_DAYS` | 否 | `7` | 备份保留天数 |

Keeper 会在每天 04:30 的维护窗口中，把早于 90 个本地自然日的原始 `usage_events` 自动移动到永久保留的 `usage_events_archive` 冷表。该冷表用于 schema migration 重建及订阅分账的精确历史查询；普通仪表盘 API 不查询 archive。

启用文件日志后，`cpa-usage-keeper-YYYY-MM-DD.log` 会记录所有已输出级别；error、fatal 和 panic 级别还会同时写入 `cpa-usage-keeper-error-YYYY-MM-DD.log`，该文件固定保留历史 30 个本地自然日及当天。

### 内置 HTTPS

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `TLS_ENABLED` | 否 | `false` | 是否让 Keeper 自己启用 HTTPS/TLS |
| `TLS_CERT_FILE` | 启用 TLS 时必填 | - | HTTPS 证书文件路径 |
| `TLS_KEY_FILE` | 启用 TLS 时必填 | - | HTTPS 私钥文件路径 |

通常建议在 nginx、Caddy 等反向代理层处理 HTTPS。只有需要 Keeper 进程直接提供 HTTPS 时，才设置 `TLS_ENABLED=true`，并填写 `TLS_CERT_FILE` 和 `TLS_KEY_FILE`；相对路径会按 `.env` 所在目录解析。

安全与数据说明：

- 浏览器 API 会脱敏 key 类字段，但 SQLite 数据库及其未加密备份仍包含原始数据。
- 认证默认启用。若显式关闭，请在部署边界限制 Keeper 访问；公网访问应在反向代理层启用 HTTPS。
- 登录 session hash 会保存在 SQLite 中，直到用户退出或超过 `AUTH_SESSION_TTL`。
- CPAMC 使用独立的 embed session：优先使用 `HttpOnly` Cookie，不可用时回退到保存在浏览器 session storage 中的单标签页请求头 token。
- 同源嵌入默认可用；跨域嵌入时，将 `CPA_PUBLIC_URL` 设置为用于 `frame-ancestors` 的公开 CPA/CPAMC 来源。
- Redis inbox 消息成功后保留到当天结束，失败后保留 7 天。

## Nginx 反向代理

部署到 `/cpa` 时设置 `APP_BASE_PATH=/cpa`，并在反向代理中保留该前缀：

```nginx
location /cpa/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

上面的本机 Nginx 配置无需额外设置 Keeper。若反向代理通过容器或其它主机访问 Keeper，或反向代理前面还有 Cloudflare 等 CDN，请加入准确的代理网段，例如 `TRUSTED_PROXY_CIDRS=172.18.0.0/16`。

CPA 与 Keeper 浏览器同源时，可以不设置 `CPA_PUBLIC_URL`，“返回 CPA”默认使用 `/management.html`。CPA 位于其它域名、端口或路径时，设置公开地址：

```env
CPA_PUBLIC_URL=https://cpa.example.com
```

## License

本项目基于 [MIT License](./LICENSE) 开源。
