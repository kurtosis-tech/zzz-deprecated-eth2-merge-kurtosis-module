package transaction_spammer

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strings"
)

const (
	imageName = "kurtosistech/tx-fuzz:0.2.0"

	serviceId = "transaction-spammer"
)

// TODO upgrade the spammer to be able to take in multiple EL addreses
func LaunchTransanctionSpammer(enclaveCtx *enclaves.EnclaveContext, prefundedAddresses []*genesis_consts.PrefundedAccount, elClientCtx *el.ELClientContext) error {
	containerConfigSupplier := getContainerConfigSupplier(prefundedAddresses, elClientCtx)

	_, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred adding the transaction spammer service")
	}

	return nil
}

func getContainerConfigSupplier(
	prefundedAddresses []*genesis_consts.PrefundedAccount,
	elClientCtx *el.ELClientContext,
) func(string) (*services.ContainerConfig, error) {
	privateKeysStrs := []string{}
	addressStrs := []string{}

	for _, prefundedAddress := range prefundedAddresses {
		privateKeysStrs = append(privateKeysStrs, prefundedAddress.PrivKey)
		addressStrs = append(addressStrs, prefundedAddress.Address)
	}

	commaSeparatedPrivateKeys := strings.Join(privateKeysStrs, ",")
	commaSeparatedAddresses := strings.Join(addressStrs, ",")
	return func(privateIpAddr string) (*services.ContainerConfig, error) {
		result := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride([]string{
			fmt.Sprintf("http://%v:%v", elClientCtx.GetIPAddress(), elClientCtx.GetRPCPortNum()),
			"spam",
			commaSeparatedPrivateKeys,
			commaSeparatedAddresses,
		}).Build()
		return result, nil
	}
}