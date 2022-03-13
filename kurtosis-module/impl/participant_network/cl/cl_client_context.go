package cl

import "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"

type CLClientContext struct {
	enr               string
	peerId            string
	ipAddr            string
	httpPortNum       uint16
	publicIpAddr      string
	publicHttpPortNum uint16
	nodesMetricsInfo []*CLNodeMetricsInfo
	restClient *cl_client_rest_client.CLClientRESTClient
}

func NewCLClientContext(
	enr string,
	peerId string,
	ipAddr string,
	httpPortNum uint16,
	publicIpAddr string,
	publicHttpPortNum uint16,
	nodesMetricsInfo []*CLNodeMetricsInfo, 
	restClient *cl_client_rest_client.CLClientRESTClient) *CLClientContext {
	return &CLClientContext{
		enr: enr,
		peerId: peerId,
		ipAddr: ipAddr,
		httpPortNum: httpPortNum,
		publicIpAddr: publicIpAddr,
		publicHttpPortNum: publicHttpPortNum,
		nodesMetricsInfo: nodesMetricsInfo,
		restClient: restClient,
	}
}

func (ctx *CLClientContext) GetENR() string {
	return ctx.enr
}

func (ctx *CLClientContext) GetPeerId() string {
	return ctx.peerId
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

func (ctx *CLClientContext) GetNodesMetricsInfo() []*CLNodeMetricsInfo {
	return ctx.nodesMetricsInfo
}

func (ctx *CLClientContext) GetPublicIPAddress() string {
	return ctx.publicIpAddr
}

func (ctx *CLClientContext) GetPublicHTTPPortNum() uint16 {
	return ctx.publicHttpPortNum
}