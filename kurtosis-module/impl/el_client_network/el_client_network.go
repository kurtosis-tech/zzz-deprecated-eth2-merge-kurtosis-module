package el_client_network

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type ethereumNode struct {
	serviceCtx *services.ServiceContext


}

type ExecutionLayerNetwork struct {
	elClientLaunche *ExecutionLayerClientLauncher

	nodes map[uint32]*ethereumNode
	nextNodeId uint32
}

// TODO constructor

/*
func (network *ExecutionLayerNetwork) AddNode() {
	isBootNode := len(network.nodeServiceCtxs) == 0
	if isBootNode {

	} else {

	}
}

 */