package geth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// From ETH HTTP API spec defined at:
	// https://playground.open-rpc.org/?schemaUrl=https://raw.githubusercontent.com/ethereum/eth1.0-apis/assembled-spec/openrpc.json&uiSchema%5BappBar%5D%5Bui:splitView%5D=false&uiSchema%5BappBar%5D%5Bui:input%5D=false&uiSchema%5BappBar%5D%5Bui:examplesDropdown%5D=false
	getBlockNumberRequestBody = `{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 0
}`

	contentTypeHeader = "Content-Type"
	jsonContentType = "application/json"

	resultBlockNumberUintBase = 16
	resultBlockNumberUintBits = 64
	resultBlockNumberHexStrPrefix = "0x"
)

type getBlockNumberResponse struct {
	BlockNumberHexStr string `json:"result"`
}

type gethMiningWaiter struct {
	privateIp         string
	privateRpcPortNum uint16
	httpClient        *http.Client
}

func newGethMiningWaiter(privateIp string, privateRpcPortNum uint16) *gethMiningWaiter {
	return &gethMiningWaiter{
		privateIp:         privateIp,
		privateRpcPortNum: privateRpcPortNum,
		httpClient:        &http.Client{},
	}
}

func (waiter *gethMiningWaiter) WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error {
	url := fmt.Sprintf("http://%v:%v", waiter.privateIp, waiter.privateRpcPortNum)

	bodyBytes := bytes.NewBufferString(getBlockNumberRequestBody)
	req, err := http.NewRequest(http.MethodGet, url, bodyBytes)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred creating the HTTP request to get the current Geth block")
	}
	req.Header.Add(contentTypeHeader, jsonContentType)

	for i := uint32(0); i < numRetries; i++ {
		if err := waiter.verifyBlockNumberGreaterThanZero(req); err == nil {
			return nil
		}
		logrus.Debugf(
			"Got an error using URL '%v' to verify that the Geth client's block number is > 0:\n%v",
			url,
			err,
		 )
		time.Sleep(timeBetweenRetries)
	}
	return stacktrace.NewError(
		"The Geth client at URL '%v' never started mining, even after %v retries with %v between retries",
		url,
		numRetries,
		timeBetweenRetries,
	)
}

func (waiter *gethMiningWaiter) verifyBlockNumberGreaterThanZero(request *http.Request) error {
	resp, err := waiter.httpClient.Do(request)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred making the request to get block number")
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred reading the response body bytes")
	}

	deserialized := &getBlockNumberResponse{}
	if err := json.Unmarshal(bodyBytes, deserialized); err != nil {
		return stacktrace.Propagate(err, "An error occurred deserializing the response body string '%v'", string(bodyBytes))
	}

	blockNumberHexStrWithoutPrefix := strings.TrimPrefix(
		deserialized.BlockNumberHexStr,
		resultBlockNumberHexStrPrefix,
	)
	blockNumber, err := strconv.ParseUint(
		blockNumberHexStrWithoutPrefix,
		resultBlockNumberUintBase,
		resultBlockNumberUintBits,
	)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred converting block number hex string '%v' into a uint with base %v and bits %v",
			blockNumberHexStrWithoutPrefix,
			resultBlockNumberUintBase,
			resultBlockNumberUintBits,
		 )
	}

	if blockNumber <= 0 {
		return stacktrace.NewError("Current block number '%v' isn't greater than 0", blockNumber)
	}

	return nil
}

