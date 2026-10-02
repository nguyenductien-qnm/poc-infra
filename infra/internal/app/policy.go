package app

import "encoding/json"

// Cac struct mo ta JSON IAM policy, dung json.Marshal thay vi ghep chuoi bang tay.

type policyDocument struct {
	Version   string            `json:"Version"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Effect    string            `json:"Effect"`
	Action    []string          `json:"Action"`
	Resource  string            `json:"Resource,omitempty"`
	Principal map[string]string `json:"Principal,omitempty"`
}

// allowPolicy tao policy document 1 statement Allow actions tren resource
func allowPolicy(resource string, actions ...string) string {
	return mustJSON(policyDocument{
		Version:   "2012-10-17",
		Statement: []policyStatement{{Effect: "Allow", Action: actions, Resource: resource}},
	})
}

// assumeRolePolicy tao trust policy cho phep service (vd ecs-tasks.amazonaws.com) assume role
func assumeRolePolicy(service string) string {
	return mustJSON(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{{
			Effect:    "Allow",
			Action:    []string{"sts:AssumeRole"},
			Principal: map[string]string{"Service": service},
		}},
	})
}

// mustJSON chi dung cho struct noi bo, Marshal khong the loi
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
