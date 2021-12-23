package cl_client_rest_client

type GetNodeIdentityResponse struct {
	Data *NodeIdentity `json:"data"`
}

type GetNodeSyncingDataResponse struct {
	Data *SyncingData `json:"data"`
}

type NodeIdentity struct {
	ENR string	`json:"enr"`
}

type SyncingData struct {
	HeadSlot int `json:"head_slot"`
	SyncDistance int `json:"sync_distance"`
	IsSyncing bool `json:"is_syncing"`
}