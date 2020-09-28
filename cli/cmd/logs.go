package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"text/template"
	"time"

	"github.com/fatih/color"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/spf13/cobra"
	"github.com/wercker/stern/stern"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

//This code replicates most of the functionality in https://github.com/wercker/stern/blob/master/cmd/cli.go
type logCmdConfig struct {
	clientset kubernetes.Interface
	*stern.Config
}

type logsOptions struct {
	container             string
	controlPlaneComponent string
	noColor               bool
	sinceSeconds          time.Duration
	tail                  int64
	timestamps            bool
}

func newLogsOptions() *logsOptions {
	return &logsOptions{
		container:             "",
		controlPlaneComponent: "",
		noColor:               false,
		sinceSeconds:          48 * time.Hour,
		tail:                  -1,
		timestamps:            false,
	}
}

func (o *logsOptions) toSternConfig(controlPlaneComponents, availableContainers []string) (*stern.Config, error) {
	config := &stern.Config{}

	if o.controlPlaneComponent == "" {
		config.LabelSelector = labels.Everything()
	} else {
		var podExists string
		for _, p := range controlPlaneComponents {
			if p == o.controlPlaneComponent {
				podExists = p
				break
			}
		}

		if podExists == "" {
			return nil, fmt.Errorf("control plane component [%s] does not exist. Must be one of %v", o.controlPlaneComponent, controlPlaneComponents)
		}
		selector, err := labels.Parse(fmt.Sprintf("linkerd.io/control-plane-component=%s", o.controlPlaneComponent))
		if err != nil {
			return nil, err
		}
		config.LabelSelector = selector
	}

	if o.container != "" {
		var matchingContainer string
		for _, c := range availableContainers {
			if o.container == c {
				matchingContainer = c
				break
			}
		}
		if matchingContainer == "" {
			return nil, fmt.Errorf("container [%s] does not exist in control plane [%s]", o.container, controlPlaneNamespace)
		}
	}

	containerFilterRgx, err := regexp.Compile(o.container)
	if err != nil {
		return nil, err
	}
	config.ContainerQuery = containerFilterRgx

	if o.tail != -1 {
		config.TailLines = &o.tail
	}

	// Do not use regex to filter pods. Instead, we provide the list of all control plane components and use
	// the label selector to filter logs.
	podFilterRgx, err := regexp.Compile("")
	if err != nil {
		return nil, err
	}

	// Based on stern/cmd/cli.go
	t := "{{color .PodColor .PodName}} {{color .ContainerColor .ContainerName}} {{.Message}}"
	if o.noColor {
		t = "{{.PodName}} {{.ContainerName}} {{.Message}}"
	}
	funs := map[string]interface{}{
		"json": func(in interface{}) (string, error) {
			b, err := json.Marshal(in)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"color": func(color color.Color, text string) string {
			return color.SprintFunc()(text)
		},
	}
	template, err := template.New("log").Funcs(funs).Parse(t)
	if err != nil {
		return nil, err
	}

	config.PodQuery = podFilterRgx
	config.Since = o.sinceSeconds
	config.Timestamps = o.timestamps
	config.Namespace = controlPlaneNamespace
	config.ContainerState = stern.RUNNING
	config.ExcludeContainerQuery = nil
	config.Template = template

	return config, nil
}

func getControlPlaneComponentsAndContainers(pods *corev1.PodList) ([]string, []string) {
	var controlPlaneComponents, containers []string
	for _, pod := range pods.Items {
		controlPlaneComponents = append(controlPlaneComponents, pod.Labels["linkerd.io/control-plane-component"])
		for _, container := range pod.Spec.Containers {
			containers = append(containers, container.Name)
		}
	}
	return controlPlaneComponents, containers
}

func newLogCmdConfig(ctx context.Context, options *logsOptions, kubeconfigPath, kubeContext, impersonate string, impersonateGroup []string) (*logCmdConfig, error) {
	kubeAPI, err := k8s.NewAPI(kubeconfigPath, kubeContext, impersonate, impersonateGroup, 0)
	if err != nil {
		return nil, err
	}

	podList, err := kubeAPI.CoreV1().Pods(controlPlaneNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	components, containers := getControlPlaneComponentsAndContainers(podList)

	c, err := options.toSternConfig(components, containers)
	if err != nil {
		return nil, err
	}

	return &logCmdConfig{
		kubeAPI,
		c,
	}, nil
}

func newCmdLogs() *cobra.Command {
	options := newLogsOptions()

	cmd := &cobra.Command{
		Use:   "logs [flags]",
		Short: "Tail logs from containers in the Linkerd control plane",
		Long:  `Tail logs from containers in the Linkerd control plane.`,
		Example: `  # Tail logs from all containers in the prometheus control plane component
  linkerd logs --control-plane-component prometheus

  # Tail logs from the linkerd-proxy container in the grafana control plane component
  linkerd logs --control-plane-component grafana --container linkerd-proxy

  # Tail logs from the linkerd-proxy container in the controller component beginning with the last two lines
  linkerd logs --control-plane-component controller --container linkerd-proxy --tail 2

  # Tail logs from the linkerd-proxy container in the controller component showing timestamps for each line
  linkerd logs --control-plane-component controller --container linkerd-proxy --timestamps
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			color.NoColor = options.noColor

			opts, err := newLogCmdConfig(cmd.Context(), options, kubeconfigPath, kubeContext, impersonate, impersonateGroup)

			if err != nil {
				return err
			}

			return runLogOutput(opts)
		},
	}

	cmd.PersistentFlags().StringVarP(&options.container, "container", "c", options.container, "Tail logs from the specified container. Options are 'public-api', 'destination', 'tap', 'prometheus', 'grafana' or 'linkerd-proxy'")
	cmd.PersistentFlags().StringVar(&options.controlPlaneComponent, "control-plane-component", options.controlPlaneComponent, "Tail logs from the specified control plane component. Default value (empty string) causes this command to tail logs from all resources marked with the 'linkerd.io/control-plane-component' label selector")
	cmd.PersistentFlags().BoolVarP(&options.noColor, "no-color", "n", options.noColor, "Disable colorized output") // needed until at least https://github.com/wercker/stern/issues/69 is resolved
	cmd.PersistentFlags().DurationVarP(&options.sinceSeconds, "since", "s", options.sinceSeconds, "Duration of how far back logs should be retrieved")
	cmd.PersistentFlags().Int64Var(&options.tail, "tail", options.tail, "Last number of log lines to show for a given container. -1 does not show previous log lines")
	cmd.PersistentFlags().BoolVarP(&options.timestamps, "timestamps", "t", options.timestamps, "Print timestamps for each given log line")

	return cmd
}

func runLogOutput(opts *logCmdConfig) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	podInterface := opts.clientset.CoreV1().Pods(opts.Namespace)
	tails := make(map[string]*stern.Tail)

	// This channel serializes all log output.
	// It is intended to workaround https://github.com/wercker/stern/issues/96,
	// and is based on
	// https://github.com/oandrew/stern/commit/8723308e46b408e239ce369ced12706d01479532
	logC := make(chan string, 1024)

	go func() {
		for {
			select {
			case str := <-logC:
				fmt.Fprintf(os.Stdout, str)
			case <-ctx.Done():
				break
			}
		}
	}()

	added, _, err := stern.Watch(
		ctx,
		podInterface,
		opts.PodQuery,
		opts.ContainerQuery,
		opts.ExcludeContainerQuery,
		opts.ContainerState,
		opts.LabelSelector,
	)

	if err != nil {
		return err
	}

	go func() {
		for a := range added {
			tailOpts := &stern.TailOptions{
				SinceSeconds: int64(opts.Since.Seconds()),
				Timestamps:   opts.Timestamps,
				TailLines:    opts.TailLines,
				Exclude:      opts.Exclude,
				Include:      opts.Include,
				Namespace:    true,
			}

			newTail := stern.NewTail(a.Namespace, a.Pod, a.Container, opts.Template, tailOpts)
			if _, ok := tails[a.GetID()]; !ok {
				tails[a.GetID()] = newTail
			}
			newTail.Start(ctx, podInterface, logC)
		}
	}()

	<-sigCh
	return nil
}
