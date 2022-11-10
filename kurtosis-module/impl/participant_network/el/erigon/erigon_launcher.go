package erigon

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"strings"
	"time"
)

const (
	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303
	engineRpcPortNum uint16 = 8551

	// Port IDs
	rpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"
	engineRpcPortId    = "engine-rpc"

	genesisDataMountDirpath = "/genesis"

	// The dirpath of the execution data directory on the client container.
	// NOTE: Container user must have permission to write to this directory.
	executionDataDirpathOnClientContainer = "/home/erigon/execution-data"

	expectedSecondsForErigonInit                            = 10
	expectedSecondsAfterNodeStartUntilHttpServerIsAvailable = 20
	getNodeInfoTimeBetweenRetries                           = 1 * time.Second

	privateIPAddressPlaceholder = "KURTOSIS_PRIVATE_IP_ADDR_PLACEHOLDER"
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	engineRpcPortId:    services.NewPortSpec(engineRpcPortNum, services.PortProtocol_TCP),
}
var entrypointArgs = []string{"sh", "-c"}
var verbosityLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "1",
	module_io.GlobalClientLogLevel_Warn:  "2",
	module_io.GlobalClientLogLevel_Info:  "3",
	module_io.GlobalClientLogLevel_Debug: "4",
	module_io.GlobalClientLogLevel_Trace: "5",
}

type ErigonELClientLauncher struct {
	genesisData *el_genesis.ELGenesisData
	networkId   string
}

func NewErigonELClientLauncher(genesisData *el_genesis.ELGenesisData, networkId string) *ErigonELClientLauncher {
	return &ErigonELClientLauncher{genesisData: genesisData, networkId: networkId}
}

func (launcher *ErigonELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	// If empty then the node will be launched as a bootnode
	existingElClients []*el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, verbosityLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	containerConfig := launcher.getContainerConfig(
		image,
		existingElClients,
		logLevel,
		extraParams,
	)

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Erigon EL client with service ID '%v'", serviceId)
	}

	restClient := el_rest_client.NewELClientRESTClient(
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
	)

	maxNumRetries := expectedSecondsForErigonInit + expectedSecondsAfterNodeStartUntilHttpServerIsAvailable
	nodeInfo, err := el.WaitForELClientAvailability(restClient, maxNumRetries, getNodeInfoTimeBetweenRetries)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the EL client to become available")
	}

	result := el.NewELClientContext(
		"erigon",
		nodeInfo.ENR,
		nodeInfo.Enode,
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
		wsPortNum,
		engineRpcPortNum,
	)

	return result, nil
}

// ====================================================================================================
//
//	Private Helper Methods
//
// ====================================================================================================
func (launcher *ErigonELClientLauncher) getContainerConfig(
	image string,
	// NOTE: If this is nil, the node will be configured as a bootnode
	existingElClients []*el.ELClientContext,
	verbosityLevel string,
	extraParams []string,
) *services.ContainerConfig {
	genesisJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetErigonGenesisJsonRelativeFilepath())
	jwtSecretJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetJWTSecretRelativeFilepath())

	initDatadirCmdStr := fmt.Sprintf(
		"erigon init --datadir=%v %v",
		executionDataDirpathOnClientContainer,
		genesisJsonFilepathOnClient,
	)

	launchNodeCmdArgs := []string{
		"erigon",
		"--verbosity=" + verbosityLevel,
		"--datadir=" + executionDataDirpathOnClientContainer,
		"--networkid=" + launcher.networkId,
		"--http",
		"--http.addr=0.0.0.0",
		"--http.corsdomain=*",
		//// WARNING: The admin info endpoint is enabled so that we can easily get ENR/enode, which means
		////  that users should NOT store private information in these Kurtosis nodes!
		"--http.api=admin,engine,net,eth",
		"--ws",
		"--allow-insecure-unlock",
		"--nat=extip:" + privateIPAddressPlaceholder,
		fmt.Sprintf("--engine.port=%v", engineRpcPortNum),
		"--engine.addr=0.0.0.0",
		fmt.Sprintf("--authrpc.jwtsecret=%v", jwtSecretJsonFilepathOnClient),
		"--nodiscover",
	}

	if len(existingElClients) > 0 {
		bootnode1ElContext := existingElClients[0]
		launchNodeCmdArgs = append(launchNodeCmdArgs, fmt.Sprintf(
			"--staticpeers=%v",
			bootnode1ElContext.GetEnode()),
		)
	}

	if len(extraParams) > 0 {
		launchNodeCmdArgs = append(launchNodeCmdArgs, extraParams...)
	}
	launchNodeCmdStr := strings.Join(launchNodeCmdArgs, " ")

	subcommandStrs := []string{
		initDatadirCmdStr,
		launchNodeCmdStr,
	}
	commandStr := strings.Join(subcommandStrs, " && ")

	containerConfig := services.NewContainerConfigBuilder(
		image,
	).WithUsedPorts(
		usedPorts,
	).WithEntrypointOverride(
		entrypointArgs,
	).WithCmdOverride([]string{
		commandStr,
	}).WithFiles(map[services.FilesArtifactUUID]string{
		launcher.genesisData.GetFilesArtifactUUID(): genesisDataMountDirpath,
	}).WithPrivateIPAddrPlaceholder(
		privateIPAddressPlaceholder,
	).Build()

	return containerConfig
}
