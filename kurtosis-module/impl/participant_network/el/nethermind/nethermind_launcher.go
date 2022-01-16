package nethermind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	//Latest Nethermind Kintsugi image version released to date, can check latest here: https://github.com/NethermindEth/nethermind/issues/3581
	imageName = "nethermindeth/nethermind:kintsugi_0.5"

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedNethermindGenesisJsonRelFilepath = "nethermind_genesis.json"

	miningRewardsAccount = "0x0000000000000000000000000000000000000001"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	rpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"

	jsonContentTypeHeader = "application/json"
	rpcRequestTimeout     = 5 * time.Second

	getNodeInfoRpcRequestBody     = `{"jsonrpc":"2.0","method": "admin_nodeInfo","params":[],"id":1}`
	getNodeInfoMaxRetries         = 20
	getNodeInfoTimeBetweenRetries = 500 * time.Millisecond
)

var usedPorts = map[string]*services.PortSpec{
	rpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}
var nethermindLogLevels = map[module_io.ParticipantLogLevel]string{
	module_io.ParticipantLogLevel_Error: "ERROR",
	module_io.ParticipantLogLevel_Warn:  "WARN",
	module_io.ParticipantLogLevel_Info:  "INFO",
	module_io.ParticipantLogLevel_Debug: "DEBUG",
}

type NethermindELClientLauncher struct {
	genesisJsonFilepathOnModule string
	totalTerminalDifficulty     uint64
}

func NewNethermindELClientLauncher(genesisJsonFilepathOnModule string, totalTerminalDifficulty uint64) *NethermindELClientLauncher {
	return &NethermindELClientLauncher{genesisJsonFilepathOnModule: genesisJsonFilepathOnModule, totalTerminalDifficulty: totalTerminalDifficulty}
}

func (launcher *NethermindELClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	logLevel module_io.ParticipantLogLevel,
	networkId string,
	bootnodeContext *el.ELClientContext,
) (resultClientCtx *el.ELClientContext, resultErr error) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(bootnodeContext, logLevel)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	nodeInfo, err := getNodeInfoWithRetry(serviceCtx.GetPrivateIPAddress())
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the newly-started node's info")
	}

	miningWaiter := &nethermindMiningWaiter{}
	result := el.NewELClientContext(
		// TODO TODO TODO TODO Get Nethermind ENR, so that CL clients can connect to it!!!
		"", //Nethermind node info endpoint doesn't return ENR field https://docs.nethermind.io/nethermind/ethereum-client/json-rpc/admin
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
func (launcher *NethermindELClientLauncher) getContainerConfigSupplier(
	bootnodeCtx *el.ELClientContext,
	logLevel module_io.ParticipantLogLevel,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		nethermindLogLevel, found := nethermindLogLevels[logLevel]
		if !found {
			return nil, stacktrace.NewError("No Nethermind log level defined for client log level '%v'; this is a bug in the module", logLevel)
		}

		nethermindGenesisJsonSharedPath := sharedDir.GetChildPath(sharedNethermindGenesisJsonRelFilepath)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisJsonFilepathOnModule, nethermindGenesisJsonSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the Nethermind genesis JSON file from '%v' into the Nethermind node being started",
				launcher.genesisJsonFilepathOnModule,
			 )
		}

		commandArgs := []string{
			"--config=kintsugi",
			"--log=" + nethermindLogLevel,
			"--datadir=" + executionDataDirpathOnClientContainer,
			"--Init.ChainSpecPath=" + nethermindGenesisJsonSharedPath.GetAbsPathOnServiceContainer(),
			"--Init.WebSocketsEnabled=true",
			"--Init.DiagnosticMode=None",
			"--JsonRpc.Enabled=true",
			"--JsonRpc.EnabledModules=net,eth,consensus,engine,admin",
			"--JsonRpc.Host=0.0.0.0",
			fmt.Sprintf("--JsonRpc.Port=%v", rpcPortNum),
			fmt.Sprintf("--JsonRpc.WebSocketsPort=%v", wsPortNum),
			fmt.Sprintf("--Network.ExternalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.LocalIp=%v", privateIpAddr),
			fmt.Sprintf("--Network.DiscoveryPort=%v", discoveryPortNum),
			fmt.Sprintf("--Network.P2PPort=%v", discoveryPortNum),
			"--Merge.Enabled=true",
			fmt.Sprintf("--Merge.TerminalTotalDifficulty=%v", launcher.totalTerminalDifficulty),
			"--Merge.BlockAuthorAccount=" + miningRewardsAccount,
		}
		if bootnodeCtx != nil {
			commandArgs = append(
				commandArgs,
				"--Discovery.Bootnodes=" + bootnodeCtx.GetEnode(),
			)
		}

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
	logrus.Debugf("Response for RPC call %v: %v", requestBody, bodyString)

	json.Unmarshal(bodyBytes, targetStruct)
	if err := json.Unmarshal(bodyBytes, targetStruct); err != nil {
		return stacktrace.Propagate(err, "Error JSON-parsing Nethermind node RPC response string '%v' into a struct", bodyString)
	}
	return nil
}
