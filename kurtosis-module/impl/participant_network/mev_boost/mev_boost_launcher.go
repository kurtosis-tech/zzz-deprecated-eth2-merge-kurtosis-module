package mev_boost

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"

type MEVBoostLauncher struct {
	// TODO Analogous to GethLauncher, BesuLauncher, NimbusLauncher, etc.
}

/*
KEVIN NOTE: You don't need this to be an interface yet because you only have one mev-boost node (the flashbots one),
but if you got multiple in the future you'd probably want a MEVBoostLauncher interface with a Launch function, and
then have implementations of it
 */
func (launcher *MEVBoostLauncher) Launch(enclaveCtx enclaves.EnclaveContext) (*MEVBoostContext, error) {
	// TODO A function for launching a MEV Boost node in the same way that GethLauncher
	//  In the future, this will probably take in relay information as well
	panic("TODO IMPLEMENT ME")
}
