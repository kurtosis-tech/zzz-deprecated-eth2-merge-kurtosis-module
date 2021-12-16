package cl_client_network

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ConsensusLayerClientContext struct {
	serviceCtx *services.ServiceContext
	enr string
}

func NewConsensusLayerClientContext(serviceCtx *services.ServiceContext, enr string) *ConsensusLayerClientContext {
	return &ConsensusLayerClientContext{serviceCtx: serviceCtx, enr: enr}
}

func (ctx *ConsensusLayerClientContext) GetServiceContext() *services.ServiceContext {
	return ctx.serviceCtx
}
func (ctx *ConsensusLayerClientContext) GetENR() string {
	return ctx.enr
}

