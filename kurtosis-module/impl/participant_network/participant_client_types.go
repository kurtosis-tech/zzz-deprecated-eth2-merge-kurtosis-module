package participant_network

// Represents teh type of consensus layer client a participant is running
type ParticipantELClientType string
const (
	ParticipantELClientType_Geth ParticipantELClientType = "geth"
	ParticipantELClientType_Nethermind ParticipantELClientType = "nethermind"
)
var ValidParticipantELClientTypes = map[ParticipantELClientType]bool{
	ParticipantELClientType_Geth: true,
	ParticipantELClientType_Nethermind: true,
}

// Represents the type of consensus layer client(s) a participant is running
// This could be "clients" because some types (like Lighthouse) actually split the Beacon and validator
//  clients into two services
type ParticipantCLClientType string
const (
	ParticipantCLClientType_Lighthouse ParticipantCLClientType = "lighthouse"
	ParticipantCLClientType_Teku ParticipantCLClientType = "teku"
	ParticipantCLClientType_Nimbus ParticipantCLClientType = "nimbus"
	ParticipantCLClientType_Prysm ParticipantCLClientType = "prysm"
	ParticipantCLClientType_Lodestar ParticipantCLClientType = "lodestar"
)
var ValidParticipantCLClientTypes = map[ParticipantCLClientType]bool{
	ParticipantCLClientType_Lighthouse: true,
	ParticipantCLClientType_Teku: true,
	ParticipantCLClientType_Nimbus: true,
	ParticipantCLClientType_Prysm: true,
	ParticipantCLClientType_Lodestar: true,
}
