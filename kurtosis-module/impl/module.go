package impl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/geth_el_client_launcher"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	consensusConfigDataDirpathOnModuleContainer = "/static-files/consensus_config_data"

	// genesisJsonFilepath = "/static-files/merge-devnet3-genesis.json"
	genesisJsonFilepath = "/static-files/genesis.json"
	// networkId = "1337602" // The merge-devnet IP address
	networkId = "3151908"

	// externalIpAddress = "189.216.206.108"
	externalIpAddress = "185.247.70.125"
	bootnodeEnr = "enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk"
	totalTerminalDifficulty = 5000000000
)
var bootnodeEnodes = []string{
	"enode://6b457d42e6301acfae11dc785b43346e195ad0974b394922b842adea5aeb4c55b02410607ba21e4a03ba53e7656091e2f990034ce3f8bad4d0cca1c6398bdbb8@137.184.55.117:30303",
	"enode://588ef56694223ce3212d7c56e5b6f3e8ba46a9c29522fdc6fef15657f505a7314b9bd32f2d53c4564bc6b9259c3d5c79fc96257eff9cd489004c4d9cbb3c0707@137.184.203.157:30303",
	"enode://46b2ecd18c24463413b7328e9a59c72d955874ad5ddb9cd9659d322bedd2758a6cefb8378e2309a028bd3cdf2beca0b18c3457f03e772f35d0cd06c37ce75eee@137.184.213.208:30303",
}

type ExampleExecutableKurtosisModule struct {
}

func NewExampleExecutableKurtosisModule() *ExampleExecutableKurtosisModule {
	return &ExampleExecutableKurtosisModule{}
}

func (e ExampleExecutableKurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	elNetwork := el_client_network.NewExecutionLayerNetwork(
		enclaveCtx,
		networkId,
		genesisJsonFilepath,
		geth_el_client_launcher.NewGethELClientLauncher(),
	)

	if err := elNetwork.AddNode(); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred adding the first EL client node")
	}

	/*
	gethElClientServiceCtx, err := geth_el_client.LaunchGethELClient(
		enclaveCtx,
		genesisJsonFilepath,
		networkId,
		externalIpAddress,
		bootnodeEnodes,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Geth EL client")
	}

	gethElClientRpcPort, found := gethElClientServiceCtx.GetPrivatePorts()[geth_el_client.RpcPortId]
	if !found {
		return "", stacktrace.NewError("Expected the Geth EL client to have a port with ID '%v' but none was found", geth_el_client.RpcPortId)
	}

	_, err = lighthouse_cl_client.LaunchLighthouseCLClient(
		enclaveCtx,
		consensusConfigDataDirpathOnModuleContainer,
		externalIpAddress,
		bootnodeEnr,
		gethElClientServiceCtx.GetPrivateIPAddress(),
		gethElClientRpcPort.GetNumber(),
		totalTerminalDifficulty,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Lighthouse CL client")
	}

	 */

	return "{}", nil
}

