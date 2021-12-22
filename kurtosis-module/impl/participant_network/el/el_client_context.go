package el

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ELClientContext struct {
	serviceCtx *services.ServiceContext
	enr string
	enode string
	ipAddr string
	rpcPortNum  uint16
	wsPortNum uint16
}

func NewELClientContext(serviceCtx *services.ServiceContext, enr string, enode string, ipAddr string, rpcPortNum uint16, wsPortNum uint16) *ELClientContext {
	return &ELClientContext{serviceCtx: serviceCtx, enr: enr, enode: enode, ipAddr: ipAddr, rpcPortNum: rpcPortNum, wsPortNum: wsPortNum}
}

func (ctx *ELClientContext) GetENR() string {
	return ctx.enr
}
func (ctx *ELClientContext) GetEnode() string {
	return ctx.enode
}
func (ctx *ELClientContext) GetIPAddress() string {
	return ctx.ipAddr
}
func (ctx *ELClientContext) GetRPCPortNum() uint16 {
	return ctx.rpcPortNum
}
func (ctx *ELClientContext) GetWSPortNum() uint16 {
	return ctx.wsPortNum
}

