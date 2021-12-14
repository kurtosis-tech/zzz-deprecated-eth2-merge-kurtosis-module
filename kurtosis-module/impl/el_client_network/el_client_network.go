package el_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

type ethereumNode struct {
	serviceCtx *services.ServiceContext
}

type ExecutionLayerNetwork struct {
	enclaveCtx *enclaves.EnclaveContext
	networkId string
	genesisJsonFilepathOnModuleContainer string

	// TODO refactor to have an ID
	elClientLauncher ExecutionLayerClientLauncher

	nodes map[uint32]*ethereumNode
	nextNodeId uint32
}

func NewExecutionLayerNetwork(enclaveCtx *enclaves.EnclaveContext, networkId string, genesisJsonFilepathOnModuleContainer string, elClientLauncher ExecutionLayerClientLauncher) *ExecutionLayerNetwork {
	return &ExecutionLayerNetwork{
		enclaveCtx: enclaveCtx,
		networkId: networkId,
		genesisJsonFilepathOnModuleContainer: genesisJsonFilepathOnModuleContainer,
		elClientLauncher: elClientLauncher,
		nodes: map[uint32]*ethereumNode{},
		nextNodeId: 0,
	}
}

func (network *ExecutionLayerNetwork) AddNode() error {
	_, _, err := network.elClientLauncher.LaunchBootNode(network.enclaveCtx, network.networkId, network.genesisJsonFilepathOnModuleContainer)
	if err != nil {
		// TODO make this error message more specific as we add non-boot nodes
		return stacktrace.Propagate(err, "An error occurred adding the node")
	}
	return nil
}