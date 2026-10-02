package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newRouting khoi tao Public & Private Route Tables va gan Associations
func newRouting(
	ctx *pulumi.Context,
	name, stack string,
	vpc *ec2.Vpc,
	igw *ec2.InternetGateway,
	subs *subnetResult,
	opt pulumi.ResourceOption,
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
		Tags: shared.Tags(fmt.Sprintf("%s-pub-rt", name), stack),
	}, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("creating public route table: %w", err)
	}
	if err := associate(ctx, fmt.Sprintf("%s-pub-rta", name), pubRouteTable, subs.public, opt); err != nil {
		return nil, nil, err
	}

	// 2. Private Route Table (Noi bo VPC, khong NAT Gateway de tiet kiem chi phi)
	privRouteTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-priv-rt", name), &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Tags:  shared.Tags(fmt.Sprintf("%s-priv-rt", name), stack),
	}, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("creating private route table: %w", err)
	}
	if err := associate(ctx, fmt.Sprintf("%s-priv-rta", name), privRouteTable, subs.private, opt); err != nil {
		return nil, nil, err
	}

	return pubRouteTable, privRouteTable, nil
}

// associate gan route table vao tung subnet; ten logical: <prefix>-1, <prefix>-2, ...
func associate(ctx *pulumi.Context, prefix string, rt *ec2.RouteTable, subnetIDs pulumi.StringArray, opt pulumi.ResourceOption) error {
	for i, subnetID := range subnetIDs {
		_, err := ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-%d", prefix, i+1), &ec2.RouteTableAssociationArgs{
			SubnetId:     subnetID,
			RouteTableId: rt.ID(),
		}, opt)
		if err != nil {
			return fmt.Errorf("creating route table association %s-%d: %w", prefix, i+1, err)
		}
	}
	return nil
}
