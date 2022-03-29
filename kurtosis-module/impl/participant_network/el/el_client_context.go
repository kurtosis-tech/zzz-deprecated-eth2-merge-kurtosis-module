package el

type ELClientContext struct {
	clientName       string
	enr              string
	enode            string
	ipAddr           string
	rpcPortNum       uint16
	wsPortNum        uint16
	engineRpcPortNum uint16
	miningWaiter     ELClientMiningWaiter
}

func NewELClientContext(clientName string, enr string, enode string, ipAddr string, rpcPortNum uint16, wsPortNum uint16, engineRpcPortNum uint16, miningWaiter ELClientMiningWaiter) *ELClientContext {
	return &ELClientContext{clientName: clientName, enr: enr, enode: enode, ipAddr: ipAddr, rpcPortNum: rpcPortNum, wsPortNum: wsPortNum, engineRpcPortNum: engineRpcPortNum, miningWaiter: miningWaiter}

}

func (ctx *ELClientContext) GetClientName() string {
	return ctx.clientName
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
func (ctx *ELClientContext) GetEngineRPCPortNum() uint16 {
	return ctx.engineRpcPortNum
}
func (ctx *ELClientContext) GetMiningWaiter() ELClientMiningWaiter {
	return ctx.miningWaiter
}
