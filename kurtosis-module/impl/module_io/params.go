package module_io

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io/participant_log_level"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lighthouse"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lodestar"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/nimbus"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/prysm"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/teku"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/besu"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/geth"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/nethermind"
)



var validParticipantLogLevels = map[participant_log_level.ParticipantLogLevel]bool{
	participant_log_level.ParticipantLogLevel_Error: true,
	participant_log_level.ParticipantLogLevel_Warn:  true,
	participant_log_level.ParticipantLogLevel_Info:  true,
	participant_log_level.ParticipantLogLevel_Debug: true,
	participant_log_level.ParticipantLogLevel_Trace: true,
}

var elClientsParticipantLogLevels = map[ParticipantELClientType]map[participant_log_level.ParticipantLogLevel]string{
	ParticipantELClientType_Geth:       geth.VerbosityLevels,
	ParticipantELClientType_Besu:       besu.BesuLogLevels,
	ParticipantELClientType_Nethermind: nethermind.NethermindLogLevels,
}

var clClientsParticipantLogLevels = map[ParticipantCLClientType]map[participant_log_level.ParticipantLogLevel]string{
	ParticipantCLClientType_Lighthouse: lighthouse.LighthouseLogLevels,
	ParticipantCLClientType_Teku:       teku.TekuLogLevels,
	ParticipantCLClientType_Nimbus:     nimbus.NimbusLogLevels,
	ParticipantCLClientType_Prysm:      prysm.PrysmLogLevels,
	ParticipantCLClientType_Lodestar:   lodestar.LodestarLogLevels,
}

// Participant EL client type "enum"
type ParticipantELClientType string

const (
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//                        If you change these in any way, you need to:
	//               1) modify the example JSON config in the README to reflect this
	//               2) update the default_params for the type you modified
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	ParticipantELClientType_Geth       ParticipantELClientType = "geth"
	ParticipantELClientType_Nethermind ParticipantELClientType = "nethermind"
	ParticipantELClientType_Besu       ParticipantELClientType = "besu"
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//                        If you change these in any way, you need to:
	//               1) modify the example JSON config in the README to reflect this
	//               2) update the default_params for the type you modified
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
)

var validParticipantELClientTypes = map[ParticipantELClientType]bool{
	ParticipantELClientType_Geth:       true,
	ParticipantELClientType_Nethermind: true,
	ParticipantELClientType_Besu:       true,
}

// Participant CL client type "enum"
type ParticipantCLClientType string

const (
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//                        If you change these in any way, you need to:
	//               1) modify the example JSON config in the README to reflect this
	//               2) update the default_params for the type you modified
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	ParticipantCLClientType_Lighthouse ParticipantCLClientType = "lighthouse"
	ParticipantCLClientType_Teku       ParticipantCLClientType = "teku"
	ParticipantCLClientType_Nimbus     ParticipantCLClientType = "nimbus"
	ParticipantCLClientType_Prysm      ParticipantCLClientType = "prysm"
	ParticipantCLClientType_Lodestar   ParticipantCLClientType = "lodestar"
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//                        If you change these in any way, you need to:
	//               1) modify the example JSON config in the README to reflect this
	//               2) update the default_params for the type you modified
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
)

var validParticipantCLClientTypes = map[ParticipantCLClientType]bool{
	ParticipantCLClientType_Lighthouse: true,
	ParticipantCLClientType_Teku:       true,
	ParticipantCLClientType_Nimbus:     true,
	ParticipantCLClientType_Prysm:      true,
	ParticipantCLClientType_Lodestar:   true,
}

type ExecuteParams struct {
	// Parameters controlling the types of clients that compose the network
	Participants []*ParticipantParams `json:"participants"`

	// Parameters controlling the settings of the network itself
	Network *NetworkParams `json:"network"`

	// If set to false, we won't wait for the EL clients to mine at least 1 block before proceeding with adding the CL clients
	// This is purely for debug purposes; waiting for blockNumber > 0 is required for the CL network to behave as
	//  expected, but that wait can be several minutes. Skipping the wait can be a good way to shorten the debug loop on a
	//  CL client that's failing to start.
	WaitForMining bool `json:"waitForMining"`

	// If set, the module will block until a finalized epoch has occurred
	WaitForFinalization bool `json:"waitForFinalization"`

	// The log level that the started clients should log at
	ClientLogLevel participant_log_level.ParticipantLogLevel `json:"logLevel"`
}

type ParticipantParams struct {
	// The type of EL client that should be started
	ELClientType ParticipantELClientType `json:"elType"`

	// The Docker image that should be used for the EL client; leave blank to use the default
	ELClientImage string `json:"elImage"`

	// The log level that the EL client should log at
	ELClientLogLevel string `json:"elLogLevel"`

	// The type of CL client that should be started
	CLClientType ParticipantCLClientType `json:"clType"`

	// The Docker image that should be used for the EL client; leave blank to use the default
	// NOTE: Prysm is different in that it requires two images - a Beacon and a validator
	//  For Prysm and Prysm only, this field should contain a comma-separated string of "beacon_image,validator_image"
	CLClientImage string `json:"clImage"`

	// The log level that the CL client should log at
	CLClientLogLevel string `json:"clLogLevel"`
}

// Parameters controlling particulars of the Eth1 & Eth2 networks
type NetworkParams struct {
	// The network ID of the Eth1 network
	NetworkID string `json:"networkId"`

	// The address of the staking contract address on the Eth1 chain
	DepositContractAddress string `json:"depositContractAddress"`

	// Number of seconds per slot on the Beacon chain
	SecondsPerSlot uint32 `json:"secondsPerSlot"`

	// Number of slots in an epoch on the Beacon chain
	SlotsPerEpoch uint32 `json:"slotsPerEpoch"`

	// Must come before the merge fork epoch
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	AltairForkEpoch uint64 `json:"altairForkEpoch"`

	// Must occur before the total terminal difficulty is hit on the Eth1 chain
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	MergeForkEpoch uint64 `json:"mergeForkEpoch"`

	// Once the total difficulty of all mined blocks crosses this threshold, the Eth1 chain will
	//  merge with the Beacon chain
	// Must happen after the merge fork epoch on the Beacon chain
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	TotalTerminalDifficulty uint64 `json:"totalTerminalDifficulty"`

	// The number of validator keys that each CL validator node should get
	NumValidatorKeysPerNode uint32 `json:"numValidatorKeysPerNode"`

	// This menmonic will a) be used to create keystores for all the types of validators that we have and b) be used to generate a CL genesis.ssz that has the children
	//  validator keys already preregistered as validators
	PreregisteredValidatorKeysMnemonic string `json:"preregisteredValidatorKeysMnemonic"`
}
