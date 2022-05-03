package cl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type CLClientLauncher interface {
	// Launches both a Beacon client AND a validator
	Launch(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		image string,
		participantLogLevel string,
		globalLogLevel module_io.GlobalClientLogLevel,
		// If nil, the node will be launched as a bootnode
		bootnodeContext *CLClientContext,
		elClientContext *el.ELClientContext,
		nodeKeystoreFiles *cl_validator_keystores.KeystoreFiles,
		extraBeaconParams []string,
		extraValidatorParams []string,
	) (
		resultClientCtx *CLClientContext,
		resultErr error,
	)
}
