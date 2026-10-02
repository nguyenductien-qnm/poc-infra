package platform

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcID           pulumi.StringOutput
	PublicSubnetIDs pulumi.StringArrayInput
}

// Platform la ComponentResource gom ECS Cluster, Log Group, ALB va Security Group ALB/App
type Platform struct {
	pulumi.ResourceState

	ClusterArn         pulumi.StringOutput
	ClusterName        pulumi.StringOutput
	AlbDnsName         pulumi.StringOutput
	AlbTargetGroupArn  pulumi.StringOutput
	AlbSecurityGroupID pulumi.StringOutput
	AppSecurityGroupID pulumi.StringOutput
	LogGroupName       pulumi.StringOutput
}

// New khoi tao toan bo ha tang Platform (ECS Cluster, CloudWatch Logs, ALB)
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Platform, error) {
	stack := ctx.Stack()

	comp := &Platform{}
	if err := ctx.RegisterComponentResource("infra-poc:platform:Platform", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering platform component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. ECS Cluster (ecs_cluster.go)
	cluster, err := newEcsCluster(ctx, name, stack, opt)
	if err != nil {
		return nil, err
	}

	// 2. CloudWatch Log Group (ecs_cluster.go)
	logGroup, err := newCloudWatchLogs(ctx, name, stack, opt)
	if err != nil {
		return nil, err
	}

	// 3. ALB, Target Group, Security Group ALB/App & Listener (alb.go)
	albRes, err := newAlb(ctx, name, stack, args, opt)
	if err != nil {
		return nil, err
	}

	comp.ClusterArn = cluster.Arn
	comp.ClusterName = cluster.Name
	comp.AlbDnsName = albRes.albDnsName
	comp.AlbTargetGroupArn = albRes.albTargetGroupArn
	comp.AlbSecurityGroupID = albRes.albSecurityGroupID
	comp.AppSecurityGroupID = albRes.appSecurityGroupID
	comp.LogGroupName = logGroup.Name
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"clusterArn":         comp.ClusterArn,
		"albDnsName":         comp.AlbDnsName,
		"albTargetGroupArn":  comp.AlbTargetGroupArn,
		"appSecurityGroupId": comp.AppSecurityGroupID,
		"logGroupName":       comp.LogGroupName,
	}); err != nil {
		return nil, fmt.Errorf("registering platform outputs: %w", err)
	}
	return comp, nil
}
