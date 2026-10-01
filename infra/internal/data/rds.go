package data

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type rdsResult struct {
	endpoint  pulumi.StringOutput
	secretArn pulumi.StringOutput
}

// newRds khoi tao DB Subnet Group, Security Group va RDS PostgreSQL Instance
func newRds(
	ctx *pulumi.Context,
	name, stack string,
	vpcIdPtr pulumi.StringPtrInput,
	vpcCidr string,
	subnetIDs pulumi.StringArrayInput,
	dbInstanceClass pulumi.StringInput,
	protectStateful bool,
	deletionProtection bool,
) (*rdsResult, error) {

	// 1. RDS DB Subnet Group (gom 2 private subnets)
	subnetGroup, err := rds.NewSubnetGroup(ctx, fmt.Sprintf("%s-sng", name), &rds.SubnetGroupArgs{
		SubnetIds: subnetIDs,
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-sng-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating db subnet group: %w", err)
	}

	// 2. RDS Security Group (chi cho phep ingress port 5432 tu VPC)
	rdsSg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-rds-sg", name), &ec2.SecurityGroupArgs{
		VpcId:       vpcIdPtr,
		Description: pulumi.String("Allow PostgreSQL traffic from VPC"),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:    pulumi.String("tcp"),
				FromPort:    pulumi.Int(5432),
				ToPort:      pulumi.Int(5432),
				CidrBlocks:  pulumi.StringArray{pulumi.String(vpcCidr)},
				Description: pulumi.String("PostgreSQL from VPC"),
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-rds-sg-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating rds security group: %w", err)
	}

	// 3. RDS PostgreSQL Instance Single-AZ
	// Mat khau duoc AWS Secrets Manager tu tao va quan ly (manageMasterUserPassword: true)
	db, err := rds.NewInstance(ctx, fmt.Sprintf("%s-db", name), &rds.InstanceArgs{
		Identifier:               pulumi.Sprintf("%s-db-%s", name, stack),
		AllocatedStorage:         pulumi.Int(20),
		StorageType:              pulumi.String("gp3"),
		Engine:                   pulumi.String("postgres"),
		EngineVersion:            pulumi.String("16.3"),
		InstanceClass:            dbInstanceClass,
		DbName:                   pulumi.String("pocdb"),
		Username:                 pulumi.String("dbadmin"),
		ManageMasterUserPassword: pulumi.Bool(true),
		DbSubnetGroupName:        subnetGroup.Name,
		VpcSecurityGroupIds:      pulumi.StringArray{rdsSg.ID()},
		PubliclyAccessible:       pulumi.Bool(false),
		SkipFinalSnapshot:        pulumi.Bool(!protectStateful),
		DeletionProtection:       pulumi.Bool(deletionProtection),
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-db-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	}, pulumi.Protect(protectStateful))
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
