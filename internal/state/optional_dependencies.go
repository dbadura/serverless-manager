package state

import (
	"context"
	"fmt"

	"github.com/kyma-project/serverless-manager/api/v1alpha1"
	"github.com/kyma-project/serverless-manager/internal/chart"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// enable or disable serverless optional dependencies based on the Serverless Spec and installed module on the cluster
func sFnOptionalDependencies() stateFn {
	return func(ctx context.Context, r *reconciler, s *systemState) (stateFn, *controllerruntime.Result, error) {
		// TODO: add functionality of auto-detecting these dependencies by checking Eventing and Tracing CRs if user does not override these values.
		// checking these URLs manually is not possible because of lack of istio-sidecar in the serverless-manager

		// update status and condition if status is not up-to-date
		if s.instance.Status.EventPublisherProxyURL != *s.instance.Spec.EventPublisherProxyURL ||
			s.instance.Status.TraceCollectorURL != *s.instance.Spec.TraceCollectorURL {

			s.instance.Status.EventPublisherProxyURL = *s.instance.Spec.EventPublisherProxyURL
			s.instance.Status.TraceCollectorURL = *s.instance.Spec.TraceCollectorURL
			return nextState(
				sFnUpdateProcessingTrueState(
					v1alpha1.ConditionTypeConfigured,
					v1alpha1.ConditionReasonConfigured,
					fmt.Sprintf("Configured with %s Publisher Proxy URL and %s Trace Collector URL.",
						dependencyState(s.instance.Status.EventPublisherProxyURL, v1alpha1.DefaultPublisherProxyURL),
						dependencyState(s.instance.Status.TraceCollectorURL, v1alpha1.DefaultTraceCollectorURL),
					),
				),
			)
		}

		s.chartConfig.Release.Flags = chart.AppendContainersFlags(
			s.chartConfig.Release.Flags,
			s.instance.Status.EventPublisherProxyURL,
			s.instance.Status.TraceCollectorURL,
		)

		return nextState(
			sFnApplyResources(),
		)
	}
}

// returns "default", "custom" or "no" based on args
func dependencyState(url, defaultUrl string) string {
	switch {
	case url == defaultUrl:
		return "default"
	case url == "":
		return "no"
	default:
		return "custom"
	}
}
