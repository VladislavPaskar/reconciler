package service

import (
	"context"
	"go.uber.org/zap"

	"github.com/avast/retry-go"
	"github.com/kyma-incubator/reconciler/pkg/reconciler"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/callback"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/heartbeat"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/adapter"
	"github.com/pkg/errors"
)

type runner struct {
	*ComponentReconciler
	install *Install

	logger *zap.SugaredLogger
}

func (r *runner) Run(ctx context.Context, model *reconciler.Reconciliation, callback callback.Handler) error {
	heartbeatSender, err := heartbeat.NewHeartbeatSender(ctx, callback, r.logger, heartbeat.Config{
		Interval: r.heartbeatSenderConfig.interval,
		Timeout:  r.heartbeatSenderConfig.timeout,
	})
	if err != nil {
		return err
	}

	retryable := func(heartbeatSender *heartbeat.Sender) func() error {
		return func() error {
			if err := heartbeatSender.Running(); err != nil {
				r.logger.Warnf("Failed to start status updater: %s", err)
				return err
			}
			err := r.reconcile(ctx, model)
			if err != nil {
				r.logger.Warnf("Failing reconciliation of '%s' in version '%s' with profile '%s': %s",
					model.Component, model.Version, model.Profile, err)
				if heartbeatErr := heartbeatSender.Failed(err); heartbeatErr != nil {
					err = errors.Wrap(err, heartbeatErr.Error())
				}
			}
			return err
		}
	}(heartbeatSender)

	//retry the reconciliation in case of an error
	err = retry.Do(retryable,
		retry.Attempts(uint(r.maxRetries)),
		retry.Delay(r.retryDelay),
		retry.LastErrorOnly(false),
		retry.Context(ctx))

	if err == nil {
		r.logger.Infof("Reconciliation of component '%s' for version '%s' finished successfully",
			model.Component, model.Version)
		if err := heartbeatSender.Success(); err != nil {
			return err
		}
	} else if ctx.Err() != nil {
		r.logger.Infof("Reconciliation of component '%s' for version '%s' terminated because context was closed",
			model.Component, model.Version)
		return err
	} else {
		r.logger.Errorf("Retryable reconciliation of component '%s' for version '%s' failed consistently: giving up",
			model.Component, model.Version)
		if heartbeatErr := heartbeatSender.Error(err); heartbeatErr != nil {
			return errors.Wrap(err, heartbeatErr.Error())
		}
	}

	return err
}

func (r *runner) reconcile(ctx context.Context, model *reconciler.Reconciliation) error {
	kubeClient, err := adapter.NewKubernetesClient(model.Kubeconfig, r.logger, &adapter.Config{
		ProgressInterval: r.progressTrackerConfig.interval,
		ProgressTimeout:  r.progressTrackerConfig.timeout,
	})
	if err != nil {
		return err
	}

	chartProvider, err := r.newChartProvider(model.Repository)
	if err != nil {
		return errors.Wrap(err, "Failed to create chart provider instance")
	}

	wsFactory, err := r.workspaceFactory(model.Repository)
	if err != nil {
		return err
	}

	actionHelper := &ActionContext{
		KubeClient:       kubeClient,
		WorkspaceFactory: *wsFactory,
		Context:          ctx,
		Logger:           r.logger,
		ChartProvider:    chartProvider,
		Model:            model,
	}

	if r.preReconcileAction != nil {
		if err := r.preReconcileAction.Run(actionHelper); err != nil {
			r.logger.Warnf("Pre-reconciliation action of '%s' with version '%s' failed: %s",
				model.Component, model.Version, err)
			return err
		}
	}

	if r.reconcileAction == nil {
		if err := r.install.Invoke(ctx, chartProvider, model, kubeClient); err != nil {
			r.logger.Warnf("Default-reconciliation of '%s' with version '%s' failed: %s",
				model.Component, model.Version, err)
			return err
		}
	} else {
		if err := r.reconcileAction.Run(actionHelper); err != nil {
			r.logger.Warnf("Reconciliation action of '%s' with version '%s' failed: %s",
				model.Component, model.Version, err)
			return err
		}
	}

	if r.postReconcileAction != nil {
		if err := r.postReconcileAction.Run(actionHelper); err != nil {
			r.logger.Warnf("Post-reconciliation action of '%s' with version '%s' failed: %s",
				model.Component, model.Version, err)
			return err
		}
	}

	return nil
}
