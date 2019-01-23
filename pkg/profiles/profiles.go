package profiles

import (
	"bytes"
	"errors"
	"io"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	pb "github.com/linkerd/linkerd2-proxy-api/go/destination"
	sp "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha1"
	"github.com/linkerd/linkerd2/pkg/util"
)

type profileTemplateConfig struct {
	ControlPlaneNamespace string
	ServiceNamespace      string
	ServiceName           string
	ClusterZone           string
}

// DefaultRetryBudget is used for routes which do not specify one.
var DefaultRetryBudget = pb.RetryBudget{
	MinRetriesPerSecond: 10,
	RetryRatio:          0.2,
	Ttl: &duration.Duration{
		Seconds: 10,
	},
}

// ToServiceProfile returns a Proxy API DestinationProfile, given a
// ServiceProfile.
func ToServiceProfile(profile *sp.ServiceProfileSpec) (*pb.DestinationProfile, error) {
	routes := make([]*pb.Route, 0)
	for _, route := range profile.Routes {
		pbRoute, err := ToRoute(route)
		if err != nil {
			return nil, err
		}
		routes = append(routes, pbRoute)
	}
	budget := DefaultRetryBudget
	if profile.RetryBudget != nil {
		budget.MinRetriesPerSecond = profile.RetryBudget.MinRetriesPerSecond
		budget.RetryRatio = profile.RetryBudget.RetryRatio
		ttl, err := time.ParseDuration(profile.RetryBudget.TTL)
		if err != nil {
			return nil, err
		}
		budget.Ttl = &duration.Duration{
			Seconds: int64(ttl / time.Second),
			Nanos:   int32(ttl % time.Second),
		}
	}
	return &pb.DestinationProfile{
		Routes:      routes,
		RetryBudget: &budget,
	}, nil
}

// ToRoute returns a Proxy API Route, given a ServiceProfile Route.
func ToRoute(route *sp.RouteSpec) (*pb.Route, error) {
	cond, err := ToRequestMatch(route.Condition)
	if err != nil {
		return nil, err
	}
	rcs := make([]*pb.ResponseClass, 0)
	for _, rc := range route.ResponseClasses {
		pbRc, err := ToResponseClass(rc)
		if err != nil {
			return nil, err
		}
		rcs = append(rcs, pbRc)
	}
	ret := pb.Route{
		Condition:       cond,
		ResponseClasses: rcs,
		MetricsLabels:   map[string]string{"route": route.Name},
		IsRetryable:     route.IsRetryable,
	}
	return &ret, nil
}

// ToResponseClass returns a Proxy API ResponseClass, given a ServiceProfile
// ResponseClass.
func ToResponseClass(rc *sp.ResponseClass) (*pb.ResponseClass, error) {
	cond, err := ToResponseMatch(rc.Condition)
	if err != nil {
		return nil, err
	}
	return &pb.ResponseClass{
		Condition: cond,
		IsFailure: rc.IsFailure,
	}, nil
}

// ToResponseMatch returns a Proxy API ResponseMatch, given a ServiceProfile
// ResponseMatch.
func ToResponseMatch(rspMatch *sp.ResponseMatch) (*pb.ResponseMatch, error) {
	if rspMatch == nil {
		return nil, errors.New("missing response match")
	}
	err := ValidateResponseMatch(rspMatch)
	if err != nil {
		return nil, err
	}

	matches := make([]*pb.ResponseMatch, 0)

	if rspMatch.All != nil {
		all := make([]*pb.ResponseMatch, 0)
		for _, m := range rspMatch.All {
			pbM, err := ToResponseMatch(m)
			if err != nil {
				return nil, err
			}
			all = append(all, pbM)
		}
		matches = append(matches, &pb.ResponseMatch{
			Match: &pb.ResponseMatch_All{
				All: &pb.ResponseMatch_Seq{
					Matches: all,
				},
			},
		})
	}

	if rspMatch.Any != nil {
		any := make([]*pb.ResponseMatch, 0)
		for _, m := range rspMatch.Any {
			pbM, err := ToResponseMatch(m)
			if err != nil {
				return nil, err
			}
			any = append(any, pbM)
		}
		matches = append(matches, &pb.ResponseMatch{
			Match: &pb.ResponseMatch_Any{
				Any: &pb.ResponseMatch_Seq{
					Matches: any,
				},
			},
		})
	}

	if rspMatch.Status != nil {
		matches = append(matches, &pb.ResponseMatch{
			Match: &pb.ResponseMatch_Status{
				Status: &pb.HttpStatusRange{
					Max: rspMatch.Status.Max,
					Min: rspMatch.Status.Min,
				},
			},
		})
	}

	if rspMatch.Not != nil {
		not, err := ToResponseMatch(rspMatch.Not)
		if err != nil {
			return nil, err
		}
		matches = append(matches, &pb.ResponseMatch{
			Match: &pb.ResponseMatch_Not{
				Not: not,
			},
		})
	}

	if len(matches) == 0 {
		return nil, errors.New("A response match must have a field set")
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return &pb.ResponseMatch{
		Match: &pb.ResponseMatch_All{
			All: &pb.ResponseMatch_Seq{
				Matches: matches,
			},
		},
	}, nil
}

// ToRequestMatch returns a Proxy API RequestMatch, given a ServiceProfile
// RequestMatch.
func ToRequestMatch(reqMatch *sp.RequestMatch) (*pb.RequestMatch, error) {
	if reqMatch == nil {
		return nil, errors.New("missing request match")
	}
	err := ValidateRequestMatch(reqMatch)
	if err != nil {
		return nil, err
	}

	matches := make([]*pb.RequestMatch, 0)

	if reqMatch.All != nil {
		all := make([]*pb.RequestMatch, 0)
		for _, m := range reqMatch.All {
			pbM, err := ToRequestMatch(m)
			if err != nil {
				return nil, err
			}
			all = append(all, pbM)
		}
		matches = append(matches, &pb.RequestMatch{
			Match: &pb.RequestMatch_All{
				All: &pb.RequestMatch_Seq{
					Matches: all,
				},
			},
		})
	}

	if reqMatch.Any != nil {
		any := make([]*pb.RequestMatch, 0)
		for _, m := range reqMatch.Any {
			pbM, err := ToRequestMatch(m)
			if err != nil {
				return nil, err
			}
			any = append(any, pbM)
		}
		matches = append(matches, &pb.RequestMatch{
			Match: &pb.RequestMatch_Any{
				Any: &pb.RequestMatch_Seq{
					Matches: any,
				},
			},
		})
	}

	if reqMatch.Method != "" {
		matches = append(matches, &pb.RequestMatch{
			Match: &pb.RequestMatch_Method{
				Method: util.ParseMethod(reqMatch.Method),
			},
		})
	}

	if reqMatch.Not != nil {
		not, err := ToRequestMatch(reqMatch.Not)
		if err != nil {
			return nil, err
		}
		matches = append(matches, &pb.RequestMatch{
			Match: &pb.RequestMatch_Not{
				Not: not,
			},
		})
	}

	if reqMatch.PathRegex != "" {
		matches = append(matches, &pb.RequestMatch{
			Match: &pb.RequestMatch_Path{
				Path: &pb.PathMatch{
					Regex: reqMatch.PathRegex,
				},
			},
		})
	}

	if len(matches) == 0 {
		return nil, errors.New("A request match must have a field set")
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return &pb.RequestMatch{
		Match: &pb.RequestMatch_All{
			All: &pb.RequestMatch_Seq{
				Matches: matches,
			},
		},
	}, nil
}

// ValidateRequestMatch validates whether a ServiceProfile RequestMatch has at
// least one field set.
func ValidateRequestMatch(reqMatch *sp.RequestMatch) error {
	matchKindSet := false
	if reqMatch.All != nil {
		matchKindSet = true
		for _, child := range reqMatch.All {
			err := ValidateRequestMatch(child)
			if err != nil {
				return err
			}
		}
	}
	if reqMatch.Any != nil {
		matchKindSet = true
		for _, child := range reqMatch.Any {
			err := ValidateRequestMatch(child)
			if err != nil {
				return err
			}
		}
	}
	if reqMatch.Method != "" {
		matchKindSet = true
	}
	if reqMatch.Not != nil {
		matchKindSet = true
		err := ValidateRequestMatch(reqMatch.Not)
		if err != nil {
			return err
		}
	}
	if reqMatch.PathRegex != "" {
		matchKindSet = true
	}

	if !matchKindSet {
		return errors.New("A request match must have a field set")
	}

	return nil
}

// ValidateResponseMatch validates whether a ServiceProfile ResponseMatch has at
// least one field set, and sanity checks the Status Range.
func ValidateResponseMatch(rspMatch *sp.ResponseMatch) error {
	invalidRangeErr := errors.New("Range maximum cannot be smaller than minimum")
	matchKindSet := false
	if rspMatch.All != nil {
		matchKindSet = true
		for _, child := range rspMatch.All {
			err := ValidateResponseMatch(child)
			if err != nil {
				return err
			}
		}
	}
	if rspMatch.Any != nil {
		matchKindSet = true
		for _, child := range rspMatch.Any {
			err := ValidateResponseMatch(child)
			if err != nil {
				return err
			}
		}
	}
	if rspMatch.Status != nil {
		if rspMatch.Status.Max != 0 && rspMatch.Status.Min != 0 && rspMatch.Status.Max < rspMatch.Status.Min {
			return invalidRangeErr
		}
		matchKindSet = true
	}
	if rspMatch.Not != nil {
		matchKindSet = true
		err := ValidateResponseMatch(rspMatch.Not)
		if err != nil {
			return err
		}
	}

	if !matchKindSet {
		return errors.New("A response match must have a field set")
	}

	return nil
}

func buildConfig(namespace, service, controlPlaneNamespace string) *profileTemplateConfig {
	return &profileTemplateConfig{
		ControlPlaneNamespace: controlPlaneNamespace,
		ServiceNamespace:      namespace,
		ServiceName:           service,
		ClusterZone:           "svc.cluster.local",
	}
}

// RenderProfileTemplate renders a ServiceProfile template to a buffer, given a
// namespace, service, and control plane namespace.
func RenderProfileTemplate(namespace, service, controlPlaneNamespace string, w io.Writer) error {
	config := buildConfig(namespace, service, controlPlaneNamespace)
	template, err := template.New("profile").Parse(Template)
	if err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	err = template.Execute(buf, config)
	if err != nil {
		return err
	}

	_, err = w.Write(buf.Bytes())
	return err
}
