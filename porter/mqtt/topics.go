package mqtt

const RootLevel string = "door_controller"

const (
	AccessListLevel   = "access_list"
	CheckInLevel      = "check_in"
	HealthCheckLevel  = "health_check"
	UnlockLevel       = "unlock"
	LockLevel         = "lock"
	DeniedAccessLevel = "denied_access"
	LogInfoLevel      = "log_info"
	LogWarnLevel      = "log_warn"
	LogFatalLevel     = "log_fatal"
)

const AccessListTopic = RootLevel + "/" + AccessListLevel
const CheckInTopic = RootLevel + "/" + CheckInLevel
const HealthCheckTopic = RootLevel + "/" + HealthCheckLevel
const UnlockTopic = RootLevel + "/" + UnlockLevel
const LockTopic = RootLevel + "/" + LockLevel
const DeniedAccessTopic = RootLevel + "/" + DeniedAccessLevel
const LogInfoTopic = RootLevel + "/" + LogInfoLevel
const LogWarnTopic = RootLevel + "/" + LogWarnLevel
const LogFatalTopic = RootLevel + "/" + LogFatalLevel
