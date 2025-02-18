package state

import (
	"errors"
	"testing"

	"github.com/kyma-project/serverless-manager/api/v1alpha1"
	"github.com/kyma-project/serverless-manager/internal/chart"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	testDeletingServerless = func() v1alpha1.Serverless {
		serverless := testInstalledServerless
		serverless.Status.State = v1alpha1.StateDeleting
		serverless.Status.Conditions = []metav1.Condition{
			{
				Type:   string(v1alpha1.ConditionTypeDeleted),
				Reason: string(v1alpha1.ConditionReasonDeletion),
				Status: metav1.ConditionUnknown,
			},
		}
		return serverless
	}()
)

func Test_sFnDeleteResources(t *testing.T) {
	t.Run("update condition", func(t *testing.T) {
		s := &systemState{
			instance: v1alpha1.Serverless{},
		}

		stateFn := sFnDeleteResources()
		next, result, err := stateFn(nil, nil, s)

		expectedNext := sFnUpdateDeletingState(
			v1alpha1.ConditionTypeDeleted,
			v1alpha1.ConditionReasonDeletion,
			"Uninstalling",
		)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})
	t.Run("choose deletion strategy", func(t *testing.T) {
		s := &systemState{
			instance: testDeletingServerless,
		}

		stateFn := sFnDeleteResources()
		next, result, err := stateFn(nil, nil, s)

		expectedNext := deletionStrategyBuilder(defaultDeletionStrategy)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})
	t.Run("cascade deletion", func(t *testing.T) {
		stateFn := deletionStrategyBuilder(cascadeDeletionStrategy)

		s := &systemState{
			instance: testDeletingServerless,
			chartConfig: &chart.Config{
				Cache: testEmptyManifestCache(),
				CacheKey: types.NamespacedName{
					Name:      testDeletingServerless.GetName(),
					Namespace: testDeletingServerless.GetNamespace(),
				},
			},
		}

		r := &reconciler{}

		next, result, err := stateFn(nil, r, s)

		expectedNext := sFnUpdateDeletingTrueState(
			v1alpha1.ConditionTypeDeleted,
			v1alpha1.ConditionReasonDeleted,
			"Serverless module deleted",
		)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})

	t.Run("upstream deletion error", func(t *testing.T) {
		stateFn := deletionStrategyBuilder(upstreamDeletionStrategy)

		s := &systemState{
			instance: testDeletingServerless,
			chartConfig: &chart.Config{
				Cache:    testEmptyManifestCache(),
				CacheKey: types.NamespacedName{},
			},
		}

		r := &reconciler{
			log: zap.NewNop().Sugar(),
		}

		next, result, err := stateFn(nil, r, s)

		expectedNext := sFnUpdateErrorState(
			v1alpha1.ConditionTypeDeleted,
			v1alpha1.ConditionReasonDeletionErr,
			errors.New("test error"),
		)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})

	t.Run("safe deletion error while checking orphan resources", func(t *testing.T) {
		wrongStrategy := deletionStrategy("test-strategy")
		stateFn := deletionStrategyBuilder(wrongStrategy)

		s := &systemState{
			instance: testDeletingServerless,
			chartConfig: &chart.Config{
				Cache:    testEmptyManifestCache(),
				CacheKey: types.NamespacedName{},
			},
		}

		r := &reconciler{
			log: zap.NewNop().Sugar(),
		}

		next, result, err := stateFn(nil, r, s)

		expectedNext := sFnUpdateErrorState(
			v1alpha1.ConditionTypeDeleted,
			v1alpha1.ConditionReasonDeletionErr,
			errors.New("test error"),
		)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})

	t.Run("safe deletion", func(t *testing.T) {
		wrongStrategy := deletionStrategy("test-strategy")
		stateFn := deletionStrategyBuilder(wrongStrategy)

		s := &systemState{
			instance: testDeletingServerless,
			chartConfig: &chart.Config{
				Cache: testEmptyManifestCache(),
				CacheKey: types.NamespacedName{
					Name:      testDeletingServerless.GetName(),
					Namespace: testDeletingServerless.GetNamespace(),
				},
			},
		}

		r := &reconciler{
			log: zap.NewNop().Sugar(),
		}

		next, result, err := stateFn(nil, r, s)

		expectedNext := sFnUpdateDeletingTrueState(
			v1alpha1.ConditionTypeDeleted,
			v1alpha1.ConditionReasonDeleted,
			"Serverless module deleted",
		)

		requireEqualFunc(t, expectedNext, next)
		require.Nil(t, result)
		require.Nil(t, err)
	})
}
