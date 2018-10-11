package public

import (
	"context"
	"io"
	"sync"
	"time"

	healthcheckPb "github.com/linkerd/linkerd2/controller/gen/common/healthcheck"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc"
)

type MockApiClient struct {
	ErrorToReturn                   error
	VersionInfoToReturn             *pb.VersionInfo
	ListPodsResponseToReturn        *pb.ListPodsResponse
	StatSummaryResponseToReturn     *pb.StatSummaryResponse
	SelfCheckResponseToReturn       *healthcheckPb.SelfCheckResponse
	Api_TapClientToReturn           pb.Api_TapClient
	Api_TapByResourceClientToReturn pb.Api_TapByResourceClient
}

func (c *MockApiClient) StatSummary(ctx context.Context, in *pb.StatSummaryRequest, opts ...grpc.CallOption) (*pb.StatSummaryResponse, error) {
	return c.StatSummaryResponseToReturn, c.ErrorToReturn
}

func (c *MockApiClient) Version(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VersionInfo, error) {
	return c.VersionInfoToReturn, c.ErrorToReturn
}

func (c *MockApiClient) ListPods(ctx context.Context, in *pb.ListPodsRequest, opts ...grpc.CallOption) (*pb.ListPodsResponse, error) {
	return c.ListPodsResponseToReturn, c.ErrorToReturn
}

func (c *MockApiClient) Tap(ctx context.Context, in *pb.TapRequest, opts ...grpc.CallOption) (pb.Api_TapClient, error) {
	return c.Api_TapClientToReturn, c.ErrorToReturn
}

func (c *MockApiClient) TapByResource(ctx context.Context, in *pb.TapByResourceRequest, opts ...grpc.CallOption) (pb.Api_TapByResourceClient, error) {
	return c.Api_TapByResourceClientToReturn, c.ErrorToReturn
}

func (c *MockApiClient) SelfCheck(ctx context.Context, in *healthcheckPb.SelfCheckRequest, _ ...grpc.CallOption) (*healthcheckPb.SelfCheckResponse, error) {
	return c.SelfCheckResponseToReturn, c.ErrorToReturn
}

type MockApi_TapClient struct {
	TapEventsToReturn []pb.TapEvent
	ErrorsToReturn    []error
	grpc.ClientStream
}

func (a *MockApi_TapClient) Recv() (*pb.TapEvent, error) {
	var eventPopped pb.TapEvent
	var errorPopped error
	if len(a.TapEventsToReturn) == 0 && len(a.ErrorsToReturn) == 0 {
		return nil, io.EOF
	}
	if len(a.TapEventsToReturn) != 0 {
		eventPopped, a.TapEventsToReturn = a.TapEventsToReturn[0], a.TapEventsToReturn[1:]
	}
	if len(a.ErrorsToReturn) != 0 {
		errorPopped, a.ErrorsToReturn = a.ErrorsToReturn[0], a.ErrorsToReturn[1:]
	}

	return &eventPopped, errorPopped
}

type MockApi_TapByResourceClient struct {
	TapEventsToReturn []pb.TapEvent
	ErrorsToReturn    []error
	grpc.ClientStream
}

func (a *MockApi_TapByResourceClient) Recv() (*pb.TapEvent, error) {
	var eventPopped pb.TapEvent
	var errorPopped error
	if len(a.TapEventsToReturn) == 0 && len(a.ErrorsToReturn) == 0 {
		return nil, io.EOF
	}
	if len(a.TapEventsToReturn) != 0 {
		eventPopped, a.TapEventsToReturn = a.TapEventsToReturn[0], a.TapEventsToReturn[1:]
	}
	if len(a.ErrorsToReturn) != 0 {
		errorPopped, a.ErrorsToReturn = a.ErrorsToReturn[0], a.ErrorsToReturn[1:]
	}

	return &eventPopped, errorPopped
}

//
// Prometheus client
//

type MockProm struct {
	Res             model.Value
	QueriesExecuted []string // expose the queries our Mock Prometheus receives, to test query generation
	rwLock          sync.Mutex
}

type PodCounts struct {
	MeshedPods  uint64
	RunningPods uint64
	FailedPods  uint64
}

// satisfies v1.API
func (m *MockProm) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	m.QueriesExecuted = append(m.QueriesExecuted, query)
	return m.Res, nil
}
func (m *MockProm) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, error) {
	m.rwLock.Lock()
	defer m.rwLock.Unlock()
	m.QueriesExecuted = append(m.QueriesExecuted, query)
	return m.Res, nil
}
func (m *MockProm) LabelValues(ctx context.Context, label string) (model.LabelValues, error) {
	return nil, nil
}
func (m *MockProm) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, error) {
	return nil, nil
}

func GenStatSummaryResponse(resName, resType string, resNs []string, counts *PodCounts) pb.StatSummaryResponse {
	rows := []*pb.StatTable_PodGroup_Row{}
	for _, ns := range resNs {
		statTableRow := &pb.StatTable_PodGroup_Row{
			Resource: &pb.Resource{
				Namespace: ns,
				Type:      resType,
				Name:      resName,
			},
			Stats: &pb.BasicStats{
				SuccessCount:    123,
				FailureCount:    0,
				LatencyMsP50:    123,
				LatencyMsP95:    123,
				LatencyMsP99:    123,
				TlsRequestCount: 123,
			},
			TimeWindow: "1m",
		}

		if counts != nil {
			statTableRow.MeshedPodCount = counts.MeshedPods
			statTableRow.RunningPodCount = counts.RunningPods
			statTableRow.FailedPodCount = counts.FailedPods
		}

		rows = append(rows, statTableRow)
	}

	resp := pb.StatSummaryResponse{
		Response: &pb.StatSummaryResponse_Ok_{ // https://github.com/golang/protobuf/issues/205
			Ok: &pb.StatSummaryResponse_Ok{
				StatTables: []*pb.StatTable{
					&pb.StatTable{
						Table: &pb.StatTable_PodGroup_{
							PodGroup: &pb.StatTable_PodGroup{
								Rows: rows,
							},
						},
					},
				},
			},
		},
	}

	return resp
}
