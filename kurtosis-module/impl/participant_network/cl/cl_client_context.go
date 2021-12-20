package cl

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type CLClientContext struct {
	serviceCtx *services.ServiceContext
	enr string
	ipAddr string
	httpPortNum uint16
}

func NewConsensusLayerClientContext(serviceCtx *services.ServiceContext, enr string, httpPortId string) *CLClientContext {
	return &CLClientContext{serviceCtx: serviceCtx, enr: enr, httpPortId: httpPortId}
}

func (ctx *CLClientContext) GetServiceContext() *services.ServiceContext {
	return ctx.serviceCtx
}
func (ctx *CLClientContext) GetENR() string {
	return ctx.enr
}
func (ctx *CLClientContext) GetIPAddress() string {
	return ctx.ipAddr
}
func (ctx *CLClientContext) GetHTTPPortNum() uint16 {
	return ctx.enr
}

