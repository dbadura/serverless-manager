package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func (s *Serverless) IsInState(state State) bool {
	return s.Status.State == state
}

func (s *Serverless) IsCondition(conditionType ConditionType) bool {
	return meta.FindStatusCondition(
		s.Status.Conditions, string(conditionType),
	) != nil
}

func (s *Serverless) IsConditionTrue(conditionType ConditionType) bool {
	condition := meta.FindStatusCondition(s.Status.Conditions, string(conditionType))
	return condition != nil && condition.Status == metav1.ConditionTrue
}

const (
	DefaultEnableInternal    = false
	DefaultRegistryAddress   = "k3d-kyma-registry:5000"
	DefaultServerAddress     = "k3d-kyma-registry:5000"
	DefaultPublisherProxyURL = "http://eventing-publisher-proxy.kyma-system.svc.cluster.local/publish"
	DefaultTraceCollectorURL = "http://telemetry-otlp-traces.kyma-system.svc.cluster.local:4318/v1/traces"
)

func (s *ServerlessSpec) Default() {
	// if DockerRegistry struct is nil configure use of k3d registry
	if s.DockerRegistry == nil {
		s.DockerRegistry = &DockerRegistry{}
	}
	if s.DockerRegistry.EnableInternal == nil {
		s.DockerRegistry.EnableInternal = pointer.Bool(DefaultEnableInternal)
	}

	if s.EventPublisherProxyURL == nil {
		s.EventPublisherProxyURL = pointer.String(DefaultPublisherProxyURL)
	}
	if s.TraceCollectorURL == nil {
		s.TraceCollectorURL = pointer.String(DefaultTraceCollectorURL)
	}
}

func (dr *DockerRegistry) IsInternalEnabled() bool {
	if dr != nil && dr.EnableInternal != nil {
		return *dr.EnableInternal
	}

	return false
}
