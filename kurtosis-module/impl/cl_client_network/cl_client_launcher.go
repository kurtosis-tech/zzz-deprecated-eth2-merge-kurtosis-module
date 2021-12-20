package cl_client_network

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ConsensusLayerClientLauncher interface {
	LaunchBootNode(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		elClientRpcSockets map[string]bool,  // IP:port of EL client RPC sockets
		nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
	) (
		resultClientCtx *ConsensusLayerClientContext,
		resultErr error,
	)

	LaunchChildNode(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		// NOTE: the ENR of the *consensus layer* boot node
		bootnodeEnr string,
		elClientRpcSockets map[string]bool,  // IP:port of EL client RPC sockets
		nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
	) (
		resultClientCtx *ConsensusLayerClientContext,
		resultErr error,
	)
}
