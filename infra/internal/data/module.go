package data

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcID              pulumi.IDInput
	VpcCidr            string
	SubnetIDs          pulumi.StringArrayInput
	PrivateSubnetList  []pulumi.StringOutput
	DbInstanceClass    pulumi.StringInput
	ProtectStateful    bool
	DeletionProtection bool
}

type Outputs struct {
	DbEndpoint  pulumi.StringOutput
	DbSecretArn pulumi.StringOutput
	EfsID       pulumi.IDOutput
}

// New khoi tao toan bo ha tang Data (RDS PostgreSQL & EFS)
func New(ctx *pulumi.Context, name string, args *Args) (*Outputs, error) {
	stack := ctx.Stack()
	vpcIdPtr := args.VpcID.ToIDOutput().ToStringOutput().ToStringPtrOutput()

	// 1. RDS PostgreSQL Instance (rds.go)
	rdsRes, err := newRds(
		ctx, name, stack,
		vpcIdPtr, args.VpcCidr,
		args.SubnetIDs, args.DbInstanceClass,
		args.ProtectStateful, args.DeletionProtection,
	)
	if err != nil {
		return nil, err
	}

	// 2. EFS File System & Mount Targets (efs.go)
	efsID, err := newEfs(
		ctx, name, stack,
		vpcIdPtr, args.VpcCidr,
		args.PrivateSubnetList,
		args.ProtectStateful,
	)
	if err != nil {
		return nil, err
	}

	return &Outputs{
		DbEndpoint:  rdsRes.endpoint,
		DbSecretArn: rdsRes.secretArn,
		EfsID:       efsID,
	}, nil
}
