package geth_execution_data_setup


import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io"
	"os"
	"path"
	"strings"
)

const (
	gethExecutionDataSetupServiceId services.ServiceID = "geth-execution-data-setup"
	gethExecutionDataSetupImage = "parithoshj/geth:merge-2219d7b"

	// The filepath inside the module where static files live
	staticFilesFilepath = "/static-files"

	genesisJsonFilename = "genesis.json"

	// The name of the shared directory (shared with the setup container) where Geth execution data will go
	executionDataDirname = "execution-data"

	executionDirInitializationSuccessfulExitCode = 0

)
var entrypointArgs = []string{"sleep"}
var cmdArgs        = []string{"999999"}

func SetupGethExecutionDataDir(enclaveCtx *enclaves.EnclaveContext) (string, error) {
	serviceCtx, err := enclaveCtx.AddService(gethExecutionDataSetupServiceId, getGethExecutionDataSetupContainerConfig)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred starting the Geth execution data setup container")
	}

	// Directory shared between this module and the service container
	sharedDir := serviceCtx.GetSharedDirectory()

	genesisJsonOnModuleContainerFilepath := path.Join(staticFilesFilepath, genesisJsonFilename)
	genesisJsonOnSetupContainerSharedPath := sharedDir.GetChildPath(genesisJsonFilename)

	srcFp, err := os.Open(genesisJsonOnModuleContainerFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred opening the genesis JSON file '%v' on the module container", genesisJsonOnModuleContainerFilepath)
	}

	destFilepath := genesisJsonOnSetupContainerSharedPath.GetAbsPathOnThisContainer()
	destFp, err := os.Create(destFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred opening the destination filepath '%v' on the module container", destFilepath)
	}

	if _, err := io.Copy(destFp, srcFp); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred copying the genesis file from the module container to the shared directory of the execution setup container")
	}

	executionDataSharedPath := sharedDir.GetChildPath(executionDataDirname)
	executionDataOnModuleContainerDirpath := executionDataSharedPath.GetAbsPathOnThisContainer()
	if err := os.Mkdir(executionDataOnModuleContainerDirpath, os.ModePerm); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred creating the execution data directory '%v' on the module container", executionDataOnModuleContainerDirpath)
	}

	executionDirInitializingCmd := []string{
		"geth",
		"--datadir=" + executionDataSharedPath.GetAbsPathOnServiceContainer(),
		"init",
		genesisJsonOnSetupContainerSharedPath.GetAbsPathOnServiceContainer(),
	}

	exitCode, logs, err := serviceCtx.ExecCommand(executionDirInitializingCmd)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred running Geth execution dir-initializing command '%v'",
			strings.Join(executionDirInitializingCmd, " "),
		)
	}
	if exitCode != executionDirInitializationSuccessfulExitCode {
		return "", stacktrace.NewError(
			"Geth execution dir-initializing command '%v' returned non-%v exit code '%v', and logs:\n%v",
			strings.Join(executionDirInitializingCmd, " "),
			executionDirInitializationSuccessfulExitCode,
			exitCode,
			logs,
		)
	}

	// TODO remove the service???

	return executionDataOnModuleContainerDirpath, nil
}

func getGethExecutionDataSetupContainerConfig(privateIpAddr string, sharedPath *services.SharedPath) (*services.ContainerConfig, error) {
	result := services.NewContainerConfigBuilder(
		gethExecutionDataSetupImage,
	).WithEntrypointOverride(
		entrypointArgs,
	).WithCmdOverride(
		cmdArgs,
	).Build()
	return result, nil
}
