package besu

import (
	"net/http"
	"time"
)

type besuMiningWaiter struct {
	privateIp         string
	privateRpcPortNum uint16
	httpClient        *http.Client
}

func newBesuMiningWaiter(privateIp string, privateRpcPortNum uint16) *besuMiningWaiter {
	return &besuMiningWaiter{
		privateIp:         privateIp,
		privateRpcPortNum: privateRpcPortNum,
		httpClient:        &http.Client{},
	}
}

func (n besuMiningWaiter) WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error {
	// TODO Fill this in
	return nil
}
