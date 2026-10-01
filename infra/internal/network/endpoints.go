package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newEndpoints khoi tao S3 Gateway VPC Endpoint (Mien phi, truy cap S3 truc tiep)
func newEndpoints(
	ctx *pulumi.Context,
	name, stack string,
	vpc *ec2.Vpc,
	pubRouteTable, privRouteTable *ec2.RouteTable,
) error {
	region, err := aws.GetRegion(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("getting aws region: %w", err)
	}

	_, err = ec2.NewVpcEndpoint(ctx, fmt.Sprintf("%s-s3-endpoint", name), &ec2.VpcEndpointArgs{
		VpcId:           vpc.ID(),
		ServiceName:     pulumi.Sprintf("com.amazonaws.%s.s3", region.Name),
		VpcEndpointType: pulumi.String("Gateway"),
		RouteTableIds: pulumi.StringArray{
			pubRouteTable.ID(),
			privRouteTable.ID(),
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-s3-ep-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return fmt.Errorf("creating s3 endpoint: %w", err)
	}

	return nil
}
