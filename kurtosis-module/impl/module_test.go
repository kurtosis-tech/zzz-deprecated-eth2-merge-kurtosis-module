package impl

import (
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	// Eth2 requires at least 64 validators, because:
	// - There are 32 slots per epoch
	// - Validators are chosen in advance for the next epoch, before it arrives
	// - There must be enough unique validators for this epoch and the next
	minNumRequiredValidators = 64

	minNumParticipants = 1
)

func TestSomeParticipants(t *testing.T) {
	require.GreaterOrEqual(t, numParticipants, minNumParticipants)
}

func TestMinimumRequiredValidators(t *testing.T) {
	require.GreaterOrEqual(t, numValidatorsToPreregister, minNumRequiredValidators)
}
