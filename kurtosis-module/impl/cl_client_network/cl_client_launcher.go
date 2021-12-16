package cl_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ConsensusLayerClientLauncher interface {
	LaunchBootNode(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		elClientRpcSockets map[string]bool,  // IP:port of EL client RPC sockets
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
	) (
		resultClientCtx *ConsensusLayerClientContext,
		resultErr error,
	)
}
