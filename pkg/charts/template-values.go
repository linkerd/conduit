package charts

type (
	// Values contains the top-level elements in the Helm charts
	Values struct {
		Namespace        string
		ClusterDomain    string
		HighAvailability bool
		Identity         *Identity

		Proxy     *Proxy
		ProxyInit *ProxyInit
	}

	// Proxy contains the fields to set the proxy sidecar container
	Proxy struct {
		Capabilities          *Capabilities
		Component             string
		DisableIdentity       bool
		DisableTap            bool
		EnableExternalProfile bool
		Image                 *Image
		LogLevel              string
		SAMountPath           *SAMountPath
		Ports                 *Ports
		Resources             *Resources
		UID                   int64
	}

	// ProxyInit contains the fields to set the proxy-init container
	ProxyInit struct {
		Capabilities        *Capabilities
		IgnoreInboundPorts  string
		IgnoreOutboundPorts string
		Image               *Image
		SAMountPath         *SAMountPath
		Resources           *Resources
	}

	// DebugContainer contains the fields to set the debugging sidecar
	DebugContainer struct {
		Image *Image
	}

	// Image contains the details to define a container image
	Image struct {
		Name       string
		PullPolicy string
		Version    string
	}

	// Ports contains all the port-related setups
	Ports struct {
		Admin    int32
		Control  int32
		Inbound  int32
		Outbound int32
	}

	// Constraints wraps the Limit and Request settings for computational resources
	Constraints struct {
		Limit   string
		Request string
	}

	// Capabilities contains the SecurityContext capabilities to add/drop into the injected
	// containers
	Capabilities struct {
		Add  []string
		Drop []string
	}

	// SAMountPath contains the details for ServiceAccount volume mount
	SAMountPath struct {
		Name      string
		MountPath string
	}

	// Resources represents the computational resources setup for a given container
	Resources struct {
		CPU    Constraints
		Memory Constraints
	}

	// Identity contains the fields to set the identity variables in the proxy
	// sidecar container
	Identity struct {
		TrustAnchorsPEM string
		TrustDomain     string
	}
)
