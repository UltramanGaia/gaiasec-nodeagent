package model

const (
	TaskEvent     = "task"
	InitEvent     = "init"
	DeployEvent   = "deploy"
	UndeployEvent = "undeploy"
	ProxyEvent    = "proxy"
	PushEvent     = "push"
	PullEvent     = "pull"
	TerminalEvent = "terminal"
	UpgradeEvent  = "upgrade"
	LoginEvent    = "login"
	LogoutEvent   = "logout"
)

const (
	EXECUTE_COMMAND = "EXECUTE_COMMAND"
	GET_PROCESSES   = "GET_PROCESSES"
	PTY_CREATE      = "PTY_CREATE"
	FS_LIST_DIR     = "FS_LIST_DIR"
	FS_READ_FILE    = "FS_READ_FILE"
	FS_WRITE_FILE   = "FS_WRITE_FILE"
	FS_CREATE_FILE  = "FS_CREATE_FILE"
	FS_CREATE_DIR   = "FS_CREATE_DIR"
	FS_DELETE       = "FS_DELETE"
	FS_RENAME       = "FS_RENAME"
	DEPLOY_PLUGIN   = "DEPLOY_PLUGIN"
)

const (
	COMMAND_RESULT          = "COMMAND_RESULT"
	PLUGIN_DEPLOY_RESPONSE  = "PLUGIN_DEPLOY_RESPONSE"
	PLUGIN_CONTROL_RESPONSE = "PLUGIN_CONTROL_RESPONSE"
	PLUGIN_STATUS_RESPONSE  = "PLUGIN_STATUS_RESPONSE"
	PLUGIN_STATUS_UPDATE    = "PLUGIN_STATUS_UPDATE"
	NODE_INFO_REPORT        = "NODE_INFO_REPORT"
	FS_RESPONSE             = "FS_RESPONSE"
	PROCESSES_RESPONSE      = "PROCESSES_RESPONSE"
	PTY_CREATED             = "PTY_CREATED"
	HEARTBEAT               = "HEARTBEAT"
	EVENT                   = "EVENT"
	ERROR_RESPONSE          = "ERROR_RESPONSE"
)
