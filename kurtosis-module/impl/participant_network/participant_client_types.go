package participant_network

// Represents teh type of consensus layer client a participant is running
type ParticipantELClientType int
const (
	ParticipantELClientType_Geth ParticipantELClientType = iota
	ParticipantELClientType_Nethermind
)

// Represents the type of consensus layer client(s) a participant is running
// This could be "clients" because some types (like Lighthouse) actually split the Beacon and validator
//  clients into two services
type ParticipantCLClientType int
const (
	ParticipantCLClientType_Lighthouse ParticipantCLClientType = iota
	ParticipantCLClientType_Teku
	ParticipantCLClientType_Nimbus
	ParticipantCLClientType_Prysm
	ParticipantCLClientType_Lodestar
)
