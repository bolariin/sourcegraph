package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
)

// This file just contains stub GraphQL resolvers and data types for Code Insights which merely
// return an error if not running in enterprise mode. The actual resolvers can be found in
// enterprise/internal/insights/resolvers

// InsightsResolver is the root resolver.
type InsightsResolver interface {
	Insights(ctx context.Context, args *InsightsArgs) (InsightConnectionResolver, error)
	InsightDashboards(ctx context.Context, args *InsightDashboardsArgs) (InsightsDashboardConnectionResolver, error)
}

type InsightsArgs struct {
	Ids *[]graphql.ID
}

type InsightsDataPointResolver interface {
	DateTime() DateTime
	Value() float64
}

type InsightStatusResolver interface {
	TotalPoints() int32
	PendingJobs() int32
	CompletedJobs() int32
	FailedJobs() int32
	BackfillQueuedAt() *DateTime
}

type InsightsPointsArgs struct {
	From             *DateTime
	To               *DateTime
	IncludeRepoRegex *string
	ExcludeRepoRegex *string
}

type InsightSeriesResolver interface {
	Label() string
	Points(ctx context.Context, args *InsightsPointsArgs) ([]InsightsDataPointResolver, error)
	Status(ctx context.Context) (InsightStatusResolver, error)
	DirtyMetadata(ctx context.Context) ([]InsightDirtyQueryResolver, error)
}

type InsightResolver interface {
	Title() string
	Description() string
	Series() []InsightSeriesResolver
	ID() string
}

type InsightConnectionResolver interface {
	Nodes(ctx context.Context) ([]InsightResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type InsightDirtyQueryResolver interface {
	Reason(ctx context.Context) string
	Time(ctx context.Context) DateTime
	Count(ctx context.Context) int32
}

type InsightDashboardsArgs struct {
	First *int32
	After *string
}

type InsightsDashboardConnectionResolver interface {
	Nodes(ctx context.Context) ([]InsightDashboardResolver, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type InsightDashboardResolver interface {
	Title() string
	ID() graphql.ID
	Views() InsightViewConnectionResolver
}

type InsightViewConnectionResolver interface {
	Nodes(ctx context.Context) ([]InsightViewResolver, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

type InsightViewResolver interface {
	ID() graphql.ID
	// Until this interface becomes uniquely identifyable in the node resolvers
	// ToXX type guard methods, we need _something_ that makes this interface
	// not match any other Node implementing type.
	VeryUniqueResolver() bool
}
