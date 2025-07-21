package model

type LoginResp struct {
	PluginName string `json:"pluginName"`
	AgentId    string `json:"agentId"`
}

type LogoutResp struct {
	AgentId string `json:"agentId"`
}
