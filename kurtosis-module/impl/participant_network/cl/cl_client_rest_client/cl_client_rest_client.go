package cl_client_rest_client

import (
	"encoding/json"
	"fmt"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type endpointClass string
type endpoint string
type HealthStatus string
const (
	// Correspond to the URL fragment outlined for each endpoint class on:
	// https://ethereum.github.io/beacon-APIs/
	endpointClass_node endpointClass = "node"

	endpoint_identity endpoint = "identity"
	endpoint_health   endpoint = "health"

	HealthStatus_Ready                     HealthStatus = "READY"
	HealthStatus_SyncingWithIncompleteData HealthStatus = "SYNCING_WITH_INCOMPLETE_DATA"
	HealthStatus_Error                     HealthStatus = "ERROR"
)
// Defined on https://ethereum.github.io/beacon-APIs/#/Node/getHealth
var healthResponseCodeToStatus = map[int]HealthStatus{
	200: HealthStatus_Ready,
	206: HealthStatus_SyncingWithIncompleteData,
	503: HealthStatus_Error,
}

// The Beacon node API is defined here: https://ethereum.github.io/beacon-APIs/
type CLClientRESTClient struct {
	ipAddr string
	portNum uint16
}

func NewCLClientRESTClient(ipAddr string, portNum uint16) *CLClientRESTClient {
	return &CLClientRESTClient{ipAddr: ipAddr, portNum: portNum}
}

func (client *CLClientRESTClient) GetHealth() (HealthStatus, error) {
	url := client.getUrl(endpointClass_node, endpoint_health)
	resp, err := http.Get(url)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred getting the node's health")
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
	url := client.getUrl(endpointClass_node, endpoint_identity)
	resp, err := http.Get(url)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the node's identity")
	}
	respBody := resp.Body
	defer respBody.Close()

	bodyBytes, err := ioutil.ReadAll(respBody)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred reading the response body")
	}

	logrus.Debugf("Identity from '%v': %v", url, string(bodyBytes))

	respObj := new(GetNodeIdentityResponse)
	if err := json.Unmarshal(bodyBytes, respObj); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred deserializing the response body")
	}
	return respObj.Data, nil
}

// ====================================================================================================
//                                    Private Helper Methods
// ====================================================================================================
func (client *CLClientRESTClient) getUrl(class endpointClass, endpoint endpoint) string {
	return fmt.Sprintf("http://%v:%v/eth/v1/%v/%v", client.ipAddr, client.portNum, class, endpoint)
}
