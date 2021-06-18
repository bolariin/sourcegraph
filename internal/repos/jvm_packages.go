package repos

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/jvmpackages/coursier"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A JvmPackagesSource yields depots from a single Maven connection configured
// in Sourcegraph via the external services configuration.
type JvmPackagesSource struct {
	svc    *types.ExternalService
	config *schema.JvmPackagesConnection
}

// NewJvmPackagesSource returns a new MavenSource from the given external
// service.
func NewJvmPackagesSource(svc *types.ExternalService) (*JvmPackagesSource, error) {
	var c schema.JvmPackagesConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, fmt.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newJvmPackagesSource(svc, &c)
}

func newJvmPackagesSource(svc *types.ExternalService, c *schema.JvmPackagesConnection) (*JvmPackagesSource, error) {
	return &JvmPackagesSource{
		svc:    svc,
		config: c,
	}, nil
}

// ListRepos returns all Maven artifacts accessible to all connections
// configured in Sourcegraph via the external services configuration.
func (s *JvmPackagesSource) ListRepos(ctx context.Context, results chan SourceResult) {
	s.listDependentRepos(ctx, results)
}

func (s *JvmPackagesSource) listDependentRepos(ctx context.Context, results chan SourceResult) {
	modules, err := MavenModules(*s.config)
	if err != nil {
		results <- SourceResult{Err: err}
		return
	}
	for _, module := range modules {
		repo := s.makeRepo(module)
		results <- SourceResult{
			Source: s,
			Repo:   repo,
		}
	}
}

func (s *JvmPackagesSource) GetRepo(ctx context.Context, artifactPath string) (*types.Repo, error) {
	module, err := reposource.ParseMavenModule(artifactPath)
	if err != nil {
		return nil, err
	}

	dependencies, err := MavenDependencies(*s.config)
	if err != nil {
		return nil, err
	}

	for _, dep := range dependencies {
		if dep.Module == module {
			exists, err := coursier.Exists(ctx, s.config, dep)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, &mavenArtifactNotFound{
					dependency: dep,
				}
			}
		}
	}

	return s.makeRepo(module), nil
}

type mavenArtifactNotFound struct {
	dependency reposource.Dependency
}

func (mavenArtifactNotFound) NotFound() bool {
	return true
}

func (e *mavenArtifactNotFound) Error() string {
	return fmt.Sprintf("not found: maven dependency '%v'", e.dependency)
}

func (s *JvmPackagesSource) makeRepo(module reposource.Module) *types.Repo {
	urn := s.svc.URN()
	cloneURL := module.CloneURL()
	return &types.Repo{
		Name: module.RepoName(),
		URI:  string(module.RepoName()),
		ExternalRepo: api.ExternalRepoSpec{
			ID:          string(module.RepoName()),
			ServiceID:   extsvc.TypeJvmPackages,
			ServiceType: extsvc.TypeJvmPackages,
		},
		Private: false,
		Sources: map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: cloneURL,
			},
		},
		Metadata: &jvmpackages.Metadata{
			Module: module,
		},
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s *JvmPackagesSource) ExternalServices() types.ExternalServices {
	return types.ExternalServices{s.svc}
}

func MavenDependencies(connection schema.JvmPackagesConnection) (dependencies []reposource.Dependency, err error) {
	for _, dep := range connection.Maven.Artifacts {
		dependency, err := reposource.ParseMavenDependency(dep)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

func MavenModules(connection schema.JvmPackagesConnection) ([]reposource.Module, error) {
	isAdded := make(map[reposource.Module]bool)
	modules := []reposource.Module{}
	dependencies, err := MavenDependencies(connection)
	if err != nil {
		return nil, err
	}
	for _, dep := range dependencies {
		module := dep.Module
		if _, added := isAdded[module]; !added {
			modules = append(modules, module)
		}
		isAdded[module] = true
	}
	return modules, nil
}