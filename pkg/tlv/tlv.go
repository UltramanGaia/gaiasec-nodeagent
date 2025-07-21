package tlv

const (
	AGENT_MSG_TYPE_LOGIN = 0
)

const (
	TaskEvent         = 0
	InitEvent         = 1
	DeployEvent       = 2
	UndeployEvent     = 3
	LoginEvent        = 4
	LogoutEvent       = 5
	PushEvent         = 6
	PullEvent         = 7
	TerminalEvent     = 8
	UpgradeEvent      = 9
	UpgradeProxyEvent = 10
	ProxyEvent        = 11
	PortForwardEvent  = 12
	FileEvent         = 13
	ProcessEvent      = 14
	NetEvent          = 15
	RouteEvent        = 16
	KickOutEvent      = 17
	UserEvent         = 18
	ContainerEvent    = 19
	ModEvent          = 20
	ServiceEvent      = 21
	CronEvent         = 22
	DetailEvent       = 23
	RebootEvent       = 24
	StatusEvent       = 25

	ErrorEvent      = 253
	DisconnectEvent = 254
	ConnectionEvent = 255
)
