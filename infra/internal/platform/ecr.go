package platform

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// newEcrRepo khoi tao ECR Repository voi bao ve stateful tuy theo stack
func newEcrRepo(ctx *pulumi.Context, name, stack string, protectStateful bool) (*ecr.Repository, error) {
	ecrRepo, err := ecr.NewRepository(ctx, fmt.Sprintf("%s-ecr", name), &ecr.RepositoryArgs{
		Name:        pulumi.Sprintf("%s-repo-%s", name, stack),
		ForceDelete: pulumi.Bool(!protectStateful),
		ImageScanningConfiguration: &ecr.RepositoryImageScanningConfigurationArgs{
			ScanOnPush: pulumi.Bool(true),
		},
		Tags: pulumi.StringMap{
			"Name":  pulumi.Sprintf("%s-repo-%s", name, stack),
			"Stack": pulumi.String(stack),
		},
	}, pulumi.Protect(protectStateful))
	if err != nil {
		return nil, fmt.Errorf("creating ecr repo: %w", err)
	}
	return ecrRepo, nil
}
