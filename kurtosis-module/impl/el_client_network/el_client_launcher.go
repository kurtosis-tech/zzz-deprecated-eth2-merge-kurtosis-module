package el_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ExecutionLayerClientLauncher interface {
	LaunchBootNode(
		enclaveCtx *enclaves.EnclaveContext,
		networkId string,
		genesisJsonFilepathOnModuleContainer string,
	) (*services.ServiceContext, string, error)

	/*
	LaunchChildNode(
		enclaveCtx *enclaves.EnclaveContext,
		networkId string,
		genesisJsonFilepathOnModuleContainer string,
		bootNodeEnr string,
	) (*services.ServiceContext, error)
	 */
}
