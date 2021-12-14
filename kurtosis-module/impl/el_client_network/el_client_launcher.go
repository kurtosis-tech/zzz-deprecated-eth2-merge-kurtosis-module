package el_client_network

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
)

type ExecutionLayerClientLauncher struct {
	enclaveCtx *enclaves.EnclaveContext
	networkId string
	genesisJsonFilepathOnModuleContainer string
}

// TODO constructor

/*
func (launcher *ExecutionLayerClientLauncher) LaunchBootNode() (
	*services.ServiceContext,
	error,
) {


}


 */