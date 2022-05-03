package nethermind

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/mining_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"time"
)

const (
	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"

	genesisDataMountDirpath = "/genesis"

	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303
	engineRpcPortNum uint16 = 8551

	// Port IDs
	rpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcpDiscovery"
	udpDiscoveryPortId = "udpDiscovery"
	engineRpcPortId    = "engineRpc"
	engineWsPortId     = "engineWs"

	getNodeInfoMaxRetries         = 30
	getNodeInfoTimeBetweenRetries = 1 * time.Second
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	engineRpcPortId:    services.NewPortSpec(engineRpcPortNum, services.PortProtocol_TCP),
}
var nethermindLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "ERROR",
	module_io.GlobalClientLogLevel_Warn:  "WARN",
	module_io.GlobalClientLogLevel_Info:  "INFO",
	module_io.GlobalClientLogLevel_Debug: "DEBUG",
	module_io.GlobalClientLogLevel_Trace: "TRACE",
}

type NethermindELClientLauncher struct {
	genesisData *el_genesis.ELGenesisData
	totalTerminalDifficulty            uint64
}

func NewNethermindELClientLauncher(genesisData *el_genesis.ELGenesisData, totalTerminalDifficulty uint64) *NethermindELClientLauncher {
	return &NethermindELClientLauncher{genesisData: genesisData, totalTerminalDifficulty: totalTerminalDifficulty}
}

func (launcher *NethermindELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	existingElClients []*el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, nethermindLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	containerConfigSupplier := launcher.getContainerConfigSupplier(image, existingElClients, logLevel, extraParams)

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	restClient := el_rest_client.NewELClientRESTClient(
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
	)

	nodeInfo, err := el.WaitForELClientAvailability(restClient, getNodeInfoMaxRetries, getNodeInfoTimeBetweenRetries)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the EL client to become available")
	}

	miningWaiter := mining_waiter.NewMiningWaiter(restClient)
	result := el.NewELClientContext(
		"nethermind",
		// TODO TODO TODO TODO Get Nethermind ENR, so that CL clients can connect to it!!!
		"", //Nethermind node info endpoint doesn't return ENR field https://docs.nethermind.io/nethermind/ethereum-client/json-rpc/admin
		nodeInfo.Enode,
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
		wsPortNum,
		engineRpcPortNum,
		miningWaiter,
	)

	return result, nil
}

// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *NethermindELClientLauncher) getContainerConfigSupplier(
	image string,
	existingElClients []*el.ELClientContext,
	logLevel string,
	extraParams []string,
) func(string) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string) (*services.ContainerConfig, error) {
		if len(existingElClients) == 0 {
			return nil, stacktrace.NewError("Nethermind nodes cannot be boot nodes")
		}
		if len(existingElClients) < 2 {
			return nil, stacktrace.NewError("Due to a bug in Nethermind peering, Nethermind requires two boot nodes (see https://discord.com/channels/783719264308953108/933134266580234290/958049716065665094https://discord.com/channels/783719264308953108/933134266580234290/958049716065665094 )")
		}
		bootnode1ElContext := existingElClients[0]
		bootnode2ElContext := existingElClients[1]
		
		genesisJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetNethermindGenesisJsonRelativeFilepath())
		jwtSecretJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetJWTSecretRelativeFilepath())

		commandArgs := []string{
			"--config=kiln",
			"--log=" + logLevel,
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--Init.ChainSpecPath=" + genesisJsonFilepathOnClient,
			"--Init.WebSocketsEnabled=true",
			"--Init.DiagnosticMode=None",
			"--JsonRpc.Enabled=true",
			"--JsonRpc.EnabledModules=net,eth,consensus,subscribe,web3,admin",
			"--JsonRpc.Host=0.0.0.0",
			// TODO Set Eth isMining?
			fmt.Sprintf("--JsonRpc.Port=%v", rpcPortNum),
			fmt.Sprintf("--JsonRpc.WebSocketsPort=%v", wsPortNum),
			fmt.Sprintf("--Network.ExternalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.LocalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.DiscoveryPort=%v", discoveryPortNum),
			fmt.Sprintf("--Network.P2PPort=%v", discoveryPortNum),
			"--Merge.Enabled=true",
			fmt.Sprintf("--Merge.TerminalTotalDifficulty=%v", launcher.totalTerminalDifficulty),
			"--Merge.FeeRecipient=" + miningRewardsAccount,
			fmt.Sprintf("--JsonRpc.JwtSecretFile=%v", jwtSecretJsonFilepathOnClient),
			fmt.Sprintf("--JsonRpc.AdditionalRpcUrls=[\"http://0.0.0.0:%v|http;ws|net;eth;subscribe;engine;web3;client\"]", engineRpcPortNum),
			fmt.Sprintf(
				 "--Discovery.Bootnodes=%v,%v",
				 bootnode1ElContext.GetEnode(),
				 bootnode2ElContext.GetEnode(),
			),
		}
		if len(extraParams) > 0 {
			commandArgs = append(commandArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			commandArgs,
		).WithFiles(map[services.FilesArtifactID]string{
			launcher.genesisData.GetFilesArtifactID(): genesisDataMountDirpath,
		}).Build()

		return containerConfig, nil
	}
	return result
}
