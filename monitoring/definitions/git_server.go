package definitions

import (
	"time"

	"github.com/grafana-tools/sdk"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func GitServer() *monitoring.Container {
	const containerName = "gitserver"

	gitserverHighMemoryNoAlertTransformer := func(observable shared.Observable) shared.Observable {
		return observable.WithNoAlerts(`Git Server is expected to use up all the memory it is provided.`)
	}

	provisioningIndicatorsOptions := &shared.ContainerProvisioningIndicatorsGroupOptions{
		LongTermMemoryUsage:  gitserverHighMemoryNoAlertTransformer,
		ShortTermMemoryUsage: gitserverHighMemoryNoAlertTransformer,
	}

	return &monitoring.Container{
		Name:        "gitserver",
		Title:       "Git Server",
		Description: "Stores, manages, and operates Git repositories.",
		Templates: []sdk.TemplateVar{
			{
				Label:      "Shard",
				Name:       "shard",
				Type:       "query",
				Datasource: monitoring.StringPtr("Prometheus"),
				Query:      "label_values(src_gitserver_exec_running, instance)",
				Multi:      true,
				Refresh:    sdk.BoolInt{Flag: true, Value: monitoring.Int64Ptr(2)}, // Refresh on time range change
				Sort:       3,
				IncludeAll: true,
				AllValue:   ".*",
				Current:    sdk.Current{Text: &sdk.StringSliceString{Value: []string{"all"}, Valid: true}, Value: "$__all"},
			},
		},
		Groups: []monitoring.Group{
			{
				Title: "General",
				Rows: []monitoring.Row{
					{
						{
							Name:        "memory_working_set",
							Description: "memory working set",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (container_memory_working_set_bytes{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"})",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.Bytes).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
						{
							Name:        "go_routines",
							Description: "go routines",
							Query:       "go_goroutines{app=\"gitserver\", instance=~\"${shard:regex}\"}",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{instance}}").With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
					},
					{
						{
							Name:        "cpu_throttling_time",
							Description: "container CPU throttling time %",
							Query:       "sum by (container_label_io_kubernetes_pod_name) ((rate(container_cpu_cfs_throttled_periods_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]) / rate(container_cpu_cfs_periods_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m])) * 100)",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.Percentage).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
						{
							Name:        "cpu_usage_seconds",
							Description: "cpu usage seconds",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (rate(container_cpu_usage_seconds_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
					},
					{
						{
							Name:        "disk_space_remaining",
							Description: "disk space remaining by instance",
							Query:       `(src_gitserver_disk_space_available / src_gitserver_disk_space_total) * 100`,
							Warning:     monitoring.Alert().LessOrEqual(25, nil),
							Critical:    monitoring.Alert().LessOrEqual(15, nil),
							Panel: monitoring.Panel().LegendFormat("{{instance}}").Unit(monitoring.Percentage).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **Provision more disk space:** Sourcegraph will begin deleting least-used repository clones at 10% disk space remaining which may result in decreased performance, users having to wait for repositories to clone, etc.
							`,
						},
					},
					{
						{
							Name:        "io_reads_total",
							Description: "i/o reads total",
							Query:       "sum by (container_label_io_kubernetes_container_name) (rate(container_fs_reads_total{container_label_io_kubernetes_container_name=\"gitserver\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.ReadsPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
						{
							Name:        "io_writes_total",
							Description: "i/o writes total",
							Query:       "sum by (container_label_io_kubernetes_container_name) (rate(container_fs_writes_total{container_label_io_kubernetes_container_name=\"gitserver\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.WritesPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
					},
					{
						{
							Name:        "io_reads",
							Description: "i/o reads",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (rate(container_fs_reads_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.ReadsPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
						{
							Name:        "io_writes",
							Description: "i/o writes",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (rate(container_fs_writes_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.WritesPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
					},
					{
						{
							Name:        "io_read_througput",
							Description: "i/o read throughput",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (rate(container_fs_reads_bytes_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.ReadsPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
						{
							Name:        "io_write_throughput",
							Description: "i/o write throughput",
							Query:       "sum by (container_label_io_kubernetes_pod_name) (rate(container_fs_writes_bytes_total{container_label_io_kubernetes_container_name=\"gitserver\", container_label_io_kubernetes_pod_name=~\"${shard:regex}\"}[5m]))",
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{container_label_io_kubernetes_pod_name}}").Unit(monitoring.WritesPerSecond).With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
						`,
						},
					},
					{
						{
							Name:        "running_git_commands",
							Description: "git commands running on each gitserver instance",
							Query:       "sum by (instance, cmd) (src_gitserver_exec_running{instance=~\"${shard:regex}\"})",
							Warning:     monitoring.Alert().GreaterOrEqual(50, nil).For(2 * time.Minute),
							Critical:    monitoring.Alert().GreaterOrEqual(100, nil).For(5 * time.Minute),
							Panel: monitoring.Panel().LegendFormat("{{instance}} {{cmd}}").With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
								A high value signals load.
							`,
							PossibleSolutions: `
								- **Check if the problem may be an intermittent and temporary peak** using the "Container monitoring" section at the bottom of the Git Server dashboard.
								- **Single container deployments:** Consider upgrading to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
								- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
							`,
						},
						{
							Name:           "git_commands_received",
							Description:    "rate of git commands received across all instances",
							Query:          "sum by (cmd) (rate(src_gitserver_exec_duration_seconds_count[5m]))",
							NoAlert:        true,
							Interpretation: "per second rate per command across all instances",
							Panel: monitoring.Panel().LegendFormat("{{cmd}}").With(func(o monitoring.Observable, p *sdk.Panel) {
								p.GraphPanel.Legend.RightSide = true
							}),
							Owner: monitoring.ObservableOwnerCoreApplication,
						},
					},
					{
						{
							Name:        "repository_clone_queue_size",
							Description: "repository clone queue size",
							Query:       "sum(src_gitserver_clone_queue)",
							Warning:     monitoring.Alert().GreaterOrEqual(25, nil),
							Panel:       monitoring.Panel().LegendFormat("queue size"),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **If you just added several repositories**, the warning may be expected.
								- **Check which repositories need cloning**, by visiting e.g. https://sourcegraph.example.com/site-admin/repositories?filter=not-cloned
							`,
						},
						{
							Name:        "repository_existence_check_queue_size",
							Description: "repository existence check queue size",
							Query:       "sum(src_gitserver_lsremote_queue)",
							Warning:     monitoring.Alert().GreaterOrEqual(25, nil),
							Panel:       monitoring.Panel().LegendFormat("queue size"),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- **Check the code host status indicator for errors:** on the Sourcegraph app homepage, when signed in as an admin click the cloud icon in the top right corner of the page.
								- **Check if the issue continues to happen after 30 minutes**, it may be temporary.
								- **Check the gitserver logs for more information.**
							`,
						},
					},
					{
						{
							Name:        "echo_command_duration_test",
							Description: "echo test command duration",
							Query:       "max(src_gitserver_echo_duration_seconds)",
							NoAlert:     true,
							Panel:       monitoring.Panel().LegendFormat("running commands").Unit(monitoring.Seconds),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							Interpretation: `
								A high value here likely indicates a problem, especially if consistently high.
								You can query for individual commands using 'sum by (cmd)(src_gitserver_exec_running)' in Grafana ('/-/debug/grafana') to see if a specific Git Server command might be spiking in frequency.

								If this value is consistently high, consider the following:

								- **Single container deployments:** Upgrade to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
								- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
							`,
						},
						shared.FrontendInternalAPIErrorResponses("gitserver", monitoring.ObservableOwnerCoreApplication).Observable(),
					},
				},
			},
			{
				Title:  "Gitserver cleanup jobs",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:           "janitor_running",
							Description:    "if the janitor process is running",
							Query:          "max by (instance) (src_gitserver_janitor_running)",
							NoAlert:        true,
							Panel:          monitoring.Panel().LegendFormat("janitor process running").Unit(monitoring.Number),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							Interpretation: "1, if the janitor process is currently running",
						},
					},
					{
						{
							Name:           "janitor_job_duration",
							Description:    "95th percentile job run duration",
							Query:          "histogram_quantile(0.95, sum(rate(src_gitserver_janitor_job_duration_seconds_bucket[5m])) by (le, job_name))",
							NoAlert:        true,
							Panel:          monitoring.Panel().LegendFormat("{{job_name}}").Unit(monitoring.Seconds),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							Interpretation: "95th percentile job run duration",
						},
					},
					{
						{
							Name:           "repos_removed",
							Description:    "repositories removed due to disk pressure",
							Query:          "sum by (instance) (rate(src_gitserver_repos_removed_disk_pressure[5m]))",
							NoAlert:        true,
							Panel:          monitoring.Panel().LegendFormat("{{instance}}").Unit(monitoring.Number),
							Owner:          monitoring.ObservableOwnerCoreApplication,
							Interpretation: "Repositories removed due to disk pressure",
						},
					},
				},
			},

			shared.CodeIntelligence.NewCoursierGroup(containerName),

			shared.NewDatabaseConnectionsMonitoringGroup(containerName),
			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerCoreApplication, provisioningIndicatorsOptions),
			shared.NewGolangMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
			shared.NewKubernetesMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
		},
	}
}
