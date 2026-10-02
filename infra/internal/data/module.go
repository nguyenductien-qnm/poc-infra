package data

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	VpcID              pulumi.StringOutput
	VpcCidr            string
	PrivateSubnetIDs   pulumi.StringArray // moi phan tu ung voi 1 AZ
	AppSecurityGroupID pulumi.StringInput // nguon duy nhat duoc ket noi toi RDS
	DbInstanceClass    pulumi.StringInput
	ProtectStateful    bool // pulumi protect cho RDS/EFS + giu final snapshot
	DeletionProtection bool // RDS deletion protection phia AWS
}

// Data la ComponentResource gom RDS PostgreSQL va EFS
type Data struct {
	pulumi.ResourceState

	DbEndpoint  pulumi.StringOutput
	DbSecretArn pulumi.StringOutput
	EfsID       pulumi.IDOutput
}

// New khoi tao toan bo ha tang Data (RDS PostgreSQL & EFS)
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Data, error) {
	stack := ctx.Stack()

	comp := &Data{}
	if err := ctx.RegisterComponentResource("infra-poc:data:Data", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering data component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. RDS PostgreSQL Instance (rds.go)
	rdsRes, err := newRds(ctx, name, stack, args, opt)
	if err != nil {
		return nil, err
	}

	// 2. EFS File System & Mount Targets (efs.go)
	efsID, err := newEfs(ctx, name, stack, args, opt)
	if err != nil {
		return nil, err
	}

	comp.DbEndpoint = rdsRes.endpoint
	comp.DbSecretArn = rdsRes.secretArn
	comp.EfsID = efsID
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"dbEndpoint":  comp.DbEndpoint,
		"dbSecretArn": comp.DbSecretArn,
		"efsId":       comp.EfsID,
	}); err != nil {
		return nil, fmt.Errorf("registering data outputs: %w", err)
	}
	return comp, nil
}
