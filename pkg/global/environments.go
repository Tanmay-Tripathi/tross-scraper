package global

// Environment enumerates the deployment environments the service can run in.
type Environment string

const (
	LocalEnv Environment = "local"
	StageEnv Environment = "stg"
	UatEnv   Environment = "uat"
	ProdEnv  Environment = "prd"
)

// IsValid reports whether e is one of the known environments.
func (e Environment) IsValid() bool {
	switch e {
	case LocalEnv, StageEnv, UatEnv, ProdEnv:
		return true
	default:
		return false
	}
}

// IsProdLike reports whether e is an environment that should run with
// production defaults (release-mode router, JSON logs, no debug output).
func (e Environment) IsProdLike() bool {
	return e == StageEnv || e == UatEnv || e == ProdEnv
}
