package cl_client_network

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ConsensusLayerClientContext struct {
	serviceCtx *services.ServiceContext
	enr string
	httpPortId string
}

func NewConsensusLayerClientContext(serviceCtx *services.ServiceContext, enr string, httpPortId string) *ConsensusLayerClientContext {
	return &ConsensusLayerClientContext{serviceCtx: serviceCtx, enr: enr, httpPortId: httpPortId}
}

func (ctx *ConsensusLayerClientContext) GetServiceContext() *services.ServiceContext {
	return ctx.serviceCtx
}
func (ctx *ConsensusLayerClientContext) GetENR() string {
	return ctx.enr
}
func (ctx *ConsensusLayerClientContext) GetHTTPPortID() string {
	return ctx.httpPortId
}

