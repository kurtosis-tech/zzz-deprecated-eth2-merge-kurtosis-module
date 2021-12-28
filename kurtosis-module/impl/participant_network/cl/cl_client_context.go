package cl

import "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"

const (
	headersEndpoint = "/eth/v1/beacon/headers"
)

type CLClientContext struct {
	enr         string
	ipAddr      string
	httpPortNum uint16
	restClient *cl_client_rest_client.CLClientRESTClient
}

func NewCLClientContext(enr string, ipAddr string, httpPortNum uint16, restClient *cl_client_rest_client.CLClientRESTClient) *CLClientContext {
	return &CLClientContext{enr: enr, ipAddr: ipAddr, httpPortNum: httpPortNum, restClient: restClient}
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

func (ctx *CLClientContext) GetRESTClient() *cl_client_rest_client.CLClientRESTClient {
	return ctx.restClient
}
