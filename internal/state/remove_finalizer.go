package state

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func sFnRemoveFinalizer() stateFn {
	return func(ctx context.Context, r *reconciler, s *systemState) (stateFn, *ctrl.Result, error) {
		if !controllerutil.RemoveFinalizer(&s.instance, r.finalizer) {
			return requeue()
		}

		return nextState(
			sFnUpdateServerless(),
		)
	}
}
