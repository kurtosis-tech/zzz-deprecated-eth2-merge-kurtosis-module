package impl

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io"
	"os"
	"path"
)

const (
	gethExecutionDataSetupServiceId services.ServiceID = "geth-execution-data-setup"
	gethExecutionDataSetupImage = "parithoshj/geth:merge-2219d7b'"

	// The filepath inside the module where static files live
	staticFilesFilepath = "/static-files"

	genesisJsonFilename = "genesis.json"

	// The name of the shared directory (shared with the setup container) where Geth execution data will go
	executionDataDirname = "execution-data"
)

type ExampleExecutableKurtosisModule struct {
}

func NewExampleExecutableKurtosisModule() *ExampleExecutableKurtosisModule {
	return &ExampleExecutableKurtosisModule{}
}

func (e ExampleExecutableKurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	gethExecutionDataSetupContainerConfigSupplier := getGethExecutionDataSetupContainerConfigSupplier()

	_, err := enclaveCtx.AddService(gethExecutionDataSetupServiceId, gethExecutionDataSetupContainerConfigSupplier)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred adding the Geth execution data setup container")
	}

	return "{}", nil
}

func getGethExecutionDataSetupContainerConfigSupplier() func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	genesisJsonOnModuleContainerFilepath := path.Join(staticFilesFilepath, genesisJsonFilename)

	result := func(privateIpAddr string, sharedPath *services.SharedPath) (*services.ContainerConfig, error) {
		genesisJsonOnSetupContainerSharedPath := sharedPath.GetChildPath(genesisJsonFilename)

		srcFp, err := os.Open(genesisJsonOnModuleContainerFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the genesis JSON file '%v' on the module container", genesisJsonOnModuleContainerFilepath)
		}

		destFilepath := genesisJsonOnSetupContainerSharedPath.GetAbsPathOnThisContainer()
		destFp, err := os.Create(destFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the destination filepath '%v' on the module container", destFilepath)
		}

		if _, err := io.Copy(srcFp, destFp); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying the genesis file from the module container to the shared directory of the execution setup container")
		}

		executionDataSharedPath := sharedPath.GetChildPath(executionDataDirname)
		executionDataOnModuleContainerDirpath := executionDataSharedPath.GetAbsPathOnThisContainer()
		if err := os.Mkdir(executionDataOnModuleContainerDirpath, os.ModeDir); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred creating the execution data directory '%v' on the module container", executionDataOnModuleContainerDirpath)
		}

		cmdArgs := []string{
			"--datadir=" + executionDataSharedPath.GetAbsPathOnServiceContainer(),
			"init",
			genesisJsonOnSetupContainerSharedPath.GetAbsPathOnServiceContainer(),
		}

		result := services.NewContainerConfigBuilder(gethExecutionDataSetupImage).WithCmdOverride(cmdArgs).Build()

		return result, nil
	}

	return result
}
