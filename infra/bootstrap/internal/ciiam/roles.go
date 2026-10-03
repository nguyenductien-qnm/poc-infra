package ciiam

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// Quyen giu nguyen nhu script cu de import khong doi gi.
// TODO(#17): bo AdministratorAccess/ReadOnlyAccess, thay bang policy toi thieu + permissions boundary.
const (
	adminPolicyArn    = "arn:aws:iam::aws:policy/AdministratorAccess"
	readOnlyPolicyArn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
)

type envRolesResult struct {
	previewArn pulumi.StringOutput
	deployArn  pulumi.StringOutput
}

// newPreviewPolicy tao policy doc state/KMS gan cho moi role preview
func newPreviewPolicy(ctx *pulumi.Context, name string, args *Args, opt pulumi.ResourceOption) (pulumi.StringOutput, error) {
	policyName := args.NamePrefix + "-ci-preview-policy"
	importID := ""
	if args.ImportExisting {
		importID = fmt.Sprintf("arn:aws:iam::%s:policy/%s", args.AccountID, policyName)
	}

	policy, err := iam.NewPolicy(ctx, fmt.Sprintf("%s-preview-policy", name), &iam.PolicyArgs{
		Name: pulumi.String(policyName),
		Policy: args.StateBucketArn.ToStringOutput().ApplyT(func(arn string) string {
			return previewPolicy(arn)
		}).(pulumi.StringOutput),
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating ci preview policy: %w", err)
	}
	return policy.Arn, nil
}

// newEnvRoles tao role preview (tin PR + push vao nhanh) va role deploy (tin GitHub Environment) cho 1 moi truong
func newEnvRoles(
	ctx *pulumi.Context,
	name string,
	args *Args,
	env Environment,
	oidcArn, previewPolicyArn pulumi.StringOutput,
	opt pulumi.ResourceOption,
) (*envRolesResult, error) {
	// 1. Role preview: job PR chua co environment nen sub la pull_request, job push co sub ref:refs/heads/<branch>
	previewName := fmt.Sprintf("%s-%s-preview-role", args.NamePrefix, env.Name)
	preview, err := newRole(ctx, fmt.Sprintf("%s-%s-preview-role", name, env.Name), previewName, env.PreviewDescription,
		[]string{args.SubjectPrefix + ":pull_request", args.SubjectPrefix + ":ref:refs/heads/" + env.Branch},
		oidcArn, args.ImportExisting, opt)
	if err != nil {
		return nil, err
	}
	if err := attach(ctx, fmt.Sprintf("%s-%s-preview-readonly", name, env.Name), preview, previewName, pulumi.String(readOnlyPolicyArn).ToStringOutput(), readOnlyPolicyArn, args.ImportExisting, opt); err != nil {
		return nil, err
	}
	statePolicyArn := fmt.Sprintf("arn:aws:iam::%s:policy/%s-ci-preview-policy", args.AccountID, args.NamePrefix)
	if err := attach(ctx, fmt.Sprintf("%s-%s-preview-state", name, env.Name), preview, previewName, previewPolicyArn, statePolicyArn, args.ImportExisting, opt); err != nil {
		return nil, err
	}

	// 2. Role deploy: chi job chay trong GitHub Environment cung ten (co approval o staging/prod)
	deployName := fmt.Sprintf("%s-%s-deploy-role", args.NamePrefix, env.Name)
	deploy, err := newRole(ctx, fmt.Sprintf("%s-%s-deploy-role", name, env.Name), deployName, env.DeployDescription,
		[]string{args.SubjectPrefix + ":environment:" + env.Name},
		oidcArn, args.ImportExisting, opt)
	if err != nil {
		return nil, err
	}
	if err := attach(ctx, fmt.Sprintf("%s-%s-deploy-admin", name, env.Name), deploy, deployName, pulumi.String(adminPolicyArn).ToStringOutput(), adminPolicyArn, args.ImportExisting, opt); err != nil {
		return nil, err
	}

	return &envRolesResult{previewArn: preview.Arn, deployArn: deploy.Arn}, nil
}

// newRole tao IAM role tin GitHub OIDC voi danh sach sub cho truoc
func newRole(ctx *pulumi.Context, resName, roleName, description string, subjects []string, oidcArn pulumi.StringOutput, importExisting bool, opt pulumi.ResourceOption) (*iam.Role, error) {
	importID := ""
	if importExisting {
		importID = roleName
	}
	role, err := iam.NewRole(ctx, resName, &iam.RoleArgs{
		Name:        pulumi.String(roleName),
		Description: pulumi.String(description),
		AssumeRolePolicy: oidcArn.ApplyT(func(arn string) string {
			return githubTrustPolicy(arn, subjects)
		}).(pulumi.StringOutput),
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true)}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating role %s: %w", roleName, err)
	}
	return role, nil
}

// attach gan managed policy vao role. policyArnForImport la ARN dang chuoi de ghep ID import "<role>/<policy-arn>".
// Role.Name lay tu resource (khong phai chuoi roleName) de attachment luon tao sau role.
func attach(ctx *pulumi.Context, resName string, role *iam.Role, roleName string, policyArn pulumi.StringOutput, policyArnForImport string, importExisting bool, opt pulumi.ResourceOption) error {
	importID := ""
	if importExisting {
		importID = roleName + "/" + policyArnForImport
	}
	_, err := iam.NewRolePolicyAttachment(ctx, resName, &iam.RolePolicyAttachmentArgs{
		Role:      role.Name,
		PolicyArn: policyArn,
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return fmt.Errorf("attaching policy %s: %w", resName, err)
	}
	return nil
}
