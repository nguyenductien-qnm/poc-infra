package statebackend

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/kms"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newKey tao KMS key + alias ma hoa secret cua Pulumi (secretsprovider awskms://<alias>).
// Rotation dang tat o key hien co: giu nguyen de import khop, bat o thay doi rieng.
func newKey(ctx *pulumi.Context, name string, args *Args, opt pulumi.ResourceOption) (pulumi.StringOutput, error) {
	// 1. Key: mat key la khong giai ma duoc secret trong state, nen protect + giu lai khi destroy
	key, err := kms.NewKey(ctx, fmt.Sprintf("%s-key", name), &kms.KeyArgs{
		Description:       pulumi.String(args.KeyDescription),
		EnableKeyRotation: pulumi.Bool(false),
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true), pulumi.RetainOnDelete(true)}, shared.ImportOpt(args.ImportKeyID)...)...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating secrets kms key: %w", err)
	}

	// 2. Alias: stack yaml tro toi alias, khong tro toi key ID
	aliasImportID := ""
	if args.ImportExisting {
		aliasImportID = args.KeyAlias
	}
	_, err = kms.NewAlias(ctx, fmt.Sprintf("%s-key-alias", name), &kms.AliasArgs{
		Name:        pulumi.String(args.KeyAlias),
		TargetKeyId: key.KeyId,
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true)}, shared.ImportOpt(aliasImportID)...)...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating secrets kms alias: %w", err)
	}

	return key.Arn, nil
}
