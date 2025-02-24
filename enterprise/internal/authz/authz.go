package authz

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/authz/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/authz/github"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/authz/gitlab"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/authz/perforce"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

type ExternalServicesStore interface {
	List(context.Context, database.ExternalServicesListOptions) ([]*types.ExternalService, error)
}

// ProvidersFromConfig returns the set of permission-related providers derived from the site config.
// It also returns any validation problems with the config, separating these into "serious problems"
// and "warnings". "Serious problems" are those that should make Sourcegraph set authz.allowAccessByDefault
// to false. "Warnings" are all other validation problems.
func ProvidersFromConfig(
	ctx context.Context,
	cfg *conf.Unified,
	store ExternalServicesStore,
) (
	allowAccessByDefault bool,
	providers []authz.Provider,
	seriousProblems []string,
	warnings []string,
) {
	allowAccessByDefault = true
	defer func() {
		if len(seriousProblems) > 0 {
			log15.Error("Repository authz config was invalid (errors are visible in the UI as an admin user, you should fix ASAP). Restricting access to repositories by default for now to be safe.", "seriousProblems", seriousProblems)
			allowAccessByDefault = false
		}
	}()

	opt := database.ExternalServicesListOptions{
		NoNamespace: true,
		Kinds: []string{
			extsvc.KindGitHub,
			extsvc.KindGitLab,
			extsvc.KindBitbucketServer,
			extsvc.KindPerforce,
		},
		LimitOffset: &database.LimitOffset{
			Limit: 500, // The number is randomly chosen
		},
	}

	var (
		gitHubConns          []*types.GitHubConnection
		gitLabConns          []*types.GitLabConnection
		bitbucketServerConns []*types.BitbucketServerConnection
		perforceConns        []*types.PerforceConnection
	)
	for {
		svcs, err := store.List(ctx, opt)
		if err != nil {
			seriousProblems = append(seriousProblems, fmt.Sprintf("Could not list external services: %v", err))
			break
		}
		if len(svcs) == 0 {
			break // No more results, exiting
		}
		opt.AfterID = svcs[len(svcs)-1].ID // Advance the cursor

		for _, svc := range svcs {
			if svc.CloudDefault { // Only public repos in CloudDefault services
				continue
			}

			cfg, err := extsvc.ParseConfig(svc.Kind, svc.Config)
			if err != nil {
				seriousProblems = append(seriousProblems, fmt.Sprintf("Could not parse config of external service %d: %v", svc.ID, err))
				continue
			}

			switch c := cfg.(type) {
			case *schema.GitHubConnection:
				gitHubConns = append(gitHubConns, &types.GitHubConnection{
					URN:              svc.URN(),
					GitHubConnection: c,
				})
			case *schema.GitLabConnection:
				gitLabConns = append(gitLabConns, &types.GitLabConnection{
					URN:              svc.URN(),
					GitLabConnection: c,
				})
			case *schema.BitbucketServerConnection:
				bitbucketServerConns = append(bitbucketServerConns, &types.BitbucketServerConnection{
					URN:                       svc.URN(),
					BitbucketServerConnection: c,
				})
			case *schema.PerforceConnection:
				perforceConns = append(perforceConns, &types.PerforceConnection{
					URN:                svc.URN(),
					PerforceConnection: c,
				})
			default:
				log15.Error("ProvidersFromConfig", "error", errors.Errorf("unexpected connection type: %T", cfg))
				continue
			}

		}

		if len(svcs) < opt.Limit {
			break // Less results than limit means we've reached end
		}
	}

	if len(gitHubConns) > 0 {
		ghProviders, ghProblems, ghWarnings := github.NewAuthzProviders(gitHubConns, cfg.AuthProviders)
		providers = append(providers, ghProviders...)
		seriousProblems = append(seriousProblems, ghProblems...)
		warnings = append(warnings, ghWarnings...)
	}

	if len(gitLabConns) > 0 {
		glProviders, glProblems, glWarnings := gitlab.NewAuthzProviders(cfg, gitLabConns)
		providers = append(providers, glProviders...)
		seriousProblems = append(seriousProblems, glProblems...)
		warnings = append(warnings, glWarnings...)
	}

	if len(bitbucketServerConns) > 0 {
		bbsProviders, bbsProblems, bbsWarnings := bitbucketserver.NewAuthzProviders(bitbucketServerConns)
		providers = append(providers, bbsProviders...)
		seriousProblems = append(seriousProblems, bbsProblems...)
		warnings = append(warnings, bbsWarnings...)
	}

	if len(perforceConns) > 0 {
		pfProviders, pfProblems, pfWarnings := perforce.NewAuthzProviders(perforceConns)
		providers = append(providers, pfProviders...)
		seriousProblems = append(seriousProblems, pfProblems...)
		warnings = append(warnings, pfWarnings...)
	}

	// 🚨 SECURITY: Warn the admin when both code host authz provider and the permissions user mapping are configured.
	if cfg.SiteConfiguration.PermissionsUserMapping != nil &&
		cfg.SiteConfiguration.PermissionsUserMapping.Enabled {
		allowAccessByDefault = false
		if len(providers) > 0 {
			serviceTypes := make([]string, len(providers))
			for i := range providers {
				serviceTypes[i] = strconv.Quote(providers[i].ServiceType())
			}
			msg := fmt.Sprintf(
				"The permissions user mapping (site configuration `permissions.userMapping`) cannot be enabled when %s authorization providers are in use. Blocking access to all repositories until the conflict is resolved.",
				strings.Join(serviceTypes, ", "))
			seriousProblems = append(seriousProblems, msg)
		}
	}

	return allowAccessByDefault, providers, seriousProblems, warnings
}
