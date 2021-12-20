package el

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ELClientLauncher interface {
	Launch(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		networkId string,
		bootnodeContext *ELClientContext,
	) (
		resultClientCtx *ELClientContext,
		resultErr error,
	)
}
