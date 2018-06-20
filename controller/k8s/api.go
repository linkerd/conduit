package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/runconduit/conduit/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	appinformers "k8s.io/client-go/informers/apps/v1beta2"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ApiResource int

const (
	CM ApiResource = iota
	Deploy
	Endpoint
	NS
	Pod
	RC
	RS
	Svc
)

// API provides shared informers for all Kubernetes objects
type API struct {
	Client kubernetes.Interface

	cm       coreinformers.ConfigMapInformer
	deploy   appinformers.DeploymentInformer
	endpoint coreinformers.EndpointsInformer
	ns       coreinformers.NamespaceInformer
	pod      coreinformers.PodInformer
	rc       coreinformers.ReplicationControllerInformer
	rs       appinformers.ReplicaSetInformer
	svc      coreinformers.ServiceInformer

	syncChecks      []cache.InformerSynced
	sharedInformers informers.SharedInformerFactory
}

// NewAPI takes a Kubernetes client and returns an initialized API
func NewAPI(k8sClient kubernetes.Interface, resources ...ApiResource) *API {
	sharedInformers := informers.NewSharedInformerFactory(k8sClient, 10*time.Minute)

	api := &API{
		Client:          k8sClient,
		syncChecks:      make([]cache.InformerSynced, 0),
		sharedInformers: sharedInformers,
	}

	for _, resource := range resources {
		switch resource {
		case CM:
			api.cm = sharedInformers.Core().V1().ConfigMaps()
			api.syncChecks = append(api.syncChecks, api.cm.Informer().HasSynced)
		case Deploy:
			api.deploy = sharedInformers.Apps().V1beta2().Deployments()
			api.syncChecks = append(api.syncChecks, api.deploy.Informer().HasSynced)
		case Endpoint:
			api.endpoint = sharedInformers.Core().V1().Endpoints()
			api.syncChecks = append(api.syncChecks, api.endpoint.Informer().HasSynced)
		case NS:
			api.ns = sharedInformers.Core().V1().Namespaces()
			api.syncChecks = append(api.syncChecks, api.ns.Informer().HasSynced)
		case Pod:
			api.pod = sharedInformers.Core().V1().Pods()
			api.syncChecks = append(api.syncChecks, api.pod.Informer().HasSynced)
		case RC:
			api.rc = sharedInformers.Core().V1().ReplicationControllers()
			api.syncChecks = append(api.syncChecks, api.rc.Informer().HasSynced)
		case RS:
			api.rs = sharedInformers.Apps().V1beta2().ReplicaSets()
			api.syncChecks = append(api.syncChecks, api.rs.Informer().HasSynced)
		case Svc:
			api.svc = sharedInformers.Core().V1().Services()
			api.syncChecks = append(api.syncChecks, api.svc.Informer().HasSynced)
		}
	}

	return api
}

// Sync waits for all informers to be synced.
// For servers, call this asynchronously.
// For testing, call this synchronously.
func (api *API) Sync(readyCh chan<- struct{}) {
	api.sharedInformers.Start(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Infof("waiting for caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), api.syncChecks...) {
		log.Fatal("failed to sync caches")
	}
	log.Infof("caches synced")

	if readyCh != nil {
		close(readyCh)
	}
}

func (api *API) NS() coreinformers.NamespaceInformer {
	if api.ns == nil {
		panic("NS informer not configured")
	}
	return api.ns
}

func (api *API) Deploy() appinformers.DeploymentInformer {
	if api.deploy == nil {
		panic("Deploy informer not configured")
	}
	return api.deploy
}

func (api *API) RS() appinformers.ReplicaSetInformer {
	if api.rs == nil {
		panic("RS informer not configured")
	}
	return api.rs
}

func (api *API) Pod() coreinformers.PodInformer {
	if api.pod == nil {
		panic("Pod informer not configured")
	}
	return api.pod
}

func (api *API) RC() coreinformers.ReplicationControllerInformer {
	if api.rc == nil {
		panic("RC informer not configured")
	}
	return api.rc
}

func (api *API) Svc() coreinformers.ServiceInformer {
	if api.svc == nil {
		panic("Svc informer not configured")
	}
	return api.svc
}

func (api *API) Endpoint() coreinformers.EndpointsInformer {
	if api.endpoint == nil {
		panic("Endpoint informer not configured")
	}
	return api.endpoint
}

func (api *API) CM() coreinformers.ConfigMapInformer {
	if api.cm == nil {
		panic("CM informer not configured")
	}
	return api.cm
}

// GetObjects returns a list of Kubernetes objects, given a namespace, type, and name.
// If namespace is an empty string, match objects in all namespaces.
// If name is an empty string, match all objects of the given type.
func (api *API) GetObjects(namespace, restype, name string) ([]runtime.Object, error) {
	switch restype {
	case k8s.Namespaces:
		return api.getNamespaces(name)
	case k8s.Deployments:
		return api.getDeployments(namespace, name)
	case k8s.Pods:
		return api.getPods(namespace, name)
	case k8s.ReplicationControllers:
		return api.getRCs(namespace, name)
	case k8s.Services:
		return api.getServices(namespace, name)
	default:
		// TODO: ReplicaSet
		return nil, status.Errorf(codes.Unimplemented, "unimplemented resource type: %s", restype)
	}
}

// GetPodsFor returns all running and pending Pods associated with a given
// Kubernetes object. Use includeFailed to also get failed Pods
func (api *API) GetPodsFor(obj runtime.Object, includeFailed bool) ([]*apiv1.Pod, error) {
	var namespace string
	var selector labels.Selector
	var pods []*apiv1.Pod
	var err error

	switch typed := obj.(type) {
	case *apiv1.Namespace:
		namespace = typed.Name
		selector = labels.Everything()

	case *appsv1beta2.Deployment:
		namespace = typed.Namespace
		selector = labels.Set(typed.Spec.Selector.MatchLabels).AsSelector()

	case *appsv1beta2.ReplicaSet:
		namespace = typed.Namespace
		selector = labels.Set(typed.Spec.Selector.MatchLabels).AsSelector()

	case *apiv1.ReplicationController:
		namespace = typed.Namespace
		selector = labels.Set(typed.Spec.Selector).AsSelector()

	case *apiv1.Service:
		namespace = typed.Namespace
		selector = labels.Set(typed.Spec.Selector).AsSelector()

	case *apiv1.Pod:
		// Special case for pods:
		// GetPodsFor a pod should just return the pod itself
		namespace = typed.Namespace
		pod, err := api.Pod().Lister().Pods(typed.Namespace).Get(typed.Name)
		if err != nil {
			return nil, err
		}
		pods = []*apiv1.Pod{pod}

	default:
		return nil, fmt.Errorf("Cannot get object selector: %v", obj)
	}

	// if obj.(type) is Pod, we've already retrieved it and put it in pods
	// for the other types, pods will still be empty
	if len(pods) == 0 {
		pods, err = api.Pod().Lister().Pods(namespace).List(selector)
		if err != nil {
			return nil, err
		}
	}

	allPods := []*apiv1.Pod{}
	for _, pod := range pods {
		if isPendingOrRunning(pod) || (includeFailed && isFailed(pod)) {
			allPods = append(allPods, pod)
		}
	}

	return allPods, nil
}

func (api *API) getNamespaces(name string) ([]runtime.Object, error) {
	var err error
	var namespaces []*apiv1.Namespace

	if name == "" {
		namespaces, err = api.NS().Lister().List(labels.Everything())
	} else {
		var namespace *apiv1.Namespace
		namespace, err = api.NS().Lister().Get(name)
		namespaces = []*apiv1.Namespace{namespace}
	}

	if err != nil {
		return nil, err
	}

	objects := []runtime.Object{}
	for _, ns := range namespaces {
		objects = append(objects, ns)
	}

	return objects, nil
}

func (api *API) getDeployments(namespace, name string) ([]runtime.Object, error) {
	var err error
	var deploys []*appsv1beta2.Deployment

	if namespace == "" {
		deploys, err = api.Deploy().Lister().List(labels.Everything())
	} else if name == "" {
		deploys, err = api.Deploy().Lister().Deployments(namespace).List(labels.Everything())
	} else {
		var deploy *appsv1beta2.Deployment
		deploy, err = api.Deploy().Lister().Deployments(namespace).Get(name)
		deploys = []*appsv1beta2.Deployment{deploy}
	}

	if err != nil {
		return nil, err
	}

	objects := []runtime.Object{}
	for _, deploy := range deploys {
		objects = append(objects, deploy)
	}

	return objects, nil
}

func (api *API) getPods(namespace, name string) ([]runtime.Object, error) {
	var err error
	var pods []*apiv1.Pod

	if namespace == "" {
		pods, err = api.Pod().Lister().List(labels.Everything())
	} else if name == "" {
		pods, err = api.Pod().Lister().Pods(namespace).List(labels.Everything())
	} else {
		var pod *apiv1.Pod
		pod, err = api.Pod().Lister().Pods(namespace).Get(name)
		pods = []*apiv1.Pod{pod}
	}

	if err != nil {
		return nil, err
	}

	objects := []runtime.Object{}
	for _, pod := range pods {
		if !isPendingOrRunning(pod) {
			continue
		}
		objects = append(objects, pod)
	}

	return objects, nil
}

func (api *API) getRCs(namespace, name string) ([]runtime.Object, error) {
	var err error
	var rcs []*apiv1.ReplicationController

	if namespace == "" {
		rcs, err = api.RC().Lister().List(labels.Everything())
	} else if name == "" {
		rcs, err = api.RC().Lister().ReplicationControllers(namespace).List(labels.Everything())
	} else {
		var rc *apiv1.ReplicationController
		rc, err = api.RC().Lister().ReplicationControllers(namespace).Get(name)
		rcs = []*apiv1.ReplicationController{rc}
	}

	if err != nil {
		return nil, err
	}

	objects := []runtime.Object{}
	for _, rc := range rcs {
		objects = append(objects, rc)
	}

	return objects, nil
}

func (api *API) getServices(namespace, name string) ([]runtime.Object, error) {
	var err error
	var services []*apiv1.Service

	if namespace == "" {
		services, err = api.Svc().Lister().List(labels.Everything())
	} else if name == "" {
		services, err = api.Svc().Lister().Services(namespace).List(labels.Everything())
	} else {
		var svc *apiv1.Service
		svc, err = api.Svc().Lister().Services(namespace).Get(name)
		services = []*apiv1.Service{svc}
	}

	if err != nil {
		return nil, err
	}

	objects := []runtime.Object{}
	for _, svc := range services {
		objects = append(objects, svc)
	}

	return objects, nil
}

func isPendingOrRunning(pod *apiv1.Pod) bool {
	pending := pod.Status.Phase == apiv1.PodPending
	running := pod.Status.Phase == apiv1.PodRunning
	terminating := pod.DeletionTimestamp != nil
	return (pending || running) && !terminating
}

func isFailed(pod *apiv1.Pod) bool {
	return pod.Status.Phase == apiv1.PodFailed
}
