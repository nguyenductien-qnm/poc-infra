// Package statebackend quan ly backend state cua workload: S3 bucket chua state va KMS key ma hoa secret Pulumi.
package statebackend

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Args struct {
	BucketName     string // ten that, co account ID nen ghep luc chay, khong commit
	KeyAlias       string // vd alias/pulumi-poc-key
	KeyDescription string
	ImportKeyID    string // != "" thi import key dang co (key ID tra tu alias), "" thi tao moi
	ImportExisting bool   // import bucket/alias dang co thay vi tao moi
}

// StateBackend la ComponentResource gom bucket state va KMS key
type StateBackend struct {
	pulumi.ResourceState

	BucketName pulumi.StringOutput
	BucketArn  pulumi.StringOutput
	KeyArn     pulumi.StringOutput
}

// New khoi tao (hoac tiep nhan) bucket state va KMS key
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*StateBackend, error) {
	comp := &StateBackend{}
	if err := ctx.RegisterComponentResource("infra-poc:statebackend:StateBackend", name, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering statebackend component: %w", err)
	}
	opt := pulumi.Parent(comp)

	// 1. S3 bucket chua state + versioning, ma hoa, chan public (bucket.go)
	bucket, err := newBucket(ctx, name, args, opt)
	if err != nil {
		return nil, err
	}

	// 2. KMS key + alias ma hoa secret trong state (key.go)
	keyArn, err := newKey(ctx, name, args, opt)
	if err != nil {
		return nil, err
	}

	comp.BucketName = bucket.Bucket
	comp.BucketArn = bucket.Arn
	comp.KeyArn = keyArn
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"bucketName": comp.BucketName,
		"bucketArn":  comp.BucketArn,
		"keyArn":     comp.KeyArn,
	}); err != nil {
		return nil, fmt.Errorf("registering statebackend outputs: %w", err)
	}
	return comp, nil
}
