package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type albResult struct {
	albDnsName         pulumi.StringOutput
	albTargetGroupArn  pulumi.StringOutput
	albSecurityGroupID pulumi.StringOutput
}

// newAlb khoi tao Security Group, Application Load Balancer, Target Group va Listener
func newAlb(
	ctx *pulumi.Context,
	name, stack string,
	vpcIdPtr pulumi.StringPtrInput,
	publicSubnetIDs pulumi.StringArrayInput,
) (*albResult, error) {

	// 1. Security Group cho ALB (Cho phep inbound HTTP port 80)
	albSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-alb-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       vpcIdPtr,
		Description: pulumi.String("Allow HTTP traffic to ALB"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol: pulumi.String("tcp"),
				FromPort: pulumi.Int(80),
				ToPort:   pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol: pulumi.String("-1"),
				FromPort: pulumi.Int(0),
				ToPort:   pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-alb-sg-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating alb security group: %w", err)
	}

	// 2. ALB (Internet-facing tren Public Subnets)
	alb, err := lb.NewLoadBalancer(ctx, fmt.Sprintf("%s-alb", name), &lb.LoadBalancerArgs{
		Name:             pulumi.Sprintf("%s-alb-%s", name, stack),
		Internal:         pulumi.Bool(false),
		LoadBalancerType: pulumi.String("application"),
		SecurityGroups: pulumi.StringArray{
			albSg.ID(),
		},
		Subnets: publicSubnetIDs,
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-alb-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating alb: %w", err)
	}

	// 3. ALB Target Group (Target Type "ip" cho ECS Fargate)
	targetGroup, err := lb.NewTargetGroup(ctx, fmt.Sprintf("%s-tg", name), &lb.TargetGroupArgs{
		Name:       pulumi.Sprintf("%s-tg-%s", name, stack),
		Port:       pulumi.Int(80),
		Protocol:   pulumi.String("HTTP"),
		VpcId:      vpcIdPtr,
		TargetType: pulumi.String("ip"),
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			Path:     pulumi.String("/"),
			Protocol: pulumi.String("HTTP"),
			Matcher:  pulumi.String("200"),
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-tg-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating target group: %w", err)
	}

	// 4. ALB Listener (HTTP Port 80 -> forward sang Target Group)
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
	})
	if err != nil {
		return nil, fmt.Errorf("creating alb listener: %w", err)
	}

	return &albResult{
		albDnsName:         alb.DnsName,
		albTargetGroupArn:  targetGroup.Arn,
		albSecurityGroupID: albSg.ID().ToStringOutput(),
	}, nil
}
