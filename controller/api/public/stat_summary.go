package public

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/prometheus/common/model"
	"github.com/runconduit/conduit/controller/api/util"
	pb "github.com/runconduit/conduit/controller/gen/public"
	"github.com/runconduit/conduit/pkg/k8s"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type promType string
type promResult struct {
	prom promType
	vec  model.Vector
	err  error
}

type resourceResult struct {
	res *pb.StatTable
	err error
}

const (
	reqQuery             = "sum(increase(response_total%s[%s])) by (%s, classification, tls)"
	latencyQuantileQuery = "histogram_quantile(%s, sum(irate(response_latency_ms_bucket%s[%s])) by (le, %s))"

	promRequests   = promType("QUERY_REQUESTS")
	promLatencyP50 = promType("0.5")
	promLatencyP95 = promType("0.95")
	promLatencyP99 = promType("0.99")

	namespaceLabel    = model.LabelName("namespace")
	dstNamespaceLabel = model.LabelName("dst_namespace")
)

var promTypes = []promType{promRequests, promLatencyP50, promLatencyP95, promLatencyP99}

type podStats struct {
	inMesh uint64
	total  uint64
	failed uint64
	errors map[string]*pb.PodErrors
}

func (s *grpcServer) StatSummary(ctx context.Context, req *pb.StatSummaryRequest) (*pb.StatSummaryResponse, error) {

	// check for well-formed request
	if req.GetSelector().GetResource() == nil {
		return statSummaryError(req, "StatSummary request missing Selector Resource"), nil
	}

	// special case to check for services as outbound only
	if isInvalidServiceRequest(req) {
		return statSummaryError(req, "service only supported as a target on 'from' queries, or as a destination on 'to' queries"), nil
	}

	switch req.Outbound.(type) {
	case *pb.StatSummaryRequest_ToResource:
		if req.Outbound.(*pb.StatSummaryRequest_ToResource).ToResource.Type == k8s.All {
			return statSummaryError(req, "resource type 'all' is not supported as a filter"), nil
		}
	case *pb.StatSummaryRequest_FromResource:
		if req.Outbound.(*pb.StatSummaryRequest_FromResource).FromResource.Type == k8s.All {
			return statSummaryError(req, "resource type 'all' is not supported as a filter"), nil
		}
	}

	statTables := make([]*pb.StatTable, 0)

	var resourcesToQuery []string
	if req.Selector.Resource.Type == k8s.All {
		resourcesToQuery = k8s.StatAllResourceTypes
	} else {
		resourcesToQuery = []string{req.Selector.Resource.Type}
	}

	// request stats for the resourcesToQuery, in parallel
	resultChan := make(chan resourceResult)

	for _, resource := range resourcesToQuery {
		statReq := proto.Clone(req).(*pb.StatSummaryRequest)
		statReq.Selector.Resource.Type = resource

		go func() {
			resultChan <- s.resourceQuery(ctx, statReq)
		}()
	}

	for i := 0; i < len(resourcesToQuery); i++ {
		result := <-resultChan
		if result.err != nil {
			return nil, util.GRPCError(result.err)
		}
		statTables = append(statTables, result.res)
	}

	rsp := pb.StatSummaryResponse{
		Response: &pb.StatSummaryResponse_Ok_{ // https://github.com/golang/protobuf/issues/205
			Ok: &pb.StatSummaryResponse_Ok{
				StatTables: statTables,
			},
		},
	}

	return &rsp, nil
}

func statSummaryError(req *pb.StatSummaryRequest, message string) *pb.StatSummaryResponse {
	return &pb.StatSummaryResponse{
		Response: &pb.StatSummaryResponse_Error{
			Error: &pb.ResourceError{
				Resource: req.GetSelector().GetResource(),
				Error:    message,
			},
		},
	}
}

func (s *grpcServer) resourceQuery(ctx context.Context, req *pb.StatSummaryRequest) resourceResult {
	objects, err := s.k8sAPI.GetObjects(req.Selector.Resource.Namespace, req.Selector.Resource.Type, req.Selector.Resource.Name)
	if err != nil {
		return resourceResult{res: nil, err: err}
	}

	// TODO: make these one struct:
	// string => {metav1.ObjectMeta, podCount}
	objectMap := map[string]metav1.Object{}
	podStatsMap := map[string]*podStats{}

	for _, object := range objects {
		key, err := cache.MetaNamespaceKeyFunc(object)
		if err != nil {
			return resourceResult{res: nil, err: err}
		}
		metaObj, err := meta.Accessor(object)
		if err != nil {
			return resourceResult{res: nil, err: err}
		}

		objectMap[key] = metaObj

		podStats, err := s.getPodStats(object)
		if err != nil {
			return resourceResult{res: nil, err: err}
		}
		podStatsMap[key] = podStats
	}

	res, err := s.objectQuery(ctx, req, objectMap, podStatsMap)
	if err != nil {
		return resourceResult{res: nil, err: err}
	}

	return resourceResult{res: res, err: nil}
}

func (s *grpcServer) objectQuery(
	ctx context.Context,
	req *pb.StatSummaryRequest,
	objects map[string]metav1.Object,
	podStats map[string]*podStats,
) (*pb.StatTable, error) {
	rows := make([]*pb.StatTable_PodGroup_Row, 0)

	requestMetrics, err := s.getPrometheusMetrics(ctx, req, req.TimeWindow)
	if err != nil {
		return nil, err
	}

	var keys []string

	if req.GetOutbound() == nil || req.GetNone() != nil {
		// if this request doesn't have outbound filtering, return all rows
		for key := range objects {
			keys = append(keys, key)
		}
	} else {
		// otherwise only return rows for which we have stats
		for key := range requestMetrics {
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		resource, ok := objects[key]
		if !ok {
			continue
		}

		row := pb.StatTable_PodGroup_Row{
			Resource: &pb.Resource{
				Namespace: resource.GetNamespace(),
				Type:      req.Selector.Resource.Type,
				Name:      resource.GetName(),
			},
			TimeWindow: req.TimeWindow,
			Stats:      requestMetrics[key],
		}

		if podStat, ok := podStats[key]; ok {
			row.MeshedPodCount = podStat.inMesh
			row.RunningPodCount = podStat.total
			row.FailedPodCount = podStat.failed
			row.ErrorsByPod = podStat.errors
		}

		rows = append(rows, &row)
	}

	rsp := pb.StatTable{
		Table: &pb.StatTable_PodGroup_{
			PodGroup: &pb.StatTable_PodGroup{
				Rows: rows,
			},
		},
	}

	return &rsp, nil
}

func promLabelNames(resource *pb.Resource) model.LabelNames {
	names := model.LabelNames{namespaceLabel}
	if resource.Type != k8s.Namespaces {
		names = append(names, promResourceType(resource))
	}
	return names
}

func promDstLabelNames(resource *pb.Resource) model.LabelNames {
	names := model.LabelNames{dstNamespaceLabel}
	if resource.Type != k8s.Namespaces {
		names = append(names, "dst_"+promResourceType(resource))
	}
	return names
}

func promLabels(resource *pb.Resource) model.LabelSet {
	set := model.LabelSet{}
	if resource.Name != "" {
		set[promResourceType(resource)] = model.LabelValue(resource.Name)
	}
	if resource.Type != k8s.Namespaces && resource.Namespace != "" {
		set[namespaceLabel] = model.LabelValue(resource.Namespace)
	}
	return set
}

func promDstLabels(resource *pb.Resource) model.LabelSet {
	set := model.LabelSet{}
	if resource.Name != "" {
		set["dst_"+promResourceType(resource)] = model.LabelValue(resource.Name)
	}
	if resource.Type != k8s.Namespaces && resource.Namespace != "" {
		set[dstNamespaceLabel] = model.LabelValue(resource.Namespace)
	}
	return set
}

func promDirectionLabels(direction string) model.LabelSet {
	return model.LabelSet{
		model.LabelName("direction"): model.LabelValue(direction),
	}
}

func promResourceType(resource *pb.Resource) model.LabelName {
	return model.LabelName(k8s.ResourceTypesToProxyLabels[resource.Type])
}

func buildRequestLabels(req *pb.StatSummaryRequest) (labels model.LabelSet, labelNames model.LabelNames) {
	// labelNames: the group by in the prometheus query
	// labels: the labels for the resource we want to query for

	switch out := req.Outbound.(type) {
	case *pb.StatSummaryRequest_ToResource:
		labelNames = promLabelNames(req.Selector.Resource)

		labels = labels.Merge(promDstLabels(out.ToResource))
		labels = labels.Merge(promLabels(req.Selector.Resource))
		labels = labels.Merge(promDirectionLabels("outbound"))

	case *pb.StatSummaryRequest_FromResource:
		labelNames = promDstLabelNames(req.Selector.Resource)
		labels = labels.Merge(promLabels(out.FromResource))
		labels = labels.Merge(promDirectionLabels("outbound"))

	default:
		labelNames = promLabelNames(req.Selector.Resource)
		labels = labels.Merge(promLabels(req.Selector.Resource))
		labels = labels.Merge(promDirectionLabels("inbound"))
	}

	return
}

func (s *grpcServer) getPrometheusMetrics(ctx context.Context, req *pb.StatSummaryRequest, timeWindow string) (map[string]*pb.BasicStats, error) {
	reqLabels, groupBy := buildRequestLabels(req)
	resultChan := make(chan promResult)

	// kick off 4 asynchronous queries: 1 request volume + 3 latency
	go func() {
		// success/failure counts
		requestsQuery := fmt.Sprintf(reqQuery, reqLabels, timeWindow, groupBy)
		resultVector, err := s.queryProm(ctx, requestsQuery)

		resultChan <- promResult{
			prom: promRequests,
			vec:  resultVector,
			err:  err,
		}
	}()

	for _, quantile := range []promType{promLatencyP50, promLatencyP95, promLatencyP99} {
		go func(quantile promType) {
			latencyQuery := fmt.Sprintf(latencyQuantileQuery, quantile, reqLabels, timeWindow, groupBy)
			latencyResult, err := s.queryProm(ctx, latencyQuery)

			resultChan <- promResult{
				prom: quantile,
				vec:  latencyResult,
				err:  err,
			}
		}(quantile)
	}

	// process results, receive one message per prometheus query type
	var err error
	results := []promResult{}
	for i := 0; i < len(promTypes); i++ {
		result := <-resultChan
		if result.err != nil {
			log.Errorf("queryProm failed with: %s", result.err)
			err = result.err
		} else {
			results = append(results, result)
		}
	}
	if err != nil {
		return nil, err
	}

	return processPrometheusMetrics(results, groupBy), nil
}

func processPrometheusMetrics(results []promResult, groupBy model.LabelNames) map[string]*pb.BasicStats {
	basicStats := make(map[string]*pb.BasicStats)

	for _, result := range results {
		for _, sample := range result.vec {
			label := metricToKey(sample.Metric, groupBy)
			if basicStats[label] == nil {
				basicStats[label] = &pb.BasicStats{}
			}

			value := extractSampleValue(sample)

			switch result.prom {
			case promRequests:
				switch string(sample.Metric[model.LabelName("classification")]) {
				case "success":
					basicStats[label].SuccessCount += value
				case "failure":
					basicStats[label].FailureCount += value
				}
				switch string(sample.Metric[model.LabelName("tls")]) {
				case "true":
					basicStats[label].TlsRequestCount += value
				}
			case promLatencyP50:
				basicStats[label].LatencyMsP50 = value
			case promLatencyP95:
				basicStats[label].LatencyMsP95 = value
			case promLatencyP99:
				basicStats[label].LatencyMsP99 = value
			}
		}
	}

	return basicStats
}

func extractSampleValue(sample *model.Sample) uint64 {
	value := uint64(0)
	if !math.IsNaN(float64(sample.Value)) {
		value = uint64(math.Round(float64(sample.Value)))
	}
	return value
}

func metricToKey(metric model.Metric, groupBy model.LabelNames) string {
	// this needs to match keys generated by MetaNamespaceKeyFunc
	values := []string{}
	for _, k := range groupBy {
		// return namespace/resource
		values = append(values, string(metric[k]))
	}
	return strings.Join(values, "/")
}

func (s *grpcServer) getPodStats(obj runtime.Object) (*podStats, error) {
	pods, err := s.k8sAPI.GetPodsFor(obj, true)
	if err != nil {
		return nil, err
	}
	podErrors := make(map[string]*pb.PodErrors)
	meshCount := &podStats{}

	for _, pod := range pods {
		if pod.Status.Phase == apiv1.PodFailed {
			meshCount.failed++
		} else {
			meshCount.total++
			if isInMesh(pod) {
				meshCount.inMesh++
			}
		}

		errors := checkContainerErrors(pod.Status.ContainerStatuses, "conduit-proxy")
		errors = append(errors, checkContainerErrors(pod.Status.InitContainerStatuses, "conduit-init")...)

		if len(errors) > 0 {
			podErrors[pod.Name] = &pb.PodErrors{Errors: errors}
		}
	}
	meshCount.errors = podErrors
	return meshCount, nil
}

func toPodError(message string) *pb.PodErrors_PodError {
	return &pb.PodErrors_PodError{
		Error: &pb.PodErrors_PodError_Unknown_{
			Unknown: &pb.PodErrors_PodError_Unknown{Message: message},
		},
	}
}

func checkContainerErrors(containerStatuses []apiv1.ContainerStatus, containerName string) []*pb.PodErrors_PodError {
	errors := []*pb.PodErrors_PodError{}
	for _, st := range containerStatuses {
		if st.Name == containerName && st.State.Waiting != nil {
			errors = append(errors, toPodError(fmt.Sprintf("[%s] container has not started. %s", st.Name, st.State.Waiting.Message)))

			if st.LastTerminationState.Waiting != nil {
				errors = append(errors, toPodError(fmt.Sprintf("[%s] [image: %s] %s", st.Name, st.Image, st.LastTerminationState.Waiting.Message)))
			}

			if st.LastTerminationState.Terminated != nil {
				errors = append(errors, toPodError(fmt.Sprintf("[%s] [image: %s] %s", st.Name, st.Image, st.LastTerminationState.Terminated.Message)))
			}
		}
	}
	return errors
}

func isInMesh(pod *apiv1.Pod) bool {
	_, ok := pod.Annotations[k8s.ProxyVersionAnnotation]
	return ok
}

func isInvalidServiceRequest(req *pb.StatSummaryRequest) bool {
	fromResource := req.GetFromResource()
	if fromResource != nil {
		return fromResource.Type == k8s.Services
	} else {
		return req.Selector.Resource.Type == k8s.Services
	}
}

func (s *grpcServer) queryProm(ctx context.Context, query string) (model.Vector, error) {
	log.Debugf("Query request:\n\t%+v", query)

	// single data point (aka summary) query
	res, err := s.prometheusAPI.Query(ctx, query, time.Time{})
	if err != nil {
		log.Errorf("Query(%+v) failed with: %+v", query, err)
		return nil, err
	}
	log.Debugf("Query response:\n\t%+v", res)

	if res.Type() != model.ValVector {
		err = fmt.Errorf("Unexpected query result type (expected Vector): %s", res.Type())
		log.Error(err)
		return nil, err
	}

	return res.(model.Vector), nil
}
