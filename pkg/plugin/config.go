package plugin

type Dependency struct {
}

type CommandParam struct {
	Type   string
	Path   string
	Params []string
}

type PluginConfig struct {
	Name         string
	Version      string
	Dependencies []Dependency
	Type         string
	Start        CommandParam
	Stop         CommandParam
}
