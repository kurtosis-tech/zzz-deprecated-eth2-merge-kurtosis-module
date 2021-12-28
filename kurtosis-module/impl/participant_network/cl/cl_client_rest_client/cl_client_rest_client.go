package cl_client_rest_client

import (
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
)

type endpointClass string
type endpoint string
type HealthStatus string

const (
	// Correspond to the URL fragment outlined for each endpoint class on:
	// https://ethereum.github.io/beacon-APIs/
	endpointClass_node endpointClass = "node"
	endpointClass_beacon endpointClass = "beacon"

	HealthStatus_Ready                     HealthStatus = "READY"
	HealthStatus_SyncingWithIncompleteData HealthStatus = "SYNCING_WITH_INCOMPLETE_DATA"
	HealthStatus_Error                     HealthStatus = "ERROR"

	headStateId = "head"

	epochUintBase = 10
	epochUintBits = 64

	slotUintBase = 10
	slotUintBits = 64
)

// Defined on https://ethereum.github.io/beacon-APIs/#/Node/getHealth
var healthResponseCodeToStatus = map[int]HealthStatus{
	200: HealthStatus_Ready,
	206: HealthStatus_SyncingWithIncompleteData,
	503: HealthStatus_Error,
}

// The Beacon node API is defined here: https://ethereum.github.io/beacon-APIs/
type CLClientRESTClient struct {
	ipAddr  string
	portNum uint16
}

func NewCLClientRESTClient(ipAddr string, portNum uint16) *CLClientRESTClient {
	return &CLClientRESTClient{ipAddr: ipAddr, portNum: portNum}
}

func (client *CLClientRESTClient) GetHealth() (HealthStatus, error) {
	url := client.getUrl(endpointClass_node, "health")
	resp, err := http.Get(url)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred calling GET on node health endpoint '%v'", url)
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode

	result, found := healthResponseCodeToStatus[statusCode]
	if !found {
		return "", stacktrace.NewError("Received unrecognized status code '%v' from the health endpoint", statusCode)
	}
	return result, nil
}

func (client *CLClientRESTClient) GetNodeIdentity() (*NodeIdentity, error) {
	respObj := new(GetNodeIdentityResponse)
	if err := client.getAndParseResponse(endpointClass_node, "identity", respObj); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the node identity and parsing the response")
	}
	return respObj.Data, nil
}

func (client *CLClientRESTClient) GetCurrentSlot() (uint64, error) {
	respObj := new(GetBlockHeadersResponse)
	if err := client.getAndParseResponse(endpointClass_beacon, "headers", respObj); err != nil {
		return 0, stacktrace.Propagate(err, "An error occurred getting the headers and parsing the response")
	}

	if len(respObj.Data) != 1 {
		return 0, stacktrace.NewError("Expected exactly only block header data object but got '%v'", len(respObj.Data))
	}
	currentSlotStr := respObj.Data[0].Header.Message.Slot
	currentSlot, err := strconv.ParseUint(currentSlotStr, slotUintBase, slotUintBits)
	if err != nil {
		return 0, stacktrace.Propagate(
			err,
			"An error occurred parsing current slot string '%v' with base %v and %v bits",
			currentSlotStr,
			slotUintBase,
			slotUintBits,
		 )
	}
	return currentSlot, nil
}

func (client *CLClientRESTClient) GetFinalizedEpoch() (uint64, error) {
	respObj := new(GetFinalityCheckpointsResponse)
	suffix := fmt.Sprintf("states/%v/finality_checkpoints", headStateId)
	if err := client.getAndParseResponse(endpointClass_beacon, suffix, respObj); err != nil {
		return 0, stacktrace.Propagate(err, "An error occurred getting the head finality checkpoints")
	}
	finalizedEpochStr := respObj.Data.Finalized.Epoch
	finalizedEpoch, err := strconv.ParseUint(finalizedEpochStr, epochUintBase, epochUintBits)
	if err != nil {
		return 0, stacktrace.Propagate(
			err,
			"An error occurred parsing current epoch string '%v' with base %v and %v bits",
			finalizedEpochStr,
			epochUintBase,
			epochUintBits,
		 )
	}
	return finalizedEpoch, nil
}

// ====================================================================================================
//                                    Private Helper Methods
// ====================================================================================================
func (client *CLClientRESTClient) getUrl(endpointClass endpointClass, suffix string) string {
	return fmt.Sprintf("http://%v:%v/eth/v1/%v/%v", client.ipAddr, client.portNum, endpointClass, suffix)
}

// Makes a GET request to the given endpointClass + suffix and JSON-parses the response into the given response object
func (client *CLClientRESTClient) getAndParseResponse(endpointClass endpointClass, suffix string, respObj interface{}) error {
	url := client.getUrl(endpointClass, suffix)
	resp, err := http.Get(url)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred making the GET request to '%v'", url)
	}
	respBody := resp.Body
	defer respBody.Close()

	bodyBytes, err := ioutil.ReadAll(respBody)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred reading the response body from '%v'", url)
	}

	logrus.Debugf("Response string from '%v': %v", url, string(bodyBytes))

	if err := json.Unmarshal(bodyBytes, respObj); err != nil {
		return stacktrace.Propagate(err, "An error occurred deserializing the response body")
	}

	return nil
}