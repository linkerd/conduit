package destination

import (
	"context"
	"errors"
	"reflect"
	"testing"

	common "github.com/runconduit/conduit/controller/gen/common"
	pb "github.com/runconduit/conduit/controller/gen/proxy/destination"
	"github.com/runconduit/conduit/controller/k8s"
	"github.com/runconduit/conduit/controller/util"
	"google.golang.org/grpc/metadata"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockDestination_GetServer struct {
	errorToReturn   error
	contextToReturn context.Context
	updatesReceived []*pb.Update
}

func (m *mockDestination_GetServer) Send(update *pb.Update) error {
	m.updatesReceived = append(m.updatesReceived, update)
	return m.errorToReturn
}

func (m *mockDestination_GetServer) SetHeader(metadata.MD) error  { return m.errorToReturn }
func (m *mockDestination_GetServer) SendHeader(metadata.MD) error { return m.errorToReturn }
func (m *mockDestination_GetServer) SetTrailer(metadata.MD)       {}
func (m *mockDestination_GetServer) Context() context.Context     { return m.contextToReturn }
func (m *mockDestination_GetServer) SendMsg(x interface{}) error  { return m.errorToReturn }
func (m *mockDestination_GetServer) RecvMsg(x interface{}) error  { return m.errorToReturn }

func TestBuildResolversList(t *testing.T) {
	endpointsWatcher := &k8s.MockEndpointsWatcher{}
	dnsWatcher := &mockDnsWatcher{}

	t.Run("Doesn't build a list if Kubernetes DNS zone isnt valid", func(t *testing.T) {
		invalidK8sDNSZones := []string{"1", "-a", "a-", "-"}
		for _, dsnZone := range invalidK8sDNSZones {
			resolvers, err := buildResolversList(dsnZone, endpointsWatcher, dnsWatcher)
			if err == nil {
				t.Fatalf("Expecting error when k8s zone is [%s], got nothing. Resolvers: %v", dsnZone, resolvers)
			}
		}
	})

	t.Run("Builds list with echo IP first, then K8s resolver", func(t *testing.T) {
		resolvers, err := buildResolversList("some.zone", endpointsWatcher, dnsWatcher)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		actualNumResolvers := len(resolvers)
		expectedNumResolvers := 2
		if actualNumResolvers != expectedNumResolvers {
			t.Fatalf("Expecting [%d] resolvers, got [%d]: %v", expectedNumResolvers, actualNumResolvers, resolvers)
		}

		if _, ok := resolvers[0].(*echoIpV4Resolver); !ok {
			t.Fatalf("Expecting first resolver to be echo IP, got [%+v]. List: %v", resolvers[0], resolvers)
		}

		if _, ok := resolvers[1].(*k8sResolver); !ok {
			t.Fatalf("Expecting second resolver to be k8s, got [%+v]. List: %v", resolvers[0], resolvers)
		}
	})
}

func TestEndpointListener(t *testing.T) {
	t.Run("Sends one update for add and another for remove", func(t *testing.T) {
		mockGetServer := &mockDestination_GetServer{updatesReceived: []*pb.Update{}}

		listener := &endpointListener{stream: mockGetServer, podsByIp: k8s.NewEmptyPodIndex()}

		addedAddress1 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 1}}, Port: 1}
		addedAddress2 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 2}}, Port: 2}
		removedAddress1 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 100}}, Port: 100}

		listener.Update([]common.TcpAddress{addedAddress1, addedAddress2}, []common.TcpAddress{removedAddress1})

		expectedNumUpdates := 2
		actualNumUpdates := len(mockGetServer.updatesReceived)
		if actualNumUpdates != expectedNumUpdates {
			t.Fatalf("Expecting [%d] updates, got [%d]. Updates: %v", expectedNumUpdates, actualNumUpdates, mockGetServer.updatesReceived)
		}
	})

	t.Run("Sends addresses as removed or added", func(t *testing.T) {
		mockGetServer := &mockDestination_GetServer{updatesReceived: []*pb.Update{}}

		listener := &endpointListener{stream: mockGetServer, podsByIp: k8s.NewEmptyPodIndex()}

		addedAddress1 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 1}}, Port: 1}
		addedAddress2 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 2}}, Port: 2}
		removedAddress1 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 100}}, Port: 100}

		listener.Update([]common.TcpAddress{addedAddress1, addedAddress2}, []common.TcpAddress{removedAddress1})

		addressesAdded := mockGetServer.updatesReceived[0].GetAdd().Addrs
		actualNumberOfAdded := len(addressesAdded)
		expectedNumberOfAdded := 2
		if actualNumberOfAdded != expectedNumberOfAdded {
			t.Fatalf("Expecting [%d] addresses to be added, got [%d]: %v", expectedNumberOfAdded, actualNumberOfAdded, addressesAdded)
		}

		addressesRemoved := mockGetServer.updatesReceived[1].GetRemove().Addrs
		actualNumberOfRemoved := len(addressesRemoved)
		expectedNumberOfRemoved := 1
		if actualNumberOfRemoved != expectedNumberOfRemoved {
			t.Fatalf("Expecting [%d] addresses to be removed, got [%d]: %v", expectedNumberOfRemoved, actualNumberOfRemoved, addressesRemoved)
		}

		checkAddress(t, addressesAdded[0], &addedAddress1)
		checkAddress(t, addressesAdded[1], &addedAddress2)

		actualAddressRemoved := addressesRemoved[0]
		expectedAddressRemoved := &removedAddress1
		if !reflect.DeepEqual(actualAddressRemoved, expectedAddressRemoved) {
			t.Fatalf("Expected remove address to be [%s], but it was [%s]", expectedAddressRemoved, actualAddressRemoved)
		}
	})

	t.Run("It returns when the underlying context is done", func(t *testing.T) {
		context, cancelFn := context.WithCancel(context.Background())
		mockGetServer := &mockDestination_GetServer{updatesReceived: []*pb.Update{}, contextToReturn: context}
		listener := &endpointListener{stream: mockGetServer, podsByIp: k8s.NewEmptyPodIndex()}

		completed := make(chan bool)
		go func() {
			<-listener.Done()
			completed <- true
		}()

		cancelFn()

		c := <-completed

		if !c {
			t.Fatalf("Expected function to be completed after the cancel()")
		}
	})

	t.Run("Sends metric labels with added addresses", func(t *testing.T) {
		expectedServiceName := "service-name"
		expectedPodName := "pod1"
		expectedNamespace := "this-namespace"

		addedAddress1 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 666}}, Port: 1}
		ipForAddr1 := util.IPToString(addedAddress1.Ip)
		podForAddedAddress1 := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      expectedPodName,
				Namespace: expectedNamespace,
			},
		}
		addedAddress2 := common.TcpAddress{Ip: &common.IPAddress{Ip: &common.IPAddress_Ipv4{Ipv4: 222}}, Port: 22}
		podIndex := &k8s.InMemoryPodIndex{BackingMap: map[string]*v1.Pod{ipForAddr1: podForAddedAddress1}}

		mockGetServer := &mockDestination_GetServer{updatesReceived: []*pb.Update{}}
		listener := &endpointListener{
			podsByIp:    podIndex,
			serviceName: expectedServiceName,
			stream:      mockGetServer,
		}

		listener.Update([]common.TcpAddress{addedAddress1, addedAddress2}, nil)

		actualGlobalMetricLabels := mockGetServer.updatesReceived[0].GetAdd().MetricLabels
		expectedGlobalMetricLabels := map[string]string{"k8s_namespace": expectedNamespace, "k8s_service": expectedServiceName}
		if !reflect.DeepEqual(actualGlobalMetricLabels, expectedGlobalMetricLabels) {
			t.Fatalf("Expected global metric labels sent to be [%v] but was [%v]", expectedGlobalMetricLabels, actualGlobalMetricLabels)
		}

		actualAddedAddress1MetricLabels := mockGetServer.updatesReceived[0].GetAdd().Addrs[0].MetricLabels
		expectedAddedAddress1MetricLabels := map[string]string{"k8s_pod": expectedPodName}
		if !reflect.DeepEqual(actualAddedAddress1MetricLabels, expectedAddedAddress1MetricLabels) {
			t.Fatalf("Expected global metric labels sent to be [%v] but was [%v]", expectedAddedAddress1MetricLabels, actualAddedAddress1MetricLabels)
		}
	})

	t.Run("It returns when the underlying context is done", func(t *testing.T) {
		context, cancelFn := context.WithCancel(context.Background())
		mockGetServer := &mockDestination_GetServer{updatesReceived: []*pb.Update{}, contextToReturn: context}
		listener := &endpointListener{stream: mockGetServer, podsByIp: k8s.NewEmptyPodIndex()}

		completed := make(chan bool)
		go func() {
			<-listener.Done()
			completed <- true
		}()

		cancelFn()

		c := <-completed

		if !c {
			t.Fatalf("Expected function to be completed after the cancel()")
		}
	})
}

type mockStreamingDestinationResolver struct {
	hostReceived             string
	portReceived             int
	listenerReceived         updateListener
	canResolveToReturn       bool
	errToReturnForCanResolve error
	errToReturnForResolution error
}

func (m *mockStreamingDestinationResolver) canResolve(host string, port int) (bool, error) {
	return m.canResolveToReturn, m.errToReturnForCanResolve
}

func (m *mockStreamingDestinationResolver) streamResolution(host string, port int, listener updateListener) error {
	m.hostReceived = host
	m.portReceived = port
	m.listenerReceived = listener
	return m.errToReturnForResolution
}

func TestStreamResolutionUsingCorrectResolverFor(t *testing.T) {
	stream := &mockDestination_GetServer{}
	host := "something"
	port := 666

	t.Run("Uses first resolver that is able to resolve the host and port", func(t *testing.T) {
		no := &mockStreamingDestinationResolver{canResolveToReturn: false}
		yes := &mockStreamingDestinationResolver{canResolveToReturn: true}
		otherYes := &mockStreamingDestinationResolver{canResolveToReturn: true}

		server := server{
			podsByIp:  k8s.NewEmptyPodIndex(),
			resolvers: []streamingDestinationResolver{no, no, yes, no, no, otherYes},
		}

		err := server.streamResolutionUsingCorrectResolverFor(host, port, stream)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if no.listenerReceived != nil {
			t.Fatalf("Expected handler [%+v] to not be called, but it was", no)
		}

		if otherYes.listenerReceived != nil {
			t.Fatalf("Expected handler [%+v] to not be called, but it was", otherYes)
		}

		if yes.listenerReceived == nil || yes.portReceived != port || yes.hostReceived != host {
			t.Fatalf("Expected resolved [%+v] to have been called with stream [%v] host [%s] and port [%d]", yes, stream, host, port)
		}
	})

	t.Run("Returns error if no resolver can resolve", func(t *testing.T) {
		no := &mockStreamingDestinationResolver{canResolveToReturn: false}

		server := server{
			podsByIp:  k8s.NewEmptyPodIndex(),
			resolvers: []streamingDestinationResolver{no, no, no, no},
		}

		err := server.streamResolutionUsingCorrectResolverFor(host, port, stream)
		if err == nil {
			t.Fatalf("Expecting error, got nothing")
		}
	})

	t.Run("Returns error if the resolver returned an error on canResolve", func(t *testing.T) {
		resolver := &mockStreamingDestinationResolver{canResolveToReturn: true, errToReturnForCanResolve: errors.New("expected for can resolve")}

		server := server{
			podsByIp:  k8s.NewEmptyPodIndex(),
			resolvers: []streamingDestinationResolver{resolver},
		}

		err := server.streamResolutionUsingCorrectResolverFor(host, port, stream)
		if err == nil {
			t.Fatalf("Expecting error, got nothing")
		}
	})

	t.Run("Returns error if the resolver returned an error on streamResolution", func(t *testing.T) {
		resolver := &mockStreamingDestinationResolver{canResolveToReturn: true, errToReturnForResolution: errors.New("expected for resolving")}

		server := server{
			podsByIp:  k8s.NewEmptyPodIndex(),
			resolvers: []streamingDestinationResolver{resolver},
		}

		err := server.streamResolutionUsingCorrectResolverFor(host, port, stream)
		if err == nil {
			t.Fatalf("Expecting error, got nothing")
		}
	})
}

func checkAddress(t *testing.T, addr *pb.WeightedAddr, expectedAddress *common.TcpAddress) {
	actualAddress := addr.Addr
	actualWeight := addr.Weight
	expectedWeight := uint32(1)

	if !reflect.DeepEqual(actualAddress, expectedAddress) || actualWeight != expectedWeight {
		t.Fatalf("Expected added address to be [%+v] and weight to be [%d], but it was [%+v] and [%d]", expectedAddress, expectedWeight, actualAddress, actualWeight)
	}
}
