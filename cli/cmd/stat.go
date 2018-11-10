package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/linkerd/linkerd2/controller/api/util"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type statOptions struct {
	namespace     string
	timeWindow    string
	toNamespace   string
	toResource    string
	fromNamespace string
	fromResource  string
	allNamespaces bool
	outputFormat  string
}

func newStatOptions() *statOptions {
	return &statOptions{
		namespace:     "default",
		timeWindow:    "1m",
		toNamespace:   "",
		toResource:    "",
		fromNamespace: "",
		fromResource:  "",
		allNamespaces: false,
		outputFormat:  "",
	}
}

func newCmdStat() *cobra.Command {
	options := newStatOptions()

	cmd := &cobra.Command{
		Use:   "stat [flags] (RESOURCES)",
		Short: "Display traffic stats about one or many resources",
		Long: `Display traffic stats about one or many resources.

  The RESOURCES argument specifies the target resource(s) to aggregate stats over:
  (TYPE [NAME] | TYPE/NAME)

  Examples:
  * deploy
  * deploy/my-deploy
  * rc/my-replication-controller
  * ns/my-ns
  * authority
  * au/my-authority
  * all

Valid resource types include:

  * deployments
  * namespaces
  * pods
  * replicationcontrollers
  * authorities (not supported in --from)
  * services (only supported if a --from is also specified, or as a --to)
  * all (all resource types, not supported in --from or --to)

This command will hide resources that have completed, such as pods that are in the Succeeded or Failed phases.
If no resource name is specified, displays stats about all resources of the specified RESOURCETYPE`,
		Example: `  # Get all deployments in the test namespace.
  linkerd stat deployments -n test

  # Get the hello1 replication controller in the test namespace.
  linkerd stat replicationcontrollers hello1 -n test

  # Get all namespaces.
  linkerd stat namespaces

  # Get all inbound stats to the web deployment.
  linkerd stat deploy/web

  # Get all pods in all namespaces that call the hello1 deployment in the test namesapce.
  linkerd stat pods --to deploy/hello1 --to-namespace test --all-namespaces

  # Get all pods in all namespaces that call the hello1 service in the test namesapce.
  linkerd stat pods --to svc/hello1 --to-namespace test --all-namespaces

  # Get all services in all namespaces that receive calls from hello1 deployment in the test namespace.
  linkerd stat services --from deploy/hello1 --from-namespace test --all-namespaces

  # Get all namespaces that receive traffic from the default namespace.
  linkerd stat namespaces --from ns/default

  # Get all inbound stats to the test namespace.
  linkerd stat ns/test`,
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: util.ValidTargets,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildStatSummaryRequest(args, options)
			if err != nil {
				return fmt.Errorf("error creating metrics request while making stats request: %v", err)
			}

			output, err := requestStatsFromAPI(validatedPublicAPIClient(time.Time{}), req, options)
			if err != nil {
				return err
			}

			_, err = fmt.Print(output)

			return err
		},
	}

	cmd.PersistentFlags().StringVarP(&options.namespace, "namespace", "n", options.namespace, "Namespace of the specified resource")
	cmd.PersistentFlags().StringVarP(&options.timeWindow, "time-window", "t", options.timeWindow, "Stat window (for example: \"10s\", \"1m\", \"10m\", \"1h\")")
	cmd.PersistentFlags().StringVar(&options.toResource, "to", options.toResource, "If present, restricts outbound stats to the specified resource name")
	cmd.PersistentFlags().StringVar(&options.toNamespace, "to-namespace", options.toNamespace, "Sets the namespace used to lookup the \"--to\" resource; by default the current \"--namespace\" is used")
	cmd.PersistentFlags().StringVar(&options.fromResource, "from", options.fromResource, "If present, restricts outbound stats from the specified resource name")
	cmd.PersistentFlags().StringVar(&options.fromNamespace, "from-namespace", options.fromNamespace, "Sets the namespace used from lookup the \"--from\" resource; by default the current \"--namespace\" is used")
	cmd.PersistentFlags().BoolVar(&options.allNamespaces, "all-namespaces", options.allNamespaces, "If present, returns stats across all namespaces, ignoring the \"--namespace\" flag")
	cmd.PersistentFlags().StringVarP(&options.outputFormat, "output", "o", options.outputFormat, "Output format; currently only \"table\" (default) and \"json\" are supported")

	return cmd
}

func requestStatsFromAPI(client pb.ApiClient, req *pb.StatSummaryRequest, options *statOptions) (string, error) {
	resp, err := client.StatSummary(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("StatSummary API error: %v", err)
	}
	if e := resp.GetError(); e != nil {
		return "", fmt.Errorf("StatSummary API response error: %v", e.Error)
	}

	return renderStats(resp, req.Selector.Resource.Type, options), nil
}

func renderStats(resp *pb.StatSummaryResponse, resourceType string, options *statOptions) string {
	var buffer bytes.Buffer
	w := tabwriter.NewWriter(&buffer, 0, 0, padding, ' ', tabwriter.AlignRight)
	writeStatsToBuffer(resp, resourceType, w, options)
	w.Flush()

	var out string
	switch options.outputFormat {
	case "table", "":
		// strip left padding on the first column
		out = string(buffer.Bytes()[padding:])
		out = strings.Replace(out, "\n"+strings.Repeat(" ", padding), "\n", -1)
	case "json":
		out = string(buffer.Bytes())
	}

	return out
}

const padding = 3

type rowStats struct {
	requestRate float64
	successRate float64
	tlsPercent  float64
	latencyP50  uint64
	latencyP95  uint64
	latencyP99  uint64
}

type row struct {
	meshed string
	*rowStats
}

var (
	nameHeader      = "NAME"
	namespaceHeader = "NAMESPACE"
)

func writeStatsToBuffer(resp *pb.StatSummaryResponse, reqResourceType string, w *tabwriter.Writer, options *statOptions) {
	maxNameLength := len(nameHeader)
	maxNamespaceLength := len(namespaceHeader)
	statTables := make(map[string]map[string]*row)

	for _, statTable := range resp.GetOk().StatTables {
		table := statTable.GetPodGroup()

		for _, r := range table.Rows {
			name := r.Resource.Name
			nameWithPrefix := name
			if reqResourceType == k8s.All {
				nameWithPrefix = getNamePrefix(r.Resource.Type) + nameWithPrefix
			}

			namespace := r.Resource.Namespace
			key := fmt.Sprintf("%s/%s", namespace, name)
			resourceKey := r.Resource.Type

			if _, ok := statTables[resourceKey]; !ok {
				statTables[resourceKey] = make(map[string]*row)
			}

			if len(nameWithPrefix) > maxNameLength {
				maxNameLength = len(nameWithPrefix)
			}

			if len(namespace) > maxNamespaceLength {
				maxNamespaceLength = len(namespace)
			}

			meshedCount := fmt.Sprintf("%d/%d", r.MeshedPodCount, r.RunningPodCount)
			if resourceKey == k8s.Authority {
				meshedCount = "-"
			}
			statTables[resourceKey][key] = &row{
				meshed: meshedCount,
			}

			if r.Stats != nil {
				statTables[resourceKey][key].rowStats = &rowStats{
					requestRate: getRequestRate(*r),
					successRate: getSuccessRate(*r),
					tlsPercent:  getPercentTls(*r),
					latencyP50:  r.Stats.LatencyMsP50,
					latencyP95:  r.Stats.LatencyMsP95,
					latencyP99:  r.Stats.LatencyMsP99,
				}
			}
		}
	}

	var resourceTypes []string
	switch reqResourceType {
	case k8s.All:
		resourceTypes = k8s.StatAllResourceTypes
	default:
		resourceTypes = []string{reqResourceType}
	}

	switch options.outputFormat {
	case "table", "":
		if len(statTables) == 0 {
			fmt.Fprintln(os.Stderr, "No traffic found.")
			os.Exit(0)
		}
		printStatTables(statTables, resourceTypes, w, maxNameLength, maxNamespaceLength, options)
	case "json":
		printStatJson(statTables, resourceTypes, w)
	}
}

func printStatTables(statTables map[string]map[string]*row, resourceTypes []string, w *tabwriter.Writer, maxNameLength int, maxNamespaceLength int, options *statOptions) {
	firstDisplayedStat := true // don't print a newline before the first stat
	for _, resourceType := range resourceTypes {
		if stats, ok := statTables[resourceType]; ok {
			if !firstDisplayedStat {
				fmt.Fprint(w, "\n")
			}
			firstDisplayedStat = false
			resourceTypeLabel := resourceType
			if len(resourceTypes) == 1 {
				resourceTypeLabel = ""
			}
			printSingleStatTable(stats, resourceTypeLabel, w, maxNameLength, maxNamespaceLength, options)
		}
	}
}

func printSingleStatTable(stats map[string]*row, resourceType string, w *tabwriter.Writer, maxNameLength int, maxNamespaceLength int, options *statOptions) {
	headers := make([]string, 0)
	if options.allNamespaces {
		headers = append(headers,
			namespaceHeader+strings.Repeat(" ", maxNamespaceLength-len(namespaceHeader)))
	}
	headers = append(headers, []string{
		nameHeader + strings.Repeat(" ", maxNameLength-len(nameHeader)),
		"MESHED",
		"SUCCESS",
		"RPS",
		"LATENCY_P50",
		"LATENCY_P95",
		"LATENCY_P99",
		"TLS\t", // trailing \t is required to format last column
	}...)

	fmt.Fprintln(w, strings.Join(headers, "\t"))

	sortedKeys := sortStatsKeys(stats)
	for _, key := range sortedKeys {
		namespace, name := namespaceName(resourceType, key)
		values := make([]interface{}, 0)
		templateString := "%s\t%s\t%.2f%%\t%.1frps\t%dms\t%dms\t%dms\t%.f%%\t\n"
		templateStringEmpty := "%s\t%s\t-\t-\t-\t-\t-\t-\t\n"

		if options.allNamespaces {
			values = append(values,
				namespace+strings.Repeat(" ", maxNamespaceLength-len(namespace)))
			templateString = "%s\t" + templateString
			templateStringEmpty = "%s\t" + templateStringEmpty
		}
		values = append(values, []interface{}{
			name + strings.Repeat(" ", maxNameLength-len(name)),
			stats[key].meshed,
		}...)

		if stats[key].rowStats != nil {
			values = append(values, []interface{}{
				stats[key].successRate * 100,
				stats[key].requestRate,
				stats[key].latencyP50,
				stats[key].latencyP95,
				stats[key].latencyP99,
				stats[key].tlsPercent * 100,
			}...)

			fmt.Fprintf(w, templateString, values...)
		} else {
			fmt.Fprintf(w, templateStringEmpty, values...)
		}
	}
}

func namespaceName(resourceType string, key string) (string, string) {
	parts := strings.Split(key, "/")
	namespace := parts[0]
	namePrefix := getNamePrefix(resourceType)
	name := namePrefix + parts[1]
	return namespace, name
}

// Using pointers there where the value is NA and the corresponding json is null
type jsonStats struct {
	Namespace    string   `json:"namespace"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Meshed       string   `json:"meshed"`
	Success      *float64 `json:"success"`
	Rps          *float64 `json:"rps"`
	LatencyMSp50 *uint64  `json:"latency_ms_p50"`
	LatencyMSp95 *uint64  `json:"latency_ms_p95"`
	LatencyMSp99 *uint64  `json:"latency_ms_p99"`
	Tls          *float64 `json:"tls"`
}

func printStatJson(statTables map[string]map[string]*row, resourceTypes []string, w *tabwriter.Writer) {
	// avoid nil initialization so that if there are not stats it gets marshalled as an empty array vs null
	entries := []*jsonStats{}
	for _, resourceType := range resourceTypes {
		if stats, ok := statTables[resourceType]; ok {
			sortedKeys := sortStatsKeys(stats)
			for _, key := range sortedKeys {
				namespace, name := namespaceName("", key)
				entry := &jsonStats{
					Namespace: namespace,
					Kind:      resourceType,
					Name:      name,
					Meshed:    stats[key].meshed,
				}
				if stats[key].rowStats != nil {
					entry.Success = &stats[key].successRate
					entry.Rps = &stats[key].requestRate
					entry.LatencyMSp50 = &stats[key].latencyP50
					entry.LatencyMSp95 = &stats[key].latencyP95
					entry.LatencyMSp99 = &stats[key].latencyP99
					entry.Tls = &stats[key].tlsPercent
				}

				entries = append(entries, entry)
			}
		}
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Fprintf(w, "%s\n", b)
}

func getNamePrefix(resourceType string) string {
	if resourceType == "" {
		return ""
	} else {
		canonicalType := k8s.ShortNameFromCanonicalResourceName(resourceType)
		return canonicalType + "/"
	}
}

func buildStatSummaryRequest(resources []string, options *statOptions) (*pb.StatSummaryRequest, error) {
	targets, err := util.BuildResources(options.namespace, resources)
	if err != nil {
		return nil, err
	}

	err = options.validate(targets[0].Type)
	if err != nil {
		return nil, err
	}

	var toRes, fromRes pb.Resource
	if options.toResource != "" {
		toRes, err = util.BuildResource(options.toNamespace, options.toResource)
		if err != nil {
			return nil, err
		}
	}
	if options.fromResource != "" {
		fromRes, err = util.BuildResource(options.fromNamespace, options.fromResource)
		if err != nil {
			return nil, err
		}
	}

	requestParams := util.StatSummaryRequestParams{
		TimeWindow:    options.timeWindow,
		ResourceName:  targets[0].Name,
		ResourceType:  targets[0].Type,
		Namespace:     options.namespace,
		ToName:        toRes.Name,
		ToType:        toRes.Type,
		ToNamespace:   options.toNamespace,
		FromName:      fromRes.Name,
		FromType:      fromRes.Type,
		FromNamespace: options.fromNamespace,
		AllNamespaces: options.allNamespaces,
	}

	return util.BuildStatSummaryRequest(requestParams)
}

func getRequestRate(r pb.StatTable_PodGroup_Row) float64 {
	success := r.Stats.SuccessCount
	failure := r.Stats.FailureCount
	windowLength, err := time.ParseDuration(r.TimeWindow)
	if err != nil {
		log.Error(err.Error())
		return 0.0
	}
	return float64(success+failure) / windowLength.Seconds()
}

func getSuccessRate(r pb.StatTable_PodGroup_Row) float64 {
	success := r.Stats.SuccessCount
	failure := r.Stats.FailureCount

	if success+failure == 0 {
		return 0.0
	}
	return float64(success) / float64(success+failure)
}

func getPercentTls(r pb.StatTable_PodGroup_Row) float64 {
	reqTotal := r.Stats.SuccessCount + r.Stats.FailureCount
	if reqTotal == 0 {
		return 0.0
	}
	return float64(r.Stats.TlsRequestCount) / float64(reqTotal)
}

func sortStatsKeys(stats map[string]*row) []string {
	var sortedKeys []string
	for key := range stats {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}

// validate performs all validation on the command-line options.
// It returns the first error encountered, or `nil` if the options are valid.
func (o *statOptions) validate(resourceType string) error {
	err := o.validateConflictingFlags()
	if err != nil {
		return err
	}

	if resourceType == k8s.Namespace {
		err := o.validateNamespaceFlags()
		if err != nil {
			return err
		}
	}

	if err := o.validateOutputFormat(); err != nil {
		return err
	}

	return nil
}

// validateConflictingFlags validates that the options do not contain mutually
// exclusive flags.
func (o *statOptions) validateConflictingFlags() error {
	if o.toResource != "" && o.fromResource != "" {
		return fmt.Errorf("--to and --from flags are mutually exclusive")
	}

	if o.toNamespace != "" && o.fromNamespace != "" {
		return fmt.Errorf("--to-namespace and --from-namespace flags are mutually exclusive")
	}

	return nil
}

// validateNamespaceFlags performs additional validation for options when the target
// resource type is a namespace.
func (o *statOptions) validateNamespaceFlags() error {
	if o.toNamespace != "" {
		return fmt.Errorf("--to-namespace flag is incompatible with namespace resource type")
	}

	if o.fromNamespace != "" {
		return fmt.Errorf("--from-namespace flag is incompatible with namespace resource type")
	}

	// Note: technically, this allows you to say `stat ns --namespace default`, but that
	// seems like an edge case.
	if o.namespace != "default" {
		return fmt.Errorf("--namespace flag is incompatible with namespace resource type")
	}

	return nil
}

func (o *statOptions) validateOutputFormat() error {
	switch o.outputFormat {
	case "table", "json", "":
		return nil
	default:
		return fmt.Errorf("--output currently only supports table and json")
	}
}
