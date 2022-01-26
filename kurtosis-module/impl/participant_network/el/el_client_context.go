package el

type ELClientContext struct {
	enode string
	ipAddr string
	rpcPortNum  uint16
	wsPortNum uint16
	miningWaiter ELClientMiningWaiter
}

func NewELClientContext(enode string, ipAddr string, rpcPortNum uint16, wsPortNum uint16, miningWaiter ELClientMiningWaiter) *ELClientContext {
	return &ELClientContext{enode: enode, ipAddr: ipAddr, rpcPortNum: rpcPortNum, wsPortNum: wsPortNum, miningWaiter: miningWaiter}
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
func (ctx *ELClientContext) GetMiningWaiter() ELClientMiningWaiter {
	return ctx.miningWaiter
}

