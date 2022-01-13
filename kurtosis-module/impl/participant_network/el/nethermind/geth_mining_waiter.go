package nethermind

import "time"

type nethermindMiningWaiter struct {}

// TODO Nethermind actually doesn't do any mining!!!! How do we handle this???
func (n nethermindMiningWaiter) WaitForMining(numRetries uint32, timeBetweenRetries time.Duration) error {
	return nil
}

