package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newEcsCluster khoi tao ECS Cluster
func newEcsCluster(ctx *pulumi.Context, name, stack string) (*ecs.Cluster, error) {
	cluster, err := ecs.NewCluster(ctx, fmt.Sprintf("%s-cluster", name), &ecs.ClusterArgs{
		Name: pulumi.Sprintf("%s-cluster-%s", name, stack),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-cluster-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating ecs cluster: %w", err)
	}
	return cluster, nil
}

// newCloudWatchLogs khoi tao CloudWatch Log Group cho container
func newCloudWatchLogs(ctx *pulumi.Context, name, stack string) (*cloudwatch.LogGroup, error) {
	logGroup, err := cloudwatch.NewLogGroup(ctx, fmt.Sprintf("%s-logs", name), &cloudwatch.LogGroupArgs{
		Name:            pulumi.Sprintf("/ecs/%s-%s", name, stack),
		RetentionInDays: pulumi.Int(7),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-logs-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating cloudwatch log group: %w", err)
	}
	return logGroup, nil
}
