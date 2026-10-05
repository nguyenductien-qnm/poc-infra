// Package ciiam quan ly danh tinh CI: GitHub OIDC provider, role preview/deploy theo moi truong va role chay chinh bootstrap.
// Thay cho scripts/setup-ci-roles.sh cu: moi thay doi quyen CI di qua PR + preview.
package ciiam

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Environment mo ta 1 moi truong CI: role preview tin PR + push vao Branch, role deploy tin GitHub Environment Name
type Environment struct {
	Name               string `json:"name"`
	Branch             string `json:"branch"`
	PreviewDescription string `json:"previewDescription"`
	DeployDescription  string `json:"deployDescription"`
}

type Args struct {
	AccountID       string // ghep ARN de import OIDC provider/policy
	NamePrefix      string // tien to ten role/policy, vd "poc"
	SubjectPrefix   string // repo:<owner>@<owner_id>/<repo>@<repo_id> (OIDC sub dang immutable)
	OidcThumbprints []string
	StateBucketArn  pulumi.StringInput
	Environments    []Environment
	ImportExisting  bool
}

// CiIam la ComponentResource gom OIDC provider va cac role CI
type CiIam struct {
	pulumi.ResourceState

	OidcProviderArn pulumi.StringOutput
	PreviewRoleArns pulumi.StringMap // key: ten moi truong
	DeployRoleArns  pulumi.StringMap

	BootstrapRoleArn pulumi.StringOutput // role workflow bootstrap assume
}

// New khoi tao (hoac tiep nhan) OIDC provider, policy preview va role preview/deploy cho tung moi truong
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*CiIam, error) {
	comp := &CiIam{}
	if err := ctx.RegisterComponentResource("infra-poc:ciiam:CiIam", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering ciiam component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. GitHub OIDC provider (oidc.go)
	oidcArn, err := newOidcProvider(ctx, name, args, opt)
	if err != nil {
		return nil, err
	}

	// 2. Policy doc state/KMS dung chung cho role preview (roles.go)
	previewPolicyArn, err := newPreviewPolicy(ctx, name, args, opt)
	if err != nil {
		return nil, err
	}

	// 3. Role preview + deploy cho tung moi truong (roles.go)
	comp.PreviewRoleArns = pulumi.StringMap{}
	comp.DeployRoleArns = pulumi.StringMap{}
	for _, env := range args.Environments {
		res, err := newEnvRoles(ctx, name, args, env, oidcArn, previewPolicyArn, opt)
		if err != nil {
			return nil, err
		}
		comp.PreviewRoleArns[env.Name] = res.previewArn
		comp.DeployRoleArns[env.Name] = res.deployArn
	}

	// 4. Role cho workflow chay chinh bootstrap (roles.go)
	comp.BootstrapRoleArn, err = newBootstrapRole(ctx, name, args, oidcArn, opt)
	if err != nil {
		return nil, err
	}

	comp.OidcProviderArn = oidcArn
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"oidcProviderArn":  comp.OidcProviderArn,
		"previewRoleArns":  comp.PreviewRoleArns,
		"deployRoleArns":   comp.DeployRoleArns,
		"bootstrapRoleArn": comp.BootstrapRoleArn,
	}); err != nil {
		return nil, fmt.Errorf("registering ciiam outputs: %w", err)
	}
	return comp, nil
}
