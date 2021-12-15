package geth_el_client_launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/el_client_network"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	imageName = "kurtosistech/go-ethereum:d99ac5a7d"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	RpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"

	// NOTE: This can't be 0x00000....000
	// See: https://github.com/ethereum/go-ethereum/issues/19547
	miningRewardsAccount = "0x0000000000000000000000000000000000000001"
	numMiningThreads = 3

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "genesis.json"

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"


	jsonContentTypeHeader = "application/json"
	rpcRequestTimeout = 5 * time.Second

	getNodeInfoRpcRequestBody = `{"jsonrpc":"2.0","method": "admin_nodeInfo","params":[],"id":1}`
	getNodeInfoMaxRetries = 10
	getNodeInfoTimeBetweenRetries = 500 * time.Millisecond

	// To start a bootnode, we provide this string to the launchNode function
	bootnodeEnodeStrForStartingBootnode = ""
)
var usedPorts = map[string]*services.PortSpec{
	RpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}
var entrypointArgs = []string{"sh", "-c"}

type GethELClientLauncher struct {}

func NewGethELClientLauncher() *GethELClientLauncher {
	return &GethELClientLauncher{}
}

func (launcher *GethELClientLauncher) LaunchBootNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	genesisJsonFilepathOnModuleContainer string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launchNode(enclaveCtx, serviceId, networkId, genesisJsonFilepathOnModuleContainer, bootnodeEnodeStrForStartingBootnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting boot node with service ID '%v'", serviceId)
	}
	return clientCtx, nil
}

func (launcher *GethELClientLauncher) LaunchChildNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	genesisJsonFilepathOnModuleContainer string,
	bootnodeEnode string,
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launchNode(enclaveCtx, serviceId, networkId, genesisJsonFilepathOnModuleContainer, bootnodeEnode)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting child node with service ID '%v' connected to boot node with enode '%v'", serviceId, bootnodeEnode)
	}
	return clientCtx, nil
}


// ====================================================================================================
//                                       Private Helper Methods
// ====================================================================================================
func launchNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	networkId string,
	genesisJsonFilepathOnModuleContainer string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be launched as a bootnode
) (
	resultClientCtx *el_client_network.ExecutionLayerClientContext,
	resultErr error,
) {
	containerConfigSupplier := getContainerConfigSupplier(genesisJsonFilepathOnModuleContainer, networkId, bootnodeEnode)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	nodeInfo, err := getNodeInfoWithRetry(serviceCtx.GetPrivateIPAddress())
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the newly-started node's info")
	}

	result := el_client_network.NewExecutionLayerClientContext(
		serviceCtx,
		nodeInfo.ENR,
		nodeInfo.Enode,
	)

	return result, nil
}

func getContainerConfigSupplier(
	genesisJsonOnModuleContainerFilepath string,
	networkId string,
	bootnodeEnode string, // NOTE: If this is emptystring, the node will be configured as a bootnode
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		genesisJsonOnModuleContainerSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)

		srcFp, err := os.Open(genesisJsonOnModuleContainerFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the genesis JSON file '%v' on the module container", genesisJsonOnModuleContainerFilepath)
		}

		destFilepath := genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer()
		destFp, err := os.Create(destFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the genesis JSON destination filepath '%v' on the module container", destFilepath)
		}

		if _, err := io.Copy(destFp, srcFp); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis file from the module container '%v' to the shared directory of the client container '%v'",
				genesisJsonOnModuleContainerFilepath,
				destFilepath,
			)
		}

		commandArgs := []string{
			"geth",
			"init",
			"--datadir=" + executionDataDirpathOnClientContainer,
			genesisJsonOnModuleContainerSharedPath.GetAbsPathOnServiceContainer(),
			"&&",
			"geth",
			"--mine",
			"--miner.etherbase=" + miningRewardsAccount,
			fmt.Sprintf("--miner.threads=%v", numMiningThreads),
			"--datadir="  + executionDataDirpathOnClientContainer,
			"--networkid=" + networkId,
			"--catalyst",
			"--http",
			"--http.addr=0.0.0.0",
			// WARNING: The admin info endpoint is enabled so that we can easily get ENR/enode, which means
			//  that users should NOT store private information in these Kurtosis nodes!
			"--http.api=admin,engine,net,eth",
			"--ws",
			"--ws.api=engine,net,eth",
			"--allow-insecure-unlock",
			"--nat=extip:" + privateIpAddr,
			"--verbosity=3",
		}
		if bootnodeEnode != bootnodeEnodeStrForStartingBootnode {
			commandArgs = append(
				commandArgs,
				"--bootnodes=" + bootnodeEnode,
			)
		}
		commandStr := strings.Join(commandArgs, " ")

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
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
		return stacktrace.Propagate(err, "Failed to send RPC request to Geth node with private IP '%v'", privateIpAddr)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return stacktrace.NewError(
			"Received non-%v status code '%v' on RPC request to Geth node with private IP '%v'",
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
	logrus.Tracef("Response for RPC call %v: %v", requestBody, bodyString)

	json.Unmarshal(bodyBytes, targetStruct)
	if err := json.Unmarshal(bodyBytes, targetStruct); err != nil {
		return stacktrace.Propagate(err, "Error JSON-parsing Geth node RPC response string '%v' into a struct", bodyString)
	}
	return nil
}
