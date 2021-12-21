package availability_waiter

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"time"
)

func WaitForCLClientAvailability(restClient *cl_client_rest_client.CLClientRESTClient, numRetries uint32, timeBetweenRetries time.Duration) error {
	for i := uint32(0); i < numRetries; i++ {
		_, err := restClient.GetHealth()
		if err == nil {
			// TODO check the healthstatus???
			return nil
		}
		logrus.Debugf(
			"CL client returned an error on GetHealth check; sleeping for %v: %v",
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
