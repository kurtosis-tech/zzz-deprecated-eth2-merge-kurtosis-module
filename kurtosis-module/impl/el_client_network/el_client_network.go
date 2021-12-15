package el_client_network

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"sync"
)

const (
	bootnodeNodeIndex = uint32(0)
	serviceIdPrefix   = "el-client-"
)


type ExecutionLayerNetwork struct {
	enclaveCtx *enclaves.EnclaveContext
	networkId string

	// TODO refactor to have an ID so we can launch different clients
	elClientLauncher ExecutionLayerClientLauncher

	nodeClientCtx map[uint32]*ExecutionLayerClientContext
	nextNodeIndex uint32
	mutex         *sync.Mutex
}

func NewExecutionLayerNetwork(enclaveCtx *enclaves.EnclaveContext, networkId string, elClientLauncher ExecutionLayerClientLauncher) *ExecutionLayerNetwork {
	return &ExecutionLayerNetwork{
		enclaveCtx:                           enclaveCtx,
		networkId:                            networkId,
		elClientLauncher:                     elClientLauncher,
		nodeClientCtx:                        map[uint32]*ExecutionLayerClientContext{},
		nextNodeIndex:                        bootnodeNodeIndex,
		mutex:                                &sync.Mutex{},
	}
}

func (network *ExecutionLayerNetwork) AddNode() error {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	newNodeIndex := network.nextNodeIndex
	serviceId := services.ServiceID(fmt.Sprintf("%v%v", serviceIdPrefix, newNodeIndex))
	var newClientCtx *ExecutionLayerClientContext
	var nodeLaunchErr error
	if network.nextNodeIndex == bootnodeNodeIndex {
		newClientCtx, nodeLaunchErr = network.elClientLauncher.LaunchBootNode(
			network.enclaveCtx,
			serviceId,
			network.networkId,
		)
	} else {
		bootnodeClientCtx, found := network.nodeClientCtx[bootnodeNodeIndex]
		if !found {
			return stacktrace.NewError("The EL client network has >= 1 nodes, but we couldn't find a node with bootnode ID '%v'; this is a bug in the module!", bootnodeNodeIndex)
		}
		newClientCtx, nodeLaunchErr = network.elClientLauncher.LaunchChildNode(
			network.enclaveCtx,
			serviceId,
			network.networkId,
			bootnodeClientCtx.GetEnode(),
		)
	}
	if nodeLaunchErr != nil {
		return stacktrace.Propagate(nodeLaunchErr, "An error occurred launching node with service ID '%v'", serviceId)
	}
	network.nextNodeIndex = network.nextNodeIndex + 1
	network.nodeClientCtx[newNodeIndex] = newClientCtx

	return nil
}