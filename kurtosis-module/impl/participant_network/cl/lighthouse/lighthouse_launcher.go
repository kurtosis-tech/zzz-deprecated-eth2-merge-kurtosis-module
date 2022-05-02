package lighthouse

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_genesis"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"os"
	"path"
	"time"
)

const (
	lighthouseBinaryCommand = "lighthouse"

	genesisDataMountpointOnClients = "/genesis"

	// ---------------------------------- Beacon client -------------------------------------
	consensusDataDirpathOnBeaconServiceContainer = "/consensus-data"
	beaconConfigDataDirpathRelToSharedDirRoot    = "config-data"
	jwtSecretFilepathRelToSharedDirRoot          = "jwtsecret"

	// Port IDs
	beaconTcpDiscoveryPortID = "tcpDiscovery"
	beaconUdpDiscoveryPortID = "udpDiscovery"
	beaconHttpPortID         = "http"
	beaconMetricsPortID      = "metrics"

	// Port nums
	beaconDiscoveryPortNum uint16 = 9000
	beaconHttpPortNum      uint16 = 4000
	beaconMetricsPortNum   uint16 = 5054

	maxNumHealthcheckRetries      = 10
	timeBetweenHealthcheckRetries = 1 * time.Second

	// ---------------------------------- Validator client -------------------------------------
	validatorConfigDataDirpathRelToSharedDirRoot = "config-data"

	validatorKeysRelDirpathInSharedDir    = "validator-keys"
	validatorSecretsRelDirpathInSharedDir = "validator-secrets"

	validatingRewardsAccount = "0x0000000000000000000000000000000000000000"

	validatorHttpPortID     = "http"
	validatorMetricsPortID  = "metrics"
	validatorHttpPortNum    = 5042
	validatorMetricsPortNum = 5064

	metricsPath = "/metrics"

	beaconSuffixServiceId    = "beacon"
	validatorSuffixServiceId = "validator"
)

var beaconUsedPorts = map[string]*services.PortSpec{
	beaconTcpDiscoveryPortID: services.NewPortSpec(beaconDiscoveryPortNum, services.PortProtocol_TCP),
	beaconUdpDiscoveryPortID: services.NewPortSpec(beaconDiscoveryPortNum, services.PortProtocol_UDP),
	beaconHttpPortID:         services.NewPortSpec(beaconHttpPortNum, services.PortProtocol_TCP),
	beaconMetricsPortID:      services.NewPortSpec(beaconMetricsPortNum, services.PortProtocol_TCP),
}
var validatorUsedPorts = map[string]*services.PortSpec{
	validatorHttpPortID:    services.NewPortSpec(validatorHttpPortNum, services.PortProtocol_TCP),
	validatorMetricsPortID: services.NewPortSpec(validatorMetricsPortNum, services.PortProtocol_TCP),
}
var lighthouseLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "error",
	module_io.GlobalClientLogLevel_Warn:  "warn",
	module_io.GlobalClientLogLevel_Info:  "info",
	module_io.GlobalClientLogLevel_Debug: "debug",
	module_io.GlobalClientLogLevel_Trace: "trace",
}

type LighthouseCLClientLauncher struct {
	genesisData *cl_genesis.CLGenesisData
}

func NewLighthouseCLClientLauncher(genesisData *cl_genesis.CLGenesisData) *LighthouseCLClientLauncher {
	return &LighthouseCLClientLauncher{genesisData: genesisData}
}

func (launcher *LighthouseCLClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *cl2.NodeTypeKeystoreDirpaths,
	extraBeaconParams []string,
	extraValidatorParams []string,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, beaconSuffixServiceId))
	validatorNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, validatorSuffixServiceId))

	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, lighthouseLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	// Launch Beacon node
	beaconContainerConfigSupplier := launcher.getBeaconContainerConfigSupplier(
		image,
		bootnodeContext,
		elClientContext,
		logLevel,
		extraBeaconParams,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconNodeServiceId, beaconContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lighthouse Beacon node with service ID '%v'", beaconNodeServiceId)
	}

	beaconHttpPort, found := beaconServiceCtx.GetPrivatePorts()[beaconHttpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lighthouse Beacon service to have port with ID '%v', but none was found", beaconHttpPortID)
	}

	// TODO This will return a 503 when genesis isn't yet ready!!! Need to fix this somehow
	beaconRestClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), beaconHttpPort.GetNumber())
	if err := cl.WaitForBeaconClientAvailability(beaconRestClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lighthouse Beacon node to become available")
	}

	// Launch validator node
	beaconHttpUrl := fmt.Sprintf("http://%v:%v", beaconServiceCtx.GetPrivateIPAddress(), beaconHttpPort.GetNumber())
	validatorContainerConfigSupplier := launcher.getValidatorContainerConfigSupplier(
		image,
		logLevel,
		beaconHttpUrl,
		nodeKeystoreDirpaths.RawKeysDirpath,
		nodeKeystoreDirpaths.RawSecretsDirpath,
		extraValidatorParams,
	)
	validatorServiceCtx, err := enclaveCtx.AddService(validatorNodeServiceId, validatorContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lighthouse validator node with service ID '%v'", validatorNodeServiceId)
	}

	// TODO add validator availability using teh validator API: https://ethereum.github.io/beacon-APIs/?urls.primaryName=v1#/ValidatorRequiredApi

	nodeIdentity, err := beaconRestClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lighthouse Beacon node's identity, which is necessary to retrieve its ENR")
	}

	beaconMetricsPort, found := beaconServiceCtx.GetPrivatePorts()[beaconMetricsPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lighthouse Beacon service to have port with ID '%v', but none was found", beaconMetricsPortID)
	}
	beaconMetricsUrl := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), beaconMetricsPort.GetNumber())

	validatorMetricsPort, found := validatorServiceCtx.GetPrivatePorts()[validatorMetricsPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lighthouse Validator service to have port with ID '%v', but none was found", validatorMetricsPortID)
	}
	validatorMetricsUrl := fmt.Sprintf("%v:%v", validatorServiceCtx.GetPrivateIPAddress(), validatorMetricsPort.GetNumber())

	beaconNodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(beaconNodeServiceId), metricsPath, beaconMetricsUrl)
	validatorNodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(validatorNodeServiceId), metricsPath, validatorMetricsUrl)
	nodesMetricsInfo := []*cl.CLNodeMetricsInfo{beaconNodeMetricsInfo, validatorNodeMetricsInfo}

	result := cl.NewCLClientContext(
		"lighthouse",
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		beaconHttpPortNum,
		nodesMetricsInfo,
		beaconRestClient,
	)

	return result, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *LighthouseCLClientLauncher) getBeaconContainerConfigSupplier(
	image string,
	bootClClientCtx *cl.CLClientContext,
	elClientCtx *el.ELClientContext,
	logLevel string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		/*
		configDataDirpathOnServiceSharedPath := sharedDir.GetChildPath(beaconConfigDataDirpathRelToSharedDirRoot)

		destConfigDataDirpathOnModule := configDataDirpathOnServiceSharedPath.GetAbsPathOnThisContainer()
		if err := recursive_copy.Copy(launcher.configDataDirpathOnModuleContainer, destConfigDataDirpathOnModule); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the config data directory on the module, '%v', into the service container, '%v'",
				launcher.configDataDirpathOnModuleContainer,
				destConfigDataDirpathOnModule,
			)
		}

		jwtSecretSharedPath := sharedDir.GetChildPath(jwtSecretFilepathRelToSharedDirRoot)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.jwtSecretFilepathOnModuleContainer, jwtSecretSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying JWT secret file '%v' into shared directory path '%v'", launcher.jwtSecretFilepathOnModuleContainer, jwtSecretFilepathRelToSharedDirRoot)
		}

		 */

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientCtx.GetIPAddress(),
			elClientCtx.GetRPCPortNum(),
		)

		elClientEngineRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientCtx.GetIPAddress(),
			elClientCtx.GetEngineRPCPortNum(),
		)

		// configDataDirpathOnService := configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer()

		// For some reason, Lighthouse takes in the parent directory of the config file (rather than the path to the config file itself)
		genesisConfigParentDirpathOnClient := path.Join(genesisDataMountpointOnClients, path.Dir(launcher.genesisData.GetConfigYMLRelativeFilepath()))
		jwtSecretFilepath := path.Join(genesisDataMountpointOnClients, launcher.genesisData.GetJWTSecretRelativeFilepath())

		// NOTE: If connecting to the merge devnet remotely we DON'T want the following flags; when they're not set, the node's external IP address is auto-detected
		//  from the peers it communicates with but when they're set they basically say "override the autodetection and
		//  use what I specify instead." This requires having a know external IP address and port, which we definitely won't
		//  have with a network running in Kurtosis.
		//    "--disable-enr-auto-update",
		//    "--enr-address=" + externalIpAddress,
		//    fmt.Sprintf("--enr-udp-port=%v", beaconDiscoveryPortNum),
		//    fmt.Sprintf("--enr-tcp-port=%v", beaconDiscoveryPortNum),
		cmdArgs := []string{
			lighthouseBinaryCommand,
			"beacon_node",
			"--debug-level=" + logLevel,
			"--datadir=" + consensusDataDirpathOnBeaconServiceContainer,
			"--testnet-dir=" + genesisConfigParentDirpathOnClient,
			"--eth1",
			// vvvvvvvvvvvvvvvvvvv REMOVE THESE WHEN CONNECTING TO EXTERNAL NET vvvvvvvvvvvvvvvvvvvvv
			"--disable-enr-auto-update",
			"--enr-address=" + privateIpAddr,
			fmt.Sprintf("--enr-udp-port=%v", beaconDiscoveryPortNum),
			fmt.Sprintf("--enr-tcp-port=%v", beaconDiscoveryPortNum),
			// ^^^^^^^^^^^^^^^^^^^ REMOVE THESE WHEN CONNECTING TO EXTERNAL NET ^^^^^^^^^^^^^^^^^^^^^
			"--listen-address=0.0.0.0",
			fmt.Sprintf("--port=%v", beaconDiscoveryPortNum), // NOTE: Remove for connecting to external net!
			"--http",
			"--http-address=0.0.0.0",
			fmt.Sprintf("--http-port=%v", beaconHttpPortNum),
			"--merge",
			"--http-allow-sync-stalled",
			// NOTE: This comes from:
			//   https://github.com/sigp/lighthouse/blob/7c88f582d955537f7ffff9b2c879dcf5bf80ce13/scripts/local_testnet/beacon_node.sh
			// and the option says it's "useful for testing in smaller networks" (unclear what happens in larger networks)
			"--disable-packet-filter",
			"--execution-endpoints=" + elClientEngineRpcUrlStr,
			"--eth1-endpoints=" + elClientRpcUrlStr,
			"--jwt-secrets=" + jwtSecretFilepath,
			"--suggested-fee-recipient=" + validatingRewardsAccount,
			// Set per Paris' recommendation to reduce noise in the logs
			"--subscribe-all-subnets",
			// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
			"--metrics",
			"--metrics-address=" + privateIpAddr,
			"--metrics-allow-origin=*",
			fmt.Sprintf("--metrics-port=%v", beaconMetricsPortNum),
			// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
		}
		if bootClClientCtx != nil {
			cmdArgs = append(cmdArgs, "--boot-nodes="+bootClClientCtx.GetENR())
		}
		if len(extraParams) > 0 {
			cmdArgs = append(cmdArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			beaconUsedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()
		return containerConfig, nil
	}
}

func (launcher *LighthouseCLClientLauncher) getValidatorContainerConfigSupplier(
	image string,
	logLevel string,
	beaconClientHttpUrl string,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		/*
		configDataDirpathOnServiceSharedPath := sharedDir.GetChildPath(validatorConfigDataDirpathRelToSharedDirRoot)

		destConfigDataDirpathOnModule := configDataDirpathOnServiceSharedPath.GetAbsPathOnThisContainer()
		if err := recursive_copy.Copy(launcher.configDataDirpathOnModuleContainer, destConfigDataDirpathOnModule); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the config data directory on the module, '%v', into the service container, '%v'",
				launcher.configDataDirpathOnModuleContainer,
				destConfigDataDirpathOnModule,
			)
		}

		 */

		validatorKeysSharedPath := sharedDir.GetChildPath(validatorKeysRelDirpathInSharedDir)
		if err := recursive_copy.Copy(
			validatorKeysDirpathOnModuleContainer,
			validatorKeysSharedPath.GetAbsPathOnThisContainer(),
			// NOTE: We have to add this because the Lighthouse validator node wants to write a slashing-protection.sqlite
			//  file to the validator keys directory, but it runs as the 'lighthouse' user (rather than 'root') so doesn't
			//  have default write access to the validator keys directory
			recursive_copy.Options{AddPermission: os.ModePerm},
		); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying the validator keys into the shared directory so the node can consume them")
		}

		validatorSecretsSharedPath := sharedDir.GetChildPath(validatorSecretsRelDirpathInSharedDir)
		if err := recursive_copy.Copy(
			validatorSecretsDirpathOnModuleContainer,
			validatorSecretsSharedPath.GetAbsPathOnThisContainer(),
		); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying the validator secrets into the shared directory so the node can consume them")
		}

		// configDataDirpathOnService := configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer()
		// For some reason, Lighthouse takes in the parent directory of the config file (rather than the path to the config file itself)
		genesisConfigParentDirpathOnClient := path.Join(genesisDataMountpointOnClients, path.Dir(launcher.genesisData.GetConfigYMLRelativeFilepath()))
		cmdArgs := []string{
			"lighthouse",
			"validator_client",
			"--debug-level=" + logLevel,
			"--testnet-dir=" + genesisConfigParentDirpathOnClient,
			"--validators-dir=" + validatorKeysSharedPath.GetAbsPathOnServiceContainer(),
			// NOTE: When secrets-dir is specified, we can't add the --data-dir flag
			"--secrets-dir=" + validatorSecretsSharedPath.GetAbsPathOnServiceContainer(),
			// The node won't have a slashing protection database and will fail to start otherwise
			"--init-slashing-protection",
			"--http",
			"--unencrypted-http-transport",
			"--http-address=0.0.0.0",
			fmt.Sprintf("--http-port=%v", validatorHttpPortNum),
			"--beacon-nodes=" + beaconClientHttpUrl,
			"--enable-doppelganger-protection=false",
			// vvvvvvvvvvvvvvvvvvv PROMETHEUS CONFIG vvvvvvvvvvvvvvvvvvvvv
			"--metrics",
			"--metrics-address=" + privateIpAddr,
			"--metrics-allow-origin=*",
			fmt.Sprintf("--metrics-port=%v", validatorMetricsPortNum),
			// ^^^^^^^^^^^^^^^^^^^ PROMETHEUS CONFIG ^^^^^^^^^^^^^^^^^^^^^
		}
		if len(extraParams) > 0 {
			cmdArgs = append(cmdArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			validatorUsedPorts,
		).WithCmdOverride(
			cmdArgs,
		).WithFiles(map[services.FilesArtifactID]string{
			launcher.genesisData.GetFilesArtifactID(): genesisDataMountpointOnClients,
		}).Build()
		return containerConfig, nil
	}
}
