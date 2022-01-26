package cl

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
		// NOTE: We don't check the return code because, per https://ethereum.github.io/beacon-APIs/#/Node/getHealth , a
		//  503 error is a node that's not initialized (which we'll almost definitely hit because we set the CL
		//  genesis time to be roughly after all nodes have started).
		// Unfortunately, each CL client interprets "genesis timestamp hasn't occurred yet" differently - Nimbus will return
		//  a 200, while Lighthouse will return a 503.
		// This means that the best we can do with this endpoint is "did we get an actual response?", and we can't check
		//  the error code
		_, err := restClient.GetHealth()
		if err == nil {
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
