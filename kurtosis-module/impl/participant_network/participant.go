package participant_network

// NOTE: We saw 1 Geth node + 3 Teku nodes causing problems, and the Teku folks
//  let us know that generally each CL node should be paired with 1 EL node
// https://discord.com/channels/697535391594446898/697539289042649190/922266717667856424
// We use this Participant class to represent a participant in the network who is running 1 of each type of client
type Participant struct {
	elClientType ParticipantELClientType
	clClientType ParticipantCLClientType

	// TODO EL & CL client contexts
}