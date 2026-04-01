# AGENTS.md

## 这个模块
- `gaiasec-nodeagent` 是部署在目标节点上的 agent，负责命令、终端、文件、进程、网络、容器和插件操作。

## 先看哪里
- `cmd/nodeagent/`
- `pkg/wsclient/`
- `pkg/terminal/`
- `pkg/filesystem/`
- `pkg/process/`
- `pkg/network/`
- `pkg/container/`
- `pkg/plugin/`
- `pkg/udsserver/`
- `pkg/pb/`

## 约束
- 改消息结构时，联动检查 `gaiasec-protobuf/` 和服务端消费者。
- `agent/` 和 `pkg/pb/` 不是手改入口。
