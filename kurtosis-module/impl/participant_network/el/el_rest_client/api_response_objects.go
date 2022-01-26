package el_rest_client

// Will be serialized
type RequestBody struct {
	JsonRPC string	`json:"jsonrpc"`
	Method string	`json:"method"`
	Params []string	`json:"params"`
	ID uint			`json:"id"`
}

type GetBlockNumberResponse struct {
	// Hex-encoded block number string
	HexEncodedBlockNumberStr string `json:"result"`
}

type GetNodeInfoResponse struct {
	Result *NodeInfo `json:"result"`
}

type NodeInfo struct {
	Enode string	`json:"enode"`
}
