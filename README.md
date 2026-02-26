# GaiaSec NodeAgent 文档

## 1. 功能概述

Node Agent 是 GaiaSec 系统中的节点代理，主要负责与 Java Agent 通信，管理节点状态，以及将收集到的信息传递给服务器。

## 2. 核心特性

- **Unix 域套接字通信**：与 Java Agent 通过 Unix 域套接字通信
- **WebSocket 通信**：与 GaiaSec 服务器建立持久连接，支持实时双向通信
- **节点管理**：管理节点状态和资源
- **数据转发**：将 Java Agent 收集的信息转发给服务器
- **命令执行**：执行来自服务器的命令并返回结果
- **进程监控**：获取系统运行进程列表
- **文件系统操作**：支持文件浏览、上传、下载
- **终端会话**：支持交互式终端操作
- **网络监控**：监控网络连接状态
- **插件部署**：支持插件下载和部署
- **自动重连**：连接断开时自动重连机制
- **监控与告警**：监控节点健康状态，发送告警信息

## 3. 技术架构

Node Agent 基于 **Go 语言** 开发（Go 1.24.3），使用高效的并发模型，支持高并发连接和实时通信。

### 技术栈

- **编程语言**：Go 1.24.3
- **通信协议**：WebSocket, Unix Domain Socket
- **序列化**：Protocol Buffers
- **日志**：logrus
- **系统信息**：gopsutil/v3

### 项目结构

```
gaiasec-nodeagent/
├── cmd/                    # 主程序入口
│   └── nodeagent/          # NodeAgent 主程序
├── pkg/                    # 核心包
│   ├── cli/                # 命令行解析
│   ├── config/             # 配置管理
│   ├── constant/           # 常量定义
│   ├── filesystem/         # 文件系统操作
│   ├── mount/              # 挂载点管理
│   ├── network/            # 网络监控
│   ├── naserver/           # NodeAgent 服务器处理
│   ├── pb/                 # Protocol Buffers 定义
│   ├── plugin/             # 插件管理
│   ├── process/            # 进程管理
│   ├── proxy/              # 代理功能
│   ├── system/             # 系统信息
│   ├── terminal/           # 终端会话
│   ├── tlv/                # TLV 编解码
│   ├── udsserver/          # Unix Domain Socket 服务器
│   ├── util/               # 工具函数
│   ├── wsclient/           # WebSocket 客户端
│   └── xdaemon/            # 守护进程
├── build.sh               # 跨平台构建脚本
├── go.mod                  # Go 模块定义
└── go.sum                  # 依赖校验和
```

## 4. 构建和部署

### 环境要求

- Go 1.24.3 或更高版本
- Git（用于版本信息）
- 交叉编译工具链（可选，用于构建 ARM64）

### 构建步骤

1. **克隆项目并进入目录**：

```bash
cd gaiasec-nodeagent
```

2. **运行构建脚本**：

```bash
# 正常构建
./build.sh

# 清理构建缓存后构建
./build.sh --clean
```

3. **构建产物**：

构建脚本会为以下平台生成二进制文件：

- `linux/amd64` - Linux x86_64
- `linux/arm64` - Linux ARM64（需要 aarch64-linux-gnu-gcc）
- `windows/amd64` - Windows x86_64

生成的二进制文件位于：`../gaiasec-server/plugins/nodeagent/`

文件命名格式：`nodeagent-{version}-{os}-{arch}`

示例：
- `nodeagent-v1.0.0-linux-amd64`
- `nodeagent-v1.0.0-linux-arm64`
- `nodeagent-v1.0.0-windows-amd64.exe`

### 手动构建

如果需要手动构建特定平台：

```bash
# 构建 Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o nodeagent-linux-amd64 ./cmd/nodeagent

# 构建 Linux ARM64（需要交叉编译器）
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -o nodeagent-linux-arm64 ./cmd/nodeagent

# 构建 Windows AMD64
GOOS=windows GOARCH=amd64 go build -o nodeagent-windows-amd64.exe ./cmd/nodeagent
```

### 运行 NodeAgent

```bash
# 基本运行
./nodeagent -project <PROJECT_ID> -server <WEBSOCKET_URL>

# 示例
./nodeagent -project 1 -server ws://localhost:9000/ws/agent
```

**参数说明**：

- `-project`：项目 ID
- `-server`：GaiaSec 服务器的 WebSocket 地址

### 交叉编译准备

如果需要构建 ARM64 版本，请确保安装了交叉编译工具链：

**Ubuntu/Debian**：
```bash
sudo apt-get install gcc-aarch64-linux-gnu
```

**CentOS/RHEL**：
```bash
sudo yum install gcc-aarch64-linux-gnu
```

## 5. 使用示例

### 启动 NodeAgent

```bash
# 开发环境
./nodeagent -project 1 -server ws://localhost:9000/ws/agent

# 生产环境（使用 TLS）
./nodeagent -project 1 -server wss://gaiasec.example.com/ws/agent
```

### 部署到目标节点

1. 将编译好的二进制文件上传到目标节点
2. 添加执行权限：`chmod +x nodeagent`
3. 创建配置文件（可选）
4. 启动服务：`./nodeagent -project <ID> -server <URL>`

### 使用 systemd 管理服务（推荐）

创建服务文件 `/etc/systemd/system/gaiasec-nodeagent.service`：

```ini
[Unit]
Description=GaiaSec NodeAgent
After=network.target

[Service]
Type=simple
User=gaiasec
WorkingDirectory=/opt/gaiasec-nodeagent
ExecStart=/opt/gaiasec-nodeagent/nodeagent -project 1 -server wss://gaiasec.example.com/ws/agent
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable gaiasec-nodeagent
sudo systemctl start gaiasec-nodeagent
sudo systemctl status gaiasec-nodeagent
```

## 6. 故障排查

### 常见问题

1. **连接 WebSocket 服务器失败**
   - 检查服务器地址是否正确
   - 确认服务器 WebSocket 端口是否开放
   - 检查网络连通性

2. **无法执行命令**
   - 确认 nodeagent 有足够的权限
   - 检查命令是否存在

3. **文件操作失败**
   - 检查文件系统权限
   - 确认目标路径可访问

### 日志查看

NodeAgent 会输出运行日志到标准输出和标准错误，建议使用 systemd journalctl 查看：

```bash
sudo journalctl -u gaiasec-nodeagent -f
```

## 7. 性能优化

- 使用 `-trimpath` 减少二进制大小
- 使用 `-ldflags="-w -s"` 去除调试信息
- 根据实际需求调整并发连接数
- 定期清理日志文件

## 8. 安全建议

- 使用 TLS/SSL 加密 WebSocket 连接
- 限制 nodeagent 的运行权限
- 定期更新到最新版本
- 使用防火墙规则限制访问
- 审计日志文件

---

**版本信息**：
- Go 版本：1.24.3
- 构建脚本：v1.0.0
- 最后更新：2024

**维护者**：GaiaSec Team

---

© 2024 GaiaSec. 保留所有权利。
