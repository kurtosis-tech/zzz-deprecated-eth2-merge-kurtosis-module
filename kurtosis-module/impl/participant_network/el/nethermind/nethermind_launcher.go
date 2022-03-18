package nethermind

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/mining_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"time"
)

const (
	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedNethermindGenesisJsonRelFilepath = "nethermind_genesis.json"
	sharedJWTSecretRelFilepath             = "jwtsecret"

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
	genesisJsonFilepathOnModule        string
	jwtSecretFilepathOnModuleContainer string
	totalTerminalDifficulty            uint64
}

func NewNethermindELClientLauncher(genesisJsonFilepathOnModule string, jwtSecretFilepathOnModuleContainer string, totalTerminalDifficulty uint64) *NethermindELClientLauncher {
	return &NethermindELClientLauncher{genesisJsonFilepathOnModule: genesisJsonFilepathOnModule, jwtSecretFilepathOnModuleContainer: jwtSecretFilepathOnModuleContainer, totalTerminalDifficulty: totalTerminalDifficulty}
}

func (launcher *NethermindELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, nethermindLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	containerConfigSupplier := launcher.getContainerConfigSupplier(image, bootnodeContext, logLevel, extraParams)

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
	bootnodeCtx *el.ELClientContext,
	logLevel string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		nethermindGenesisJsonSharedPath := sharedDir.GetChildPath(sharedNethermindGenesisJsonRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisJsonFilepathOnModule, nethermindGenesisJsonSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the Nethermind genesis JSON file from '%v' into the Nethermind node being started",
				launcher.genesisJsonFilepathOnModule,
			)
		}

		jwtSecretSharedPath := sharedDir.GetChildPath(sharedJWTSecretRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.jwtSecretFilepathOnModuleContainer, jwtSecretSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying JWT secret file '%v' into shared directory path '%v'", launcher.jwtSecretFilepathOnModuleContainer, sharedJWTSecretRelFilepath)
		}

		commandArgs := []string{
			"--config=kiln",
			"--log=" + logLevel,
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--Init.ChainSpecPath=" + nethermindGenesisJsonSharedPath.GetAbsPathOnServiceContainer(),
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
			fmt.Sprintf("--JsonRpc.JwtSecretFile==%v", jwtSecretSharedPath.GetAbsPathOnServiceContainer()),
			fmt.Sprintf("--JsonRpc.AdditionalRpcUrls=[\"http://0.0.0.0:%v|http;ws|net;eth;subscribe;engine;web3;client\"]", engineRpcPortNum),
		}
		if bootnodeCtx != nil {
			commandArgs = append(
				commandArgs,
				"--Discovery.Bootnodes="+bootnodeCtx.GetEnode(),
			)
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
		).Build()

		return containerConfig, nil
	}
	return result
}
