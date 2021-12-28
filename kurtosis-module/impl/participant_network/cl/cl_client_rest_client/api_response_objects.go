package cl_client_rest_client

type GetNodeIdentityResponse struct {
	Data *NodeIdentity `json:"data"`
}

type NodeIdentity struct {
	ENR string `json:"enr"`
}

type GetBlockHeadersResponse struct {
	Data []*BlockHeaderData `json:"data"`
}

type BlockHeaderData struct {
	Header *BlockHeaderInfo`json:"header"`
}

type BlockHeaderInfo struct {
	Message	*BlockHeaderMessage `json:"message"`
}

type BlockHeaderMessage struct {
	Slot string	`json:"slot"`
}

type GetFinalityCheckpointsResponse struct {
	Data *FinalityCheckpoints `json:"data"`
}

// https://ethereum.github.io/beacon-APIs/#/Beacon/getStateFinalityCheckpoints
type FinalityCheckpoints struct {
	Finalized *FinalityCheckpointInfo	`json:"finalized"`
}

type FinalityCheckpointInfo struct {
	Epoch string	`json:"epoch"`
}