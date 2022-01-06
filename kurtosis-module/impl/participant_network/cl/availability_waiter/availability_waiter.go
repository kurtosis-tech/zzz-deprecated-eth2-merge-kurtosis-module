package availability_waiter

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"time"
)

//READY state means that the Beacon node is synced
//the eth beacon endpoint /eth/v1/node/health has returned 200 status code, see more here: https://ethereum.github.io/beacon-APIs/#/Node/getHealth
const waitForAvailabilityExpectedStatus = "READY"

func WaitForBeaconClientAvailability(restClient *cl_client_rest_client.CLClientRESTClient, numRetries uint32, timeBetweenRetries time.Duration) error {
	for i := uint32(0); i < numRetries; i++ {
		status, err := restClient.GetHealth()
		if err == nil {
			if status == waitForAvailabilityExpectedStatus {
				return nil
			}
			return nil
		}
		logrus.Debugf(
			"CL client returned an error on GetHealth check; sleeping for %v: %v",
			timeBetweenRetries,
			err,
		)
		time.Sleep(timeBetweenRetries)
	}
	return stacktrace.NewError(
		"CL client didn't become available even after %v retries with %v between retries",
		numRetries,
		timeBetweenRetries,
	)
}
