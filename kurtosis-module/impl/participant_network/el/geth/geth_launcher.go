package geth

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/mining_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"path"
	"strings"
	"time"
)

const (
	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303
	engineRpcPortNum uint16 = 8550
	engineWsPortNum  uint16 = 8551

	// Port IDs
	rpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcpDiscovery"
	udpDiscoveryPortId = "udpDiscovery"
	engineRpcPortId    = "engineRpc"
	engineWsPortId    = "engineWs"

	// NOTE: This can't be 0x00000....000
	// See: https://github.com/ethereum/go-ethereum/issues/19547
	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	// TODO Scale this dynamically based on CPUs available and Geth nodes mining
	numMiningThreads = 1

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "genesis.json"
	sharedJWTSecretRelFilepath   = "jwtsecret"

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"
	keystoreDirpathOnClientContainer      = executionDataDirpathOnClientContainer + "/keystore"

	gethKeysRelDirpathInSharedDir = "geth-keys"

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
	engineRpcPortId:    services.NewPortSpec(engineRpcPortNum, services.PortProtocol_UDP),
	engineWsPortId:    services.NewPortSpec(engineWsPortNum, services.PortProtocol_UDP),
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
	genesisJsonFilepathOnModuleContainer string
	jwtSecretFilepathOnModuleContainer string
	prefundedAccountInfo                 []*genesis_consts.PrefundedAccount
	networkId                            string
}

func NewGethELClientLauncher(genesisJsonFilepathOnModuleContainer string, jwtSecretFilepathOnModuleContainer string, prefundedAccountInfo []*genesis_consts.PrefundedAccount, networkId string) *GethELClientLauncher {
	return &GethELClientLauncher{genesisJsonFilepathOnModuleContainer: genesisJsonFilepathOnModuleContainer, jwtSecretFilepathOnModuleContainer: jwtSecretFilepathOnModuleContainer, prefundedAccountInfo: prefundedAccountInfo, networkId: networkId}
}

func (launcher *GethELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *el.ELClientContext,
	extraParams []string,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, verbosityLevels)
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
		nodeInfo.ENR,
		nodeInfo.Enode,
		serviceCtx.GetPrivateIPAddress(),
		rpcPortNum,
		wsPortNum,
		engineRpcPortNum,
		engineWsPortNum,
		miningWaiter,
	)

	return result, nil
}

// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *GethELClientLauncher) getContainerConfigSupplier(
	image string,
	networkId string,
	bootnodeContext *el.ELClientContext, // NOTE: If this is empty, the node will be configured as a bootnode
	verbosityLevel string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		genesisJsonSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisJsonFilepathOnModuleContainer, genesisJsonSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying genesis JSON file '%v' into shared directory path '%v'", launcher.genesisJsonFilepathOnModuleContainer, sharedGenesisJsonRelFilepath)
		}

		jwtSecretSharedPath := sharedDir.GetChildPath(sharedJWTSecretRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.jwtSecretFilepathOnModuleContainer, jwtSecretSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying JWT secret file '%v' into shared directory path '%v'", launcher.jwtSecretFilepathOnModuleContainer, sharedJWTSecretRelFilepath)
		}

		gethKeysDirSharedPath := sharedDir.GetChildPath(gethKeysRelDirpathInSharedDir)
		if err := os.Mkdir(gethKeysDirSharedPath.GetAbsPathOnThisContainer(), os.ModePerm); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred creating the Geth keys directory in the shared dir")
		}

		accountAddressesToUnlock := []string{}
		for _, prefundedAccount := range launcher.prefundedAccountInfo {
			keyFilepathOnModuleContainer := prefundedAccount.GethKeyFilepath
			keyFilename := path.Base(keyFilepathOnModuleContainer)
			keyRelFilepathInSharedDir := path.Join(gethKeysRelDirpathInSharedDir, keyFilename)
			keyFileSharedPath := sharedDir.GetChildPath(keyRelFilepathInSharedDir)
			if err := service_launch_utils.CopyFileToSharedPath(keyFilepathOnModuleContainer, keyFileSharedPath); err != nil {
				return nil, stacktrace.Propagate(err, "An error occurred copying key file '%v' to the shared directory", keyFilepathOnModuleContainer)
			}

			accountAddressesToUnlock = append(accountAddressesToUnlock, prefundedAccount.Address)
		}

		initDatadirCmdStr := fmt.Sprintf(
			"geth init --datadir=%v %v",
			executionDataDirpathOnClientContainer,
			genesisJsonSharedPath.GetAbsPathOnServiceContainer(),
		)

		copyKeysIntoKeystoreCmdStr := fmt.Sprintf(
			"cp -r %v/* %v/",
			gethKeysDirSharedPath.GetAbsPathOnServiceContainer(),
			keystoreDirpathOnClientContainer,
		)

		createPasswordsFileCmdStr := fmt.Sprintf(
			"{ for i in $(seq 1 %v); do echo \"%v\" >> %v; done; }",
			len(launcher.prefundedAccountInfo),
			gethAccountPassword,
			gethAccountPasswordsFile,
		)

		accountsToUnlockStr := strings.Join(accountAddressesToUnlock, ",")
		launchNodeCmdArgs := []string{
			"geth",
			"--verbosity=" + verbosityLevel,
			"--unlock=" + accountsToUnlockStr,
			"--password=" + gethAccountPasswordsFile,
			"--mine",
			"--miner.etherbase=" + miningRewardsAccount,
			fmt.Sprintf("--miner.threads=%v", numMiningThreads),
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--networkid=" + networkId,
			"--http",
			"--http.addr=0.0.0.0",
			// WARNING: The admin info endpoint is enabled so that we can easily get ENR/enode, which means
			//  that users should NOT store private information in these Kurtosis nodes!
			"--http.api=admin,engine,net,eth",
			"--ws",
			"--ws.addr=0.0.0.0",
			fmt.Sprintf("--ws.port=%v", wsPortNum),
			"--ws.api=engine,net,eth",
			"--allow-insecure-unlock",
			"--nat=extip:" + privateIpAddr,
			"--verbosity=" + verbosityLevel,
			fmt.Sprintf("--authrpc.port=%v", engineRpcPortNum),
			fmt.Sprintf("--authrpc.jwtsecret=%v", jwtSecretSharedPath.GetAbsPathOnServiceContainer()),
		}
		if bootnodeContext != nil {
			launchNodeCmdArgs = append(
				launchNodeCmdArgs,
				"--bootnodes="+bootnodeContext.GetEnode(),
			)
		}
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
		}).Build()

		return containerConfig, nil
	}
	return result
}
