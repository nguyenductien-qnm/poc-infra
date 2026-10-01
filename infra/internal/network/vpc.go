package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newVpc khoi tao VPC va Internet Gateway
func newVpc(ctx *pulumi.Context, name, stack, vpcCidr string) (*ec2.Vpc, *ec2.InternetGateway, error) {
	// 1. VPC
	vpc, err := ec2.NewVpc(ctx, fmt.Sprintf("%s-vpc", name), &ec2.VpcArgs{
		CidrBlock:          pulumi.String(vpcCidr),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-vpc-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating vpc: %w", err)
	}

	// 2. Internet Gateway
	igw, err := ec2.NewInternetGateway(ctx, fmt.Sprintf("%s-igw", name), &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-igw-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating igw: %w", err)
	}

	return vpc, igw, nil
}
