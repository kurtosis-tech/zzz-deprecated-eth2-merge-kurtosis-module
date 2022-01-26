package mining_waiter

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/el_rest_client"
	"github.com/kurtosis-tech/stacktrace"
	"time"
)

type miningWaiter struct {
	restClient *el_rest_client.ELClientRESTClient
}

func NewMiningWaiter(restClient *el_rest_client.ELClientRESTClient) *miningWaiter {
	return &miningWaiter{restClient: restClient}
}

func (waiter *miningWaiter) WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error {
	for i := uint32(0); i < numRetries; i++ {
		blockNumber, err := waiter.restClient.GetBlockNumber()
		if err == nil && blockNumber > 0 {
			return nil
		}
		time.Sleep(timeBetweenRetries)
	}
	return stacktrace.NewError(
		"The EL client never started mining, even after %v retries with %v between retries",
		numRetries,
		timeBetweenRetries,
	)
}
