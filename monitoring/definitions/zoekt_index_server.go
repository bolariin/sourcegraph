package definitions

import (
	"time"

	"github.com/grafana-tools/sdk"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func ZoektIndexServer() *monitoring.Container {
	const (
		containerName        = "zoekt-indexserver"
		bundledContainerName = "indexed-search"
	)

	return &monitoring.Container{
		Name: "zoekt-indexserver",

		Title:                    "Zoekt Index Server",
		Description:              "Indexes repositories and populates the search index.",
		NoSourcegraphDebugServer: true,
		Groups: []monitoring.Group{
			{
				Title: "General",
				Rows: []monitoring.Row{
					{
						{
							Name:        "repos_assigned",
							Description: "total number of repos",
							Query:       `sum(index_num_assigned)`,
							NoAlert:     true,
							Panel: monitoring.Panel().With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.Current = true
								p.GraphPanel.Targets = []sdk.Target{{
									Expr:         o.Query,
									LegendFormat: "assigned",
								}, {
									Expr:         `sum(index_num_indexed)`,
									LegendFormat: "indexed",
								}}
								p.GraphPanel.Tooltip.Shared = true
							}),
							Owner:          monitoring.ObservableOwnerSearchCore,
							Interpretation: "Sudden changes should be caused by indexing configuration changes.",
						},
					},
					{
						{
							Name:        "repo_index_state",
							Description: "indexing results over 5m (noop=no changes, empty=no branches to index)",
							Query:       `sum by (state) (increase(index_repo_seconds_count[5m]))`,
							NoAlert:     true,
							Owner:       monitoring.ObservableOwnerSearchCore,
							Panel: monitoring.Panel().LegendFormat("{{state}}").With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Yaxes[0].LogBase = 2  // log to show the huge number of "noop" or "empty"
								p.GraphPanel.Tooltip.Shared = true // show multiple lines simultaneously
							}),
							Interpretation: "A persistent failing state indicates some repositories cannot be indexed, perhaps due to size and timeouts.",
						},
						{
							Name:        "repo_index_success_speed",
							Description: "successful indexing durations",
							Query:       `sum by (le, state) (increase(index_repo_seconds_bucket{state="success"}[$__rate_interval]))`,
							NoAlert:     true,
							Panel: monitoring.PanelHeatmap().With(func(o monitoring.Observable, p *sdk.Panel) {
								p.HeatmapPanel.YAxis.Format = string(monitoring.Seconds)
							}),
							Owner:          monitoring.ObservableOwnerSearchCore,
							Interpretation: "Latency increases can indicate bottlenecks in the indexserver.",
						},
						{
							Name:        "repo_index_fail_speed",
							Description: "failed indexing durations",
							Query:       `sum by (le, state) (increase(index_repo_seconds_bucket{state="fail"}[$__rate_interval]))`,
							NoAlert:     true,
							Panel: monitoring.PanelHeatmap().With(func(o monitoring.Observable, p *sdk.Panel) {
								p.HeatmapPanel.YAxis.Format = string(monitoring.Seconds)
							}),
							Owner:          monitoring.ObservableOwnerSearchCore,
							Interpretation: "Failures happening after a long time indicates timeouts.",
						},
					},
					{
						{
							Name:              "average_resolve_revision_duration",
							Description:       "average resolve revision duration over 5m",
							Query:             `sum(rate(resolve_revision_seconds_sum[5m])) / sum(rate(resolve_revision_seconds_count[5m]))`,
							Warning:           monitoring.Alert().GreaterOrEqual(15, nil),
							Critical:          monitoring.Alert().GreaterOrEqual(30, nil),
							Panel:             monitoring.Panel().LegendFormat("{{duration}}").Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerSearchCore,
							PossibleSolutions: "none",
						},
						{
							Name:        "get_index_options_error_increase",
							Description: "the number of repositories we failed to get indexing options over 5m",
							Query:       `sum(increase(get_index_options_error_total[5m]))`,
							// This value can spike, so only if we have a
							// sustained error rate do we alert.
							Warning:  monitoring.Alert().GreaterOrEqual(100, nil).For(time.Minute),
							Critical: monitoring.Alert().GreaterOrEqual(100, nil).For(10 * time.Minute),
							Panel:    monitoring.Panel().Min(0),
							Owner:    monitoring.ObservableOwnerSearchCore,
							PossibleSolutions: `
								- View error rates on gitserver and frontend to identify root cause.
								- Rollback frontend/gitserver deployment if due to a bad code change.
								- View error logs for 'getIndexOptions' via net/trace debug interface. For example click on a 'indexed-search-indexer-' on https://sourcegraph.com/-/debug/. Then click on Traces. Replace sourcegraph.com with your instance address.
							`,
							Interpretation: `
								When considering indexing a repository we ask for the index configuration
								from frontend per repository. The most likely reason this would fail is
								failing to resolve branch names to git SHAs.

								This value can spike up during deployments/etc. Only if you encounter
								sustained periods of errors is there an underlying issue. When sustained
								this indicates repositories will not get updated indexes.
							`,
						},
					},
				},
			},

			// Note:
			// zoekt_indexserver and zoekt_webserver are deployed together as part of the indexed-search service
			// We show pod availability here for both the webserver and indexserver as they are bundled together.

			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerSearchCore, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerSearchCore, nil),
			shared.NewKubernetesMonitoringGroup(bundledContainerName, monitoring.ObservableOwnerSearchCore, nil),
		},
	}
}
