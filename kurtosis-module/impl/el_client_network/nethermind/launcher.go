package nethermind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"
)

const (
	imageName = "nethermindeth/nethermind:kintsugi_0.5"
	// To start a bootnode, we provide this string to the launchNode function
	bootnodeEnodeStrForStartingBootnode = ""

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "nethermind_genesis.json"

	configDirpath = "configs"

	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	rpcPortId = "rpc"
	wsPortId  = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"

	jsonContentTypeHeader = "application/json"
	rpcRequestTimeout = 5 * time.Second

	getNodeInfoRpcRequestBody = `{"jsonrpc":"2.0","method": "admin_nodeInfo","params":[],"id":1}`
	getNodeInfoMaxRetries = 60 //TODO try to adjust
	getNodeInfoTimeBetweenRetries = 500 * time.Millisecond

	kintsugiConfigFilename ="kurtosis-config.cfg"
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId: services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId: services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}

type nethermindTemplateData struct {
	NetworkID string
}

type nethermindConfigTemplateData struct {
	TestNodeKey string
}

type NethermindELClientLauncher struct {
	genesisJsonTemplate *template.Template
	configTemplate *template.Template
}

func NewNethermindELClientLauncher(genesisJsonTemplate *template.Template, configTemplate *template.Template) *NethermindELClientLauncher {
	return &NethermindELClientLauncher{
		genesisJsonTemplate: genesisJsonTemplate,
		configTemplate: configTemplate,
	}
}

func (launcher *NethermindELClientLauncher) LaunchBootNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, networkId, bootnodeEnodeStrForStartingBootnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting boot node with service ID '%v'", serviceId)
	}
	return clientCtx, nil
}

func (launcher *NethermindELClientLauncher) LaunchChildNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	bootnodeEnode string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, networkId, bootnodeEnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting child node with service ID '%v' connected to boot node with enode '%v'", serviceId, bootnodeEnode)
	}
	return clientCtx, nil
}

// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func (launcher *NethermindELClientLauncher) launchNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be launched as a bootnode
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(networkId, bootnodeEnode)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	nodeInfo, err := getNodeInfoWithRetry(serviceCtx.GetPrivateIPAddress())
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the newly-started node's info")
	}

	logrus.Infof("Node info: %+v", nodeInfo)

	result := el_client_network.NewExecutionLayerClientContext(
		serviceCtx,
		"",  //Nethermind node info endpoint doesn't return ENR field https://docs.nethermind.io/nethermind/ethereum-client/json-rpc/admin
		nodeInfo.Enode,
		rpcPortId,
	)


	return result, nil
}

func (launcher *NethermindELClientLauncher) getContainerConfigSupplier(
	networkId string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be configured as a bootnode
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		genesisJsonOnModuleContainerSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)

		networkIdHexStr, err := getNetworkIdHexSting(networkId)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred getting network ID in Hex")
		}

		nethermindTmplData := nethermindTemplateData{
			NetworkID: networkIdHexStr,
		}

		fp, err := os.Create(genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer())
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening file '%v' for writing", genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer())
		}
		defer fp.Close()

		if err = launcher.genesisJsonTemplate.Execute(fp, nethermindTmplData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the template")
		}

		kintsugiConfigSharedPath := sharedDir.GetChildPath(kintsugiConfigFilename)

		nethermindConfigTmplData := nethermindConfigTemplateData{
			TestNodeKey: getRandomTestNodeKey(),
		}

		kintsugiConfigFilePath, err := os.Create(kintsugiConfigSharedPath.GetAbsPathOnThisContainer())
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening file '%v' for writing", kintsugiConfigSharedPath.GetAbsPathOnThisContainer())
		}
		defer fp.Close()

		if err = launcher.configTemplate.Execute(kintsugiConfigFilePath, nethermindConfigTmplData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the template")
		}


		/*if err := service_launch_utils.CopyFileToSharedPath(kintsugiConfigFilepathInModuleContainer, kintsugiConfigSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying genesis JSON file '%v' into shared directory path '%v'", kintsugiConfigFilepathInModuleContainer, kintsugiConfigSharedPath)
		}*/

		commandArgs := []string{
			"--config",
			kintsugiConfigSharedPath.GetAbsPathOnServiceContainer(),
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--Init.ChainSpecPath=" + genesisJsonOnModuleContainerSharedPath.GetAbsPathOnServiceContainer(),
			"--Init.WebSocketsEnabled=true",
			"--Init.IsMining=true",
			//"--Init.DiscoveryEnabled=true",
			"--Init.DiagnosticMode=None",
			//"--Init.StoreReceipts=true",
			//"--Init.EnableUnsecuredDevWallet=true",
			"--JsonRpc.Enabled=true",
			"--JsonRpc.EnabledModules=net,eth,consensus,engine,admin,subscribe,trace,txpool,web3,personal,proof,parity,health,debug",
			fmt.Sprintf("--JsonRpc.Port=%v", rpcPortNum),
			fmt.Sprintf("--JsonRpc.WebSocketsPort=%v", wsPortNum),
			"--JsonRpc.Host=0.0.0.0",
			fmt.Sprintf("--Network.ExternalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.LocalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.DiscoveryPort=%v", discoveryPortNum),
			fmt.Sprintf("--Network.P2PPort=%v", discoveryPortNum),
		}
		if bootnodeEnode != bootnodeEnodeStrForStartingBootnode {
			logrus.Infof("Entra a setear el bootnode: %v", bootnodeEnode)
			commandArgs = append(
				commandArgs,
				"--Discovery.Bootnodes=" + bootnodeEnode,
			)
		}

		logrus.Infof("Command Args: %+v", commandArgs)

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			commandArgs,
		).Build()

		return containerConfig, nil
	}
	return result
}

func getNodeInfoWithRetry(privateIpAddr string) (NodeInfo, error) {
	getNodeInfoResponse := new(GetNodeInfoResponse)
	for i := 0; i < getNodeInfoMaxRetries; i++ {
		if err := sendRpcCall(privateIpAddr, getNodeInfoRpcRequestBody, getNodeInfoResponse); err == nil {
			return getNodeInfoResponse.Result, nil
		} else {
			logrus.Debugf("Getting the node info via RPC failed with error: %v", err)
		}
		time.Sleep(getNodeInfoTimeBetweenRetries)
	}
	return NodeInfo{}, stacktrace.NewError("Couldn't get the node's info even after %v retries with %v between retries", getNodeInfoMaxRetries, getNodeInfoTimeBetweenRetries)
}

func sendRpcCall(privateIpAddr string, requestBody string, targetStruct interface{}) error {
	url := fmt.Sprintf("http://%v:%v", privateIpAddr, rpcPortNum)
	var jsonByteArray = []byte(requestBody)

	logrus.Debugf("Sending RPC call to '%v' with JSON body '%v'...", url, requestBody)

	client := http.Client{
		Timeout: rpcRequestTimeout,
	}
	resp, err := client.Post(url, jsonContentTypeHeader, bytes.NewBuffer(jsonByteArray))
	if err != nil {
		return stacktrace.Propagate(err, "Failed to send RPC request to Nethermind node with private IP '%v'", privateIpAddr)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stacktrace.NewError(
			"Received non-%v status code '%v' on RPC request to Nethermind node with private IP '%v'",
			http.StatusOK,
			resp.StatusCode,
			privateIpAddr,
		)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return stacktrace.Propagate(err, "Error reading the RPC call response body")
	}
	bodyString := string(bodyBytes)
	logrus.Infof("Response for RPC call %v: %v", requestBody, bodyString)

	json.Unmarshal(bodyBytes, targetStruct)
	if err := json.Unmarshal(bodyBytes, targetStruct); err != nil {
		return stacktrace.Propagate(err, "Error JSON-parsing Nethermind node RPC response string '%v' into a struct", bodyString)
	}
	return nil
}

func getNetworkIdHexSting(networkId string) (string, error){
	uintBase := 10
	uintBits := 64
	networkIdUint64, err := strconv.ParseUint(networkId, uintBase, uintBits)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred parsing network ID string '%v' to uint with base %v and %v bits",
			networkId,
			uintBase,
			uintBits,
		)
	}
	return fmt.Sprintf("0x%x", networkIdUint64), nil
}

func getRandomTestNodeKey() string {
	rand.Seed(time.Now().UnixNano())
	min := 10
	max := 99
	randomNumber := rand.Intn(max - min + 1) + min
	randomTestNodeKey := fmt.Sprintf("0x45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2%v", randomNumber)
	logrus.Infof("New random TestNodeKey: %v", randomTestNodeKey)
	return randomTestNodeKey
}