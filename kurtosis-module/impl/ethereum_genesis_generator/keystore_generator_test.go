package ethereum_genesis_generator

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenerateKeyStartAndStopIndices(t *testing.T) {
	startIndices, stopIndices := generateKeyStartAndStopIndices(10, 3)
	require.Equal(t, []uint32{0, 4, 7}, startIndices)
	require.Equal(t, []uint32{4, 7, 10}, stopIndices)
}
