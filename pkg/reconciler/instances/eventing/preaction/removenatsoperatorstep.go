package preaction

import (
	"strings"

	"github.com/kyma-incubator/reconciler/pkg/reconciler/chart"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/instances/eventing/log"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/service"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	removeNatsOperatorStepName = "removeNatsOperator"
	natsOperatorDeploymentName = "nats-operator"
	natsOperatorLastVersion    = "1.24.7"
	natsSubChartPath           = "eventing/charts/nats"
	eventingNats               = "eventing-nats"
	oldConfigValue             = "global.image.repository"
	newConfigValue             = "eu.gcr.io/kyma-project"
)

var (
	natsOperatorCRDsToDelete = []string{"natsclusters.nats.io", "natsserviceroles.nats.io"}
)

type kubeClientProvider func(context *service.ActionContext, logger *zap.SugaredLogger) (*kubeclient.KubeClient, error)

type removeNatsOperatorStep struct {
	kubeClientProvider
}

func newRemoveNatsOperatorStep() *removeNatsOperatorStep {
	return &removeNatsOperatorStep{
		kubeClientProvider: defaultKubeClientProvider,
	}
}

func defaultKubeClientProvider(context *service.ActionContext, logger *zap.SugaredLogger) (*kubeclient.KubeClient, error) {
	kubeClient, err := kubeclient.NewKubeClient(context.Task.Kubeconfig, logger)
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

func (r *removeNatsOperatorStep) Execute(context *service.ActionContext, logger *zap.SugaredLogger) error {
	kubeClient, err := r.kubeClientProvider(context, logger)
	if err != nil {
		return err
	}

	// decorate logger
	logger = logger.With(log.KeyStep, removeNatsOperatorStepName)

	err = r.removeNatsOperatorResources(context, kubeClient, logger)
	if err != nil {
		return err
	}

	//err = r.removeNatsOperatorCRDs(kubeClient, logger)
	return err
}

func (r *removeNatsOperatorStep) removeNatsOperatorResources(context *service.ActionContext, kubeClient *kubeclient.KubeClient, logger *zap.SugaredLogger) error {
	// get charts from the version 1.24.x, where the NATS-operator resources still exist
	comp := chart.NewComponentBuilder(natsOperatorLastVersion, natsSubChartPath).
		WithConfiguration(map[string]interface{}{ //todo add more logging
			oldConfigValue: newConfigValue, // replace the missing global value, as we are rendering on the subchart level
		}).
		WithNamespace(namespace).
		Build()

	manifest, err := context.ChartProvider.RenderManifest(comp)
	if err != nil {
		return err
	}

	// set the right eventing name, which went lost after rendering
	manifest.Manifest = strings.ReplaceAll(manifest.Manifest, natsSubChartPath, eventingNats)

	// remove all the existing nats-operator resources, installed via charts
	_, err = context.KubeClient.Delete(context.Context, manifest.Manifest, namespace)
	return err
}

// delete the leftover CRDs, which were outside of charts
func (r *removeNatsOperatorStep) removeNatsOperatorCRDs(kubeClient *kubeclient.KubeClient, log *zap.SugaredLogger) error {
	for _, crdName := range natsOperatorCRDsToDelete {
		_, err := kubeClient.DeleteResourceByKindAndNameAndNamespace("customresourcedefinitions", crdName, namespace, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Errorf("Failed to delete the nats-operator CRDs, name='%s', namespace='%s': %s", crdName, namespace, err)
			return err
		}
	}
	return nil
}
