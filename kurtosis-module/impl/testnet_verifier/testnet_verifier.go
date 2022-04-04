package testnet_verifier

import (
	"fmt"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

// We use Docker exec commands to run the commands we need, so we override the default
var entrypointArgs = []string{
	"sleep",
	"999999",
}

const (
	imageName = "marioevz/merge-testnet-verifier:latest"
	serviceId = "testnet-verifier"
)

func LaunchTestnetVerifier(params *module_io.ExecuteParams, enclaveCtx *enclaves.EnclaveContext, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) error {
	containerConfigSupplier := getContainerConfigSupplier(params, elClientCtxs, clClientCtxs, ttd)

	_, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred adding the testnet verifier service")
	}

	return nil
}

func RunTestnetVerifier(params *module_io.ExecuteParams, enclaveCtx *enclaves.EnclaveContext, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) (int32, string, error) {
	containerConfigSupplier := getSleepContainerConfigSupplier()

	svcCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
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

	if params.VerificationsTTDEpochLimit != nil {
		cmd = append(cmd, "--ttd-epoch-limit")
		cmd = append(cmd, fmt.Sprintf("%d", *params.VerificationsTTDEpochLimit))
	}

	if params.VerificationsEpochLimit != nil {
		cmd = append(cmd, "--verif-epoch-limit")
		cmd = append(cmd, fmt.Sprintf("%d", *params.VerificationsEpochLimit))
	}

	return cmd
}

func getContainerConfigSupplier(params *module_io.ExecuteParams, elClientCtxs []*el.ELClientContext, clClientCtxs []*cl.CLClientContext, ttd uint64) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		cmd := getCmd(params, elClientCtxs, clClientCtxs, ttd, false)
		result := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride(cmd).Build()
		return result, nil
	}
}

func getSleepContainerConfigSupplier() func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		result := services.NewContainerConfigBuilder(
			imageName,
		).WithEntrypointOverride(
			entrypointArgs,
		).Build()
		return result, nil
	}
}
