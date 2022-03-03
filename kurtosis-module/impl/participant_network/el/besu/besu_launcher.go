package besu
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
	"strings"
	"time"
)

const (
	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/opt/besu/execution-data"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "genesis.json"

	// NOTE: This can't be 0x00000....000
	// See: https://github.com/ethereum/go-ethereum/issues/19547
	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	rpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcpDiscovery"
	udpDiscoveryPortId = "udpDiscovery"

	getNodeInfoMaxRetries         = 20
	getNodeInfoTimeBetweenRetries = 1 * time.Second
)
var usedPorts = map[string]*services.PortSpec{
	rpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	// TODO Remove if there's no UDP discovery port?????
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}
var entrypointArgs = []string{"sh", "-c"}
var besuLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "ERROR",
	module_io.GlobalClientLogLevel_Warn:  "WARN",
	module_io.GlobalClientLogLevel_Info:  "INFO",
	module_io.GlobalClientLogLevel_Debug: "DEBUG",
	module_io.GlobalClientLogLevel_Trace: "TRACE",
}

type BesuELClientLauncher struct {
	genesisJsonFilepathOnModuleContainer string
	networkId string
}

func NewBesuELClientLauncher(genesisJsonFilepathOnModuleContainer string, networkId string) *BesuELClientLauncher {
	return &BesuELClientLauncher{genesisJsonFilepathOnModuleContainer: genesisJsonFilepathOnModuleContainer, networkId: networkId}
}

func (launcher *BesuELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, besuLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	containerConfigSupplier := launcher.getContainerConfigSupplier(
		image,
		launcher.networkId,
		bootnodeContext,
		logLevel,
		extraParams,
	)

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Besu EL client with service ID '%v'", serviceId)
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
		// TODO Figure out how to get the ENR so CL clients can connect to it!!
		"", // Besu node info endpoint doesn't return an ENR
		nodeInfo.Enode,
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
		wsPortNum,
		miningWaiter,
	)

	return result, nil
}


// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *BesuELClientLauncher) getContainerConfigSupplier(
	image string,
	networkId string,
	bootnodeContext *el.ELClientContext, // NOTE: If this is empty, the node will be configured as a bootnode
	logLevel string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		genesisJsonSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisJsonFilepathOnModuleContainer, genesisJsonSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying genesis JSON file '%v' into shared directory path '%v'", launcher.genesisJsonFilepathOnModuleContainer, sharedGenesisJsonRelFilepath)
		}

		launchNodeCmdArgs := []string{
			"besu",
			"--logging=" + logLevel,
			"--data-path=" + executionDataDirpathOnClientContainer,
			"--genesis-file=" + genesisJsonSharedPath.GetAbsPathOnServiceContainer(),
			"--network-id=" + networkId,
			"--host-allowlist=*",
			"--Xmerge-support=true",
			"--miner-enabled=true",
			"--miner-coinbase=" + miningRewardsAccount,
			"--rpc-http-enabled=true",
			"--rpc-http-host=0.0.0.0",
			fmt.Sprintf("--rpc-http-port=%v", rpcPortNum),
			"--rpc-http-api=ADMIN,CLIQUE,MINER,ETH,NET,DEBUG,TXPOOL",
			"--rpc-http-cors-origins=*",
			"--rpc-ws-enabled=true",
			"--rpc-ws-host=0.0.0.0",
			fmt.Sprintf("--rpc-ws-port=%v", wsPortNum),
			"--rpc-ws-api=ADMIN,CLIQUE,MINER,ETH,NET,DEBUG,TXPOOL",
			"--p2p-enabled=true",
			"--p2p-host=" + privateIpAddr,
			fmt.Sprintf("--p2p-port=%v", discoveryPortNum),
		}
		if bootnodeContext != nil {
			launchNodeCmdArgs = append(
				launchNodeCmdArgs,
				"--bootnodes=" + bootnodeContext.GetEnode(),
			)
		}
		if len(extraParams) > 0 {
			launchNodeCmdArgs = append(launchNodeCmdArgs, extraParams...)
		}
		launchNodeCmdStr := strings.Join(launchNodeCmdArgs, " ")

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			usedPorts,
		).WithEntrypointOverride(
			entrypointArgs,
		).WithCmdOverride([]string{
			launchNodeCmdStr,
		}).Build()

		return containerConfig, nil
	}
	return result
}