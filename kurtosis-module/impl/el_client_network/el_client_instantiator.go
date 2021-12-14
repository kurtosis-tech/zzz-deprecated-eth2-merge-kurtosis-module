package el_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ExecutionLayerClientInstantiator interface {
	LaunchBootNode(
	) (*services.ServiceContext, error)

	LaunchChildNode(
		enclaveCtx *enclaves.EnclaveContext,
		networkId string,
		genesisJsonFilepathOnModuleContainer string,
		bootNodeEnr string,
	) (*services.ServiceContext, error)
}
