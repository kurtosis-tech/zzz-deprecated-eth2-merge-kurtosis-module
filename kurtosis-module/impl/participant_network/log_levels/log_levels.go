package log_levels

type ParticipantLogLevel string

const (
	ParticipantLogLevel_Error ParticipantLogLevel = "error"
	ParticipantLogLevel_Warn  ParticipantLogLevel = "warn"
	ParticipantLogLevel_Info  ParticipantLogLevel = "info"
	ParticipantLogLevel_Debug ParticipantLogLevel = "debug"
)
// "Set" of the allowed client log levels
var ValidParticipantLogLevels = map[ParticipantLogLevel]bool{
	ParticipantLogLevel_Error: true,
	ParticipantLogLevel_Warn:  true,
	ParticipantLogLevel_Info:  true,
	ParticipantLogLevel_Debug: true,
}



