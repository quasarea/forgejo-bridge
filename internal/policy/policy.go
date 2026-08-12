package policy

import "github.com/quasarea/forgejo-bridge/internal/contracts"

type Risk string

const (
	RiskRead           Risk = "R0"
	RiskReversible     Risk = "R1"
	RiskConsequential  Risk = "R2"
	RiskAdministrative Risk = "R3"
)

type Decision struct {
	Allowed              bool
	ConfirmationRequired bool
	Risk                 Risk
	Reason               string
}

type Operation struct {
	Name       string
	Risk       Risk
	Write      bool
	Repository string
}

type InstancePolicy struct {
	ReadOnly bool
}

func Evaluate(operation Operation, instance InstancePolicy) (Decision, *contracts.BridgeError) {
	if operation.Write && instance.ReadOnly {
		return Decision{}, contracts.NewError("permission_denied", "instance policy is read-only")
	}
	if operation.Risk == RiskAdministrative {
		return Decision{}, contracts.NewError("capability_unsupported", "administrative operations are outside the MVP")
	}
	return Decision{
		Allowed:              true,
		ConfirmationRequired: operation.Risk == RiskConsequential,
		Risk:                 operation.Risk,
	}, nil
}
