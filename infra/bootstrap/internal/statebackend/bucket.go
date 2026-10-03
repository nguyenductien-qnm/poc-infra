package statebackend

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

// newBucket tao bucket state va cau hinh di kem.
// Gia tri khop dung bucket hien co (doc bang aws s3api get-*) de import khong lam thay doi gi.
// Chua co tag/bucket policy: them sau buoc import, la thay doi rieng co review.
func newBucket(ctx *pulumi.Context, name string, args *Args, opt pulumi.ResourceOption) (*s3.Bucket, error) {
	importID := ""
	if args.ImportExisting {
		importID = args.BucketName // ID import cua bucket va cac cau hinh con deu la ten bucket
	}

	// 1. Bucket: protect + giu lai khi destroy, mat bucket la mat toan bo state workload
	bucket, err := s3.NewBucket(ctx, fmt.Sprintf("%s-bucket", name), &s3.BucketArgs{
		Bucket: pulumi.String(args.BucketName),
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true), pulumi.RetainOnDelete(true)}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating state bucket: %w", err)
	}

	// 2. Versioning: khoi phuc duoc ban state cu
	_, err = s3.NewBucketVersioning(ctx, fmt.Sprintf("%s-versioning", name), &s3.BucketVersioningArgs{
		Bucket: bucket.Bucket,
		VersioningConfiguration: &s3.BucketVersioningVersioningConfigurationArgs{
			Status: pulumi.String("Enabled"),
		},
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating state bucket versioning: %w", err)
	}

	// 3. Ma hoa object bang SSE-S3. Secret trong state da ma hoa rieng bang KMS key cua Pulumi.
	_, err = s3.NewBucketServerSideEncryptionConfiguration(ctx, fmt.Sprintf("%s-sse", name), &s3.BucketServerSideEncryptionConfigurationArgs{
		Bucket: bucket.Bucket,
		Rules: s3.BucketServerSideEncryptionConfigurationRuleArray{
			&s3.BucketServerSideEncryptionConfigurationRuleArgs{
				ApplyServerSideEncryptionByDefault: &s3.BucketServerSideEncryptionConfigurationRuleApplyServerSideEncryptionByDefaultArgs{
					SseAlgorithm: pulumi.String("AES256"),
				},
				BucketKeyEnabled:       pulumi.Bool(false),
				BlockedEncryptionTypes: pulumi.StringArray{pulumi.String("SSE-C")},
			},
		},
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating state bucket encryption: %w", err)
	}

	// 4. Chan moi duong public
	_, err = s3.NewBucketPublicAccessBlock(ctx, fmt.Sprintf("%s-public-block", name), &s3.BucketPublicAccessBlockArgs{
		Bucket:                bucket.Bucket,
		BlockPublicAcls:       pulumi.Bool(true),
		IgnorePublicAcls:      pulumi.Bool(true),
		BlockPublicPolicy:     pulumi.Bool(true),
		RestrictPublicBuckets: pulumi.Bool(true),
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating state bucket public access block: %w", err)
	}

	// 5. Tat ACL, quyen chi qua IAM/bucket policy
	_, err = s3.NewBucketOwnershipControls(ctx, fmt.Sprintf("%s-ownership", name), &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.Bucket,
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("BucketOwnerEnforced"),
		},
	}, append([]pulumi.ResourceOption{opt}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return nil, fmt.Errorf("creating state bucket ownership controls: %w", err)
	}

	return bucket, nil
}
