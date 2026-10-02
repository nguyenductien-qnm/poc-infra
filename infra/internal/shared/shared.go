// Package shared chua hang so va helper dung chung giua cac package infra.
package shared

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Hang so dung chung giua data (RDS) va app (container) - doi o day, ca hai ben cung doi.
const (
	AppPort = 80
	DbPort  = 5432
	DbName  = "pocdb"
	DbUser  = "dbadmin"
)

// Tags tra ve tag chuan cho moi resource: Name = "<resourceName>-<stack>" va Stack.
// extra (neu co) duoc gop them, vi du {"Type": "public"}.
func Tags(resourceName, stack string, extra ...pulumi.StringMap) pulumi.StringMap {
	tags := pulumi.StringMap{
		"Name":  pulumi.String(resourceName + "-" + stack),
		"Stack": pulumi.String(stack),
	}
	for _, m := range extra {
		for k, v := range m {
			tags[k] = v
		}
	}
	return tags
}

// AllowAllEgress tra ve rule egress cho phep moi traffic di ra (dung cho moi Security Group).
func AllowAllEgress() ec2.SecurityGroupEgressArray {
	return ec2.SecurityGroupEgressArray{
		&ec2.SecurityGroupEgressArgs{
			Protocol:   pulumi.String("-1"),
			FromPort:   pulumi.Int(0),
			ToPort:     pulumi.Int(0),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
	}
}
