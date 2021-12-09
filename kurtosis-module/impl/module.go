package impl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/geth_el_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/geth_execution_data_setup"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
)

const (
	networkId = "1337602"
	externalIpAddress = "189.216.206.108"
	bootnodeEnode = "enode://6b457d42e6301acfae11dc785b43346e195ad0974b394922b842adea5aeb4c55b02410607ba21e4a03ba53e7656091e2f990034ce3f8bad4d0cca1c6398bdbb8@137.184.55.117:30303"
)

type ExampleExecutableKurtosisModule struct {
}

func NewExampleExecutableKurtosisModule() *ExampleExecutableKurtosisModule {
	return &ExampleExecutableKurtosisModule{}
}

func (e ExampleExecutableKurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	executionDataDirpath, err := geth_execution_data_setup.SetupGethExecutionDataDir(enclaveCtx)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred setting up the Geth execution data directory")
	}
	logrus.Infof("Geth execution data directory initialized at '%v'", executionDataDirpath)

	_, err = geth_el_client.LaunchGethELClient(
		enclaveCtx,
		executionDataDirpath,
		networkId,
		externalIpAddress,
		bootnodeEnode,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Geth EL client")
	}

	return "{}", nil
}

