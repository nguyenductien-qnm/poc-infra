package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newEcsCluster khoi tao ECS Cluster
func newEcsCluster(ctx *pulumi.Context, name, stack string, opt pulumi.ResourceOption) (*ecs.Cluster, error) {
	clusterName := fmt.Sprintf("%s-cluster", name)
	cluster, err := ecs.NewCluster(ctx, clusterName, &ecs.ClusterArgs{
		Name: pulumi.String(clusterName + "-" + stack),
		Tags: shared.Tags(clusterName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating ecs cluster: %w", err)
	}
	return cluster, nil
}

// newCloudWatchLogs khoi tao CloudWatch Log Group cho container
func newCloudWatchLogs(ctx *pulumi.Context, name, stack string, opt pulumi.ResourceOption) (*cloudwatch.LogGroup, error) {
	logGroup, err := cloudwatch.NewLogGroup(ctx, fmt.Sprintf("%s-logs", name), &cloudwatch.LogGroupArgs{
		Name:            pulumi.Sprintf("/ecs/%s-%s", name, stack),
		RetentionInDays: pulumi.Int(7),
		Tags:            shared.Tags(fmt.Sprintf("%s-logs", name), stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating cloudwatch log group: %w", err)
	}
	return logGroup, nil
}
