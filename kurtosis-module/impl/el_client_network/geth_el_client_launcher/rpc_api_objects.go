package geth_el_client_launcher

type GetNodeInfoResponse struct {
	Result NodeInfo `json:"result"`
}

type NodeInfo struct {
	Enode string	`json:"enode"`
	ENR   string	`json:"enr"`
}
