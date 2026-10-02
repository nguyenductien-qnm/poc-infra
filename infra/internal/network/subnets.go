package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

type subnetResult struct {
	public  pulumi.StringArray
	private pulumi.StringArray
}

// newSubnets khoi tao public/private subnets, moi CIDR dat o 1 AZ khac nhau
func newSubnets(ctx *pulumi.Context, name, stack string, vpc *ec2.Vpc, publicCidrs, privateCidrs []string, opt pulumi.ResourceOption) (*subnetResult, error) {
	// Lay danh sach Availability Zones kha dung
	azs, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{
		State: pulumi.StringRef("available"),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getting azs: %w", err)
	}

	// 1. Public Subnets (ALB + Fargate task, co public IP)
	public, err := createSubnets(ctx, name, stack, vpc, azs.Names, publicCidrs, "public", opt)
	if err != nil {
		return nil, err
	}

	// 2. Private Subnets (RDS / EFS)
	private, err := createSubnets(ctx, name, stack, vpc, azs.Names, privateCidrs, "private", opt)
	if err != nil {
		return nil, err
	}

	return &subnetResult{public: public, private: private}, nil
}

// createSubnets tao 1 subnet cho moi CIDR; subnetType la "public" hoac "private"
func createSubnets(
	ctx *pulumi.Context,
	name, stack string,
	vpc *ec2.Vpc,
	azNames, cidrs []string,
	subnetType string,
	opt pulumi.ResourceOption,
) (pulumi.StringArray, error) {
	if len(azNames) < len(cidrs) {
		return nil, fmt.Errorf("need %d azs for %s subnets, region only has %d", len(cidrs), subnetType, len(azNames))
	}

	// Ten logical ngan: "pub-sub" / "priv-sub"
	short := map[string]string{"public": "pub", "private": "priv"}[subnetType]

	ids := make(pulumi.StringArray, len(cidrs))
	for i, cidr := range cidrs {
		resName := fmt.Sprintf("%s-%s-sub-%d", name, short, i+1)
		subnet, err := ec2.NewSubnet(ctx, resName, &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(cidr),
			AvailabilityZone:    pulumi.String(azNames[i]),
			MapPublicIpOnLaunch: pulumi.Bool(subnetType == "public"),
			Tags:                shared.Tags(resName, stack, pulumi.StringMap{"Type": pulumi.String(subnetType)}),
		}, opt)
		if err != nil {
			return nil, fmt.Errorf("creating %s subnet %d: %w", subnetType, i+1, err)
		}
		ids[i] = subnet.ID()
	}
	return ids, nil
}
