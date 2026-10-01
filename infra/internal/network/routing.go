package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newRouting khoi tao Public & Private Route Tables va gan Associations
func newRouting(
	ctx *pulumi.Context,
	name, stack string,
	vpc *ec2.Vpc,
	igw *ec2.InternetGateway,
	subs *subnetResult,
) (*ec2.RouteTable, *ec2.RouteTable, error) {

	// 1. Public Route Table (Tro default route ra IGW)
	pubRouteTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-pub-rt", name), &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-pub-rt-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating public route table: %w", err)
	}

	for i := 0; i < 2; i++ {
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-pub-rta-%d", name, i+1), &ec2.RouteTableAssociationArgs{
			SubnetId:     subs.publicSubnetIDs[i],
			RouteTableId: pubRouteTable.ID(),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("associating public route table %d: %w", i+1, err)
		}
	}

	// 2. Private Route Table (Noi bo VPC, khong NAT Gateway de tiet kiem chi phi)
	privRouteTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-priv-rt", name), &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-priv-rt-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating private route table: %w", err)
	}

	for i := 0; i < 2; i++ {
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-priv-rta-%d", name, i+1), &ec2.RouteTableAssociationArgs{
			SubnetId:     subs.privateSubnetIDs[i],
			RouteTableId: privRouteTable.ID(),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("associating private route table %d: %w", i+1, err)
		}
	}

	return pubRouteTable, privRouteTable, nil
}
