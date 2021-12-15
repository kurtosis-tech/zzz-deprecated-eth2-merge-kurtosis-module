package impl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network/lighthouse"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network/geth"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/ethereum_genesis_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"path"
	"text/template"

	// "path"
	// "text/template"
)

const (
	networkId = "3151908"

	staticFilesDirpath                    = "/static-files"
	gethGenesisGenerationConfigYmlTemplateFilepath = staticFilesDirpath + "/el/genesis-config.yaml.tmpl"
	clGenesisGenerationConfigYmlTemplateFilepath = staticFilesDirpath + "/cl/config.yaml.tmpl"
	nethermindGenesisJsonTemplateFilepath = staticFilesDirpath + "/nethermind-genesis.json.tmpl"

	totalTerminalDifficulty         = 60000000 //This value is the one that the genesis generator creates in the genesis file

	// TODO uncomment these when the module can either start a private network OR connect to an existing devnet
	// mergeDevnet3NetworkId = "1337602"
	// mergeDevnet3ClClientBootnodeEnr = "enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk"
)
/*
var mergeDevnet3BootnodeEnodes = []string{
	"enode://6b457d42e6301acfae11dc785b43346e195ad0974b394922b842adea5aeb4c55b02410607ba21e4a03ba53e7656091e2f990034ce3f8bad4d0cca1c6398bdbb8@137.184.55.117:30303",
	"enode://588ef56694223ce3212d7c56e5b6f3e8ba46a9c29522fdc6fef15657f505a7314b9bd32f2d53c4564bc6b9259c3d5c79fc96257eff9cd489004c4d9cbb3c0707@137.184.203.157:30303",
	"enode://46b2ecd18c24463413b7328e9a59c72d955874ad5ddb9cd9659d322bedd2758a6cefb8378e2309a028bd3cdf2beca0b18c3457f03e772f35d0cd06c37ce75eee@137.184.213.208:30303",
}
 */

type ExampleExecutableKurtosisModule struct {
}

func NewExampleExecutableKurtosisModule() *ExampleExecutableKurtosisModule {
	return &ExampleExecutableKurtosisModule{}
}

func (e ExampleExecutableKurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	logrus.Info("Generating genesis information for EL & CL clients...")
	gethGenesisConfigTemplate, err := parseTemplate(gethGenesisGenerationConfigYmlTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Geth genesis generation config YAML template")
	}
	clGenesisConfigTemplate, err := parseTemplate(clGenesisGenerationConfigYmlTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the CL genesis generation config YAML template")
	}
	gethGenesisJsonFilepath, clClientConfigDataDirpath, err := ethereum_genesis_generator.GenerateELAndCLGenesisConfig(
		enclaveCtx,
		gethGenesisConfigTemplate,
		clGenesisConfigTemplate,
		networkId,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis generator Service")
	}
	logrus.Info("Successfully generated genesis information for EL & CL clients")

	// TODO Nethermind template-filling here
	/*
	tmpl, err := template.New(templateFilename).ParseFiles(templateFilepath)
	template.New(
		// For some reason, the template name has to match the basename of the file:
		//  https://stackoverflow.com/questions/49043292/error-template-is-an-incomplete-or-empty-template
		path.Base(nethermindGenesisJsonTemplateFilepath),
	).Parse(
		gethGenesisJsonFilepath,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Nethermind genesis JSON template file '%v'", nethermindGenesisJsonTemplateFilepath)
	}
	 */

	logrus.Info("Launching a network of EL clients...")
	gethClientLauncher := geth.NewGethELClientLauncher(gethGenesisJsonFilepath)
	elNetwork := el_client_network.NewExecutionLayerNetwork(
		enclaveCtx,
		networkId,
		gethClientLauncher,
	)

	// TODO Make the number of nodes a dynamic argument
	allElClientContexts := []*el_client_network.ExecutionLayerClientContext{}
	for i := 0; i < 1; i++ {
		elClientCtx, err := elNetwork.AddNode()
		if err != nil {
			return "", stacktrace.Propagate(err, "An error occurred adding EL client node %v", i)
		}
		allElClientContexts = append(allElClientContexts, elClientCtx)
	}
	logrus.Info("Successfully launched a network of EL clients")

	logrus.Info("Launching a network of CL clients...")
	lighthouseClientLauncher := lighthouse.NewLighthouseCLClientLauncher(clClientConfigDataDirpath)
	clNetwork := cl_client_network.NewConsensusLayerNetwork(
		enclaveCtx,
		allElClientContexts,
		totalTerminalDifficulty,
		lighthouseClientLauncher,
	)

	// TODO Make this dynamic
	for i := 0; i < 1; i++ {
		if err := clNetwork.AddNode(); err != nil {
			return "", stacktrace.Propagate(err, "An error occurred adding CL client node %v", i)
		}

	}
	logrus.Info("Successfully launched a network of CL clients")


	/*
	gethElClientServiceCtx, err := geth_el_client.LaunchGethELClient(
		enclaveCtx,
		gethGenesisJsonFilepath,
		networkId,
		externalIpAddress,
		bootnodeEnodes,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Geth EL client")
	}

	gethElClientRpcPort, found := gethElClientServiceCtx.GetPrivatePorts()[geth_el_client.rpcPortId]
	if !found {
		return "", stacktrace.NewError("Expected the Geth EL client to have a port with ID '%v' but none was found", geth_el_client.rpcPortId)
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

func parseTemplate(filepath string) (*template.Template, error) {
	tmpl, err := template.New(
		// For some reason, the template name has to match the basename of the file:
		//  https://stackoverflow.com/questions/49043292/error-template-is-an-incomplete-or-empty-template
		path.Base(filepath),
	).Parse(
		filepath,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing template file '%v'", filepath)
	}
	return tmpl, nil
}