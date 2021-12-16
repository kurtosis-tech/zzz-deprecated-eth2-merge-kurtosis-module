package nethermind

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"text/template"
)

const (
	imageName = "nethermindeth/nethermind:kintsugi_v3_0.1"
	// To start a bootnode, we provide this string to the launchNode function
	bootnodeEnodeStrForStartingBootnode = ""

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "nethermind_genesis.json"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	rpcPortId = "rpc"
	wsPortId  = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId: services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId: services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}

type nethermindTemplateData struct {
	NetworkID string
}

type NethermindELClientLauncher struct {
	genesisJsonFilepathOnModuleContainer string
	genesisJsonTemplate *template.Template
}

func NewNethermindELClientLauncher(genesisJsonFilepathOnModuleContainer string, genesisJsonTemplate *template.Template) *NethermindELClientLauncher {
	return &NethermindELClientLauncher{
		genesisJsonFilepathOnModuleContainer: genesisJsonFilepathOnModuleContainer,
		genesisJsonTemplate: genesisJsonTemplate,
	}
}

func (launcher *NethermindELClientLauncher) LaunchBootNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, networkId, bootnodeEnodeStrForStartingBootnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting boot node with service ID '%v'", serviceId)
	}
	return clientCtx, nil
}

func (launcher *NethermindELClientLauncher) LaunchChildNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	bootnodeEnode string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, networkId, bootnodeEnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting child node with service ID '%v' connected to boot node with enode '%v'", serviceId, bootnodeEnode)
	}
	return clientCtx, nil
}

// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *NethermindELClientLauncher) launchNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be launched as a bootnode
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(networkId, bootnodeEnode)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	/*nodeInfo, err := getNodeInfoWithRetry(serviceCtx.GetPrivateIPAddress())
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the newly-started node's info")
	}

	result := el_client_network.NewExecutionLayerClientContext(
		serviceCtx,
		nodeInfo.ENR,
		nodeInfo.Enode,
	)*/

	result := el_client_network.NewExecutionLayerClientContext(serviceCtx, "", "")

	return result, nil
}

func (launcher *NethermindELClientLauncher) getContainerConfigSupplier(
	networkId string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be configured as a bootnode
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		genesisJsonOnModuleContainerSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)

		nethermindTmplData := nethermindTemplateData{
			NetworkID: networkId,
		}

		fp, err := os.Create(genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer())
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening file '%v' for writing", genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer())
		}
		defer fp.Close()

		if err = launcher.genesisJsonTemplate.Execute(fp, nethermindTmplData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the template")
		}

		commandArgs := []string{
			"--config",
			"kintsugi",
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--Init.ChainSpecPath=" + genesisJsonOnModuleContainerSharedPath.GetAbsPathOnServiceContainer(),
			"--Init.WebSocketsEnabled=true",
			"--JsonRpc.Enabled=true",
			"--JsonRpc.EnabledModules=net,eth,consensus,engine",
			fmt.Sprintf("--JsonRpc.Port=%v", rpcPortNum),
			fmt.Sprintf("--JsonRpc.WebSocketsPort=%v", wsPortNum),
			"--JsonRpc.Host=0.0.0.0",
			fmt.Sprintf("--Network.DiscoveryPort=%v", discoveryPortNum),
			fmt.Sprintf("--Network.P2PPort=%v", discoveryPortNum),
			"--Merge.Enabled=true",
			"--Merge.TerminalTotalDifficulty=60000000", //TODO it has to be dynamic, I got this value from genesis generator genesis_config.yaml file
			"--Init.DiagnosticMode=None",
		}
		if bootnodeEnode != bootnodeEnodeStrForStartingBootnode {
			commandArgs = append(
				commandArgs,
				"--Discovery.Bootnodes=" + bootnodeEnode,
			)
		}

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			commandArgs,
		).Build()

		return containerConfig, nil
	}
	return result
}