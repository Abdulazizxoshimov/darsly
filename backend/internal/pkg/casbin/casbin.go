package casbin

import (
	"github.com/casbin/casbin/v2"
)

// NewEnforcer loads the RBAC model and policy from disk and returns a ready enforcer.
func NewEnforcer(modelPath, policyPath string) (*casbin.Enforcer, error) {
	return casbin.NewEnforcer(modelPath, policyPath)
}
