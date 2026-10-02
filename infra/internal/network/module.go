package network

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcCidr   string
	Region    string
	EnableTgw bool
}

// Network la ComponentResource gom moi resource network lam con (hien thanh 1 nhom trong preview/console)
type Network struct {
	pulumi.ResourceState

	VpcID            pulumi.StringOutput
	PublicSubnetIDs  pulumi.StringArray // moi phan tu ung voi 1 AZ
	PrivateSubnetIDs pulumi.StringArray // moi phan tu ung voi 1 AZ
}

// New khoi tao toan bo ha tang Network (VPC, Subnets, Routing, Endpoints)
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Network, error) {
	stack := ctx.Stack()

	comp := &Network{}
	if err := ctx.RegisterComponentResource("infra-poc:network:Network", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering network component: %w", err)
	}
	opt := pulumi.Parent(comp)

	publicCidrs, privateCidrs, err := subnetPlan(args.VpcCidr)
	if err != nil {
		return nil, fmt.Errorf("planning subnets: %w", err)
	}

	// 1. VPC & Internet Gateway (vpc.go)
	vpc, igw, err := newVpc(ctx, name, stack, args.VpcCidr, opt)
	if err != nil {
		return nil, err
	}

	// 2. Public & Private Subnets (subnets.go)
	subs, err := newSubnets(ctx, name, stack, vpc, publicCidrs, privateCidrs, opt)
	if err != nil {
		return nil, err
	}

	// 3. Route Tables & Associations (routing.go)
	pubRt, privRt, err := newRouting(ctx, name, stack, vpc, igw, subs, opt)
	if err != nil {
		return nil, err
	}

	// 4. S3 Gateway VPC Endpoint (endpoints.go)
	err = newEndpoints(ctx, name, stack, args.Region, vpc, pubRt, privRt, opt)
	if err != nil {
		return nil, err
	}

	comp.VpcID = vpc.ID().ToStringOutput()
	comp.PublicSubnetIDs = subs.public
	comp.PrivateSubnetIDs = subs.private
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"vpcId":            comp.VpcID,
		"publicSubnetIds":  comp.PublicSubnetIDs,
		"privateSubnetIds": comp.PrivateSubnetIDs,
	}); err != nil {
		return nil, fmt.Errorf("registering network outputs: %w", err)
	}
	return comp, nil
}
