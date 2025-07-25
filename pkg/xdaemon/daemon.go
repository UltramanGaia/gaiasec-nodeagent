package xdaemon

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
)

const ENV_NAME = "XW_DAEMON_IDX"

// 运行时调用background的次数
var runIdx int = 0

// 守护进程
type Daemon struct {
	LogFile     string //日志文件, 记录守护进程和子进程的标准输出和错误输出. 若为空则不记录
	MaxCount    int    //循环重启最大次数, 若为0则无限重启
	MaxError    int    //连续启动失败或异常退出的最大次数, 超过此数, 守护进程退出, 不再重启子进程
	MinExitTime int64  //子进程正常退出的最小时间(秒). 小于此时间则认为是异常退出
}

// 把本身程序转化为后台运行(启动一个子进程, 然后自己退出)
// logFile 若不为空,子程序的标准输出和错误输出将记入此文件
// isExit  启动子加进程后是否直接退出主程序, 若为false, 主程序返回*os.Process, 子程序返回 nil. 需自行判断处理
func Background(logFile string, isExit bool) (*exec.Cmd, error) {
	//判断子进程还是父进程
	runIdx++
	envIdx, err := strconv.Atoi(os.Getenv(ENV_NAME))
	if err != nil {
		envIdx = 0
	}
	if runIdx <= envIdx { //子进程, 退出
		return nil, nil
	}

	//设置子进程环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", ENV_NAME, runIdx))

	//启动子进程
	cmd, err := startProc(os.Args, env, logFile)
	if err != nil {
		log.Info(os.Getpid(), "start child process failed:", err)
		return nil, err
	} else {
		//执行成功
		log.Info(os.Getpid(), ":", "start child process success:", "->", cmd.Process.Pid, "\n ")
	}

	if isExit {
		os.Exit(0)
	}

	return cmd, nil
}

func NewDaemon(logFile string) *Daemon {
	return &Daemon{
		LogFile:     logFile,
		MaxCount:    0,
		MaxError:    3,
		MinExitTime: 10,
	}
}

func startProc(args, env []string, logFile string) (*exec.Cmd, error) {
	cmd := &exec.Cmd{
		Path:        args[0],
		Args:        args,
		Env:         env,
		SysProcAttr: NewSysProcAttr(),
	}

	if logFile != "" {
		stdout, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Info(os.Getpid(), ": open log file error:", err)
			return nil, err
		}
		cmd.Stderr = stdout
		cmd.Stdout = stdout
	}

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
