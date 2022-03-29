package testnet_verifier

import (
	"fmt"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	imageName = "marioevz/merge-testnet-verifier:latest"
	serviceId = "testnet-verifier"
)

// TODO upgrade the spammer to be able to take in multiple EL addreses
func LaunchTestnetVerifier(enclaveCtx *enclaves.EnclaveContext, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) error {
	containerConfigSupplier := getContainerConfigSupplier(elClientCtxs, clClientCtxs, ttd)

	_, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred adding the testnet verifier service")
	}

	return nil
}

func getContainerConfigSupplier(elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) func(string, *services.SharedPath) (*services.ContainerConfig, error) {

	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		cmd := []string{
			"--ttd",
			fmt.Sprintf("%d", ttd),
		}
		for _, elClientCtx := range elClientCtxs {
			cmd = append(cmd, "--client")
			cmd = append(cmd, fmt.Sprintf("%s,http://%v:%v", elClientCtx.GetClientName(), elClientCtx.GetIPAddress(), elClientCtx.GetRPCPortNum()))
		}
		for _, clClientCtx := range clClientCtxs {
			cmd = append(cmd, "--client")
			cmd = append(cmd, fmt.Sprintf("%s,http://%v:%v", clClientCtx.GetClientName(), clClientCtx.GetIPAddress(), clClientCtx.GetHTTPPortNum()))
		}
		result := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride(cmd).Build()
		return result, nil
	}
}
