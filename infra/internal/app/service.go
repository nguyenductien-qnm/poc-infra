package app

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newFargateService khoi tao App Security Group, Task Definition va ECS Service
func newFargateService(
	ctx *pulumi.Context,
	name, stack string,
	vpcIdPtr pulumi.StringPtrInput,
	albSecurityGroupID pulumi.StringInput,
	subnetIDs pulumi.StringArrayInput,
	clusterArn pulumi.StringInput,
	albTargetGroupArn pulumi.StringInput,
	desiredCount pulumi.IntInput,
	ecrRepoUrl pulumi.StringInput,
	dbEndpoint pulumi.StringInput,
	dbSecretArn pulumi.StringInput,
	logGroupName pulumi.StringInput,
	iamRes *appIamResult,
) error {

	// 1. Security Group cho App (Chi cho phep traffic HTTP port 80 tu ALB)
	appSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       vpcIdPtr,
		Description: pulumi.String("Allow HTTP from ALB and all egress"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(80),
				ToPort:         pulumi.Int(80),
				SecurityGroups: pulumi.StringArray{albSecurityGroupID},
				Description:    pulumi.String("HTTP from ALB SG"),
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-sg-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return fmt.Errorf("creating app security group: %w", err)
	}

	// 2. ECS Task Definition (Fargate 0.25 vCPU, 512 MB RAM)
	containerDef := pulumi.Sprintf(`[
		{
			"name": "app",
			"image": "%s:latest",
			"essential": true,
			"portMappings": [
				{
					"containerPort": 80,
					"hostPort": 80,
					"protocol": "tcp"
				}
			],
			"environment": [
				{
					"name": "DB_ENDPOINT",
					"value": "%s"
				},
				{
					"name": "DB_NAME",
					"value": "pocdb"
				},
				{
					"name": "DB_USER",
					"value": "dbadmin"
				}
			],
			"secrets": [
				{
					"name": "DB_PASSWORD",
					"valueFrom": "%s"
				}
			],
			"logConfiguration": {
				"logDriver": "awslogs",
				"options": {
					"awslogs-group": "%s",
					"awslogs-region": "ap-southeast-1",
					"awslogs-stream-prefix": "app"
				}
			}
		}
	]`, ecrRepoUrl, dbEndpoint, dbSecretArn, logGroupName)

	taskDef, err := ecs.NewTaskDefinition(ctx, fmt.Sprintf("%s-taskdef", name), &ecs.TaskDefinitionArgs{
		Family:                  pulumi.Sprintf("%s-%s", name, stack),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		NetworkMode:             pulumi.String("awsvpc"),
		Cpu:                     pulumi.String("256"),
		Memory:                  pulumi.String("512"),
		ExecutionRoleArn:        iamRes.execRole.Arn,
		TaskRoleArn:             iamRes.taskRole.Arn,
		ContainerDefinitions:    containerDef,
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-taskdef-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return fmt.Errorf("creating task definition: %w", err)
	}

	// 3. ECS Service Fargate (chay tren public subnets, gan vao ALB Target Group)
	_, err = ecs.NewService(ctx, fmt.Sprintf("%s-service", name), &ecs.ServiceArgs{
		Cluster:        clusterArn,
		TaskDefinition: taskDef.Arn,
		LaunchType:     pulumi.String("FARGATE"),
		DesiredCount:   desiredCount,
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			Subnets:        subnetIDs,
			SecurityGroups: pulumi.StringArray{appSg.ID()},
			AssignPublicIp: pulumi.Bool(true),
		},
		LoadBalancers: ecs.ServiceLoadBalancerArray{
			&ecs.ServiceLoadBalancerArgs{
				TargetGroupArn: albTargetGroupArn,
				ContainerName:  pulumi.String("app"),
				ContainerPort:  pulumi.Int(80),
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-svc-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return fmt.Errorf("creating ecs service: %w", err)
	}

	return nil
}
