# Sothoth NodeAgent (Go版本)

Sothoth NodeAgent是用Golang重写的高性能节点代理程序，用于在目标节点上执行命令和收集系统信息。

## 功能特性

- **跨平台支持**: 支持Linux、macOS和Windows系统
- **WebSocket通信**: 与Sothoth服务器进行实时双向通信
- **进程监控**: 获取系统运行进程列表
- **命令执行**: 远程执行系统命令
- **自动重连**: 连接断开时自动重连
- **优雅关闭**: 支持信号处理和优雅关闭

## 编译

### 前置要求
- Go 1.21或更高版本

### 编译命令

```bash
# 编译当前平台
go build -o sothoth-nodeagent

# 交叉编译Linux版本
GOOS=linux GOARCH=amd64 go build -o sothoth-nodeagent-linux

# 交叉编译Windows版本
GOOS=windows GOARCH=amd64 go build -o sothoth-nodeagent.exe

# 交叉编译macOS版本
GOOS=darwin GOARCH=amd64 go build -o sothoth-nodeagent-darwin
```

## 使用方法

### 基本用法

```bash
./sothoth-nodeagent -project <PROJECT_ID> -server <WEBSOCKET_URL>
```

### 参数说明

- `-project`: 项目ID（必需）
- `-server`: 服务器WebSocket URL（必需）
- `-version`: 显示版本信息

### 示例

```bash
# 连接到本地服务器
./sothoth-nodeagent -project 1 -server ws://localhost:8080/ws/nodeagent

# 连接到远程服务器
./sothoth-nodeagent -project 1 -server ws://192.168.1.100:8080/ws/nodeagent
```

## 通信协议

NodeAgent与服务器之间使用WebSocket进行通信，支持以下消息类型：

### 客户端发送的消息

#### 注册消息
```json
{
  "type": "REGISTER",
  "data": {
    "hostname": "server1",
    "ip": "192.168.1.100",
    "project_id": 1,
    "agent_version": "1.0.0"
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
  "request_id": "req-123",
  "data": {
    "processes": [
      {
        "pid": 1234,
        "ppid": 1,
        "comm": "systemd",
        "command_line": "/sbin/init"
      }
    ]
  }
}
```

#### 命令执行结果
```json
{
  "type": "COMMAND_RESULT",
  "request_id": "req-456",
  "data": {
    "exit_code": 0,
    "stdout": "Hello World\n",
    "stderr": "",
    "execution_time": 123
  }
}
```

### 服务器发送的消息

#### 获取进程列表
```json
{
  "type": "GET_PROCESSES",
  "request_id": "req-123"
}
```

#### 执行命令
```json
{
  "type": "EXECUTE_COMMAND",
  "request_id": "req-456",
  "data": {
    "command": "ls -la"
  }
}
```

## 系统支持

### Linux
- 通过读取`/proc`文件系统获取进程信息
- 使用`sh -c`执行命令

### macOS
- 使用`ps`命令获取进程信息
- 使用`sh -c`执行命令

### Windows
- 使用`wmic`命令获取进程信息
- 使用`cmd /C`执行命令

## 安全特性

- **命令超时**: 所有命令执行都有30秒超时限制
- **优雅关闭**: 支持SIGINT和SIGTERM信号处理
- **错误处理**: 完善的错误处理和日志记录

## 日志

NodeAgent会输出详细的运行日志，包括：
- 连接状态
- 消息处理
- 错误信息
- 命令执行结果

## 故障排除

### 连接问题
1. 检查服务器地址和端口是否正确
2. 确认防火墙设置允许WebSocket连接
3. 检查网络连通性

### 权限问题
1. 确保NodeAgent有足够权限读取进程信息
2. 检查命令执行权限

### 性能问题
1. 监控内存和CPU使用情况
2. 检查网络延迟
3. 调整心跳间隔（如需要）

## 开发

### 项目结构
```
sothoth-nodeagent/
├── main.go          # 主程序入口
├── system.go        # 系统相关功能
├── go.mod          # Go模块定义
├── go.sum          # 依赖校验和
└── README.md       # 说明文档
```

### 依赖项
- `github.com/gorilla/websocket`: WebSocket客户端库

## 版本历史

### v1.0.0
- 初始版本
- 支持基本的进程监控和命令执行
- 跨平台支持
- WebSocket通信
