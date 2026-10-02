package app

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	Region             string
	EcrRepoUrl         pulumi.StringInput
	ImageTag           pulumi.StringInput
	SubnetIDs          pulumi.StringArrayInput
	AppSecurityGroupID pulumi.StringInput
	ClusterArn         pulumi.StringInput
	AlbTargetGroupArn  pulumi.StringInput
	AlbDnsName         pulumi.StringInput
	LogGroupName       pulumi.StringInput
	DesiredCount       pulumi.IntInput
	DbEndpoint         pulumi.StringInput
	DbSecretArn        pulumi.StringInput
}

// App la ComponentResource gom IAM Roles, Task Definition va ECS Service
type App struct {
	pulumi.ResourceState

	ServiceUrl pulumi.StringOutput
}

// New khoi tao toan bo ha tang App (IAM Roles, Task Definition & Fargate Service)
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*App, error) {
	stack := ctx.Stack()

	comp := &App{}
	if err := ctx.RegisterComponentResource("infra-poc:app:App", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering app component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. IAM Roles & Policies (iam.go)
	iamRes, err := newAppIamRoles(ctx, name, stack, args.DbSecretArn, opt)
	if err != nil {
		return nil, err
	}

	// 2. Task Definition & Fargate Service (service.go)
	if err := newFargateService(ctx, name, stack, args, iamRes, opt); err != nil {
		return nil, err
	}

	comp.ServiceUrl = pulumi.Sprintf("http://%s", args.AlbDnsName)
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"serviceUrl": comp.ServiceUrl,
	}); err != nil {
		return nil, fmt.Errorf("registering app outputs: %w", err)
	}
	return comp, nil
}
