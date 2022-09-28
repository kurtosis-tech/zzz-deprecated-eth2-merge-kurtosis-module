package testnet_verifier

import (
	"fmt"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

// We use Docker exec commands to run the commands we need, so we override the default
var synchronousEntrypointArgs = []string{
	"sleep",
	"999999",
}

const (
	imageName = "marioevz/merge-testnet-verifier:latest"
	serviceId = "testnet-verifier"
)

func LaunchAsynchronousTestnetVerifier(params *module_io.ExecuteParams, enclaveCtx *enclaves.EnclaveContext, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) error {
	containerConfig := getAsynchronousVerificationContainerConfig(params, elClientCtxs, clClientCtxs, ttd)

	_, err := enclaveCtx.AddService(serviceId, containerConfig)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred adding the testnet verifier service")
	}

	return nil
}

func RunSynchronousTestnetVerification(params *module_io.ExecuteParams, enclaveCtx *enclaves.EnclaveContext, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) (int32, string, error) {
	containerConfig := getSynchronousVerificationContainerConfig()

	svcCtx, err := enclaveCtx.AddService(serviceId, containerConfig)
	if err != nil {
		return 1, "", stacktrace.Propagate(err, "An error occurred adding the testnet verifier service")
	}

	cmd := getCmd(params, elClientCtxs, clClientCtxs, ttd, true)

	return svcCtx.ExecCommand(cmd)

}

func getCmd(params *module_io.ExecuteParams, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64, addBinaryName bool) []string {
	cmd := make([]string, 0)
	if addBinaryName {
		cmd = append(cmd, "./merge_testnet_verifier")
	}
	cmd = append(cmd, "--ttd")
	cmd = append(cmd, fmt.Sprintf("%d", ttd))

	for _, elClientCtx := range elClientCtxs {
		cmd = append(cmd, "--client")
		cmd = append(cmd, fmt.Sprintf("%s,http://%v:%v", elClientCtx.GetClientName(), elClientCtx.GetIPAddress(), elClientCtx.GetRPCPortNum()))
	}
	for _, clClientCtx := range clClientCtxs {
		cmd = append(cmd, "--client")
		cmd = append(cmd, fmt.Sprintf("%s,http://%v:%v", clClientCtx.GetClientName(), clClientCtx.GetIPAddress(), clClientCtx.GetHTTPPortNum()))
	}

	cmd = append(cmd, "--ttd-epoch-limit")
	cmd = append(cmd, fmt.Sprintf("%d", params.VerificationsTTDEpochLimit))

	cmd = append(cmd, "--verif-epoch-limit")
	cmd = append(cmd, fmt.Sprintf("%d", params.VerificationsEpochLimit))

	return cmd
}

func getAsynchronousVerificationContainerConfig(
	params *module_io.ExecuteParams,
	elClientCtxs []*el.ELClientContext,
	clClientCtxs []*cl.CLClientContext,
	ttd uint64,
) *services.ContainerConfig {
	cmd := getCmd(params, elClientCtxs, clClientCtxs, ttd, false)
	result := services.NewContainerConfigBuilder(
		imageName,
	).WithCmdOverride(cmd).Build()
	return result
}

func getSynchronousVerificationContainerConfig() *services.ContainerConfig {
	result := services.NewContainerConfigBuilder(
		imageName,
	).WithEntrypointOverride(
		synchronousEntrypointArgs,
	).Build()
	return result
}
