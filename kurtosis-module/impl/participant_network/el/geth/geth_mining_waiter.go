package geth

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/stacktrace"
	"time"
)

type gethMiningWaiter struct {
	restClient *el_rest_client.ELClientRESTClient
}

func newGethMiningWaiter(elRestClient *el_rest_client.ELClientRESTClient) *gethMiningWaiter {
	return &gethMiningWaiter{
		restClient: elRestClient,
	}
}

func (waiter *gethMiningWaiter) WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error {
	for i := uint32(0); i < numRetries; i++ {
		blockNumber, err := waiter.restClient.GetBlockNumber()
		if err == nil && blockNumber > 0 {
			return nil
		}
		time.Sleep(timeBetweenRetries)
	}
	return stacktrace.NewError(
		"The Geth client never started mining, even after %v retries with %v between retries",
		numRetries,
		timeBetweenRetries,
	)
}
