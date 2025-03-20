# Auto Deploy Platform 🚀

一个基于 Go 语言 + Gin + Docker + Ansible 构建的前后端一体化 **自动化发布平台**。

支持容器、Compose、系统监控实时推送、Ansible 自动化发布。

------

## 🧩 功能概览

| 功能                     | 描述                                             |
| ------------------------ | ------------------------------------------------ |
| 📦 容器管理               | 列出、启动、停止容器，实时日志推送               |
| 🚀 容器快速部署           | 通过前端填写镜像名、端口映射、环境变量等创建容器 |
| 🖥️ 系统状态监控           | 实时 WebSocket 推送 uptime、CPU、内存、磁盘使用  |
| 🧩 Docker Compose 管理    | 上传 Compose 文件、启动、停止、查看容器详情      |
| 📄 Ansible Playbook 执行  | 运行指定 Inventory 和 Playbook，并实时推送日志   |
| 🌐 多主机 Docker API 管理 | 支持切换管理多个 Docker 主机                     |
| 📜 日志实时推送           | WebSocket 日志实时推送                           |
| ⚙️ 配置动态化             | 支持 config.json 配置接口、WebSocket 地址        |
| 🔒 权限控制 (可扩展)      | 未来支持多用户权限                               |

------

## 📂 已实现模块

### 1️⃣ 容器管理

- GET `/api/v1/containers` → 列出容器
- POST `/api/v1/container/start/:id` → 启动容器
- POST `/api/v1/container/stop/:id` → 停止容器
- GET `/api/v1/ws/container-logs/:id` → 实时日志推送
- POST `/api/v1/container/create` → 创建新容器 (带端口映射、变量、挂载、高级选项)

------

### 2️⃣ 系统监控

- GET `/api/v1/ws-system` → 实时推送系统 uptime、CPU、内存、磁盘

------

### 3️⃣ Compose 管理

- POST `/api/v1/compose/upload` → 上传 Compose
- GET `/api/v1/compose/status` → 获取所有上传 Compose 状态 & 容器列表
- POST `/api/v1/compose/up` → 启动 Compose
- POST `/api/v1/compose/down` → 停止 Compose
- POST `/api/v1/compose/delete` → 删除 Compose 应用
- GET `/api/v1/ws/compose-logs` → 实时推送 Compose 启停日志

✅ 已实现 **即使未运行的 Compose 也显示（从 compose-files 目录读取）**

------

### 4️⃣ Ansible Playbook

- POST `/api/v1/run-ansible` → 指定 Inventory + Playbook 文件，运行并推送日志

------

### 5️⃣ 多主机切换

支持配置多个 Docker 主机，通过下拉选择切换。

------

## 📄 配置文件结构 (config.json)

```
{
  "apiBaseUrl": "/api/v1",
  "wsBaseUrl": "ws://localhost:8081/api/v1",
  "dockerHosts": [
    { "name": "Localhost", "host": "" },
    { "name": "Server1", "host": "192.168.1.101" }
  ]
}
```

------

## 🚧 **待实现模块**

| 功能模块          | 描述                                            |
| ----------------- | ----------------------------------------------- |
| 📂 文件管理        | 支持通过页面管理挂载到容器/主机的文件，增删改查 |
| 🔒 用户权限 & 登录 | 多用户、角色权限系统                            |
| 📊 任务历史        | 查看历史发布任务记录及状态                      |
| 💾 自动备份 & 回滚 | 容器和 Compose 的数据卷快照备份/一键回滚        |
| 💬 消息通知        | 发布成功/失败可邮件/钉钉/微信通知               |
| 🌍 多云环境适配    | Azure、AWS、Aliyun 扩展                         |

------

## 🚀 快速启动

```
git clone https://github.com/xk-7/auto-deploy-platform.git
cd auto-deploy-platform

go mod tidy
go run cmd/server/main.go
```

访问：

```
http://localhost:8081
```

------

## 📁 目录结构

```
├── cmd/server/main.go      // 入口
├── controllers             // API 控制器
│   ├── docker_controller.go
│   ├── compose_controller.go
│   ├── system_controller.go
│   └── ansible_controller.go
├── api/v1/router.go        // 路由
├── static/                 // 前端页面
│   ├── index.html
│   ├── containers.html
│   ├── containers-create.html
│   └── compose.html
└── compose-files/          // Compose 文件存放
```

------

## 💡 下一步

**文件管理模块** 计划设计：

- 页面展示：
  - 文件树 + 权限标识
  - 可上传 / 删除 / 下载
- 接口设计：
  - GET `/api/v1/files/list`
  - POST `/api/v1/files/upload`
  - DELETE `/api/v1/files/:path`
  - GET `/api/v1/files/download/:path`