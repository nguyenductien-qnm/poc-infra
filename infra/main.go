package main

import (
	"infra-poc/internal/app"
	"infra-poc/internal/data"
	"infra-poc/internal/network"
	"infra-poc/internal/platform"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")

		// 1. Doc config theo PLAN.md tu Pulumi.<stack>.yaml
		vpcCidr := cfg.Require("vpcCidr")
		desiredCount := cfg.RequireInt("desiredCount")
		dbInstanceClass := cfg.Require("dbInstanceClass")
		protectStateful := cfg.RequireBool("protectStateful")
		deletionProtection := cfg.RequireBool("deletionProtection")
		enableTgw := cfg.GetBool("enableTgw")
		imageTag := cfg.Get("imageTag")
		if imageTag == "" {
			imageTag = "latest"
		}

		// Lay thong tin AWS Account va Region de ghep ECR URL dung chung (poc-app)
		caller, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}
		region, err := aws.GetRegion(ctx, nil, nil)
		if err != nil {
			return err
		}
		sharedEcrUrl := pulumi.Sprintf("%s.dkr.ecr.%s.amazonaws.com/poc-app", caller.AccountId, region.Name)

		// 2. Network Package
		netOut, err := network.New(ctx, "network", &network.Args{
			VpcCidr:   vpcCidr,
			EnableTgw: enableTgw,
		})
		if err != nil {
			return err
		}

		// 3. Platform Package (ECS Cluster, CloudWatch Logs, ALB)
		platOut, err := platform.New(ctx, "platform", &platform.Args{
			VpcID:           netOut.VpcID,
			PublicSubnetIDs: netOut.PublicSubnetIDs,
			ProtectStateful: protectStateful,
		})
		if err != nil {
			return err
		}

		// 4. Data Package
		dataOut, err := data.New(ctx, "data", &data.Args{
			VpcID:              netOut.VpcID,
			VpcCidr:            vpcCidr,
			SubnetIDs:          netOut.PrivateSubnetIDs,
			PrivateSubnetList:  netOut.PrivateSubnetList,
			DbInstanceClass:    pulumi.String(dbInstanceClass),
			ProtectStateful:    protectStateful,
			DeletionProtection: deletionProtection,
		})
		if err != nil {
			return err
		}

		// 5. App Package (ECS Fargate Service tro ECR chung poc-app kem imageTag)
		appOut, err := app.New(ctx, "app", &app.Args{
			EcrRepoUrl:         sharedEcrUrl,
			ImageTag:           pulumi.String(imageTag),
			VpcID:              netOut.VpcID,
			SubnetIDs:          netOut.PublicSubnetIDs,
			ClusterArn:         platOut.ClusterArn,
			AlbTargetGroupArn:  platOut.AlbTargetGroupArn,
			AlbSecurityGroupID: platOut.AlbSecurityGroupID,
			AlbDnsName:         platOut.AlbDnsName,
			LogGroupName:       platOut.LogGroupName,
			DesiredCount:       pulumi.Int(desiredCount),
			DbEndpoint:         dataOut.DbEndpoint,
			DbSecretArn:        dataOut.DbSecretArn,
		})
		if err != nil {
			return err
		}

		// 6. Export outputs
		ctx.Export("vpcId", netOut.VpcID)
		ctx.Export("vpcCidr", pulumi.String(vpcCidr))
		ctx.Export("publicSubnetIds", netOut.PublicSubnetIDs)
		ctx.Export("privateSubnetIds", netOut.PrivateSubnetIDs)
		ctx.Export("clusterArn", platOut.ClusterArn)
		ctx.Export("albDnsName", platOut.AlbDnsName)
		ctx.Export("ecrRepoUrl", sharedEcrUrl)
		ctx.Export("imageTag", pulumi.String(imageTag))
		ctx.Export("dbEndpoint", dataOut.DbEndpoint)
		ctx.Export("efsId", dataOut.EfsID)
		ctx.Export("serviceUrl", appOut.ServiceUrl)

		return nil
	})
}
