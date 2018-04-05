package srv

import (
	"encoding/json"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/julienschmidt/httprouter"
	"github.com/runconduit/conduit/controller/api/util"
	pb "github.com/runconduit/conduit/controller/gen/public"
	log "github.com/sirupsen/logrus"
)

type (
	jsonError struct {
		Error string `json:"error"`
	}
)

var (
	defaultMetricTimeWindow      = pb.TimeWindow_ONE_MIN
	defaultMetricAggregationType = pb.AggregationType_TARGET_DEPLOY

	allMetrics = []pb.MetricName{
		pb.MetricName_REQUEST_RATE,
		pb.MetricName_SUCCESS_RATE,
		pb.MetricName_LATENCY,
	}

	meshMetrics = []pb.MetricName{
		pb.MetricName_REQUEST_RATE,
		pb.MetricName_SUCCESS_RATE,
	}

	pbMarshaler = jsonpb.Marshaler{EmitDefaults: true}
)

func renderJsonError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	log.Error(err.Error())
	rsp, _ := json.Marshal(jsonError{Error: err.Error()})
	w.WriteHeader(status)
	w.Write(rsp)
}

func renderJson(w http.ResponseWriter, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		renderJsonError(w, err, http.StatusInternalServerError)
		return
	}
	w.Write(jsonResp)
}

func renderJsonPb(w http.ResponseWriter, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	pbMarshaler.Marshal(w, msg)
}

func (h *handler) handleApiVersion(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	version, err := h.apiClient.Version(req.Context(), &pb.Empty{})

	if err != nil {
		renderJsonError(w, err, http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"version": version,
	}
	renderJson(w, resp)
}

func validateMetricParams(metricNameParam, aggParam, timeWindowParam string) (
	metrics []pb.MetricName,
	groupBy pb.AggregationType,
	window pb.TimeWindow,
	err error,
) {
	groupBy = defaultMetricAggregationType
	if aggParam != "" {
		groupBy, err = util.GetAggregationType(aggParam)
		if err != nil {
			return
		}
	}

	metrics = allMetrics
	if metricNameParam != "" {
		var requestedMetricName pb.MetricName
		requestedMetricName, err = util.GetMetricName(metricNameParam)
		if err != nil {
			return
		}
		metrics = []pb.MetricName{requestedMetricName}
	} else if groupBy == pb.AggregationType_MESH {
		metrics = meshMetrics
	}

	window = defaultMetricTimeWindow
	if timeWindowParam != "" {
		var requestedWindow pb.TimeWindow
		requestedWindow, err = util.GetWindow(timeWindowParam)
		if err != nil {
			return
		}
		window = requestedWindow
	}

	return
}

func (h *handler) handleApiMetrics(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	metricNameParam := req.FormValue("metric")
	timeWindowParam := req.FormValue("window")
	aggParam := req.FormValue("aggregation")
	timeseries := req.FormValue("timeseries") == "true"

	filterBy := pb.MetricMetadata{
		TargetDeploy: req.FormValue("target_deploy"),
		SourceDeploy: req.FormValue("source_deploy"),
		Component:    req.FormValue("component"),
	}

	metrics, groupBy, window, err := validateMetricParams(metricNameParam, aggParam, timeWindowParam)
	if err != nil {
		renderJsonError(w, err, http.StatusBadRequest)
		return
	}

	metricsRequest := &pb.MetricRequest{
		Metrics:   metrics,
		Window:    window,
		FilterBy:  &filterBy,
		GroupBy:   groupBy,
		Summarize: !timeseries,
	}

	metricsResponse, err := h.apiClient.Stat(req.Context(), metricsRequest)
	if err != nil {
		renderJsonError(w, err, http.StatusInternalServerError)
		return
	}

	renderJsonPb(w, metricsResponse)
}

func (h *handler) handleApiPods(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	pods, err := h.apiClient.ListPods(req.Context(), &pb.Empty{})
	if err != nil {
		renderJsonError(w, err, http.StatusInternalServerError)
		return
	}

	renderJsonPb(w, pods)
}

func (h *handler) handleApiStat(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
	timeWindow := req.FormValue("window")
	resourceName := req.FormValue("resource_name")
	resourceType := req.FormValue("resource_type")
	namespace := req.FormValue("namespace")
	outToResourceName := req.FormValue("out_to_resource_name")
	outToResourceType := req.FormValue("out_to_resource_type")
	outToResourceNamespace := req.FormValue("out_to_resource_namespace")
	outFromResourceName := req.FormValue("out_from_resource_name")
	outFromResourceType := req.FormValue("out_from_resource_type")
	outFromResourceNamespace := req.FormValue("out_from_resource_namespace")

	window := defaultMetricTimeWindow
	if timeWindow != "" {
		var requestedWindow pb.TimeWindow
		requestedWindow, err := util.GetWindow(timeWindow)
		if err != nil {
			renderJsonError(w, err, http.StatusInternalServerError)
			return
		}
		window = requestedWindow
	}

	statRequest := &pb.StatSummaryRequest{
		Resource: &pb.ResourceSelection{
			Spec: &pb.Resource{
				Namespace: namespace,
				Name:      resourceName,
				Type:      resourceType,
			},
		},
		TimeWindow: window,
	}

	if outToResourceName != "" || outToResourceType != "" || outToResourceNamespace != "" {
		if outToResourceNamespace == "" {
			outToResourceNamespace = namespace
		}

		outToResource := pb.StatSummaryRequest_OutToResource{
			OutToResource: &pb.Resource{
				Namespace: outToResourceNamespace,
				Type:      outToResourceType,
				Name:      outToResourceName,
			},
		}
		statRequest.Outbound = &outToResource
	}

	if outFromResourceName != "" || outFromResourceType != "" || outFromResourceNamespace != "" {
		if outFromResourceNamespace == "" {
			outFromResourceNamespace = namespace
		}

		outFromResource := pb.StatSummaryRequest_OutFromResource{
			OutFromResource: &pb.Resource{
				Namespace: outFromResourceNamespace,
				Type:      outFromResourceType,
				Name:      outFromResourceName,
			},
		}
		statRequest.Outbound = &outFromResource
	}

	result, err := h.apiClient.StatSummary(req.Context(), statRequest)
	if err != nil {
		renderJsonError(w, err, http.StatusInternalServerError)
		return
	}
	renderJsonPb(w, result)
}
