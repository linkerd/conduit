package main

import (
	validator "github.com/linkerd/linkerd2/controller/sp-validator"
	"github.com/linkerd/linkerd2/controller/sp-validator/tmpl"
	"github.com/linkerd/linkerd2/controller/webhook"
	pkgK8s "github.com/linkerd/linkerd2/pkg/k8s"
)

func main() {
	config := &webhook.Config{
		TemplateStr: tmpl.ValidatingWebhookConfigurationSpec,
		Ops:         &validator.Ops{},
	}
	webhook.Launch(
		config,
		nil,
		9997,
		pkgK8s.SPValidatorWebhookServiceName,
		validator.AdmitSP,
	)
}
