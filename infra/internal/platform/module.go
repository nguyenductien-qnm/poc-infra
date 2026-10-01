package platform

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcID           pulumi.StringInput
	PublicSubnetIDs pulumi.StringArrayInput
	ProtectStateful bool
}

type Outputs struct {
	ClusterArn         pulumi.StringOutput
	ClusterName        pulumi.StringOutput
	AlbDnsName         pulumi.StringOutput
	AlbTargetGroupArn  pulumi.StringOutput
	AlbSecurityGroupID pulumi.StringOutput
	EcrRepoUrl         pulumi.StringOutput
	LogGroupName       pulumi.StringOutput
}

// New khoi tao toan bo ha tang Platform (ECS Cluster, ECR, CloudWatch Logs, ALB)
func New(ctx *pulumi.Context, name string, args *Args) (*Outputs, error) {
	stack := ctx.Stack()
	vpcIdPtr := args.VpcID.ToStringOutput().ToStringPtrOutput()

	// 1. ECS Cluster (ecs_cluster.go)
	cluster, err := newEcsCluster(ctx, name, stack)
	if err != nil {
		return nil, err
	}

	// 2. ECR Repository (ecr.go)
	ecrRepo, err := newEcrRepo(ctx, name, stack, args.ProtectStateful)
	if err != nil {
		return nil, err
	}

	// 3. CloudWatch Log Group (ecs_cluster.go)
	logGroup, err := newCloudWatchLogs(ctx, name, stack)
	if err != nil {
		return nil, err
	}

	// 4. ALB, Target Group, Security Group & Listener (alb.go)
	albRes, err := newAlb(ctx, name, stack, vpcIdPtr, args.PublicSubnetIDs)
	if err != nil {
		return nil, err
	}

	return &Outputs{
		ClusterArn:         cluster.Arn,
		ClusterName:        cluster.Name,
		AlbDnsName:         albRes.albDnsName,
		AlbTargetGroupArn:  albRes.albTargetGroupArn,
		AlbSecurityGroupID: albRes.albSecurityGroupID,
		EcrRepoUrl:         ecrRepo.RepositoryUrl,
		LogGroupName:       logGroup.Name,
	}, nil
}
