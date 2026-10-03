package ciiam

import "encoding/json"

// Cac struct mo ta JSON IAM policy, dung json.Marshal thay vi ghep chuoi bang tay.
// Action/Resource la any vi AWS chap nhan ca chuoi lan mang; giu dung dang cua policy hien co.

type policyDocument struct {
	Version   string            `json:"Version"`
	Statement []policyStatement `json:"Statement"`
}

type policyStatement struct {
	Sid       string                    `json:"Sid,omitempty"`
	Effect    string                    `json:"Effect"`
	Principal map[string]string         `json:"Principal,omitempty"`
	Action    any                       `json:"Action"`
	Resource  any                       `json:"Resource,omitempty"`
	Condition map[string]map[string]any `json:"Condition,omitempty"`
}

// githubTrustPolicy cho phep GitHub Actions assume role qua OIDC khi aud va sub khop chinh xac (StringEquals, khong wildcard)
func githubTrustPolicy(oidcArn string, subjects []string) string {
	var sub any = subjects
	if len(subjects) == 1 {
		sub = subjects[0]
	}
	return mustJSON(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{{
			Effect:    "Allow",
			Principal: map[string]string{"Federated": oidcArn},
			Action:    "sts:AssumeRoleWithWebIdentity",
			Condition: map[string]map[string]any{
				"StringEquals": {
					githubOidcHost + ":aud": "sts.amazonaws.com",
					githubOidcHost + ":sub": sub,
				},
			},
		}},
	})
}

// previewPolicy cho role preview doc state va giai ma secret Pulumi
func previewPolicy(stateBucketArn string) string {
	return mustJSON(policyDocument{
		Version: "2012-10-17",
		Statement: []policyStatement{
			{
				Sid:      "AllowS3ReadState",
				Effect:   "Allow",
				Action:   []string{"s3:GetObject", "s3:ListBucket"},
				Resource: []string{stateBucketArn, stateBucketArn + "/*"},
			},
			{
				// TODO(#17): gioi han theo ARN key va theo moi truong
				Sid:      "AllowKmsDecryptForPreview",
				Effect:   "Allow",
				Action:   []string{"kms:Decrypt", "kms:DescribeKey"},
				Resource: "*",
			},
		},
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
