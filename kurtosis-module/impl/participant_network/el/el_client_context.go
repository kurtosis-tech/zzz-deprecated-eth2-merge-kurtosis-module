package el

type ELClientContext struct {
	enr string
	enode string
	ipAddr string
	rpcPortNum  uint16
	discoveryPortNum uint16
	publicIpAddr string
	publicDiscoveryPortNum uint16
	wsPortNum uint16
	miningWaiter ELClientMiningWaiter
}

func NewELClientContext(enr string, enode string, ipAddr string, rpcPortNum uint16,discoveryPortNum uint16, publicIpAddr string, publicDiscoveryPortNum uint16, wsPortNum uint16, miningWaiter ELClientMiningWaiter) *ELClientContext {
	return &ELClientContext{enr: enr, enode: enode, ipAddr: ipAddr, rpcPortNum: rpcPortNum, discoveryPortNum: discoveryPortNum, publicIpAddr: publicIpAddr, publicDiscoveryPortNum: publicDiscoveryPortNum, wsPortNum: wsPortNum, miningWaiter: miningWaiter}
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
func (ctx *ELClientContext) GetDiscoveryPortNum() uint16 {
	return ctx.discoveryPortNum
}
func (ctx *ELClientContext) GetWSPortNum() uint16 {
	return ctx.wsPortNum
}
func (ctx *ELClientContext) GetMiningWaiter() ELClientMiningWaiter {
	return ctx.miningWaiter
}
func (ctx *ELClientContext) GetPublicIPAddress() string {
	return ctx.publicIpAddr
}
func (ctx *ELClientContext) GetPublicDiscoveryPortNum() uint16 {
	return ctx.publicDiscoveryPortNum
}
