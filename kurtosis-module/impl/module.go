package impl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network/nethermind"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/ethereum_genesis_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"text/template"

	// "path"
	// "text/template"
)

const (
	// networkId = "1337602" // The merge-devnet IP address
	networkId = "3151908"

	staticFilesDirpath                    = "/static-files"


	externalIpAddress = "185.247.70.125"
	bootnodeEnr = "enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk"
	totalTerminalDifficulty = 60000000 //This value is the one that the genesis generator creates in the genesis file

	//Nethermind
	nethermindGenesisJsonTemplateFilename = "nethermind-genesis.json.tmpl"
	nethermindGenesisJsonTemplateFilepath = staticFilesDirpath + "/" + nethermindGenesisJsonTemplateFilename
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
	_, gethGenesisJsonFilepath, _, err := ethereum_genesis_generator.LaunchEthereumGenesisGenerator(enclaveCtx)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis generator Service")
	}

	tmpl, err := template.New(nethermindGenesisJsonTemplateFilename).ParseFiles(nethermindGenesisJsonTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Nethermind genesis JSON template file '%v'", nethermindGenesisJsonTemplateFilepath)
	}

	nethermindClientLauncher := nethermind.NewNethermindELClientLauncher(gethGenesisJsonFilepath, tmpl)
	elNetwork := el_client_network.NewExecutionLayerNetwork(
		enclaveCtx,
		networkId,
		nethermindClientLauncher,
	)
	if err := elNetwork.AddNode(); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred adding the EL client node")
	}

	return "{}", nil
}

