package data

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/efs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newEfs khoi tao EFS Security Group, File System va cac Mount Targets
func newEfs(ctx *pulumi.Context, name, stack string, args *Args, opt pulumi.ResourceOption) (pulumi.IDOutput, error) {
	// 1. EFS Security Group (ingress port 2049 NFS tu VPC)
	efsSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-efs-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       args.VpcID,
		Description: pulumi.String("Allow NFS traffic to EFS from VPC"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:    pulumi.String("tcp"),
				FromPort:    pulumi.Int(2049),
				ToPort:      pulumi.Int(2049),
				CidrBlocks:  pulumi.StringArray{pulumi.String(args.VpcCidr)},
				Description: pulumi.String("NFS from VPC"),
			},
		},
		Egress: shared.AllowAllEgress(),
		Tags:   shared.Tags(fmt.Sprintf("%s-efs-sg", name), stack),
	}, opt)
	if err != nil {
		return pulumi.IDOutput{}, fmt.Errorf("creating efs security group: %w", err)
	}

	// 2. EFS File System
	efsFs, err := efs.NewFileSystem(ctx, fmt.Sprintf("%s-efs", name), &efs.FileSystemArgs{
		Encrypted: pulumi.Bool(true),
		Tags:      shared.Tags(fmt.Sprintf("%s-efs", name), stack),
	}, opt, pulumi.Protect(args.ProtectStateful))
	if err != nil {
		return pulumi.IDOutput{}, fmt.Errorf("creating efs filesystem: %w", err)
	}

	// 3. EFS Mount Targets tren tung private subnet
	for i, subnetID := range args.PrivateSubnetIDs {
		_, err := efs.NewMountTarget(ctx, fmt.Sprintf("%s-efs-mt-%d", name, i+1), &efs.MountTargetArgs{
			FileSystemId:   efsFs.ID(),
			SubnetId:       subnetID,
			SecurityGroups: pulumi.StringArray{efsSg.ID()},
		}, opt)
		if err != nil {
			return pulumi.IDOutput{}, fmt.Errorf("creating efs mount target %d: %w", i+1, err)
		}
	}

	return efsFs.ID(), nil
}
