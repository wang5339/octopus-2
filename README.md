<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus
1
**A self-hosted AI API gateway for individuals and small teams, unifying multi-provider model access, protocol conversion, load balancing, model management, and usage analytics**

 English | [简体中文](README_zh.md)

</div>


## 🐙 Overview

Octopus is a self-hosted LLM API aggregation gateway. It brings OpenAI-compatible APIs, Anthropic, Gemini, Volcengine, GitHub Copilot, Antigravity, OpenCode Zen, and other upstream providers into one management panel, while exposing APIs that follow familiar OpenAI / Anthropic usage patterns.

With Octopus, you can manage channels, Base URLs, API keys, model lists, group routing, load-balancing policies, and model prices in one place. It also provides request logs, token usage, cost analytics, channel health information, and built-in API usage documentation. It is suitable for personal use, multi-model aggregation, centralized API key distribution, model routing experiments, and lightweight team gateway scenarios.


## ✨ Features

- 🔀 **Multi-provider aggregation** - Connect OpenAI-compatible APIs, Anthropic, Gemini, Volcengine, GitHub Copilot, Antigravity, OpenCode Zen, and more
- 🔄 **Protocol conversion** - Convert between OpenAI Chat, OpenAI Responses, Anthropic Messages, Embeddings, Image Generation, and related request formats
- 🧠 **Per-model protocol overrides** - Assign different outbound protocols to different models inside the same channel, useful for mixed-model gateways and Zen-style routing
- 🔑 **Multi-key management** - Configure multiple API keys per channel and automatically select available keys based on runtime state
- 🪪 **OAuth helpers** - Built-in entry points for GitHub Copilot Device Flow and Antigravity Web OAuth
- ⚡ **Smart endpoint selection** - Configure multiple Base URLs per channel and prefer lower-latency endpoints
- ⚖️ **Load balancing and resilience** - Supports round robin, random, failover, weighted routing, retries, circuit breaking, and sticky sessions
- 🔃 **Model sync and detection** - Sync upstream models, detect added / removed models, configure ignore rules, and apply updates manually
- 📊 **Analytics and logs** - Track requests, tokens, costs, latency, and view statistics by total, channel, API key, and model group
- 📚 **API usage docs** - Built-in OpenAI / Anthropic examples for quick copy into clients or CLI tools
- 🎨 **Web management panel** - Visual management for channels, groups, model prices, logs, settings, appearance, and API documentation
- 🗄️ **Multi-database support** - Supports SQLite, MySQL, and PostgreSQL, with Docker Compose using a local persistent data directory by default


## 🚀 Docker Compose Deployment

Octopus recommends Docker Compose for deployment. This method builds the image, starts the service, and persists runtime data to the local `./data` directory.

### 1. Requirements

- Docker installed
- Docker Compose installed (usually bundled with Docker Desktop)

### 2. Clone the repository

```bash
git clone https://github.com/wang5339/octopus-2.git
cd octopus-2
```

### 3. Start the service

```bash
docker compose up -d --build
```

After startup, visit:

```text
http://localhost:8080
```

### 4. Check service status

```bash
docker compose ps
docker compose logs -f octopus
```

### 5. Stop or restart the service

```bash
# Stop the service
docker compose down

# Restart the service
docker compose restart
```

### 6. Data persistence

`docker-compose.yml` mounts the local data directory by default:

```yaml
volumes:
  - './data:/app/data'
```

Configuration files and the SQLite database are stored in the `data` directory under the project root.

### 🔐 Default Credentials

After first launch, visit http://localhost:8080 and log in to the management panel with:

- **Username**: `admin`
- **Password**: `admin`

> ⚠️ **Security Notice**: Please change the default password immediately after first login.

### 📝 Configuration File

The configuration file is located at `data/config.json` by default and is automatically generated on first startup.

**Complete Configuration Example:**

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

**Configuration Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `server.host` | Listen address | `0.0.0.0` |
| `server.port` | Server port | `8080` |
| `database.type` | Database type | `sqlite` |
| `database.path` | Database connection string | `data/data.db` |
| `log.level` | Log level | `info` |

**Database Configuration:**

Three database types are supported:

| Type | `database.type` | `database.path` Format |
|------|-----------------|-----------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL Configuration Example:**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL Configuration Example:**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **Tip**: MySQL and PostgreSQL require manual database creation. The application will automatically create the table structure.

### 🌐 Environment Variables

All configuration options can be overridden via environment variables using the format `OCTOPUS_` + configuration path (joined with `_`):

| Environment Variable | Configuration Option |
|---------------------|---------------------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_GITHUB_PAT` | For rate limiting when getting the latest version (optional) |
| `OCTOPUS_RELAY_MAX_SSE_EVENT_SIZE` | Maximum SSE event size (optional) |

## 📸 Screenshots

### 🖥️ Desktop

<div align="center">
<table>
<tr>
<td align="center"><b>Dashboard</b></td>
<td align="center"><b>Channel Management</b></td>
<td align="center"><b>Group Management</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="Dashboard" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="Channel" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="Group" width="400"></td>
</tr>
<tr>
<td align="center"><b>Price Management</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="Price Management" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="Logs" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="Settings" width="400"></td>
</tr>
</table>
</div>

### 📱 Mobile

<div align="center">
<table>
<tr>
<td align="center"><b>Home</b></td>
<td align="center"><b>Channel</b></td>
<td align="center"><b>Group</b></td>
<td align="center"><b>Price</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="Mobile Home" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="Mobile Channel" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="Mobile Group" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="Mobile Price" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="Mobile Logs" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="Mobile Settings" width="140"></td>
</tr>
</table>
</div>


## 📖 Documentation

### 📡 Channel Management

Channels are the basic configuration units for connecting to LLM providers and model services. A channel can define its type, Base URLs, multiple API keys, proxy, custom headers, model list, per-model protocol overrides, and upstream model synchronization strategy.

**Supported capabilities:**

- Provider presets: quickly fill common channel types and Base URLs
- Model fetching: fetch upstream model lists and select models into a channel
- Model testing: test model availability before or after saving
- Upstream update detection: detect added and removed models; added models can be appended, while removed models require manual confirmation
- Ignore rules: skip unwanted models with exact model names or `regex:` rules
- OAuth channels: GitHub Copilot Device Flow and Antigravity Web OAuth entry points

**Base URL Guide:**

The program automatically appends API paths based on channel type. You only need to provide the base URL:

| Channel Type | Auto-appended Path | Base URL | Full Request URL Example |
|--------------|-------------------|----------|--------------------------|
| OpenAI Chat | `/chat/completions` | `https://api.openai.com/v1` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/responses` | `https://api.openai.com/v1` | `https://api.openai.com/v1/responses` |
| Anthropic | `/messages` | `https://api.anthropic.com/v1` | `https://api.anthropic.com/v1/messages` |
| Gemini | `/models/:model:generateContent` | `https://generativelanguage.googleapis.com/v1beta` | `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` |
| Volcengine | `/responses` | Provider base URL | `{base_url}/responses` |
| OpenAI Embedding | `/embeddings` | `https://api.openai.com/v1` | `https://api.openai.com/v1/embeddings` |
| OpenAI Image Generation | `/images/generations` | `https://api.openai.com/v1` | `https://api.openai.com/v1/images/generations` |
| GitHub Copilot | `/chat/completions` | `https://api.githubcopilot.com` | `https://api.githubcopilot.com/chat/completions` |
| Antigravity | `/v1internal:generateContent` / `/v1internal:streamGenerateContent` | `https://cloudcode-pa.googleapis.com` | `https://cloudcode-pa.googleapis.com/v1internal:generateContent` |
| OpenCode Zen | Dynamically chooses Chat / Responses / Anthropic / Gemini paths by model | Zen service base URL | Routed automatically by model prefix |

> 💡 **Tip**: No need to include specific API endpoint paths in the Base URL - the program handles this automatically.

---

### 📁 Group Management

Groups aggregate multiple channels into a unified external model name.

**Core Concepts:**

- **Group name** is the model name exposed by the program
- When calling the API, set the `model` parameter to the group name

**Load Balancing Modes:**

| Mode | Description |
|------|-------------|
| 🔄 **Round Robin** | Cycles through channels sequentially for each request |
| 🎲 **Random** | Randomly selects an available channel for each request |
| 🛡️ **Failover** | Prioritizes high-priority channels, switches to lower priority only on failure |
| ⚖️ **Weighted** | Distributes requests based on configured channel weights |

> 💡 **Example**: Create a group named `gpt-4o`, add multiple providers' GPT-4o channels to it, then access all channels via a unified `model: gpt-4o`.

---

### 💰 Price Management

Manage model pricing information in the system.

**Data Sources:**

- The system periodically syncs model pricing data from [models.dev](https://github.com/sst/models.dev)
- When creating a channel, if the channel contains models not in models.dev, the system automatically creates pricing information for those models on this page, so this page displays models that haven't had their prices fetched from upstream, allowing users to set prices manually
- Manual creation of models that exist in models.dev is also supported for custom pricing

**Price Priority:**

| Priority | Source | Description |
|:--------:|--------|-------------|
| 🥇 High | This Page | Prices set by user in price management page |
| 🥈 Low | models.dev | Auto-synced default prices |

> 💡 **Tip**: To override a model's default price, simply set a custom price for it in the price management page.

---

### ⚙️ Settings

Global system configuration.

**Statistics Save Interval (minutes):**

Since the program handles numerous statistics, writing to the database on every request would impact read/write performance. The program uses this strategy:

- Statistics are first stored in **memory**
- Periodically **batch-written** to the database at the configured interval

> ⚠️ **Important**: When exiting the program, use proper shutdown methods (like `Ctrl+C` or sending `SIGTERM` signal) to ensure in-memory statistics are correctly written to the database. **Do NOT use `kill -9` or other forced termination methods**, as this may result in statistics data loss.

---

## 🔌 Client Integration

### OpenAI SDK

```python
from openai import OpenAI
import os

client = OpenAI(   
    base_url="http://127.0.0.1:8080/v1",   
    api_key="sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg", 
)
completion = client.chat.completions.create(
    model="octopus-openai",  # Use the correct group name
    messages = [
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

Edit `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8080",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg",
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

Edit `~/.codex/config.toml`

```toml
model = "octopus-codex" # Use the correct group name

model_provider = "octopus"

[model_providers.octopus]
name = "octopus"
base_url = "http://127.0.0.1:8080/v1"
```

Edit `~/.codex/auth.json`

```json
{
  "OPENAI_API_KEY": "sk-octopus-P48ROljwJmWBYVARjwQM8Nkiezlg7WOrXXOWDYY8TI5p9Mzg"
}
```

---

## 🤝 Acknowledgments

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - The LLM API adaptation module in this project is directly derived from this repository
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI model database providing model pricing data
