# AGENTS.md

## 模块定位
`gaiasec-nodeagent` 是部署在目标节点上的代理，负责与服务端保持 WebSocket 连接，并在节点本地执行命令、终端、文件、网络、容器与插件相关操作。

## 技术栈与入口
- Go 1.24
- 入口：`cmd/nodeagent/nodeagent.go`
- 关键依赖：WebSocket、Docker API、CRI、protobuf、gopsutil

## 关键目录
- `cmd/nodeagent/`: 启动入口
- `pkg/wsclient/`: 服务端连接与消息收发
- `pkg/terminal/`: 终端代理
- `pkg/filesystem/`: 文件系统操作
- `pkg/process/`: 进程管理
- `pkg/network/`: 网络信息采集
- `pkg/container/`: 容器运行时适配
- `pkg/plugin/`: 插件下载与管理
- `pkg/udsserver/`: 与 Java Agent 的本地 socket 通信
- `pkg/pb/`: protobuf 生成代码
- `agent/`: 发布二进制产物

## 常用命令
```bash
go test ./...
go run ./cmd/nodeagent
go build ./cmd/nodeagent
bash build.sh
```

## 协作约定
- 修改消息结构时优先看 `pkg/wsclient/`、`pkg/udsserver/` 与 `pkg/pb/` 的边界，不要只改单侧实现。
- `pkg/pb/` 是由 `../gaiasec-protobuf` 生成的，不要手改。
- `agent/` 下是交付产物，不是源码；跨平台差异应收敛在源码和构建脚本里。
- 涉及终端、文件、容器、HTTP 发送等能力变更时，要联动核对 `gaiasec-server`、`gaiasec-worker`、`gaiasec-terminal`、`gaiasec-fileserver` 的协议兼容性。
