package cl_client_network

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ConsensusLayerClientContext struct {
	serviceCtx *services.ServiceContext
	enode string
}

func NewConsensusLayerClientContext(serviceCtx *services.ServiceContext, enode string) *ConsensusLayerClientContext {
	return &ConsensusLayerClientContext{serviceCtx: serviceCtx, enode: enode}
}

func (ctx *ConsensusLayerClientContext) GetServiceContext() *services.ServiceContext {
	return ctx.serviceCtx
}
func (ctx *ConsensusLayerClientContext) GetEnode() string {
	return ctx.enode
}
