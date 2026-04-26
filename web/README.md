# Octopus Web

Octopus Web 是 Octopus 的 Web 管理面板前端，负责提供渠道、模型、分组、API Key、模型价格、请求日志、系统设置、外观配置和 API 使用文档等可视化管理能力。

它基于 Next.js、React 和 Tailwind CSS 构建，但不建议脱离根项目单独部署。Docker Compose 部署时，根目录 `Dockerfile` 会在 frontend 阶段构建本目录，并把静态产物输出到后端服务使用的 `static/out` 目录。

## 核心职责

- 渠道管理：维护 Provider 类型、Base URL、API Key、自定义 Header、代理和模型列表
- 模型能力：支持模型拉取、模型测试、模型级协议覆盖和上游模型更新检测
- OAuth 入口：提供 GitHub Copilot Device Flow 与 Antigravity Web OAuth 授权辅助
- 运营视图：展示请求量、Token、费用、耗时、日志和渠道健康状态
- 使用文档：内置 OpenAI / Anthropic 等客户端调用示例，方便复制接入

## 部署方式

请在项目根目录使用 Docker Compose 统一部署：

```bash
docker compose up -d --build
```

启动后访问：

```text
http://localhost:8080
```

> 前端不再维护单独部署说明，生产和日常使用都以根目录 Docker Compose 流程为准。

## 相关目录

- `app/`：Next.js 页面、布局和路由入口
- `components/`：通用 UI 组件和业务组件
- `lib/`：前端 API 请求、配置和工具函数
- `public/`：图标、PWA Manifest 和静态资源
- `messages/`：国际化文案
