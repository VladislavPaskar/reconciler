package preaction

import (
	"github.com/kyma-incubator/reconciler/pkg/reconciler/chart"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/instances/eventing/log"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/service"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

const (
	removeNatsOperatorStepName = "removeNatsOperator"

	natsOperatorDeploymentName = "nats-operator"
	natsOperatorLastVersion    = "1.24.7"
	natsSubChartPath           = "eventing/charts/nats"
	eventingNats               = "eventing-nats"
	CRDKind                    = "customresourcedefinitions"
)

var (
	natsOperatorCRDsToDelete = []string{"natsclusters.nats.io", "natsserviceroles.nats.io"}
)

type removeNatsOperatorStep struct{}

func (r removeNatsOperatorStep) Execute(context *service.ActionContext, logger *zap.SugaredLogger) error {

	// decorate logger
	logger = logger.With(log.KeyStep, removeNatsOperatorStepName)
	// prepare Kubernetes clientset
	var clientset kubernetes.Interface
	var err error
	if clientset, err = context.KubeClient.Clientset(); err != nil {
		return err
	}

	// remove the old NATS-operator resources if the NATS-operator deployment still exists
	natsOperatorDeployment, err := getDeployment(context, clientset, natsOperatorDeploymentName)
	if err != nil {
		return err
	}

	err = removeNatsOperatorResources(context, logger, natsOperatorDeployment, clientset)
	if err != nil {
		return err
	}

	return nil
}

func removeNatsOperatorResources(context *service.ActionContext, logger *zap.SugaredLogger, natsOperatorDeployment *appsv1.Deployment, clientset kubernetes.Interface) error {
	if natsOperatorDeployment == nil {
		logger.With(log.KeyReason, "no nats-operator deployment found").Info("Step skipped")
		return nil
	}
	// get charts from the version when the old NATS-operator yaml resource definitions still exist
	comp := chart.NewComponentBuilder(natsOperatorLastVersion, natsSubChartPath).
		WithConfiguration(map[string]interface{}{
			"global.image.repository": "eu.gcr.io/kyma-project", // replace the missing global value, as we are rendering on the subchart level
		}).
		WithNamespace(namespace).
		Build()

	manifest, err := context.ChartProvider.RenderManifest(comp)
	if err != nil {
		return err
	}

	// set the right eventing name, which went lost after rendering
	manifest.Manifest = strings.ReplaceAll(manifest.Manifest, natsSubChartPath, eventingNats)


	kubeClient, err := kubeclient.NewKubeClient(context.Task.Kubeconfig, logger)
	if err != nil {
		return err
	}


	// remove all the nats-operator resources, installed via charts
	_, err = context.KubeClient.Delete(context.Context, manifest.Manifest, namespace)
	if err != nil {
		return err
	}

	err = removeNatsOperatorCRDs(kubeClient, logger)
	return err
}

// delete the leftover CRDs, which were outside of charts
func removeNatsOperatorCRDs(kubeClient *kubeclient.KubeClient, log *zap.SugaredLogger) error {
	for _, crdName := range natsOperatorCRDsToDelete {
		_, err := kubeClient.DeleteResourceByKindAndNameAndNamespace("customresourcedefinitions", crdName, namespace, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Errorf("Failed to delete the nats-operator CRDs, name='%s', namespace='%s': %s", crdName, namespace, err)
			return err
		}
	}
	return nil
}
