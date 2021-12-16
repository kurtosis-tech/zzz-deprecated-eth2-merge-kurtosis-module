package cl_client_rest_client

type GetNodeIdentityResponse struct {
	Data *NodeIdentity `json:"data"`
}

type NodeIdentity struct {
	ENR string	`json:"enr"`
}