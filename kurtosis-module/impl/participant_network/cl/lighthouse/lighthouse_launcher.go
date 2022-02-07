package lighthouse

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"os"
	"time"
)

const (
	lighthouseBinaryCommand = "lighthouse"

	// ---------------------------------- Beacon client -------------------------------------
	consensusDataDirpathOnBeaconServiceContainer = "/consensus-data"
	beaconConfigDataDirpathRelToSharedDirRoot = "config-data"

	// Port IDs
	beaconTcpDiscoveryPortID = "tcp-discovery"
	beaconUdpDiscoveryPortID = "udp-discovery"
	beaconHttpPortID         = "http"

	// Port nums
	beaaconDiscoveryPortNum uint16 = 9000
	beaconHttpPortNum              = 4000

	maxNumHealthcheckRetries = 10
	timeBetweenHealthcheckRetries = 1 * time.Second

	// ---------------------------------- Validator client -------------------------------------
	validatorConfigDataDirpathRelToSharedDirRoot = "config-data"

	validatorKeysRelDirpathInSharedDir = "validator-keys"
	validatorSecretsRelDirpathInSharedDir = "validator-secrets"

	validatorHttpPortID = "http"
	validatorHttpPortNum = 5042
)
var beaconUsedPorts = map[string]*services.PortSpec{
	beaconTcpDiscoveryPortID: services.NewPortSpec(beaaconDiscoveryPortNum, services.PortProtocol_TCP),
	beaconUdpDiscoveryPortID: services.NewPortSpec(beaaconDiscoveryPortNum, services.PortProtocol_UDP),
	beaconHttpPortID:         services.NewPortSpec(beaconHttpPortNum, services.PortProtocol_TCP),
}
var validatorUsedPorts = map[string]*services.PortSpec{
	validatorHttpPortID: services.NewPortSpec(validatorHttpPortNum, services.PortProtocol_TCP),
}
var lighthouseLogLevels = map[module_io.ParticipantLogLevel]string{
	module_io.ParticipantLogLevel_Error: "error",
	module_io.ParticipantLogLevel_Warn:  "warn",
	module_io.ParticipantLogLevel_Info:  "info",
	module_io.ParticipantLogLevel_Debug: "debug",
	module_io.ParticipantLogLevel_Trace: "trace",
}

type LighthouseCLClientLauncher struct {
	// The dirpath on the module container where the CL genesis config data directory exists
	configDataDirpathOnModuleContainer string
}

func NewLighthouseCLClientLauncher(configDataDirpathOnModuleContainer string) *LighthouseCLClientLauncher {
	return &LighthouseCLClientLauncher{configDataDirpathOnModuleContainer: configDataDirpathOnModuleContainer}
}

func (launcher *LighthouseCLClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	logLevel module_io.ParticipantLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *cl2.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconNodeServiceId := services.ServiceID(fmt.Sprintf("%v-beacon", serviceId))
	validatorNodeServiceId := services.ServiceID(fmt.Sprintf("%v-validator", serviceId))

	// Launch Beacon node
	beaconContainerConfigSupplier := launcher.getBeaconContainerConfigSupplier(image, bootnodeContext, elClientContext, logLevel)
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
	)
	if _, err := enclaveCtx.AddService(validatorNodeServiceId, validatorContainerConfigSupplier); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lighthouse validator node with service ID '%v'", validatorNodeServiceId)
	}

	// TODO add validator availability using teh validator API: https://ethereum.github.io/beacon-APIs/?urls.primaryName=v1#/ValidatorRequiredApi

	nodeIdentity, err := beaconRestClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lighthouse Beacon node's identity, which is necessary to retrieve its ENR")
	}

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		beaconHttpPortNum,
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
	logLevel module_io.ParticipantLogLevel,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		lighthouseLogLevel, found := lighthouseLogLevels[logLevel]
		if !found {
			return nil, stacktrace.NewError("No Lighthouse log level defined for client log level '%v'; this is a bug in the module", logLevel)
		}

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

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientCtx.GetIPAddress(),
			elClientCtx.GetRPCPortNum(),
		)

		configDataDirpathOnService := configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer()
		// NOTE: If connecting to the merge devnet remotely we DON'T want the following flags; when they're not set, the node's external IP address is auto-detected
		//  from the peers it communicates with but when they're set they basically say "override the autodetection and
		//  use what I specify instead." This requires having a know external IP address and port, which we definitely won't
		//  have with a network running in Kurtosis.
		//    "--disable-enr-auto-update",
		//    "--enr-address=" + externalIpAddress,
		//    fmt.Sprintf("--enr-udp-port=%v", beaaconDiscoveryPortNum),
		//    fmt.Sprintf("--enr-tcp-port=%v", beaaconDiscoveryPortNum),
		cmdArgs := []string{
			lighthouseBinaryCommand,
			"beacon_node",
			"--debug-level=" + lighthouseLogLevel,
			"--datadir=" + consensusDataDirpathOnBeaconServiceContainer,
			"--testnet-dir=" + configDataDirpathOnService,
			"--eth1",
			// vvvvvvvvvvvvvvvvvvv REMOVE THESE WHEN CONNECTING TO EXTERNAL NET vvvvvvvvvvvvvvvvvvvvv
			"--disable-enr-auto-update",
			"--enr-address=" + privateIpAddr,
			fmt.Sprintf("--enr-udp-port=%v", beaaconDiscoveryPortNum),
			fmt.Sprintf("--enr-tcp-port=%v", beaaconDiscoveryPortNum),
			// ^^^^^^^^^^^^^^^^^^^ REMOVE THESE WHEN CONNECTING TO EXTERNAL NET ^^^^^^^^^^^^^^^^^^^^^
			"--listen-address=0.0.0.0",
			fmt.Sprintf("--port=%v", beaaconDiscoveryPortNum), // NOTE: Remove for connecting to external net!
			"--http",
			"--http-address=0.0.0.0",
			fmt.Sprintf("--http-port=%v", beaconHttpPortNum),
			"--merge",
			"--http-allow-sync-stalled",
			// NOTE: This comes from:
			//   https://github.com/sigp/lighthouse/blob/7c88f582d955537f7ffff9b2c879dcf5bf80ce13/scripts/local_testnet/beacon_node.sh
			// and the option says it's "useful for testing in smaller networks" (unclear what happens in larger networks)
			"--disable-packet-filter",
			"--execution-endpoints=" + elClientRpcUrlStr,
			"--eth1-endpoints=" + elClientRpcUrlStr,
			// Set per Paris' recommendation to reduce noise in the logs
			"--subscribe-all-subnets",
		}
		if bootClClientCtx != nil {
			cmdArgs = append(cmdArgs, "--boot-nodes=" + bootClClientCtx.GetENR())
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
	logLevel module_io.ParticipantLogLevel,
	beaconClientHttpUrl string,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		lighthouseLogLevel, found := lighthouseLogLevels[logLevel]
		if !found {
			return nil, stacktrace.NewError("No Lighthouse log level defined for client log level '%v'; this is a bug in the module", logLevel)
		}

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

		configDataDirpathOnService := configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer()
		cmdArgs := []string{
			"lighthouse",
			"validator_client",
			"--debug-level=" + lighthouseLogLevel,
			"--testnet-dir=" + configDataDirpathOnService,
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
		}
		if len(extraParams) > 0 {
			launchNodeCmdArgs = append(launchNodeCmdArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			validatorUsedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()
		return containerConfig, nil
	}
}