package cl

type CLClientContext struct {
	enr string
	ipAddr string
	httpPortNum uint16
}

func NewCLClientContext(enr string, ipAddr string, httpPortNum uint16) *CLClientContext {
	return &CLClientContext{enr: enr, ipAddr: ipAddr, httpPortNum: httpPortNum}
}

func (ctx *CLClientContext) GetENR() string {
	return ctx.enr
}

func (ctx *CLClientContext) GetIPAddress() string {
	return ctx.ipAddr
}

func (ctx *CLClientContext) GetHTTPPortNum() uint16 {
	return ctx.httpPortNum
}

