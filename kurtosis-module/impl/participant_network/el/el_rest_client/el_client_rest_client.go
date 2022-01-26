package el_rest_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	jsonRpcVersion = "2.0"
	requestId = 0

	getNodeInfoMethod = "admin_nodeInfo"
	getBlockNumberMethod = "eth_blockNumber"

	jsonContentType = "application/json"

	rpcRequestTimeout = 5 * time.Second

	hexDecodeBase = 16
	hexDecodeBits = 64
)

// API defined here:
// https://playground.open-rpc.org/?schemaUrl=https://raw.githubusercontent.com/ethereum/eth1.0-apis/assembled-spec/openrpc.json&uiSchema%5BappBar%5D%5Bui:splitView%5D=true&uiSchema%5BappBar%5D%5Bui:input%5D=false&uiSchema%5BappBar%5D%5Bui:examplesDropdown%5D=false
type ELClientRESTClient struct {
	ipAddr  string
	portNum uint16
	client *http.Client
}

func NewELClientRESTClient(ipAddr string, portNum uint16) *ELClientRESTClient {
	return &ELClientRESTClient{
		ipAddr: ipAddr,
		portNum: portNum,
		client: &http.Client{
			Timeout: rpcRequestTimeout,
		},
	}
}

func (client *ELClientRESTClient) GetBlockNumber() (uint64, error) {
	respObj := &GetBlockNumberResponse{}
	if err := client.makeRequest(getBlockNumberMethod, []string{}, respObj); err != nil {
		return 0, stacktrace.Propagate(err, "An error occurred getting the block number")
	}

	prefixedHexEncodedBlockNumberStr := respObj.HexEncodedBlockNumberStr
	hexEncodedBlockNumberStr := prefixedHexEncodedBlockNumberStr[2:]
	blockNumber, err := strconv.ParseUint(hexEncodedBlockNumberStr, hexDecodeBase, hexDecodeBits)
	if err != nil {
		return 0, stacktrace.Propagate(
			err,
			"An error occurred parsing block number string '%v' using base '%v' and bits '%v'",
			hexEncodedBlockNumberStr,
			hexDecodeBase,
			hexDecodeBits,
		)
	}
	return blockNumber, nil
}

func (client *ELClientRESTClient) GetNodeInfo() (*NodeInfo, error) {
	respObj := &GetNodeInfoResponse{}
	if err := client.makeRequest(getNodeInfoMethod, []string{}, respObj); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the node info")
	}
	return respObj.Result, nil
}

func (client *ELClientRESTClient) makeRequest(method string, params []string, respObj interface{}) error {
	url := fmt.Sprintf("http://%v:%v", client.ipAddr, client.portNum)

	requestBodyObj := &RequestBody{
		JsonRPC: jsonRpcVersion,
		Method:  method,
		Params:  params,
		ID:      requestId,
	}

	requestBodyBytes, err := json.Marshal(requestBodyObj)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred serializing the body of request to URL '%v'", url)
	}

	resp, err := client.client.Post(url, jsonContentType, bytes.NewReader(requestBodyBytes))
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred making the request to URL '%v' with body '%v'",
			url,
			string(requestBodyBytes),
		 )
	}
	defer resp.Body.Close()

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred reading the response body bytes")
	}

	if err := json.Unmarshal(respBodyBytes, respObj); err != nil {
		return stacktrace.Propagate(err, "An error occurred deserializing response body string '%v'", string(respBodyBytes))
	}
	return nil
}
