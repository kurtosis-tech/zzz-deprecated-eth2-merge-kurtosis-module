package cl_client_network

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"sync"
)

const (
	bootnodeNodeIndex = uint32(0)
	serviceIdPrefix   = "cl-client-"
)

type ConsensusLayerNetwork struct {
	enclaveCtx       *enclaves.EnclaveContext
	elClientContexts []*el_client_network.ExecutionLayerClientContext

	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths

	// TODO refactor to have an ID so we can launch different clients
	clientLauncher ConsensusLayerClientLauncher

	nodeClientCtx map[uint32]*ConsensusLayerClientContext
	nextNodeIndex uint32
	mutex         *sync.Mutex
}

func NewConsensusLayerNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	elClientContexts []*el_client_network.ExecutionLayerClientContext,
	clientLauncher ConsensusLayerClientLauncher,
	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) *ConsensusLayerNetwork {
	return &ConsensusLayerNetwork{
		enclaveCtx:                         enclaveCtx,
		elClientContexts:                   elClientContexts,
		preregisteredValidatorKeysForNodes: preregisteredValidatorKeysForNodes,
		clientLauncher:                     clientLauncher,
		nodeClientCtx:                      map[uint32]*ConsensusLayerClientContext{},
		nextNodeIndex:                      bootnodeNodeIndex,
		mutex:                              &sync.Mutex{},
	}
}

func (network *ConsensusLayerNetwork) AddNode() (*ConsensusLayerClientContext, error) {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	elClientRpcSocketStrs, err := network.getElClientRpcSockets()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred converting the EL client contexts into RPC IP:port socket strings")
	}

	newNodeIndex := network.nextNodeIndex
	if newNodeIndex >= uint32(len(network.preregisteredValidatorKeysForNodes)) {
		return nil, stacktrace.NewError(
			"Cannot add new node; when the CL genesis was generated with preregistered validator keys, only %v validator nodes were expected",
			len(network.preregisteredValidatorKeysForNodes),
		)
	}
	newNodeKeystores := network.preregisteredValidatorKeysForNodes[newNodeIndex]

	serviceId := services.ServiceID(fmt.Sprintf("%v%v", serviceIdPrefix, newNodeIndex))
	var newClientCtx *ConsensusLayerClientContext
	var nodeLaunchErr error
	if network.nextNodeIndex == bootnodeNodeIndex {
		newClientCtx, nodeLaunchErr = network.clientLauncher.LaunchBootNode(
			network.enclaveCtx,
			serviceId,
			elClientRpcSocketStrs,
			newNodeKeystores,
		)
	} else {
		bootnodeClientCtx, found := network.nodeClientCtx[bootnodeNodeIndex]
		if !found {
			return nil, stacktrace.NewError("The EL client network has >= 1 nodes, but we couldn't find a node with bootnode ID '%v'; this is a bug in the module!", bootnodeNodeIndex)
		}
		newClientCtx, nodeLaunchErr = network.clientLauncher.LaunchChildNode(
			network.enclaveCtx,
			serviceId,
			bootnodeClientCtx.GetENR(),
			elClientRpcSocketStrs,
			newNodeKeystores,
		)
	}
	if nodeLaunchErr != nil {
		return nil, stacktrace.Propagate(nodeLaunchErr, "An error occurred launching node with service ID '%v'", serviceId)
	}
	network.nextNodeIndex = network.nextNodeIndex + 1
	network.nodeClientCtx[newNodeIndex] = newClientCtx

	return newClientCtx, nil
}

// ====================================================================================================
//                                    Private Helper Functions
// ====================================================================================================
// Returns a "set" of IP:port info for each of the execution layer clients' RPC ports
func (network *ConsensusLayerNetwork) getElClientRpcSockets() (map[string]bool, error) {
	result := map[string]bool{}
	for _, elClientCtx := range network.elClientContexts {
		rpcPortId := elClientCtx.GetRPCPortID()
		serviceCtx := elClientCtx.GetServiceContext()
		privateIp := serviceCtx.GetPrivateIPAddress()
		rpcPort, found := serviceCtx.GetPrivatePorts()[rpcPortId]
		if !found {
			return nil, stacktrace.NewError(
				"Expected a port with ID '%v' for execution layer client '%v' but none was found",
				rpcPortId,
				serviceCtx.GetServiceID(),
			)
		}
		rpcSocketStr := fmt.Sprintf("%v:%v", privateIp, rpcPort.GetNumber())
		result[rpcSocketStr] = true
	}
	return result, nil
}
