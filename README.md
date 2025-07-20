# Sothoth NodeAgent - 节点代理程序

Sothoth NodeAgent是用Golang重写的高性能节点代理程序，用于在目标节点上执行命令、收集系统信息，并与Sothoth服务器进行实时通信。它是Sothoth渗透测试管理系统的核心组件之一。

## 功能特性

### 核心功能
- **跨平台支持**: 支持Linux、macOS和Windows系统
- **多架构支持**: 支持x86_64、ARM64等多种CPU架构
- **WebSocket通信**: 与Sothoth服务器进行实时双向通信
- **进程监控**: 获取系统运行进程列表
- **命令执行**: 远程执行系统命令并返回结果
- **文件系统操作**: 支持文件浏览、上传、下载
- **终端会话**: 支持交互式终端操作
- **自动重连**: 连接断开时自动重连机制
- **优雅关闭**: 支持信号处理和优雅关闭

### 高级特性
- **独立连接管理**: 支持多会话独立连接架构
- **守护进程模式**: 支持后台运行
- **代理模式**: 支持代理网络环境
- **进程管理**: PID文件管理和进程唯一性检查
- **环境初始化**: 自动创建工作目录和环境变量
- **日志记录**: 详细的运行日志和错误处理

## 系统架构

### 技术栈
- **Go 1.21+** - 核心开发语言
- **Gorilla WebSocket** - WebSocket客户端库
- **跨平台系统调用** - 系统信息获取和命令执行

### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                  Sothoth NodeAgent                         │
├─────────────────────────────────────────────────────────────┤
│  CLI层 (Command Line Interface)                            │
│  └── parse.go           - 命令行参数解析和程序入口         │
├─────────────────────────────────────────────────────────────┤
│  核心层 (Core Agent)                                       │
│  ├── agent.go           - 主Agent结构和生命周期管理        │
│  ├── websocket.go       - WebSocket通信协议实现            │
│  ├── independent_connection_manager.go - 独立连接管理器    │
│  └── independent_server.go - 独立服务器实现                │
├─────────────────────────────────────────────────────────────┤
│  系统层 (System Interface)                                 │
│  ├── info.go            - 系统信息获取                     │
│  ├── command.go         - 命令执行                         │
│  └── process.go         - 进程管理                         │
├─────────────────────────────────────────────────────────────┤
│  文件系统层 (File System)                                  │
│  ├── api.go             - 文件系统API                      │
│  └── explorer.go        - 文件浏览器                       │
├─────────────────────────────────────────────────────────────┤
│  终端层 (Terminal)                                         │
│  └── pty.go             - 伪终端(PTY)管理                  │
├─────────────────────────────────────────────────────────────┤
│  工具层 (Utilities)                                        │
│  ├── config.go          - 配置管理                         │
│  ├── file.go            - 文件工具                         │
│  └── daemon.go          - 守护进程工具                     │
└─────────────────────────────────────────────────────────────┘
```

## 编译构建

### 前置要求
- **Go 1.21或更高版本**
- **Git** (用于获取依赖)

### 编译命令

#### 本地编译
```bash
# 编译当前平台
go build -o sothoth-nodeagent cmd/nodeagent/nodeagent.go

# 或使用make（如果有Makefile）
make build
```

#### 交叉编译
```bash
# 编译Linux x86_64版本
GOOS=linux GOARCH=amd64 go build -o sothoth-nodeagent-linux-amd64 cmd/nodeagent/nodeagent.go

# 编译Linux ARM64版本
GOOS=linux GOARCH=arm64 go build -o sothoth-nodeagent-linux-arm64 cmd/nodeagent/nodeagent.go

# 编译Windows x86_64版本
GOOS=windows GOARCH=amd64 go build -o sothoth-nodeagent-windows-amd64.exe cmd/nodeagent/nodeagent.go

# 编译macOS x86_64版本
GOOS=darwin GOARCH=amd64 go build -o sothoth-nodeagent-darwin-amd64 cmd/nodeagent/nodeagent.go

# 编译macOS ARM64版本（Apple Silicon）
GOOS=darwin GOARCH=arm64 go build -o sothoth-nodeagent-darwin-arm64 cmd/nodeagent/nodeagent.go
```

#### 批量编译脚本
```bash
#!/bin/bash
# build-all.sh - 批量编译所有平台版本

platforms=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "darwin/amd64"
    "darwin/arm64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS=${array[0]}
    GOARCH=${array[1]}
    
    output="sothoth-nodeagent-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output="${output}.exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "bin/$output" cmd/nodeagent/nodeagent.go
done

echo "Build completed!"
```

### 构建优化

#### 减小二进制文件大小
```bash
# 使用ldflags减小文件大小
go build -ldflags="-s -w" -o sothoth-nodeagent cmd/nodeagent/nodeagent.go

# 使用UPX进一步压缩（需要安装UPX）
upx --best sothoth-nodeagent
```

#### 静态链接
```bash
# 静态链接，避免依赖问题
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o sothoth-nodeagent cmd/nodeagent/nodeagent.go
```

## 使用方法

### 基本用法

```bash
./sothoth-nodeagent -projectId <PROJECT_ID> -nodeId <NODE_ID> -server <SERVER_URL>
```

### 命令行参数

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `-projectId` | string | ✅ | - | 项目ID |
| `-nodeId` | string | ✅ | - | 节点ID |
| `-server` | string | ✅ | - | 服务器WebSocket URL |
| `-sothothDir` | string | ❌ | `/sothoth` | Sothoth工作目录 |
| `-d` | bool | ❌ | `false` | 以守护进程模式运行 |
| `-p` | bool | ❌ | `false` | 启用代理模式 |
| `-version` | bool | ❌ | `false` | 显示版本信息 |
| `-logflags` | string | ❌ | `log.LstdFlags` | 日志标志 |

### 使用示例

#### 基本连接
```bash
# 连接到本地服务器
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server localhost:9000

# 连接到远程服务器
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server 192.168.1.100:9000
```

#### 守护进程模式
```bash
# 后台运行
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server localhost:9000 -d

# 检查运行状态
ps aux | grep sothoth-nodeagent

# 查看日志
tail -f /sothoth/logs/nodeagent/000000000000/agent.log
```

#### 自定义工作目录
```bash
# 指定工作目录
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server localhost:9000 -sothothDir /opt/sothoth
```

#### 代理模式
```bash
# 启用代理模式（适用于复杂网络环境）
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server localhost:9000 -p
```

### 服务管理

#### 创建systemd服务
```bash
# 创建服务文件
sudo tee /etc/systemd/system/sothoth-nodeagent.service > /dev/null <<EOF
[Unit]
Description=Sothoth NodeAgent
After=network.target

[Service]
Type=simple
User=sothoth
Group=sothoth
ExecStart=/usr/local/bin/sothoth-nodeagent -projectId 1 -nodeId \$(hostname) -server your-server:9000 -sothothDir /opt/sothoth
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 启用并启动服务
sudo systemctl daemon-reload
sudo systemctl enable sothoth-nodeagent
sudo systemctl start sothoth-nodeagent

# 查看服务状态
sudo systemctl status sothoth-nodeagent
```

## 通信协议

### WebSocket连接

NodeAgent与服务器之间使用WebSocket进行通信，支持以下消息类型：

### 消息格式

```json
{
  "type": "MESSAGE_TYPE",
  "requestId": "unique_request_id",
  "data": {
    // 消息数据
  }
}
```

### 客户端发送的消息

#### 节点注册
```json
{
  "type": "REGISTER",
  "data": {
    "hostname": "server1",
    "ipAddress": "192.168.1.100",
    "projectId": "1",
    "nodeId": "node-001",
    "agentVersion": "1.0.0",
    "osType": "linux"
  }
}
```

#### 心跳消息
```json
{
  "type": "HEARTBEAT",
  "data": {}
}
```

#### 进程列表响应
```json
{
  "type": "PROCESSES_RESPONSE",
  "requestId": "req-123",
  "data": {
    "processes": [
      {
        "pid": 1234,
        "ppid": 1,
        "name": "systemd",
        "command": "/sbin/init",
        "cpu": 0.1,
        "memory": 1024
      }
    ]
  }
}
```

#### 命令执行结果
```json
{
  "type": "COMMAND_RESULT",
  "requestId": "req-456",
  "data": {
    "exitCode": 0,
    "stdout": "Hello World\n",
    "stderr": "",
    "executionTime": 123
  }
}
```

#### 终端输出
```json
{
  "type": "PTY_OUTPUT",
  "requestId": "req-789",
  "data": {
    "sessionId": "session-001",
    "output": "user@host:~$ "
  }
}
```

#### 文件系统响应
```json
{
  "type": "FS_RESPONSE",
  "requestId": "req-101",
  "data": {
    "path": "/home/user",
    "files": [
      {
        "name": "document.txt",
        "size": 1024,
        "isDir": false,
        "modTime": "2024-01-01T12:00:00Z"
      }
    ]
  }
}
```

### 服务器发送的消息

#### 获取进程列表
```json
{
  "type": "GET_PROCESSES",
  "requestId": "req-123"
}
```

#### 执行命令
```json
{
  "type": "EXECUTE_COMMAND",
  "requestId": "req-456",
  "data": {
    "command": "ls -la",
    "timeout": 30
  }
}
```

#### 创建终端会话
```json
{
  "type": "CREATE_PTY",
  "requestId": "req-789",
  "data": {
    "sessionId": "session-001",
    "cols": 80,
    "rows": 24
  }
}
```

#### 终端输入
```json
{
  "type": "PTY_INPUT",
  "requestId": "req-790",
  "data": {
    "sessionId": "session-001",
    "input": "ls -la\n"
  }
}
```

#### 文件系统操作
```json
{
  "type": "FS_REQUEST",
  "requestId": "req-101",
  "data": {
    "operation": "list",
    "path": "/home/user"
  }
}
```

## 系统支持

### Linux
- **进程信息**: 通过读取`/proc`文件系统获取进程信息
- **命令执行**: 使用`sh -c`执行命令
- **终端支持**: 支持PTY伪终端
- **文件系统**: 完整的文件系统操作支持

### macOS
- **进程信息**: 使用`ps`命令获取进程信息
- **命令执行**: 使用`sh -c`执行命令
- **终端支持**: 支持PTY伪终端
- **文件系统**: 完整的文件系统操作支持

### Windows
- **进程信息**: 使用`wmic`命令获取进程信息
- **命令执行**: 使用`cmd /C`执行命令
- **终端支持**: 基础终端支持
- **文件系统**: 基础文件系统操作支持

## 配置管理

### 配置文件结构

```go
type Config struct {
    ServerURL    string // 服务器WebSocket URL
    ProjectID    string // 项目ID
    NodeID       string // 节点ID
    SothothDir   string // Sothoth工作目录
    DaemonMode   bool   // 守护进程模式
    Proxy        bool   // 代理模式
    Version      bool   // 版本信息
    Logflags     string // 日志标志
}
```

### 环境变量

NodeAgent支持通过环境变量进行配置：

```bash
# 服务器配置
export SOTHOTH_SERVER_URL="localhost:9000"
export SOTHOTH_PROJECT_ID="1"
export SOTHOTH_NODE_ID="node-001"

# 工作目录
export SOTHOTH_DIR="/opt/sothoth"

# 运行模式
export SOTHOTH_DAEMON_MODE="true"
export SOTHOTH_PROXY_MODE="false"
```

## 安全特性

### 网络安全
- **WebSocket加密**: 支持WSS加密传输
- **连接验证**: 服务器端连接验证
- **超时控制**: 连接和命令执行超时控制
- **错误处理**: 安全的错误信息处理

### 系统安全
- **权限控制**: 以指定用户权限运行
- **命令限制**: 可配置的命令执行限制
- **文件访问**: 受限的文件系统访问
- **进程隔离**: 独立的进程空间

### 运行安全
- **PID管理**: 防止重复运行
- **优雅关闭**: 信号处理和资源清理
- **日志记录**: 详细的操作日志
- **错误恢复**: 自动错误恢复机制

## 故障排除

### 常见问题

#### 1. 连接失败
**问题**: 无法连接到服务器
```
Error: dial tcp: connection refused
```
**解决方案**:
```bash
# 检查服务器状态
telnet server-ip 9000

# 检查网络连通性
ping server-ip

# 检查防火墙设置
sudo ufw status
```

#### 2. 权限问题
**问题**: 文件或目录权限不足
```
Error: permission denied
```
**解决方案**:
```bash
# 检查工作目录权限
ls -la /sothoth

# 修改权限
sudo chown -R user:group /sothoth
sudo chmod -R 755 /sothoth
```

#### 3. 进程已存在
**问题**: 另一个实例正在运行
```
Error: Another instance of the agent is already running
```
**解决方案**:
```bash
# 查找运行中的进程
ps aux | grep sothoth-nodeagent

# 停止现有进程
kill -TERM <pid>

# 删除PID文件
rm /sothoth/nodeagent.pid
```

#### 4. 内存不足
**问题**: 系统内存不足
```
Error: cannot allocate memory
```
**解决方案**:
```bash
# 检查内存使用
free -h

# 检查进程内存使用
ps aux --sort=-%mem | head

# 优化系统内存
sudo sysctl vm.swappiness=10
```

### 调试模式

启用详细日志：
```bash
# 设置日志级别
export SOTHOTH_LOG_LEVEL=DEBUG

# 运行时启用调试
./sothoth-nodeagent -projectId 1 -nodeId node-001 -server localhost:9000 -logflags "log.LstdFlags|log.Lshortfile"
```

查看日志：
```bash
# 查看实时日志
tail -f /sothoth/logs/nodeagent/000000000000/agent.log

# 查看系统日志
journalctl -u sothoth-nodeagent -f
```

## 性能优化

### 系统调优

#### 网络优化
```bash
# 增加TCP连接数限制
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf

# 优化TCP参数
echo 'net.ipv4.tcp_fin_timeout = 30' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_keepalive_time = 1200' >> /etc/sysctl.conf

# 应用配置
sysctl -p
```

#### 文件描述符优化
```bash
# 增加文件描述符限制
echo '* soft nofile 65535' >> /etc/security/limits.conf
echo '* hard nofile 65535' >> /etc/security/limits.conf

# 重新登录生效
ulimit -n 65535
```

### 应用优化

#### 编译优化
```bash
# 性能优化编译
go build -ldflags="-s -w" -gcflags="-N -l" -o sothoth-nodeagent cmd/nodeagent/nodeagent.go

# 启用竞态检测（开发环境）
go build -race -o sothoth-nodeagent cmd/nodeagent/nodeagent.go
```

#### 运行时优化
```bash
# 设置GOMAXPROCS
export GOMAXPROCS=4

# 设置垃圾回收参数
export GOGC=100

# 启用性能分析
export GODEBUG=gctrace=1
```

## 监控和指标

### 内置监控

NodeAgent提供内置的监控指标：

- **连接状态**: WebSocket连接状态
- **消息统计**: 发送/接收消息数量
- **命令执行**: 命令执行次数和耗时
- **系统资源**: CPU和内存使用情况
- **错误统计**: 错误类型和频率

### 外部监控

#### Prometheus集成
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'sothoth-nodeagent'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

#### 日志监控
```bash
# 使用filebeat收集日志
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /sothoth/logs/nodeagent/*/*.log
  fields:
    service: sothoth-nodeagent
```

## 开发指南

### 项目结构

```
sothoth-nodeagent/
├── cmd/nodeagent/
│   └── nodeagent.go              # 主程序入口
├── pkg/
│   ├── cli/
│   │   └── parse.go              # 命令行解析
│   ├── config/
│   │   └── config.go             # 配置管理
│   ├── naserver/
│   │   ├── agent.go              # 主Agent实现
│   │   ├── websocket.go          # WebSocket通信
│   │   ├── independent_connection_manager.go
│   │   └── independent_server.go
│   ├── system/
│   │   ├── info.go               # 系统信息
│   │   ├── command.go            # 命令执行
│   │   └── process.go            # 进程管理
│   ├── filesystem/
│   │   ├── api.go                # 文件系统API
│   │   └── explorer.go           # 文件浏览器
│   ├── terminal/
│   │   └── pty.go                # 终端管理
│   ├── util/
│   │   └── file.go               # 文件工具
│   └── xdaemon/
│       └── daemon.go             # 守护进程
├── go.mod                        # Go模块定义
├── go.sum                        # 依赖校验和
└── README.md                     # 项目文档
```

### 添加新功能

#### 1. 添加新的消息类型
```go
// 在websocket.go中定义新的消息类型
const (
    MessageTypeNewFeature = "NEW_FEATURE"
)

// 在handleMessage中添加处理逻辑
case MessageTypeNewFeature:
    return a.handleNewFeature(msg)
```

#### 2. 添加新的系统接口
```go
// 在system包中添加新的功能
func GetSystemMetrics() (*SystemMetrics, error) {
    // 实现系统指标获取
}
```

#### 3. 扩展配置选项
```go
// 在config.go中添加新的配置项
type Config struct {
    // 现有配置...
    NewOption string `json:"new_option"`
}
```

### 测试

#### 单元测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/system

# 运行测试并生成覆盖率报告
go test -cover ./...
```

#### 集成测试
```bash
# 启动测试服务器
go run cmd/nodeagent/nodeagent.go -projectId test -nodeId test-node -server localhost:9000

# 运行集成测试
go test -tags=integration ./tests/
```

#### 性能测试
```bash
# 运行基准测试
go test -bench=. ./pkg/system

# 生成性能分析文件
go test -bench=. -cpuprofile=cpu.prof ./pkg/system
```

## 版本历史

### v1.0.0 (当前版本)
- ✅ 基础Agent功能
- ✅ WebSocket通信
- ✅ 跨平台支持
- ✅ 命令执行和进程监控
- ✅ 文件系统操作
- ✅ 终端会话支持
- ✅ 守护进程模式
- ✅ 独立连接管理

### 计划功能
- 🔄 插件系统
- 🔄 加密通信
- 🔄 负载均衡
- 🔄 集群支持
- 🔄 性能监控
- 🔄 自动更新

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献指南

欢迎提交Issue和Pull Request来改进Sothoth NodeAgent。

### 贡献流程

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范

- 遵循Go代码规范
- 使用`gofmt`格式化代码
- 添加适当的注释和文档
- 编写单元测试
- 更新相关文档

### 提交规范

```
type(scope): description

[optional body]

[optional footer]
```

类型：
- `feat`: 新功能
- `fix`: 错误修复
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 [GitHub Issue](https://github.com/UltramanGaia/sothoth/issues)
- 发送邮件至项目维护者

---

**注意**: 本程序仅用于授权的渗透测试活动，请遵守相关法律法规。
