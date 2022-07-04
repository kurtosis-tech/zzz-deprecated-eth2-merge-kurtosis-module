package geth

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/mining_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
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
	engineWsPortId     = "engineWs"

	// NOTE: This can't be 0x00000....000
	// See: https://github.com/ethereum/go-ethereum/issues/19547
	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	// TODO Scale this dynamically based on CPUs available and Geth nodes mining
	numMiningThreads = 1

	genesisDataMountDirpath = "/genesis"

	prefundedKeysMountDirpath = "/prefunded-keys"

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"
	keystoreDirpathOnClientContainer      = executionDataDirpathOnClientContainer + "/keystore"

	expectedSecondsForGethInit                              = 10
	expectedSecondsPerKeyImport                             = 8
	expectedSecondsAfterNodeStartUntilHttpServerIsAvailable = 20
	getNodeInfoTimeBetweenRetries                           = 1 * time.Second

	gethAccountPassword      = "password"          // Password that the Geth accounts will be locked with
	gethAccountPasswordsFile = "/tmp/password.txt" // Importing an account to
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

type GethELClientLauncher struct {
	genesisData                   *el_genesis.ELGenesisData
	prefundedGethKeysArtifactUuid services.FilesArtifactUUID
	prefundedAccountInfo          []*genesis_consts.PrefundedAccount
	networkId                     string
}

func NewGethELClientLauncher(genesisData *el_genesis.ELGenesisData, prefundedGethKeysArtifactUuid services.FilesArtifactUUID, prefundedAccountInfo []*genesis_consts.PrefundedAccount, networkId string) *GethELClientLauncher {
	return &GethELClientLauncher{genesisData: genesisData, prefundedGethKeysArtifactUuid: prefundedGethKeysArtifactUuid, prefundedAccountInfo: prefundedAccountInfo, networkId: networkId}
}

func (launcher *GethELClientLauncher) Launch(
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

	containerConfigSupplier := launcher.getContainerConfigSupplier(
		image,
		existingElClients,
		logLevel,
		extraParams,
	)

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	restClient := el_rest_client.NewELClientRESTClient(
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
	)

	maxNumRetries := expectedSecondsForGethInit + len(launcher.prefundedAccountInfo)*expectedSecondsPerKeyImport + expectedSecondsAfterNodeStartUntilHttpServerIsAvailable
	nodeInfo, err := el.WaitForELClientAvailability(restClient, maxNumRetries, getNodeInfoTimeBetweenRetries)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the EL client to become available")
	}

	miningWaiter := mining_waiter.NewMiningWaiter(restClient)
	result := el.NewELClientContext(
		"geth",
		nodeInfo.ENR,
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
func (launcher *GethELClientLauncher) getContainerConfigSupplier(
	image string,
	// NOTE: If this is nil, the node will be configured as a bootnode
	existingElClients []*el.ELClientContext,
	verbosityLevel string,
	extraParams []string,
) func(string) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string) (*services.ContainerConfig, error) {
		genesisJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetGethGenesisJsonRelativeFilepath())
		jwtSecretJsonFilepathOnClient := path.Join(genesisDataMountDirpath, launcher.genesisData.GetJWTSecretRelativeFilepath())

		accountAddressesToUnlock := []string{}
		for _, prefundedAccount := range launcher.prefundedAccountInfo {
			accountAddressesToUnlock = append(accountAddressesToUnlock, prefundedAccount.Address)
		}
		accountsToUnlockStr := strings.Join(accountAddressesToUnlock, ",")

		initDatadirCmdStr := fmt.Sprintf(
			"geth init --datadir=%v %v",
			executionDataDirpathOnClientContainer,
			genesisJsonFilepathOnClient,
		)

		// We need to put the keys into the right spot
		copyKeysIntoKeystoreCmdStr := fmt.Sprintf(
			"cp -r %v/* %v/",
			// TODO We have to do this because Kurtosis' current method for storing directories in files artifacts
			//  We can remove this when Kurtosis can "flatten" directories when storing files artifacts
			path.Join(prefundedKeysMountDirpath, path.Base(static_files.GethPrefundedKeysDirpath)),
			keystoreDirpathOnClientContainer,
		)

		createPasswordsFileCmdStr := fmt.Sprintf(
			"{ for i in $(seq 1 %v); do echo \"%v\" >> %v; done; }",
			len(launcher.prefundedAccountInfo),
			gethAccountPassword,
			gethAccountPasswordsFile,
		)

		launchNodeCmdArgs := []string{
			"geth",
			"--verbosity=" + verbosityLevel,
			"--unlock=" + accountsToUnlockStr,
			"--password=" + gethAccountPasswordsFile,
			"--mine",
			"--miner.etherbase=" + miningRewardsAccount,
			fmt.Sprintf("--miner.threads=%v", numMiningThreads),
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--networkid=" + launcher.networkId,
			"--http",
			"--http.addr=0.0.0.0",
			"--http.vhosts=*",
			"--http.corsdomain=*",
			// WARNING: The admin info endpoint is enabled so that we can easily get ENR/enode, which means
			//  that users should NOT store private information in these Kurtosis nodes!
			"--http.api=admin,engine,net,eth",
			"--ws",
			"--ws.addr=0.0.0.0",
			fmt.Sprintf("--ws.port=%v", wsPortNum),
			"--ws.api=engine,net,eth",
			"--ws.origins=*",
			"--allow-insecure-unlock",
			"--nat=extip:" + privateIpAddr,
			"--verbosity=" + verbosityLevel,
			fmt.Sprintf("--authrpc.port=%v", engineRpcPortNum),
			"--authrpc.addr=0.0.0.0",
			"--authrpc.vhosts=*",
			fmt.Sprintf("--authrpc.jwtsecret=%v", jwtSecretJsonFilepathOnClient),
		}
		var bootnodeEnode string
		if len(existingElClients) > 0 {
			bootnodeContext := existingElClients[0]
			bootnodeEnode = bootnodeContext.GetEnode()
		}
		launchNodeCmdArgs = append(
			launchNodeCmdArgs,
			fmt.Sprintf(`--bootnodes="%s"`, bootnodeEnode),
		)
		if len(extraParams) > 0 {
			launchNodeCmdArgs = append(launchNodeCmdArgs, extraParams...)
		}
		launchNodeCmdStr := strings.Join(launchNodeCmdArgs, " ")

		subcommandStrs := []string{
			initDatadirCmdStr,
			copyKeysIntoKeystoreCmdStr,
			createPasswordsFileCmdStr,
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
			launcher.prefundedGethKeysArtifactUuid:      prefundedKeysMountDirpath,
		}).Build()

		return containerConfig, nil
	}
	return result
}
