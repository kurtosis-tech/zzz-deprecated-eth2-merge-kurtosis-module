package el

import "time"

type ELClientMiningWaiter interface {
	WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error
}
