package el_client_network

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ExecutionLayerClientContext struct {
	serviceCtx *services.ServiceContext
	enr string
	enode string
	rpcPortId string
}

func NewExecutionLayerClientContext(serviceCtx *services.ServiceContext, enr string, enode string, rpcPortId string) *ExecutionLayerClientContext {
	return &ExecutionLayerClientContext{serviceCtx: serviceCtx, enr: enr, enode: enode, rpcPortId: rpcPortId}
}

func (ctx *ExecutionLayerClientContext) GetServiceContext() *services.ServiceContext {
	return ctx.serviceCtx
}
func (ctx *ExecutionLayerClientContext) GetENR() string {
	return ctx.enr
}
func (ctx *ExecutionLayerClientContext) GetEnode() string {
	return ctx.enode
}
func (ctx *ExecutionLayerClientContext) GetRPCPortID() string {
	return ctx.rpcPortId
}
