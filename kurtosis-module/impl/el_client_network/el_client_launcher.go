package el_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ExecutionLayerClientLauncher interface {
	LaunchBootNode(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		networkId string,
		genesisJsonFilepathOnModuleContainer string,
	) (
		resultClientCtx *ExecutionLayerClientContext,
		resultErr error,
	)

	LaunchChildNode(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		networkId string,
		genesisJsonFilepathOnModuleContainer string,
		bootnodeEnode string,
	) (
		resultClientCtx *ExecutionLayerClientContext,
		resultErr error,
	)
}
