package besu

// TODO Copied from Geth; need to be updated for Besu
type GetNodeInfoResponse struct {
	Result NodeInfo `json:"result"`
}

// TODO Copied from Geth; need to be updated for Besu
type NodeInfo struct {
	Enode string	`json:"enode"`
	ENR   string	`json:"enr"`
}
