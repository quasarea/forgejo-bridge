package policy

import "testing"

func TestPolicyRiskClasses(t *testing.T) {
	decision, err := Evaluate(Operation{Name: "repo.get", Risk: RiskRead}, InstancePolicy{ReadOnly: true})
	if err != nil || !decision.Allowed || decision.ConfirmationRequired {
		t.Fatalf("unexpected read decision: %#v err=%v", decision, err)
	}
	decision, err = Evaluate(Operation{Name: "pr.merge", Risk: RiskConsequential, Write: true}, InstancePolicy{})
	if err != nil || !decision.ConfirmationRequired {
		t.Fatalf("expected confirmation: %#v err=%v", decision, err)
	}
	if _, err := Evaluate(Operation{Name: "repo.delete", Risk: RiskAdministrative, Write: true}, InstancePolicy{}); err == nil {
		t.Fatal("administrative operation should be unsupported")
	}
}
