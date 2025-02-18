package state

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/kyma-project/serverless-manager/api/v1alpha1"
	"github.com/kyma-project/serverless-manager/internal/chart"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultResult = ctrl.Result{}
	secretCache   = types.NamespacedName{
		Name:      "serverless-manifest-cache",
		Namespace: "kyma-system", // TODO: detect serverless-manager's namespace
	}
)

type stateFn func(context.Context, *reconciler, *systemState) (stateFn, *ctrl.Result, error)

type cfg struct {
	finalizer string
	chartPath string
	namespace string
}

type systemState struct {
	instance    v1alpha1.Serverless
	snapshot    v1alpha1.ServerlessStatus
	chartConfig *chart.Config
}

func (s *systemState) saveServerlessStatus() {
	result := s.instance.Status.DeepCopy()
	if result == nil {
		result = &v1alpha1.ServerlessStatus{}
	}
	s.snapshot = *result
}

func (s *systemState) setState(state v1alpha1.State) {
	s.instance.Status.State = state
}

func (s *systemState) setServed(served v1alpha1.Served) {
	s.instance.Status.Served = served
}

func (s *systemState) Setup(ctx context.Context, r *reconciler) error {
	s.instance.Spec.Default()

	chartConfig, err := chartConfig(ctx, r, s)
	if err != nil {
		r.log.Errorf("error while preparing chart config: %s", err.Error())
		return err
	}

	s.chartConfig = chartConfig
	return nil
}

func chartConfig(ctx context.Context, r *reconciler, s *systemState) (*chart.Config, error) {
	flags, err := chart.BuildFlags(ctx, r.client, &s.instance)
	if err != nil {
		return nil, fmt.Errorf("resolving manifest failed: %w", err)
	}

	return &chart.Config{
		Ctx:      ctx,
		Log:      r.log,
		Cache:    r.cache,
		CacheKey: secretCache,
		Cluster: chart.Cluster{
			Client: r.client,
			Config: r.config,
		},
		Release: chart.Release{
			Flags:     flags,
			ChartPath: r.chartPath,
			Namespace: s.instance.GetNamespace(),
			Name:      "serverless",
		},
	}, nil
}

type k8s struct {
	client client.Client
	config *rest.Config
	record.EventRecorder
}

type reconciler struct {
	fn     stateFn
	log    *zap.SugaredLogger
	cache  chart.ManifestCache
	result ctrl.Result
	k8s
	cfg
}

func (m *reconciler) stateFnName() string {
	fullName := runtime.FuncForPC(reflect.ValueOf(m.fn).Pointer()).Name()
	splitFullName := strings.Split(fullName, ".")

	if len(splitFullName) < 3 {
		return fullName
	}

	shortName := splitFullName[2]
	return shortName
}

func (m *reconciler) Reconcile(ctx context.Context, v v1alpha1.Serverless) (ctrl.Result, error) {
	state := systemState{instance: v}
	var err error
	var result *ctrl.Result
loop:
	for m.fn != nil && err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break loop

		default:
			m.log.Info(fmt.Sprintf("switching state: %s", m.stateFnName()))
			m.fn, result, err = m.fn(ctx, m, &state)
		}
	}

	if result == nil {
		result = &defaultResult
	}

	m.log.
		With("error", err).
		With("result", result).
		Info("reconciliation done")

	return *result, err
}
