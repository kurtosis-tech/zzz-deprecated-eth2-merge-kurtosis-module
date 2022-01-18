package module_io

// Participant log level "enum"
type ParticipantLogLevel string
const (
	ParticipantLogLevel_Error ParticipantLogLevel = "error"
	ParticipantLogLevel_Warn ParticipantLogLevel = "warn"
	ParticipantLogLevel_Info  ParticipantLogLevel = "info"
	ParticipantLogLevel_Debug ParticipantLogLevel = "debug"
)
var validParticipantLogLevels = map[ParticipantLogLevel]bool{
	ParticipantLogLevel_Error: true,
	ParticipantLogLevel_Warn:  true,
	ParticipantLogLevel_Info:  true,
	ParticipantLogLevel_Debug: true,
}

// Participant EL client type "enum"
type ParticipantELClientType string
const (
	ParticipantELClientType_Geth       ParticipantELClientType = "geth"
	ParticipantELClientType_Nethermind ParticipantELClientType = "nethermind"
)
var validParticipantELClientTypes = map[ParticipantELClientType]bool{
	ParticipantELClientType_Geth:       true,
	ParticipantELClientType_Nethermind: true,
}

// Participant CL client type "enum"
type ParticipantCLClientType string
const (
	ParticipantCLClientType_Lighthouse ParticipantCLClientType = "lighthouse"
	ParticipantCLClientType_Teku       ParticipantCLClientType = "teku"
	ParticipantCLClientType_Nimbus     ParticipantCLClientType = "nimbus"
	ParticipantCLClientType_Prysm      ParticipantCLClientType = "prysm"
	ParticipantCLClientType_Lodestar   ParticipantCLClientType = "lodestar"
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
	Network *NetworkParams	`json:"network"`

	// If set, the module will block until a finalized epoch has occurred
	WaitForFinalization bool	`json:"waitForFinalization"`

	// The log level that the started clients should log at
	ClientLogLevel ParticipantLogLevel `json:"logLevel"`
}

type ParticipantParams struct {
	ELClientType ParticipantELClientType `json:"el"`
	CLClientType ParticipantCLClientType `json:"cl"`
}

// Parameters controlling particulars of the Eth1 & Eth2 networks
type NetworkParams struct {
	// The network ID of the Eth1 network
	NetworkID string	`json:"networkId"`

	// The address of the staking contract address on the Eth1 chain
	DepositContractAddress string	`json:"depositContractAddress"`

	// Number of seconds per slot on the Beacon chain
	SecondsPerSlot uint32	`json:"secondsPerSlot"`

	// Number of slots in an epoch on the Beacon chain
	SlotsPerEpoch uint32	`json:"slotsPerEpoch"`

	// Must come before the merge fork epoch
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	AltairForkEpoch uint64	`json:"altairForkEpoch"`

	// Must occur before the total terminal difficulty is hit on the Eth1 chain
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	MergeForkEpoch uint64	`json:"mergeForkEpoch"`

	// Once the total difficulty of all mined blocks crosses this threshold, the Eth1 chain will
	//  merge with the Beacon chain
	// Must happen after the merge fork epoch on the Beacon chain
	// See https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	TotalTerminalDifficulty uint64	`json:"totalTerminalDifficulty"`

	// The number of validator keys that each CL validator node should get
	NumValidatorKeysPerNode uint32	`json:"numValidatorKeysPerNode"`

	// This menmonic will a) be used to create keystores for all the types of validators that we have and b) be used to generate a CL genesis.ssz that has the children
	//  validator keys already preregistered as validators
	// preregisteredValidatorKeysMnemonic = "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete"
	PreregisteredValidatorKeysMnemonic string	`json:"preregisteredValidatorKeysMnemonic"`
}
