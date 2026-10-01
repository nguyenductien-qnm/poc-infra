package network

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type subnetResult struct {
	publicSubnetIDs   pulumi.StringArray
	privateSubnetIDs  pulumi.StringArray
	privateSubnetList []pulumi.StringOutput
}

// newSubnets khoi tao 2 Public Subnets va 2 Private Subnets o 2 AZs khac nhau
func newSubnets(ctx *pulumi.Context, name, stack string, vpc *ec2.Vpc, vpcCidr string) (*subnetResult, error) {
	// Lay danh sach Availability Zones kha dung
	azs, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{
		State: pulumi.StringRef("available"),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("getting azs: %w", err)
	}

	// Tinh toan dai CIDR dua vao vpcCidr (vi du: 10.10.0.0/16 -> prefix "10.10")
	parts := strings.Split(vpcCidr, ".")
	prefix := fmt.Sprintf("%s.%s", parts[0], parts[1])
	pubCidrs := []string{fmt.Sprintf("%s.1.0/24", prefix), fmt.Sprintf("%s.2.0/24", prefix)}
	privCidrs := []string{fmt.Sprintf("%s.10.0/24", prefix), fmt.Sprintf("%s.20.0/24", prefix)}

	// 1. Public Subnets (2 AZs)
	publicSubnetIDs := make(pulumi.StringArray, 2)
	for i := 0; i < 2; i++ {
		az := azs.Names[i]
		subnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-pub-sub-%d", name, i+1), &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(pubCidrs[i]),
			AvailabilityZone:    pulumi.String(az),
			MapPublicIpOnLaunch: pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name":  pulumi.Sprintf("%s-pub-sub-%d-%s", name, i+1, stack),
				"Stack": pulumi.String(stack),
				"Type":  pulumi.String("public"),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("creating public subnet %d: %w", i+1, err)
		}
		publicSubnetIDs[i] = subnet.ID()
	}

	// 2. Private Subnets (2 AZs - cho RDS / EFS)
	privateSubnetIDs := make(pulumi.StringArray, 2)
	privateSubnetList := make([]pulumi.StringOutput, 2)
	for i := 0; i < 2; i++ {
		az := azs.Names[i]
		subnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-priv-sub-%d", name, i+1), &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(privCidrs[i]),
			AvailabilityZone:    pulumi.String(az),
			MapPublicIpOnLaunch: pulumi.Bool(false),
			Tags: pulumi.StringMap{
				"Name":  pulumi.Sprintf("%s-priv-sub-%d-%s", name, i+1, stack),
				"Stack": pulumi.String(stack),
				"Type":  pulumi.String("private"),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("creating private subnet %d: %w", i+1, err)
		}
		privateSubnetIDs[i] = subnet.ID()
		privateSubnetList[i] = subnet.ID().ToStringOutput()
	}

	return &subnetResult{
		publicSubnetIDs:   publicSubnetIDs,
		privateSubnetIDs:  privateSubnetIDs,
		privateSubnetList: privateSubnetList,
	}, nil
}
