package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

type albResult struct {
	albDnsName         pulumi.StringOutput
	albTargetGroupArn  pulumi.StringOutput
	albSecurityGroupID pulumi.StringOutput
	appSecurityGroupID pulumi.StringOutput
}

// newAlb khoi tao Security Group ALB/App, Application Load Balancer, Target Group va Listener
func newAlb(ctx *pulumi.Context, name, stack string, args *Args, opt pulumi.ResourceOption) (*albResult, error) {
	// 1. Security Group cho ALB (Cho phep inbound HTTP tu Internet)
	albSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-alb-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       args.VpcID,
		Description: pulumi.String("Allow HTTP traffic to ALB"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(80),
				ToPort:     pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: shared.AllowAllEgress(),
		Tags:   shared.Tags(fmt.Sprintf("%s-alb-sg", name), stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating alb security group: %w", err)
	}

	// 2. Security Group cho App (chi cho phep traffic tu ALB vao port app).
	// Tao o platform de package data dung lam nguon ingress cho RDS (tranh phu thuoc vong data <-> app).
	appSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-app-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       args.VpcID,
		Description: pulumi.String("Allow HTTP from ALB and all egress"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(shared.AppPort),
				ToPort:         pulumi.Int(shared.AppPort),
				SecurityGroups: pulumi.StringArray{albSg.ID()},
				Description:    pulumi.String("HTTP from ALB SG"),
			},
		},
		Egress: shared.AllowAllEgress(),
		Tags:   shared.Tags(fmt.Sprintf("%s-app-sg", name), stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating app security group: %w", err)
	}

	// 3. ALB (Internet-facing tren Public Subnets)
	albName := fmt.Sprintf("%s-alb", name)
	alb, err := lb.NewLoadBalancer(ctx, albName, &lb.LoadBalancerArgs{
		Name:             pulumi.String(albName + "-" + stack),
		Internal:         pulumi.Bool(false),
		LoadBalancerType: pulumi.String("application"),
		SecurityGroups:   pulumi.StringArray{albSg.ID()},
		Subnets:          args.PublicSubnetIDs,
		Tags:             shared.Tags(albName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating alb: %w", err)
	}

	// 4. ALB Target Group (Target Type "ip" cho ECS Fargate)
	tgName := fmt.Sprintf("%s-tg", name)
	targetGroup, err := lb.NewTargetGroup(ctx, tgName, &lb.TargetGroupArgs{
		Name:       pulumi.String(tgName + "-" + stack),
		Port:       pulumi.Int(shared.AppPort),
		Protocol:   pulumi.String("HTTP"),
		VpcId:      args.VpcID,
		TargetType: pulumi.String("ip"),
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			Path:     pulumi.String("/health"),
			Protocol: pulumi.String("HTTP"),
			Matcher:  pulumi.String("200"),
		},
		Tags: shared.Tags(tgName, stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating target group: %w", err)
	}

	// 5. ALB Listener (HTTP Port 80 -> forward sang Target Group)
	_, err = lb.NewListener(ctx, fmt.Sprintf("%s-listener", name), &lb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(80),
		Protocol:        pulumi.String("HTTP"),
		DefaultActions: lb.ListenerDefaultActionArray{
			&lb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: targetGroup.Arn,
			},
		},
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating alb listener: %w", err)
	}

	return &albResult{
		albDnsName:         alb.DnsName,
		albTargetGroupArn:  targetGroup.Arn,
		albSecurityGroupID: albSg.ID().ToStringOutput(),
		appSecurityGroupID: appSg.ID().ToStringOutput(),
	}, nil
}
