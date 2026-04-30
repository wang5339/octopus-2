<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**面向个人与小团队的自托管 AI API 网关，统一聚合多家模型服务、协议转换、负载均衡、模型管理与用量统计**

简体中文 | [English](README.md)

</div>


## 🐙 项目简介

Octopus 是一个自托管的 LLM API 聚合网关。它可以把 OpenAI 兼容接口、Anthropic、Gemini、Volcengine、OpenCode Zen 等不同上游统一接入到一个管理面板中，并对外提供兼容 OpenAI / Anthropic 使用习惯的 API。

你可以通过 Octopus 统一管理渠道、Base URL、API Key、模型列表、分组路由、负载均衡策略和模型价格；同时还能查看请求日志、Token 消耗、费用统计、渠道健康状态和 API 使用文档。适合个人自用、多模型聚合、API Key 统一分发、模型路由实验和轻量级团队网关场景。


## ✨ 特性

- 🔀 **多上游聚合** - 统一接入 OpenAI 兼容接口、Anthropic、Gemini、Volcengine、OpenCode Zen 等渠道
- 🔄 **协议转换** - 支持 OpenAI Chat、OpenAI Responses、Anthropic Messages、Embeddings、Image Generation 等请求格式转换
- 🧠 **模型级协议覆盖** - 同一渠道内可为不同模型指定不同出站协议，适配混合模型网关和 Zen 类路由
- 🔑 **多 Key 管理** - 单渠道支持多个 API Key，可按运行状态自动选择可用 Key
- ⚡ **智能端点优选** - 单渠道可配置多个 Base URL，并优先使用延迟更低的端点
- ⚖️ **负载均衡与容错** - 支持轮询、随机、故障转移、加权分配，并包含重试、熔断和会话保持能力
- 🔃 **模型同步与检测** - 自动同步上游模型，支持手动检测新增 / 消失模型、忽略规则和一键应用
- 📊 **统计与日志** - 统计请求量、Token、费用、耗时，并按总量、渠道、API Key、模型分组等维度展示
- 📚 **API 使用文档** - 内置 OpenAI / Anthropic 等调用示例，方便复制到客户端或 CLI 工具
- 🎨 **Web 管理面板** - 提供渠道、分组、模型价格、日志、设置、外观和 API 文档等可视化管理
- 🗄️ **多数据库支持** - 支持 SQLite、MySQL、PostgreSQL，Docker Compose 默认使用本地持久化数据目录


## 🚀 Docker Compose 部署

Octopus 推荐使用 Docker Compose 部署。该方式会自动构建镜像、启动服务，并将运行数据持久化到本机 `./data` 目录。

### 1. 环境要求

- 已安装 Docker
- 已安装 Docker Compose（Docker Desktop 通常已内置）

### 2. 克隆项目

```bash
git clone https://github.com/wang5339/octopus-2.git
cd octopus-2
```

### 3. 启动服务

```bash
docker compose up -d --build
```

启动后访问：

```text
http://localhost:8080
```

### 4. 查看运行状态

```bash
docker compose ps
docker compose logs -f octopus
```

### 5. 停止或重启服务

```bash
# 停止服务
docker compose down

# 重启服务
docker compose restart
```

### 6. 数据持久化

`docker-compose.yml` 默认挂载：

```yaml
volumes:
  - './data:/app/data'
```

因此配置文件和 SQLite 数据库会保存在项目根目录的 `data` 目录中。

### 🔐 默认账户

首次启动后，访问 http://localhost:8080 使用以下默认账户登录管理面板：

- **用户名**：`admin`
- **密码**：`admin`

> ⚠️ **安全提示**：请在首次登录后立即修改默认密码。

### 📝 配置文件

配置文件默认位于 `data/config.json`，首次启动时自动生成。

**完整配置示例：**

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "type": "sqlite",
    "path": "data/data.db"
  },
  "log": {
    "level": "info"
  }
}
```

**配置项说明：**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.port` | 服务端口 | `8080` |
| `database.type` | 数据库类型 | `sqlite` |
| `database.path` | 数据库连接地址 | `data/data.db` |
| `log.level` | 日志级别 | `info` |

**数据库配置：**

支持三种数据库：

| 类型 | `database.type` | `database.path` 格式 |
|------|-----------------|---------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL 配置示例：**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL 配置示例：**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **提示**：MySQL 和 PostgreSQL 需要先手动创建数据库，程序会自动创建表结构。

**环境变量：**

所有配置项均可通过环境变量覆盖，格式为 `OCTOPUS_` + 配置路径（用 `_` 连接）：

| 环境变量 | 对应配置项 |
|----------|-----------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE` | 最大 SSE 事件大小(可选) |


## 📸 界面预览

### 🖥️ 桌面端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="首页" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="渠道" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="分组" width="400"></td>
</tr>
<tr>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="价格" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="日志" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="设置" width="400"></td>
</tr>
</table>
</div>

### 📱 移动端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="移动端首页" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="移动端渠道" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="移动端分组" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="移动端价格" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="移动端日志" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="移动端设置" width="140"></td>
</tr>
</table>
</div>


## 📖 功能说明

### 📡 渠道管理

渠道是连接 LLM 供应商和模型服务的基础配置单元。一个渠道可以配置类型、Base URL、多个 API Key、代理、自定义 Header、模型列表、模型协议覆盖和上游模型同步策略。

**支持能力：**

- Provider 预设：快速填充常见渠道类型和 Base URL
- 模型拉取：从上游获取模型列表并选择写入渠道
- 模型测试：在保存前或保存后测试模型是否可用
- 上游更新检测：检测新增模型和消失模型，新增可追加，消失需手动确认删除
- 忽略规则：支持精确模型名和 `regex:` 正则规则，跳过不想自动同步的模型
**Base URL 说明：**

程序会根据渠道类型自动补全 API 路径，您只需填写基础 URL 即可：

| 渠道类型 | 自动补全路径 | 填写 URL | 完整请求地址示例 |
|----------|-------------|----------|-----------------|
| OpenAI Chat | `/chat/completions` | `https://api.openai.com/v1` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/responses` | `https://api.openai.com/v1` | `https://api.openai.com/v1/responses` |
| Anthropic | `/messages` | `https://api.anthropic.com/v1` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/models/:model:generateContent` | `https://generativelanguage.googleapis.com/v1beta` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |
| Volcengine | `/responses` | 供应商提供的基础 URL | `{base_url}/responses` |
| OpenAI Embedding | `/embeddings` | `https://api.openai.com/v1` | `https://api.openai.com/v1/embeddings` |
| OpenAI Image Generation | `/images/generations` | `https://api.openai.com/v1` | `https://api.openai.com/v1/images/generations` |
| OpenCode Zen | 按模型动态选择 Chat / Responses / Anthropic / Gemini 路径 | Zen 服务基础 URL | 根据模型名前缀自动路由 |

> 💡 **提示**：填写 Base URL 时无需包含具体的 API 端点路径，程序会自动处理。

---

### 📁 分组管理

分组用于将多个渠道聚合为一个统一的对外模型名称。

**核心概念：**

- **分组名称** 即程序对外暴露的模型名称
- 调用 API 时，将请求中的 `model` 参数设置为分组名称即可

**负载均衡模式：**

| 模式 | 说明 |
|------|------|
| 🔄 **轮询** | 每次请求依次切换到下一个渠道 |
| 🎲 **随机** | 每次请求随机选择一个可用渠道 |
| 🛡️ **故障转移** | 优先使用高优先级渠道，仅当其故障时才切换到低优先级渠道 |
| ⚖️ **加权分配** | 根据渠道设置的权重比例分配请求 |

> 💡 **示例**：创建分组名称为 `gpt-4o`，将多个供应商的 GPT-4o 渠道加入该分组，即可通过统一的 `model: gpt-4o` 访问所有渠道。

---

### 💰 价格管理

管理系统中的模型价格信息。

**数据来源：**

- 系统会定期从 [models.dev](https://github.com/sst/models.dev) 同步更新模型价格数据
- 当创建渠道时，若渠道包含的模型不在 models.dev 中，系统会自动在此页面创建该模型的价格信息,所以此页面显示的是没有从上游获取到价格的模型，用户可以手动设置价格
- 也支持手动创建 models.dev 中已存在的模型，用于自定义价格

**价格优先级：**

| 优先级 | 来源 | 说明 |
|:------:|------|------|
| 🥇 高 | 本页面 | 用户在价格管理页面设置的价格 |
| 🥈 低 | models.dev | 自动同步的默认价格 |

> 💡 **提示**：如需覆盖某个模型的默认价格，只需在价格管理页面为其设置自定义价格即可。

---

### ⚙️ 设置

系统全局配置项。

**统计保存周期（分钟）：**

由于程序涉及大量统计项目，若每次请求都直接写入数据库会影响读写性能。因此程序采用以下策略：

- 统计数据先保存在 **内存** 中
- 按设定的周期 **定期批量写入** 数据库

> ⚠️ **重要提示**：退出程序时，请使用正常的关闭方式（如 `Ctrl+C` 或发送 `SIGTERM` 信号），以确保内存中的统计数据能正确写入数据库。**请勿使用 `kill -9` 等强制终止方式**，否则可能导致统计数据丢失。




## 🔌 客户端接入

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:8080/v1",   
    api_key="sk-octopus-your-api-key",
)
completion = client.chat.completions.create(
    model="octopus-openai",  // 填写正确的分组名称
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

编辑 `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8080",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-your-api-key",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_SMALL_FAST_MODEL": "octopus-haiku-4-5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "octopus-haiku-4-5"
  }
}
```

### Codex

编辑 `~/.codex/config.toml`

```toml
model = "octopus-codex" # 填写正确的分组名称

model_provider = "octopus"

[model_providers.octopus]
name = "octopus"
base_url = "http://127.0.0.1:8080/v1"
```
编辑 `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": "sk-octopus-your-api-key"
}
```


---

## 🤝 致谢

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - 本项目的 LLM API 适配模块直接源自该仓库的实现
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI 模型数据库，提供模型价格数据
