package el

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ELClientLauncher interface {
	Launch(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		image string,
		participantLogLevel string,
		globalLogLevel module_io.GlobalClientLogLevel,
		// If empty then the node will be launched as a bootnode
		existingElClients []*ELClientContext,
		extraParams []string,
	) (
		resultClientCtx *ELClientContext,
		resultErr error,
	)
}
