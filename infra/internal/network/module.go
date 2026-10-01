package network

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcCidr   string
	EnableTgw bool
}

type Outputs struct {
	VpcID             pulumi.IDOutput
	PublicSubnetIDs   pulumi.StringArrayOutput
	PrivateSubnetIDs  pulumi.StringArrayOutput
	PrivateSubnetList []pulumi.StringOutput
}

// New khoi tao toan bo ha tang Network (VPC, Subnets, Routing, Endpoints)
func New(ctx *pulumi.Context, name string, args *Args) (*Outputs, error) {
	stack := ctx.Stack()

	// 1. VPC & Internet Gateway (vpc.go)
	vpc, igw, err := newVpc(ctx, name, stack, args.VpcCidr)
	if err != nil {
		return nil, err
	}

	// 2. Public & Private Subnets (subnets.go)
	subs, err := newSubnets(ctx, name, stack, vpc, args.VpcCidr)
	if err != nil {
		return nil, err
	}

	// 3. Route Tables & Associations (routing.go)
	pubRt, privRt, err := newRouting(ctx, name, stack, vpc, igw, subs)
	if err != nil {
		return nil, err
	}

	// 4. S3 Gateway VPC Endpoint (endpoints.go)
	err = newEndpoints(ctx, name, stack, vpc, pubRt, privRt)
	if err != nil {
		return nil, err
	}

	return &Outputs{
		VpcID:             vpc.ID(),
		PublicSubnetIDs:   subs.publicSubnetIDs.ToStringArrayOutput(),
		PrivateSubnetIDs:  subs.privateSubnetIDs.ToStringArrayOutput(),
		PrivateSubnetList: subs.privateSubnetList,
	}, nil
}
