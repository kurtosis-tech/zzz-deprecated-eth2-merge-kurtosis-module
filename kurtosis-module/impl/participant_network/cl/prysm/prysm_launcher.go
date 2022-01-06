package prysm

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/availability_waiter"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"path"
	"text/template"
	"time"
)

const (
	beaconNodeImageName    = "prysmaticlabs/prysm-beacon-chain:latest"
	validatorNodeImageName = "prysmaticlabs/prysm-validator:latest"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	// Port IDs
	tcpDiscoveryPortID        = "tcp-discovery"
	udpDiscoveryPortID        = "udp-discovery"
	rpcPortID                 = "rpc"
	httpPortID                = "http"
	beaconMonitoringPortID    = "beacon-monitoring"
	validatorMonitoringPortID = "validator-monitoring"

	// Port nums
	discoveryTCPPortNum        uint16 = 13000
	discoveryUDPPortNum        uint16 = 12000
	rpcPortNum                 uint16 = 4000
	httpPortNum                uint16 = 3500
	beaconMonitoringPortNum    uint16 = 8080
	validatorMonitoringPortNum uint16 = 8081

	genesisConfigYmlRelFilepathInSharedDir = "genesis-config.yml"
	genesisSszRelFilepathInSharedDir       = "genesis.ssz"
	prysmPasswordTxtRelFilepathInSharedDir = "prysm-password.txt"

	validatorKeysRelDirpathInSharedDir    = "validator-keys"
	validatorSecretsRelDirpathInSharedDir = "validator-secrets"

	maxNumHealthcheckRetries      = 20
	timeBetweenHealthcheckRetries = 1 * time.Second

	beaconSuffixServiceId    = "beacon"
	validatorSuffixServiceId = "validator"
)

var beaconNodeUsedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID:     services.NewPortSpec(discoveryTCPPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID:     services.NewPortSpec(discoveryUDPPortNum, services.PortProtocol_UDP),
	rpcPortID:              services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	httpPortID:             services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	beaconMonitoringPortID: services.NewPortSpec(beaconMonitoringPortNum, services.PortProtocol_TCP),
}

var validatorNodeUsedPorts = map[string]*services.PortSpec{
	validatorMonitoringPortID: services.NewPortSpec(validatorMonitoringPortNum, services.PortProtocol_TCP),
}

type prysmPasswordTemplateData struct {
	Password string
}

type PrysmClientLauncher struct {
	genesisConfigYmlFilepathOnModuleContainer string
	genesisSszFilepathOnModuleContainer       string
	prysmPassword                             string
	prysmPasswordTxtTemplate                  *template.Template
}

func NewPrysmCLCLientLauncher(
	genesisConfigYmlFilepathOnModuleContainer string,
	genesisSszFilepathOnModuleContainer string,
	prysmPassword string,
	prysmPasswordTxtTemplate *template.Template,
) *PrysmClientLauncher {
	return &PrysmClientLauncher{
		genesisConfigYmlFilepathOnModuleContainer: genesisConfigYmlFilepathOnModuleContainer,
		genesisSszFilepathOnModuleContainer:       genesisSszFilepathOnModuleContainer,
		prysmPassword:                             prysmPassword,
		prysmPasswordTxtTemplate:                  prysmPasswordTxtTemplate,
	}
}

func (launcher *PrysmClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconServiceId := serviceId + "-" + beaconSuffixServiceId
	validatorServiceId := serviceId + "-" + validatorSuffixServiceId

	beaconContainerConfigSupplier := getBeaconContainerConfigSupplier(
		bootnodeContext,
		elClientContext,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		launcher.genesisSszFilepathOnModuleContainer,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconServiceId, beaconContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Prysm CL beacon client with service ID '%v'", serviceId)
	}

	httpPort, found := beaconServiceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Prysm beacon service to have port with ID '%v', but none was found", httpPortID)
	}

	beaconRestClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), httpPort.GetNumber())
	if err := availability_waiter.WaitForBeaconClientAvailability(beaconRestClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Prysm beacon node to become available")
	}

	nodeIdentity, err := beaconRestClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Prysm beacon node's identity, which is necessary to retrieve its ENR")
	}

	beaconRPCEndpoint := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), rpcPortNum)
	beaconHTTPEndpoint := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), httpPortNum)
	validatorContainerConfigSupplier := getValidatorContainerConfigSupplier(
		validatorServiceId,
		beaconRPCEndpoint,
		beaconHTTPEndpoint,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		nodeKeystoreDirpaths.RawKeysDirpath,
		nodeKeystoreDirpaths.PrysmDirpath,
		launcher.prysmPassword,
		launcher.prysmPasswordTxtTemplate,
	)
	_, err = enclaveCtx.AddService(validatorServiceId, validatorContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Prysm CL validator client with service ID '%v'", serviceId)
	}

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		httpPortNum,
		beaconRestClient,
	)

	return result, nil

}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func getBeaconContainerConfigSupplier(
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	genesisConfigYmlFilepathOnModuleContainer string,
	genesisSszFilepathOnModuleContainer string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		genesisConfigYmlSharedPath := sharedDir.GetChildPath(genesisConfigYmlRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisConfigYmlFilepathOnModuleContainer, genesisConfigYmlSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis config YML from '%v' to shared dir relative path '%v'",
				genesisConfigYmlFilepathOnModuleContainer,
				genesisConfigYmlRelFilepathInSharedDir,
			)
		}

		genesisSszSharedPath := sharedDir.GetChildPath(genesisSszRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisSszFilepathOnModuleContainer, genesisSszSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis SSZ from '%v' to shared dir relative path '%v'",
				genesisSszFilepathOnModuleContainer,
				genesisSszRelFilepathInSharedDir,
			)
		}

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetRPCPortNum(),
		)

		cmdArgs := []string{
			"--accept-terms-of-use=true", //it's mandatory in order to run the node
			"--prater",                   //it's a tesnet setup, it's mandatory to set a network (https://docs.prylabs.network/docs/install/install-with-script#before-you-begin-pick-your-network-1)
			"--datadir=" + consensusDataDirpathOnServiceContainer,
			"--chain-config-file=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--genesis-state=" + genesisSszSharedPath.GetAbsPathOnServiceContainer(),
			"--http-web3provider=" + elClientRpcUrlStr,
			"--http-modules=prysm,eth",
			"--rpc-host=" + privateIpAddr,
			fmt.Sprintf("--rpc-port=%v", rpcPortNum),
			"--grpc-gateway-host=0.0.0.0",
			fmt.Sprintf("--grpc-gateway-port=%v", httpPortNum),
			fmt.Sprintf("--p2p-tcp-port=%v", discoveryTCPPortNum),
			fmt.Sprintf("--p2p-udp-port=%v", discoveryUDPPortNum),
			"--monitoring-host=" + privateIpAddr,
			fmt.Sprintf("--monitoring-port=%v", beaconMonitoringPortNum),
			"--verbosity=debug",
		}
		if bootnodeContext != nil {
			cmdArgs = append(cmdArgs, "--bootstrap-node=" + bootnodeContext.GetENR())
		}

		containerConfig := services.NewContainerConfigBuilder(
			beaconNodeImageName,
		).WithUsedPorts(
			beaconNodeUsedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return containerConfigSupplier
}

func getValidatorContainerConfigSupplier(
	serviceId services.ServiceID,
	beaconRPCEndpoint string,
	beaconHTTPEndpoint string,
	genesisConfigYmlFilepathOnModuleContainer string,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
	prysmPassword string,
	prysmPasswordTxtTemplate *template.Template,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		genesisConfigYmlSharedPath := sharedDir.GetChildPath(genesisConfigYmlRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisConfigYmlFilepathOnModuleContainer, genesisConfigYmlSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis config YML from '%v' to shared dir relative path '%v'",
				genesisConfigYmlFilepathOnModuleContainer,
				genesisConfigYmlRelFilepathInSharedDir,
			)
		}

		validatorKeysSharedPath := sharedDir.GetChildPath(validatorKeysRelDirpathInSharedDir)
		if err := recursive_copy.Copy(
			validatorKeysDirpathOnModuleContainer,
			validatorKeysSharedPath.GetAbsPathOnThisContainer(),
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

		prysmPasswordTemplData := &prysmPasswordTemplateData{
			Password: prysmPassword,
		}

		prysmPasswordTxtSharedPath := sharedDir.GetChildPath(prysmPasswordTxtRelFilepathInSharedDir)
		if err := service_launch_utils.FillTemplateToSharedPath(prysmPasswordTxtTemplate, prysmPasswordTemplData, prysmPasswordTxtSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the Prysm password txt template")
		}

		rootDirpath := path.Join(consensusDataDirpathOnServiceContainer, string(serviceId))

		cmdArgs := []string{
			"--accept-terms-of-use=true", //it's mandatory in order to run the node
			"--prater",                   //it's a tesnet setup, it's mandatory to set a network (https://docs.prylabs.network/docs/install/install-with-script#before-you-begin-pick-your-network-1)
			"--beacon-rpc-gateway-provider=" + beaconHTTPEndpoint,
			"--beacon-rpc-provider=" + beaconRPCEndpoint,
			"--wallet-dir=" + validatorSecretsSharedPath.GetAbsPathOnServiceContainer(),
			"--wallet-password-file=" + prysmPasswordTxtSharedPath.GetAbsPathOnServiceContainer(),
			"--datadir=" + rootDirpath,
			"--monitoring-host=" + privateIpAddr,
			fmt.Sprintf("--monitoring-port=%v", validatorMonitoringPortNum),
			"--verbosity=debug",
		}

		containerConfig := services.NewContainerConfigBuilder(
			validatorNodeImageName,
		).WithUsedPorts(
			validatorNodeUsedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return containerConfigSupplier
}
