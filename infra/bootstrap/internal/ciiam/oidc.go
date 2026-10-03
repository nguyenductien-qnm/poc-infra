package ciiam

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"infra-poc/internal/shared"
)

const githubOidcHost = "token.actions.githubusercontent.com"

// newOidcProvider tao GitHub OIDC provider. Moi account chi co 1 provider cho host nay.
func newOidcProvider(ctx *pulumi.Context, name string, args *Args, opt pulumi.ResourceOption) (pulumi.StringOutput, error) {
	importID := ""
	if args.ImportExisting {
		importID = fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", args.AccountID, githubOidcHost)
	}

	provider, err := iam.NewOpenIdConnectProvider(ctx, fmt.Sprintf("%s-oidc", name), &iam.OpenIdConnectProviderArgs{
		Url:             pulumi.String("https://" + githubOidcHost),
		ClientIdLists:   pulumi.StringArray{pulumi.String("sts.amazonaws.com")},
		ThumbprintLists: pulumi.ToStringArray(args.OidcThumbprints),
	}, append([]pulumi.ResourceOption{opt, pulumi.Protect(true)}, shared.ImportOpt(importID)...)...)
	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("creating github oidc provider: %w", err)
	}
	return provider.Arn, nil
}
