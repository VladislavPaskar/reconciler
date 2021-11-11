package preaction

import (
	v1 "k8s.io/api/apps/v1"
	apiextensionsapis "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nats operator resources to deploy
var (
	natsClusterCRD     = &apiextensionsapis.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: natsOperatorCRDsToDelete[0]}}
	natsServiceRoleCRD = &apiextensionsapis.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: natsOperatorCRDsToDelete[1]}}
	natsDeployment     = &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      natsOperatorDeploymentName,
			Namespace: namespace,
		}}
)

