// Package registry quan ly ECR repository dung chung cho moi stack workload.
// Thuoc bootstrap vi CI push image truoc khi up workload, va destroy workload khong duoc xoa image.
package registry

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

type Args struct {
	RepositoryName string
	ImportExisting bool
}

// Registry la ComponentResource gom ECR repository
type Registry struct {
	pulumi.ResourceState

	RepositoryUrl pulumi.StringOutput
	RepositoryArn pulumi.StringOutput
}

// New khoi tao (hoac tiep nhan) ECR repository.
// Gia tri khop repo hien co (MUTABLE, scanOnPush, AES256) de import khong thay doi gi.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Registry, error) {
	comp := &Registry{}
	if err := ctx.RegisterComponentResource("infra-poc:registry:Registry", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering registry component: %w", err)
	}
	opt := pulumi.Parent(comp)

	importID := ""
	if args.ImportExisting {
		importID = args.RepositoryName
	}

	// 1. ECR repository: protect + giu lai khi destroy de khong mat image dang chay
	repo, err := ecr.NewRepository(ctx, fmt.Sprintf("%s-repo", name), &ecr.RepositoryArgs{
		Name:               pulumi.String(args.RepositoryName),
		ImageTagMutability: pulumi.String("MUTABLE"),
		ImageScanningConfiguration: &ecr.RepositoryImageScanningConfigurationArgs{
			ScanOnPush: pulumi.Bool(true),
		},
		EncryptionConfigurations: ecr.RepositoryEncryptionConfigurationArray{
			&ecr.RepositoryEncryptionConfigurationArgs{EncryptionType: pulumi.String("AES256")},
		},
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true), pulumi.RetainOnDelete(true)}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating ecr repository: %w", err)
	}

	comp.RepositoryUrl = repo.RepositoryUrl
	comp.RepositoryArn = repo.Arn
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"repositoryUrl": comp.RepositoryUrl,
		"repositoryArn": comp.RepositoryArn,
	}); err != nil {
		return nil, fmt.Errorf("registering registry outputs: %w", err)
	}
	return comp, nil
}
