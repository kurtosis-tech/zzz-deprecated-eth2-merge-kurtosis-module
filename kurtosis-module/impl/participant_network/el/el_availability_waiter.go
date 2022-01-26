package el

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"time"
)

func WaitForELClientAvailability(restClient *el_rest_client.ELClientRESTClient, numRetries int, timeBetweenRetries time.Duration) (*el_rest_client.NodeInfo, error) {
	for i := 0; i < numRetries; i++ {
		nodeInfo, err := restClient.GetNodeInfo()
		if err == nil {
			return nodeInfo, nil
		}
		logrus.Debugf("Getting the node info via RPC failed with error: %v", err)
		time.Sleep(timeBetweenRetries)
	}
	return nil, stacktrace.NewError("Couldn't get the node's info even after %v retries with %v between retries", numRetries, timeBetweenRetries)
}
