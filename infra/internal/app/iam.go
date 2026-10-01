package app

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type appIamResult struct {
	execRole *iam.Role
	taskRole *iam.Role
}

// newAppIamRoles tao ECS Task Execution Role va Task Role
func newAppIamRoles(ctx *pulumi.Context, name, stack string, dbSecretArn pulumi.StringInput) (*appIamResult, error) {
	// IAM Assume Role Policy chung cho ECS Tasks
	ecsTasksAssumeRolePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Action": "sts:AssumeRole",
				"Principal": {
					"Service": "ecs-tasks.amazonaws.com"
				},
				"Effect": "Allow"
			}
		]
	}`

	// 1. ECS Task Execution Role (keo image, ghi log CloudWatch, doc secrets)
	execRole, err := iam.NewRole(ctx, fmt.Sprintf("%s-exec-role", name), &iam.RoleArgs{
		Name:             pulumi.Sprintf("%s-exec-role-%s", name, stack),
		AssumeRolePolicy: pulumi.String(ecsTasksAssumeRolePolicy),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-exec-role-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating task exec role: %w", err)
	}

	// Gan policy chuan cua AWS cho Task Execution Role
	_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("%s-exec-policy-attach", name), &iam.RolePolicyAttachmentArgs{
		Role:      execRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	})
	if err != nil {
		return nil, fmt.Errorf("attaching exec policy: %w", err)
	}

	// Cap quyen doc Secret tu Secrets Manager cho Execution Role
	_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("%s-exec-secret-policy", name), &iam.RolePolicyArgs{
		Role: execRole.Name,
		Policy: pulumi.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"secretsmanager:GetSecretValue"
					],
					"Resource": "%s"
				}
			]
		}`, dbSecretArn),
	})
	if err != nil {
		return nil, fmt.Errorf("attaching secret policy to exec role: %w", err)
	}

	// 2. ECS Task Role (danh rieng cho code chay ben trong container)
	taskRole, err := iam.NewRole(ctx, fmt.Sprintf("%s-task-role", name), &iam.RoleArgs{
		Name:             pulumi.Sprintf("%s-task-role-%s", name, stack),
		AssumeRolePolicy: pulumi.String(ecsTasksAssumeRolePolicy),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-task-role-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating task role: %w", err)
	}

	// Theo PLAN.md: Task role co policy bedrock:InvokeModel
	_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("%s-bedrock-policy", name), &iam.RolePolicyArgs{
		Role: taskRole.Name,
		Policy: pulumi.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"bedrock:InvokeModel"
					],
					"Resource": "*"
				}
			]
		}`),
	})
	if err != nil {
		return nil, fmt.Errorf("attaching bedrock policy: %w", err)
	}

	return &appIamResult{
		execRole: execRole,
		taskRole: taskRole,
	}, nil
}
