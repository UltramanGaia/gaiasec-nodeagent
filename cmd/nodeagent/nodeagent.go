// Package main 是Sothoth NodeAgent的主程序入口
// 
// Sothoth NodeAgent是一个用Go语言编写的轻量级节点代理程序，
// 用于在目标节点上执行命令、收集系统信息，并与Sothoth服务器进行实时通信。
//
// 主要功能：
// - WebSocket通信：与Sothoth服务器建立持久连接
// - 命令执行：远程执行系统命令并返回结果
// - 进程监控：获取系统运行进程列表
// - 文件系统操作：支持文件浏览、上传、下载
// - 终端会话：支持交互式终端操作
// - 自动重连：连接断开时自动重连机制
//
// 使用方法：
//   ./sothoth-nodeagent -project <PROJECT_ID> -server <WEBSOCKET_URL>
//
// 示例：
//   ./sothoth-nodeagent -project 1 -server ws://localhost:9000/ws/nodeagent
//
// @author UltramanGaia
// @version 1.0.0
package main

import (
	"sothoth-nodeagent/pkg/cli"
)

// main 是程序的主入口函数
// 调用CLI解析器来处理命令行参数并启动NodeAgent
func main() {
	cli.ParseMain()
}
