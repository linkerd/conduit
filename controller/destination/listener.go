package destination

import (
	pb "github.com/linkerd/linkerd2-proxy-api/go/destination"
	net "github.com/linkerd/linkerd2-proxy-api/go/net"
	"github.com/linkerd/linkerd2/pkg/addr"
	pkgK8s "github.com/linkerd/linkerd2/pkg/k8s"
	log "github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
)

type updateListener interface {
	Update(add, remove []*updateAddress)
	ClientClose() <-chan struct{}
	ServerClose() <-chan struct{}
	NoEndpoints(exists bool)
	SetServiceId(id *serviceId)
	Stop()
}

type ownerKindAndNameFn func(*coreV1.Pod) (string, string)

// updateAddress is a pairing of TCP address to Kubernetes pod object
type updateAddress struct {
	address *net.TcpAddress
	pod     *coreV1.Pod
}

func diffUpdateAddresses(oldAddrs, newAddrs []*updateAddress) ([]*updateAddress, []*updateAddress) {
	addSet := make(map[string]*updateAddress)
	removeSet := make(map[string]*updateAddress)

	for _, a := range newAddrs {
		key := addr.ProxyAddressToString(a.address)
		addSet[key] = a
	}

	for _, a := range oldAddrs {
		key := addr.ProxyAddressToString(a.address)
		delete(addSet, key)
		removeSet[key] = a
	}

	for _, a := range newAddrs {
		key := addr.ProxyAddressToString(a.address)
		delete(removeSet, key)
	}

	add := make([]*updateAddress, 0)
	for _, a := range addSet {
		add = append(add, a)
	}

	remove := make([]*updateAddress, 0)
	for _, a := range removeSet {
		remove = append(remove, a)
	}

	return add, remove
}

// implements the updateListener interface
type endpointListener struct {
	stream           pb.Destination_GetServer
	ownerKindAndName ownerKindAndNameFn
	labels           map[string]string
	enableTLS        bool
	stopCh           chan struct{}
}

func newEndpointListener(
	stream pb.Destination_GetServer,
	ownerKindAndName ownerKindAndNameFn,
	enableTLS bool,
) *endpointListener {
	return &endpointListener{
		stream:           stream,
		ownerKindAndName: ownerKindAndName,
		labels:           make(map[string]string),
		enableTLS:        enableTLS,
		stopCh:           make(chan struct{}),
	}
}

func (l *endpointListener) ClientClose() <-chan struct{} {
	return l.stream.Context().Done()
}

func (l *endpointListener) ServerClose() <-chan struct{} {
	return l.stopCh
}

func (l *endpointListener) Stop() {
	close(l.stopCh)
}

func (l *endpointListener) SetServiceId(id *serviceId) {
	if id != nil {
		l.labels = map[string]string{
			"namespace": id.namespace,
			"service":   id.name,
		}
	}
}

func (l *endpointListener) Update(add, remove []*updateAddress) {
	if len(add) > 0 {
		update := &pb.Update{
			Update: &pb.Update_Add{
				Add: l.toWeightedAddrSet(add),
			},
		}
		err := l.stream.Send(update)
		if err != nil {
			log.Error(err)
		}
	}
	if len(remove) > 0 {
		update := &pb.Update{
			Update: &pb.Update_Remove{
				Remove: l.toAddrSet(remove),
			},
		}
		err := l.stream.Send(update)
		if err != nil {
			log.Error(err)
		}
	}
}

func (l *endpointListener) NoEndpoints(exists bool) {
	update := &pb.Update{
		Update: &pb.Update_NoEndpoints{
			NoEndpoints: &pb.NoEndpoints{
				Exists: exists,
			},
		},
	}
	l.stream.Send(update)
}

func (l *endpointListener) toWeightedAddrSet(addresses []*updateAddress) *pb.WeightedAddrSet {
	addrs := make([]*pb.WeightedAddr, 0)
	for _, a := range addresses {
		addrs = append(addrs, l.toWeightedAddr(a))
	}

	return &pb.WeightedAddrSet{
		Addrs:        addrs,
		MetricLabels: l.labels,
	}
}

func (l *endpointListener) toWeightedAddr(address *updateAddress) *pb.WeightedAddr {
	metricLabelsForPod := pkgK8s.GetOwnerLabels(address.pod.ObjectMeta)
	metricLabelsForPod["pod"] = address.pod.Name

	return &pb.WeightedAddr{
		Addr:         address.address,
		Weight:       1,
		MetricLabels: metricLabelsForPod,
		TlsIdentity:  l.toTlsIdentity(address.pod),
	}
}

func (l *endpointListener) toAddrSet(addresses []*updateAddress) *pb.AddrSet {
	addrs := make([]*net.TcpAddress, 0)
	for _, a := range addresses {
		addrs = append(addrs, a.address)
	}
	return &pb.AddrSet{Addrs: addrs}
}

func (l *endpointListener) toTlsIdentity(pod *coreV1.Pod) *pb.TlsIdentity {
	if !l.enableTLS {
		return nil
	}

	controllerNs := pkgK8s.GetControllerNs(pod.ObjectMeta)
	ownerKind, ownerName := l.ownerKindAndName(pod)

	identity := pkgK8s.TLSIdentity{
		Name:                ownerName,
		Kind:                ownerKind,
		Namespace:           pod.Namespace,
		ControllerNamespace: controllerNs,
	}

	return &pb.TlsIdentity{
		Strategy: &pb.TlsIdentity_K8SPodIdentity_{
			K8SPodIdentity: &pb.TlsIdentity_K8SPodIdentity{
				PodIdentity:  identity.ToDNSName(),
				ControllerNs: controllerNs,
			},
		},
	}
}
