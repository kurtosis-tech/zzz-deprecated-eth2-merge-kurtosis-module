package besu

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/mining_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"strings"
	"time"
)

const (
	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/opt/besu/execution-data"

	genesisDataDirpathOnClientContainer = "/opt/besu/genesis"

	// NOTE: This can't be 0x00000....000
	// See: https://github.com/ethereum/go-ethereum/issues/19547
	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	rpcPortNum           uint16 = 8545
	wsPortNum            uint16 = 8546
	discoveryPortNum     uint16 = 30303
	engineHttpRpcPortNum uint16 = 8550
	engineWsRpcPortNum   uint16 = 8551

	// Port IDs
	rpcPortId           = "rpc"
	wsPortId            = "ws"
	tcpDiscoveryPortId  = "tcp-discovery"
	udpDiscoveryPortId  = "udp-discovery"
	engineHttpRpcPortId = "engineHttpRpc"
	engineWsRpcPortId   = "engineWsRpc"

	getNodeInfoMaxRetries         = 20
	getNodeInfoTimeBetweenRetries = 1 * time.Second

	privateIPAddressPlaceholder = "KURTOSIS_IP_ADDRESS_PLACEHOLDER"
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	// TODO Remove if there's no UDP discovery port?????
	udpDiscoveryPortId:  services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	engineHttpRpcPortId: services.NewPortSpec(engineHttpRpcPortNum, services.PortProtocol_TCP),
	engineWsRpcPortId:   services.NewPortSpec(engineWsRpcPortNum, services.PortProtocol_TCP),
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
	genesisData *el_genesis.ELGenesisData
	networkId   string
}

func NewBesuELClientLauncher(genesisData *el_genesis.ELGenesisData, networkId string) *BesuELClientLauncher {
	return &BesuELClientLauncher{genesisData: genesisData, networkId: networkId}
}

func (launcher *BesuELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	existingElClients []*el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, besuLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	containerConfig, err := launcher.getContainerConfig(
		image,
		launcher.networkId,
		existingElClients,
		logLevel,
		extraParams,
	)

	if err != nil {
		stacktrace.Propagate(err, "There was an error while generating the container config")
	}

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfig)
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
		"besu",
		// TODO Figure out how to get the ENR so CL clients can connect to it!!
		"", // Besu node info endpoint doesn't return an ENR
		nodeInfo.Enode,
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
		wsPortNum,
		engineHttpRpcPortNum,
		miningWaiter,
	)

	return result, nil
}

// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *BesuELClientLauncher) getContainerConfig(
	image string,
	networkId string,
	existingElClients []*el.ELClientContext,
	logLevel string,
	extraParams []string,
) (*services.ContainerConfig, error){
	if len(existingElClients) == 0 {
		return nil, stacktrace.NewError("Besu nodes cannot be boot nodes")
	}
	if len(existingElClients) < 2 {
		return nil, stacktrace.NewError("Due to a bug in Besu peering, Besu requires two boot nodes")
	}
	bootnode1ElContext := existingElClients[0]
	bootnode2ElContext := existingElClients[1]

	genesisJsonFilepathOnClient := path.Join(genesisDataDirpathOnClientContainer, launcher.genesisData.GetBesuGenesisJsonRelativeFilepath())
	jwtSecretJsonFilepathOnClient := path.Join(genesisDataDirpathOnClientContainer, launcher.genesisData.GetJWTSecretRelativeFilepath())

	launchNodeCmdArgs := []string{
		"besu",
		"--logging=" + logLevel,
		"--data-path=" + executionDataDirpathOnClientContainer,
		"--genesis-file=" + genesisJsonFilepathOnClient,
		"--network-id=" + networkId,
		"--host-allowlist=*",
		"--miner-enabled=true",
		"--miner-coinbase=" + miningRewardsAccount,
		"--rpc-http-enabled=true",
		"--rpc-http-host=0.0.0.0",
		fmt.Sprintf("--rpc-http-port=%v", rpcPortNum),
		"--rpc-http-api=ADMIN,CLIQUE,MINER,ETH,NET,DEBUG,TXPOOL,ENGINE",
		"--rpc-http-cors-origins=*",
		"--rpc-ws-enabled=true",
		"--rpc-ws-host=0.0.0.0",
		fmt.Sprintf("--rpc-ws-port=%v", wsPortNum),
		"--rpc-ws-api=ADMIN,CLIQUE,MINER,ETH,NET,DEBUG,TXPOOL,ENGINE",
		"--p2p-enabled=true",
		"--p2p-host=" + privateIPAddressPlaceholder,
		fmt.Sprintf("--p2p-port=%v", discoveryPortNum),
		"--engine-rpc-enabled=true",
		fmt.Sprintf("--engine-jwt-secret=%v", jwtSecretJsonFilepathOnClient),
		"--engine-host-allowlist=*",
		fmt.Sprintf("--engine-rpc-port=%v", engineHttpRpcPortNum),
	}
	if len(existingElClients) > 0 {
		launchNodeCmdArgs = append(
			launchNodeCmdArgs,
			fmt.Sprintf(
				"--bootnodes=%v,%v",
				bootnode1ElContext.GetEnode(),
				bootnode2ElContext.GetEnode(),
			),
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
	}).WithFiles(map[services.FilesArtifactUUID]string{
		launcher.genesisData.GetFilesArtifactUUID(): genesisDataDirpathOnClientContainer,
	}).WithPrivateIPAddrPlaceholder(
		privateIPAddressPlaceholder,
	).Build()

	return containerConfig, nil
}
