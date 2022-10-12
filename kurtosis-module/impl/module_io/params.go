package module_io

// GlobalClient log level "enum"
type GlobalClientLogLevel string

const (
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	GlobalClientLogLevel_Error GlobalClientLogLevel = "error"
	GlobalClientLogLevel_Warn  GlobalClientLogLevel = "warn"
	GlobalClientLogLevel_Info  GlobalClientLogLevel = "info"
	GlobalClientLogLevel_Debug GlobalClientLogLevel = "debug"
	GlobalClientLogLevel_Trace GlobalClientLogLevel = "trace"
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
)

var validGlobalClientLogLevels = map[GlobalClientLogLevel]bool{
	GlobalClientLogLevel_Error: true,
	GlobalClientLogLevel_Warn:  true,
	GlobalClientLogLevel_Info:  true,
	GlobalClientLogLevel_Debug: true,
	GlobalClientLogLevel_Trace: true,
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
	ParticipantELClientType_Erigon     ParticipantELClientType = "erigon"
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
	ParticipantELClientType_Erigon:     true,
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
	Participants []*ParticipantParams `yaml:"participants"`

	// Parameters controlling the settings of the network itself
	Network *NetworkParams `yaml:"network"`

	// True by defaults such that in addition to the Ethereum network:
	//  - A transaction spammer is launched to fake transactions sent to the network
	//  - Forkmon will be launched after CL genesis has happened
	//  - a prometheus will be started, coupled with grafana
	// If set to false:
	//  - only Ethereum network (EL and CL nodes) will be launched. Nothing else (no transaction spammer)
	//  - params for the CL nodes will be ignored (e.g. CL node image, CL node extra params)
	// This is a hack - it's not very elegant - but this is a commonly-requested feature
	// The longterm solution is making the module trivial to decompose so we don't need flags like this; we're working
	//  on this at the Kurtosis product level
	LaunchAdditionalServices bool `yaml:"launchAdditionalServices"`

	// The log level that the started clients should log at
	ClientLogLevel GlobalClientLogLevel `yaml:"logLevel"`
}

// Parameters for clients to connect to a network of external block builders
type BuilderNetworkParams struct {
	// A list of endpoints to reach block builder relays
	RelayEndpoints []string `yaml:"relayEndpoints"`
}

type ParticipantParams struct {
	// The type of EL client that should be started
	ELClientType ParticipantELClientType `yaml:"elType"`

	// The Docker image that should be used for the EL client; leave blank to use the default
	ELClientImage string `yaml:"elImage"`

	// The log level string that this participant's EL client should log at
	// If this is emptystring then the global `logLevel` parameter's value will be translated into a string appropriate for the client (e.g. if
	//  global `logLevel` = `info` then Geth would receive `3`, Besu would receive `INFO`, etc.)
	// If this is not emptystring, then this value will override the global `logLevel` setting to allow for fine-grained control
	//  over a specific participant's logging
	ELClientLogLevel string `yaml:"elLogLevel"`

	// Optional extra parameters that will be passed to the EL client
	ELExtraParams []string `yaml:"elExtraParams"`

	// The type of CL client that should be started
	CLClientType ParticipantCLClientType `yaml:"clType"`

	// The Docker image that should be used for the EL client; leave blank to use the default
	// NOTE: Prysm is different in that it requires two images - a Beacon and a validator
	//  For Prysm and Prysm only, this field should contain a comma-separated string of "beacon_image,validator_image"
	CLClientImage string `yaml:"clImage"`

	// The log level string that this participant's CL client should log at
	// If this is emptystring then the global `logLevel` parameter's value will be translated into a string appropriate for the client (e.g. if
	//  global `logLevel` = `info` then Nimbus would receive `INFO`, Prysm would receive `info`, etc.)
	// If this is not emptystring, then this value will override the global `logLevel` setting to allow for fine-grained control
	//  over a specific participant's logging
	CLClientLogLevel string `yaml:"clLogLevel"`

	// Extra parameters that will be passed to the Beacon container (if a separate one exists), or to the combined node if
	// the Beacon and validator are combined
	BeaconExtraParams []string `yaml:"beaconExtraParams"`

	// Extra parameters that will be passed to the validator container (if a separate one exists), or to the combined node if
	// the Beacon and validator are combined
	ValidatorExtraParams []string `yaml:"validatorExtraParams"`

	BuilderNetworkParams *BuilderNetworkParams `yaml:"builderNetworkParams"`
}

// Parameters controlling particulars of the Eth1 & Eth2 networks
type NetworkParams struct {
	// The network ID of the Eth1 network
	NetworkID string `yaml:"networkId"`

	// The address of the staking contract address on the Eth1 chain
	DepositContractAddress string `yaml:"depositContractAddress"`

	// Number of seconds per slot on the Beacon chain
	SecondsPerSlot uint32 `yaml:"secondsPerSlot"`

	// Number of slots in an epoch on the Beacon chain
	SlotsPerEpoch uint32 `yaml:"slotsPerEpoch"`

	// The number of validator keys that each CL validator node should get
	NumValidatorKeysPerNode uint32 `yaml:"numValidatorKeysPerNode"`

	// This menmonic will a) be used to create keystores for all the types of validators that we have and b) be used to generate a CL genesis.ssz that has the children
	//  validator keys already preregistered as validators
	PreregisteredValidatorKeysMnemonic string `yaml:"preregisteredValidatorKeysMnemonic"`
}
