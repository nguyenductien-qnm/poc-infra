// Chuong trinh bootstrap: nen tang ma moi stack workload dua vao (state backend, KMS, ECR, danh tinh CI).
// Admin chay tay theo docs/SETUP.md, state luu o bucket rieng pulumi-bootstrap-poc-<account>.
// Pipeline workload khong co quyen sua project nay.
package main

import (
	"fmt"

	"infra-poc/bootstrap/internal/ciiam"
	"infra-poc/bootstrap/internal/registry"
	"infra-poc/bootstrap/internal/statebackend"
	"infra-poc/internal/shared"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/kms"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// githubRepo dung de ghep OIDC sub dang immutable: repo:<owner>@<owner_id>/<repo>@<repo_id>
type githubRepo struct {
	Owner   string `json:"owner"`
	OwnerID string `json:"ownerId"`
	Repo    string `json:"repo"`
	RepoID  string `json:"repoId"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")

		// 1. Doc config tu Pulumi.<stack>.yaml
		expectedAccount := cfg.Require("expectedAccount") // secret: khong commit account ID dang plaintext
		importExisting := cfg.RequireBool("importExisting")
		namePrefix := cfg.Require("namePrefix")
		stateBucketPrefix := cfg.Require("stateBucketPrefix")
		secretsKeyAlias := cfg.Require("secretsKeyAlias")
		secretsKeyDescription := cfg.Require("secretsKeyDescription")
		ecrRepositoryName := cfg.Require("ecrRepositoryName")
		var gh githubRepo
		cfg.RequireObject("github", &gh)
		var thumbprints []string
		cfg.RequireObject("oidcThumbprints", &thumbprints)
		var envs []ciiam.Environment
		cfg.RequireObject("ciEnvironments", &envs)

		caller, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}
		// Sai account thi dung truoc khi tao resource nao
		if err := shared.CheckAccount(caller.AccountId, expectedAccount); err != nil {
			return err
		}

		// Key hien co chi biet qua alias: lay key ID de import (khong commit ID vao config)
		importKeyID := ""
		if importExisting {
			alias, err := kms.LookupAlias(ctx, &kms.LookupAliasArgs{Name: secretsKeyAlias})
			if err != nil {
				return fmt.Errorf("looking up kms alias %s: %w", secretsKeyAlias, err)
			}
			importKeyID = alias.TargetKeyId
		}

		// 2. State backend cua workload: bucket + KMS key
		backend, err := statebackend.New(ctx, "state", &statebackend.Args{
			BucketName:     fmt.Sprintf("%s-%s", stateBucketPrefix, caller.AccountId),
			KeyAlias:       secretsKeyAlias,
			KeyDescription: secretsKeyDescription,
			ImportKeyID:    importKeyID,
			ImportExisting: importExisting,
		})
		if err != nil {
			return err
		}

		// 3. ECR dung chung cho moi stack workload
		reg, err := registry.New(ctx, "registry", &registry.Args{
			RepositoryName: ecrRepositoryName,
			ImportExisting: importExisting,
		})
		if err != nil {
			return err
		}

		// 4. Danh tinh CI: OIDC provider + role preview/deploy cho tung moi truong
		ci, err := ciiam.New(ctx, "ci", &ciiam.Args{
			AccountID:       caller.AccountId,
			NamePrefix:      namePrefix,
			SubjectPrefix:   fmt.Sprintf("repo:%s@%s/%s@%s", gh.Owner, gh.OwnerID, gh.Repo, gh.RepoID),
			OidcThumbprints: thumbprints,
			StateBucketArn:  backend.BucketArn,
			Environments:    envs,
			ImportExisting:  importExisting,
		})
		if err != nil {
			return err
		}

		// 5. Export outputs: ARN role la gia tri GitHub secret AWS_ROLE_<ENV>_<PREVIEW|DEPLOY> va AWS_ROLE_BOOTSTRAP
		ctx.Export("stateBucketName", backend.BucketName)
		ctx.Export("secretsKeyArn", backend.KeyArn)
		ctx.Export("ecrRepositoryUrl", reg.RepositoryUrl)
		ctx.Export("oidcProviderArn", ci.OidcProviderArn)
		ctx.Export("previewRoleArns", ci.PreviewRoleArns)
		ctx.Export("deployRoleArns", ci.DeployRoleArns)
		ctx.Export("bootstrapRoleArn", ci.BootstrapRoleArn) // GitHub secret AWS_ROLE_BOOTSTRAP

		return nil
	})
}
