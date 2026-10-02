package app

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

const containerName = "app"

// Cac struct mo ta JSON container definition cua ECS (chi cac field dang dung)
type containerDefinition struct {
	Name             string           `json:"name"`
	Image            string           `json:"image"`
	Essential        bool             `json:"essential"`
	PortMappings     []portMapping    `json:"portMappings"`
	Environment      []keyValue       `json:"environment"`
	Secrets          []secretRef      `json:"secrets"`
	LogConfiguration logConfiguration `json:"logConfiguration"`
}

type portMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	Protocol      string `json:"protocol"`
}

type keyValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type secretRef struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"`
}

type logConfiguration struct {
	LogDriver string            `json:"logDriver"`
	Options   map[string]string `json:"options"`
}

// newFargateService khoi tao Task Definition va ECS Service (App SG tao o package platform)
func newFargateService(ctx *pulumi.Context, name, stack string, args *Args, iamRes *appIamResult, opt pulumi.ResourceOption) error {
	// 1. ECS Task Definition (Fargate 0.25 vCPU, 512 MB RAM)
	containerDefs := pulumi.All(args.EcrRepoUrl, args.ImageTag, args.DbEndpoint, args.DbSecretArn, args.LogGroupName).
		ApplyT(func(v []any) string {
			repoUrl, imageTag, dbEndpoint, dbSecretArn, logGroup := v[0].(string), v[1].(string), v[2].(string), v[3].(string), v[4].(string)
			return mustJSON([]containerDefinition{{
				Name:      containerName,
				Image:     repoUrl + ":" + imageTag,
				Essential: true,
				PortMappings: []portMapping{
					{ContainerPort: shared.AppPort, HostPort: shared.AppPort, Protocol: "tcp"},
				},
				Environment: []keyValue{
					{Name: "DB_ENDPOINT", Value: dbEndpoint},
					{Name: "DB_PORT", Value: strconv.Itoa(shared.DbPort)},
					{Name: "DB_NAME", Value: shared.DbName},
					{Name: "DB_USER", Value: shared.DbUser},
					{Name: "PORT", Value: strconv.Itoa(shared.AppPort)},
				},
				Secrets: []secretRef{
					// Secret RDS dang JSON {"username","password"}, chi lay key password
					{Name: "DB_PASSWORD", ValueFrom: dbSecretArn + ":password::"},
				},
				LogConfiguration: logConfiguration{
					LogDriver: "awslogs",
					Options: map[string]string{
						"awslogs-group":         logGroup,
						"awslogs-region":        args.Region,
						"awslogs-stream-prefix": containerName,
					},
				},
			}})
		}).(pulumi.StringOutput)

	taskDef, err := ecs.NewTaskDefinition(ctx, fmt.Sprintf("%s-taskdef", name), &ecs.TaskDefinitionArgs{
		Family:                  pulumi.String(name + "-" + stack),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		NetworkMode:             pulumi.String("awsvpc"),
		Cpu:                     pulumi.String("256"),
		Memory:                  pulumi.String("512"),
		ExecutionRoleArn:        iamRes.execRole.Arn,
		TaskRoleArn:             iamRes.taskRole.Arn,
		ContainerDefinitions:    containerDefs,
		Tags:                    shared.Tags(fmt.Sprintf("%s-taskdef", name), stack),
	}, opt)
	if err != nil {
		return fmt.Errorf("creating task definition: %w", err)
	}

	// 2. ECS Service Fargate (chay tren public subnets, gan vao ALB Target Group)
	_, err = ecs.NewService(ctx, fmt.Sprintf("%s-service", name), &ecs.ServiceArgs{
		Cluster:        args.ClusterArn,
		TaskDefinition: taskDef.Arn,
		LaunchType:     pulumi.String("FARGATE"),
		DesiredCount:   args.DesiredCount,
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        args.SubnetIDs,
			SecurityGroups: pulumi.StringArray{args.AppSecurityGroupID},
			AssignPublicIp: pulumi.Bool(true),
		},
		LoadBalancers: ecs.ServiceLoadBalancerArray{
			&ecs.ServiceLoadBalancerArgs{
				TargetGroupArn: args.AlbTargetGroupArn,
				ContainerName:  pulumi.String(containerName),
				ContainerPort:  pulumi.Int(shared.AppPort),
			},
		},
		Tags: shared.Tags(fmt.Sprintf("%s-svc", name), stack),
	}, opt)
	if err != nil {
		return fmt.Errorf("creating ecs service: %w", err)
	}

	return nil
}
