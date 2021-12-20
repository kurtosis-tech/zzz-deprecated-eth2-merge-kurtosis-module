package el

type ELClientContext struct {
	enr string
	enode string
	ipAddr string
	rpcPortNum  uint16
}

func NewELClientContext(enr string, enode string, ipAddr string, rpcPortNum uint16) *ELClientContext {
	return &ELClientContext{enr: enr, enode: enode, ipAddr: ipAddr, rpcPortNum: rpcPortNum}
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

