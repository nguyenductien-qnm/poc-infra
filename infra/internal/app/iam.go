package app

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

type appIamResult struct {
	execRole *iam.Role
	taskRole *iam.Role
}

// newAppIamRoles tao ECS Task Execution Role va Task Role
func newAppIamRoles(ctx *pulumi.Context, name, stack string, dbSecretArn pulumi.StringInput, opt pulumi.ResourceOption) (*appIamResult, error) {
	ecsTasksTrust := pulumi.String(assumeRolePolicy("ecs-tasks.amazonaws.com"))

	// 1. ECS Task Execution Role (keo image, ghi log CloudWatch, doc secrets)
	execRoleName := fmt.Sprintf("%s-exec-role", name)
	execRole, err := iam.NewRole(ctx, execRoleName, &iam.RoleArgs{
		Name:             pulumi.String(execRoleName + "-" + stack),
		AssumeRolePolicy: ecsTasksTrust,
		Tags:             shared.Tags(execRoleName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating task exec role: %w", err)
	}

	// Gan policy chuan cua AWS cho Task Execution Role
	_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-exec-policy-attach", name), &iam.RolePolicyAttachmentArgs{
		Role:      execRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("attaching exec policy: %w", err)
	}

	// Cap quyen doc Secret tu Secrets Manager cho Execution Role
	_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("%s-exec-secret-policy", name), &iam.RolePolicyArgs{
		Role: execRole.Name,
		Policy: dbSecretArn.ToStringOutput().ApplyT(func(arn string) string {
			return allowPolicy(arn, "secretsmanager:GetSecretValue")
		}).(pulumi.StringOutput),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("attaching secret policy to exec role: %w", err)
	}

	// 2. ECS Task Role (danh rieng cho code chay ben trong container)
	taskRoleName := fmt.Sprintf("%s-task-role", name)
	taskRole, err := iam.NewRole(ctx, taskRoleName, &iam.RoleArgs{
		Name:             pulumi.String(taskRoleName + "-" + stack),
		AssumeRolePolicy: ecsTasksTrust,
		Tags:             shared.Tags(taskRoleName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating task role: %w", err)
	}

	// Task role co policy bedrock:InvokeModel
	_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("%s-bedrock-policy", name), &iam.RolePolicyArgs{
		Role:   taskRole.Name,
		Policy: pulumi.String(allowPolicy("*", "bedrock:InvokeModel")),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("attaching bedrock policy: %w", err)
	}

	return &appIamResult{
		execRole: execRole,
		taskRole: taskRole,
	}, nil
}
