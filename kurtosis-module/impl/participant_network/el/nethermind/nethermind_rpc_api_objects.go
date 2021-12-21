package nethermind

type GetNodeInfoResponse struct {
	Result NodeInfo `json:"result"`
}

type NodeInfo struct {
	Enode string	`json:"enode"`
}
