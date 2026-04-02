package escape

// Linux 默认 capabilities（容器默认开启的）
var DefaultCapabilities = []string{
	"CAP_CHOWN",
	"CAP_DAC_OVERRIDE",
	"CAP_FOWNER",
	"CAP_FSETID",
	"CAP_KILL",
	"CAP_SETGID",
	"CAP_SETUID",
	"CAP_SETPCAP",
	"CAP_NET_BIND_SERVICE",
	"CAP_NET_RAW",
	"CAP_IPC_LOCK",
	"CAP_IPC_OWNER",
	"CAP_SYS_MODULE",
	"CAP_SYS_RAWIO",
	"CAP_SYS_CHROOT",
	"CAP_SYS_PTRACE",
	"CAP_SYS_PACCT",
	"CAP_SYS_ADMIN",
	"CAP_SYS_BOOT",
	"CAP_SYS_NICE",
	"CAP_SYS_RESOURCE",
	"CAP_SYS_TIME",
	"CAP_SYS_TTY_CONFIG",
	"CAP_MKNOD",
	"CAP_LEASE",
	"CAP_AUDIT_WRITE",
	"CAP_AUDIT_CONTROL",
	"CAP_SETFCAP",
}

// 危险的 capabilities 列表
var DangerousCapabilities = map[string]string{
	"CAP_SYS_ADMIN":   "允许执行系统管理操作，如加载内核模块、修改内核参数",
	"CAP_SYS_MODULE":  "允许加载/卸载内核模块",
	"CAP_SYS_RAWIO":   "允许直接访问原始设备",
	"CAP_SYS_CHROOT":  "允许使用 chroot",
	"CAP_SYS_PTRACE":  "允许使用 ptrace 追踪进程",
	"CAP_SYS_TIME":    "允许修改系统时间",
	"CAP_SYS_BOOT":    "允许重新启动系统",
	"CAP_NET_ADMIN":   "允许网络管理操作",
	"CAP_NET_RAW":     "允许原始套接字",
	"CAP_IPC_OWNER":   "允许忽略 IPC 权限检查",
	"CAP_SYS_RESOURCE": "允许修改资源限制",
	"CAP_DAC_READ_SEARCH": "忽略文件读取权限和目录搜索权限",
	"CAP_DAC_OVERRIDE":    "忽略文件访问权限检查",
}

// 危险的挂载点
var DangerousMounts = map[string]string{
	"/":                     "根目录挂载，可访问宿主机文件系统",
	"/proc":                 "proc 挂载，可获取宿主机进程信息",
	"/root":                 "root 目录挂载，可访问宿主机 root 文件",
	"/etc":                  "etc 目录挂载，可修改系统配置",
	"/var/run/docker.sock": "Docker socket，可通过 docker escape",
	"/var/run/containerd":  "containerd socket",
	"/var/run/crio":        "crio socket",
	"/host":                "宿主机文件系统",
	"/mnt/host":            "宿主机文件系统",
	"/dev":                 "设备文件目录",
	"/sys":                 "sys 文件系统",
	"/sys/kernel":          "内核信息",
	"/etc/shadow":          "shadow 文件，可尝试读取密码哈希",
	"/etc/passwd":          "passwd 文件，可修改用户信息",
}

// 危险的 SUID 文件
var DangerousSuidBinaries = map[string]string{
	"/bin/bash":   "可执行 bash，获得 shell",
	"/bin/sh":     "可执行 sh，获得 shell",
	"/bin/dash":   "可执行 dash，获得 shell",
	"/usr/bin/bash": "可执行 bash，获得 shell",
	"/usr/bin/sh":   "可执行 sh，获得 shell",
	"/usr/bin/python": "可执行 python，逃逸容器",
	"/usr/bin/perl":   "可执行 perl，逃逸容器",
	"/usr/bin/ruby":   "可执行 ruby，逃逸容器",
	"/usr/bin/vim":    "可执行 vim，可读取敏感文件",
	"/usr/bin/vi":     "可执行 vi，可读取敏感文件",
	"/usr/bin/nano":   "可执行 nano，可读取敏感文件",
	"/usr/bin/less":   "可执行 less，可读取文件",
	"/usr/bin/more":   "可执行 more，可读取文件",
	"/usr/bin/find":   "可执行 find，可用于提权",
	"/usr/bin/tar":    "可执行 tar，可用于文件操作",
	"/usr/bin/wget":   "可执行 wget，可下载恶意程序",
	"/usr/bin/curl":   "可执行 curl，可下载恶意程序",
	"/usr/bin/nc":     "可执行 netcat，可用于反弹 shell",
	"/usr/bin/socat":  "可执行 socat，可用于端口转发",
	"/bin/nc":         "可执行 netcat",
	"/bin/socat":      "可执行 socat",
	"/sbin/nologin":   "可尝试修改为可登录",
}

// 可写入的敏感文件
var SensitiveWritableFiles = []string{
	"/etc/passwd",
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/cron.d",
	"/etc/cron.daily",
	"/etc/cron.hourly",
	"/etc/cron.monthly",
	"/etc/cron.weekly",
	"/var/spool/cron",
	"/root/.ssh/authorized_keys",
	"/home/*/.ssh/authorized_keys",
}