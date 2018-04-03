package destination

import (
	common "github.com/runconduit/conduit/controller/gen/common"
	pb "github.com/runconduit/conduit/controller/gen/proxy/destination"
	"github.com/runconduit/conduit/controller/k8s"
	"github.com/runconduit/conduit/controller/util"
	log "github.com/sirupsen/logrus"
)

type updateListener interface {
	Update(add []common.TcpAddress, remove []common.TcpAddress)
	Done() <-chan struct{}
	NoEndpoints(exists bool)
}

// implements the updateListener interface
type endpointListener struct {
	serviceName string
	stream      pb.Destination_GetServer
	podsByIp    k8s.PodIndex
}

func (l *endpointListener) Done() <-chan struct{} {
	return l.stream.Context().Done()
}

func (l *endpointListener) Update(add []common.TcpAddress, remove []common.TcpAddress) {
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

func (l *endpointListener) toWeightedAddrSet(endpoints []common.TcpAddress) *pb.WeightedAddrSet {
	var namespace string
	addrs := make([]*pb.WeightedAddr, 0)
	for i, address := range endpoints {
		metricLabelsForPod := map[string]string{}

		ipAsString := util.IPToString(address.Ip)
		resultingPods, err := l.podsByIp.GetPodsByIndex(ipAsString)
		if err != nil {
			log.Errorf("Error while finding pod for IP [%s], this IP will be sent with no metric labels: %v", ipAsString, err)
		} else {
			if len(resultingPods) == 0 || resultingPods[0] == nil {
				log.Errorf("Could not find pod for IP [%s], this IP will be sent with no metric labels.", ipAsString)
			} else {
				pod := resultingPods[0]
				metricLabelsForPod = map[string]string{
					"k8s_pod": pod.Name,
				}

				namespace = pod.Namespace
			}
		}

		addrs = append(addrs, &pb.WeightedAddr{
			Addr:         &endpoints[i],
			Weight:       1,
			MetricLabels: metricLabelsForPod,
		})
	}

	globalMetricLabels := map[string]string{
		"k8s_service":   l.serviceName,
		"k8s_namespace": namespace,
	}

	return &pb.WeightedAddrSet{
		Addrs:        addrs,
		MetricLabels: globalMetricLabels,
	}
}

func (l *endpointListener) toAddrSet(endpoints []common.TcpAddress) *pb.AddrSet {
	addrs := make([]*common.TcpAddress, 0)
	for i := range endpoints {
		addrs = append(addrs, &endpoints[i])
	}
	return &pb.AddrSet{Addrs: addrs}
}
