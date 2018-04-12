package public

import (
	"context"
	"io"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	common "github.com/runconduit/conduit/controller/gen/common"
	healthcheckPb "github.com/runconduit/conduit/controller/gen/common/healthcheck"
	pb "github.com/runconduit/conduit/controller/gen/public"
	"google.golang.org/grpc"
)

type MockConduitApiClient struct {
	ErrorToReturn               error
	VersionInfoToReturn         *pb.VersionInfo
	ListPodsResponseToReturn    *pb.ListPodsResponse
	MetricResponseToReturn      *pb.MetricResponse
	StatSummaryResponseToReturn *pb.StatSummaryResponse
	SelfCheckResponseToReturn   *healthcheckPb.SelfCheckResponse
	Api_TapClientToReturn       pb.Api_TapClient
}

func (c *MockConduitApiClient) Stat(ctx context.Context, in *pb.MetricRequest, opts ...grpc.CallOption) (*pb.MetricResponse, error) {
	return c.MetricResponseToReturn, c.ErrorToReturn
}

func (c *MockConduitApiClient) StatSummary(ctx context.Context, in *pb.StatSummaryRequest, opts ...grpc.CallOption) (*pb.StatSummaryResponse, error) {
	return c.StatSummaryResponseToReturn, c.ErrorToReturn
}

func (c *MockConduitApiClient) Version(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.VersionInfo, error) {
	return c.VersionInfoToReturn, c.ErrorToReturn
}

func (c *MockConduitApiClient) ListPods(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.ListPodsResponse, error) {
	return c.ListPodsResponseToReturn, c.ErrorToReturn
}

func (c *MockConduitApiClient) Tap(ctx context.Context, in *pb.TapRequest, opts ...grpc.CallOption) (pb.Api_TapClient, error) {
	return c.Api_TapClientToReturn, c.ErrorToReturn
}

func (c *MockConduitApiClient) SelfCheck(ctx context.Context, in *healthcheckPb.SelfCheckRequest, _ ...grpc.CallOption) (*healthcheckPb.SelfCheckResponse, error) {
	return c.SelfCheckResponseToReturn, c.ErrorToReturn
}

type MockApi_TapClient struct {
	TapEventsToReturn []common.TapEvent
	ErrorsToReturn    []error
	grpc.ClientStream
}

func (a *MockApi_TapClient) Recv() (*common.TapEvent, error) {
	var eventPopped common.TapEvent
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

type MockProm struct {
	api v1.API
	Res model.Value
}

// satisfies v1.API
func (m *MockProm) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	return m.Res, nil
}
func (m *MockProm) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, error) {
	return m.Res, nil
}
func (m *MockProm) LabelValues(ctx context.Context, label string) (model.LabelValues, error) {
	return nil, nil
}
func (m *MockProm) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, error) {
	return nil, nil
}
