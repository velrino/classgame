package ratelimit

import "context"

type Rule struct {
	Identifier string
	Limiter    Limiter
}

type PolicyDecision struct {
	Decision  Decision
	Evaluated bool
}

func Evaluate(ctx context.Context, rules ...Rule) (PolicyDecision, error) {
	result := PolicyDecision{}
	for _, rule := range rules {
		if rule.Limiter == nil {
			continue
		}

		decision, err := rule.Limiter.Allow(ctx, rule.Identifier)
		if err != nil {
			return PolicyDecision{}, err
		}
		if !result.Evaluated {
			result = PolicyDecision{Decision: decision, Evaluated: true}
		}
		if !decision.Allowed {
			return PolicyDecision{Decision: decision, Evaluated: true}, nil
		}
	}

	return result, nil
}
