package cl

import "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"

type CLClientContext struct {
	enr         string
	ipAddr      string
	httpPortNum uint16
	metricsInfo *CLClientMetricsInfo
	restClient *cl_client_rest_client.CLClientRESTClient
}

func NewCLClientContext(enr string, ipAddr string, httpPortNum uint16, metricsInfo *CLClientMetricsInfo, restClient *cl_client_rest_client.CLClientRESTClient) *CLClientContext {
	return &CLClientContext{enr: enr, ipAddr: ipAddr, httpPortNum: httpPortNum, metricsInfo: metricsInfo, restClient: restClient}
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

func (ctx *CLClientContext) GetMetricsInfo() *CLClientMetricsInfo {
	return ctx.metricsInfo
}