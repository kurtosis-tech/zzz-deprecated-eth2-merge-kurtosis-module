package el

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ELClientLauncher interface {
	Launch(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		image string,
		loglevel string,
		// If nil, then the node will be launched as a bootnode
		bootnodeContext *ELClientContext,
	) (
		resultClientCtx *ELClientContext,
		resultErr error,
	)
}
