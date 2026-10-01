package app

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcID              pulumi.IDInput
	SubnetIDs          pulumi.StringArrayInput
	ClusterArn         pulumi.StringInput
	AlbTargetGroupArn  pulumi.StringInput
	AlbSecurityGroupID pulumi.StringInput
	AlbDnsName         pulumi.StringInput
	LogGroupName       pulumi.StringInput
	DesiredCount       pulumi.IntInput
	DbEndpoint         pulumi.StringInput
	DbSecretArn        pulumi.StringInput
}

type Outputs struct {
	ServiceUrl pulumi.StringOutput
}

// New khoi tao toan bo ha tang App (IAM Roles, Task Definition & Fargate Service)
func New(ctx *pulumi.Context, name string, args *Args) (*Outputs, error) {
	stack := ctx.Stack()
	vpcIdPtr := args.VpcID.ToIDOutput().ToStringOutput().ToStringPtrOutput()

	// 1. IAM Roles & Policies (iam.go)
	iamRes, err := newAppIamRoles(ctx, name, stack, args.DbSecretArn)
	if err != nil {
		return nil, err
	}

	// 2. Security Group, Task Definition & Fargate Service (service.go)
	err = newFargateService(
		ctx, name, stack,
		vpcIdPtr, args.AlbSecurityGroupID,
		args.SubnetIDs, args.ClusterArn,
		args.AlbTargetGroupArn, args.DesiredCount,
		args.DbEndpoint, args.DbSecretArn,
		args.LogGroupName, iamRes,
	)
	if err != nil {
		return nil, err
	}

	return &Outputs{
		ServiceUrl: pulumi.Sprintf("http://%s", args.AlbDnsName),
	}, nil
}
