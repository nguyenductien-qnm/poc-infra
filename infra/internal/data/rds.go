package data

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

type rdsResult struct {
	endpoint  pulumi.StringOutput
	secretArn pulumi.StringOutput
}

// newRds khoi tao DB Subnet Group, Security Group va RDS PostgreSQL Instance
func newRds(ctx *pulumi.Context, name, stack string, args *Args, opt pulumi.ResourceOption) (*rdsResult, error) {
	// 1. RDS DB Subnet Group (gom cac private subnets)
	subnetGroup, err := rds.NewSubnetGroup(ctx, fmt.Sprintf("%s-sng", name), &rds.SubnetGroupArgs{
		SubnetIds: args.PrivateSubnetIDs,
		Tags:      shared.Tags(fmt.Sprintf("%s-sng", name), stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating db subnet group: %w", err)
	}

	// 2. RDS Security Group (chi cho phep ingress port DB tu App SG)
	rdsSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-rds-sg", name), &ec2.SecurityGroupArgs{
		VpcId: args.VpcID,
		// Giu nguyen description cu: doi description se lam replace Security Group
		Description: pulumi.String("Allow PostgreSQL traffic from VPC"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(shared.DbPort),
				ToPort:         pulumi.Int(shared.DbPort),
				SecurityGroups: pulumi.StringArray{args.AppSecurityGroupID},
				Description:    pulumi.String("PostgreSQL from App SG"),
			},
		},
		Egress: shared.AllowAllEgress(),
		Tags:   shared.Tags(fmt.Sprintf("%s-rds-sg", name), stack),
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("creating rds security group: %w", err)
	}

	// 3. RDS PostgreSQL Instance Single-AZ
	// Mat khau duoc AWS Secrets Manager tu tao va quan ly (manageMasterUserPassword: true)
	dbName := fmt.Sprintf("%s-db", name)
	db, err := rds.NewInstance(ctx, dbName, &rds.InstanceArgs{
		Identifier:               pulumi.String(dbName + "-" + stack),
		AllocatedStorage:         pulumi.Int(20),
		StorageType:              pulumi.String("gp3"),
		Engine:                   pulumi.String("postgres"),
		EngineVersion:            pulumi.String("16.11"),
		InstanceClass:            args.DbInstanceClass,
		Port:                     pulumi.Int(shared.DbPort),
		DbName:                   pulumi.String(shared.DbName),
		Username:                 pulumi.String(shared.DbUser),
		ManageMasterUserPassword: pulumi.Bool(true),
		DbSubnetGroupName:        subnetGroup.Name,
		VpcSecurityGroupIds:      pulumi.StringArray{rdsSg.ID()},
		PubliclyAccessible:       pulumi.Bool(false),
		SkipFinalSnapshot:        pulumi.Bool(!args.ProtectStateful),
		DeletionProtection:       pulumi.Bool(args.DeletionProtection),
		Tags:                     shared.Tags(dbName, stack),
	}, opt, pulumi.Protect(args.ProtectStateful))
	if err != nil {
		return nil, fmt.Errorf("creating rds instance: %w", err)
	}

	// Lay ARN cua Master User Secret tu AWS Secrets Manager
	dbSecretArn := db.MasterUserSecrets.ApplyT(func(secrets []rds.InstanceMasterUserSecret) string {
		if len(secrets) > 0 && secrets[0].SecretArn != nil {
			return *secrets[0].SecretArn
		}
		return ""
	}).(pulumi.StringOutput)

	return &rdsResult{
		endpoint:  db.Endpoint,
		secretArn: dbSecretArn,
	}, nil
}
