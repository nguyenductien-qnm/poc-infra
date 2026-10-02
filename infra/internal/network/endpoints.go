package network

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newEndpoints khoi tao S3 Gateway VPC Endpoint (Mien phi, truy cap S3 truc tiep)
func newEndpoints(
	ctx *pulumi.Context,
	name, stack, region string,
	vpc *ec2.Vpc,
	pubRouteTable, privRouteTable *ec2.RouteTable,
	opt pulumi.ResourceOption,
) error {
	_, err := ec2.NewVpcEndpoint(ctx, fmt.Sprintf("%s-s3-endpoint", name), &ec2.VpcEndpointArgs{
		VpcId:           vpc.ID(),
		ServiceName:     pulumi.String(fmt.Sprintf("com.amazonaws.%s.s3", region)),
		VpcEndpointType: pulumi.String("Gateway"),
		RouteTableIds: pulumi.StringArray{
			pubRouteTable.ID(),
			privRouteTable.ID(),
		},
		Tags: shared.Tags(fmt.Sprintf("%s-s3-ep", name), stack),
	}, opt)
	if err != nil {
		return fmt.Errorf("creating s3 endpoint: %w", err)
	}

	return nil
}
