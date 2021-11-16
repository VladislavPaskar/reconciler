package progress

import (
	"context"
	"encoding/json"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const expectedReadyReplicas = 1
const expectedReadyDaemonSet = 1

var DeployGVK = schema.GroupVersionResource{
	Version:  appsv1.SchemeGroupVersion.Version,
	Group:    appsv1.SchemeGroupVersion.Group,
	Resource: "deployments",
}

func isDeploymentReady(ctx context.Context, client dynamic.Interface, object *resource) (bool, error) {
	deploymentUnstrct, err := client.Resource(DeployGVK).Namespace(object.namespace).Get(ctx, object.name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	deployment, err := toDeploy(deploymentUnstrct)
	if err != nil {
		return false, err
	}
	replicaSet, err := getLatestReplicaSet(ctx, deployment, client)
	if err != nil || replicaSet == nil {
		return false, err
	}

	isReady := replicaSet.Status.ReadyReplicas >= expectedReadyReplicas
	return isReady, nil
}

func isStatefulSetReady(ctx context.Context, client kubernetes.Interface, object *resource) (bool, error) {
	statefulSet, err := client.AppsV1().StatefulSets(object.namespace).Get(ctx, object.name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	var partition, replicas = 0, 1
	if statefulSet.Spec.UpdateStrategy.RollingUpdate != nil && statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
		partition = int(*statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	}

	if statefulSet.Spec.Replicas != nil {
		replicas = int(*statefulSet.Spec.Replicas)
	}

	expectedReplicas := replicas - partition
	if int(statefulSet.Status.UpdatedReplicas) != expectedReplicas {
		return false, nil
	}

	isReady := int(statefulSet.Status.ReadyReplicas) == replicas
	return isReady, nil
}

func isPodReady(ctx context.Context, client dynamic.Interface, object *resource) (bool, error) {
	podGVK := schema.GroupVersionResource{
		Version:  corev1.SchemeGroupVersion.Version,
		Group:    corev1.SchemeGroupVersion.Group,
		Resource: "pods",
	}
	podUnstr, err := client.Resource(podGVK).Namespace(object.namespace).Get(ctx, object.name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	pod, err := toPod(podUnstr)
	if err != nil {
		return false, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false, nil
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			return false, nil
		}
	}
	//deletion timestamp determines whether pod is terminating or running (nil == running)
	return pod.ObjectMeta.DeletionTimestamp == nil, nil
}

func toDeploy(unstucrdDeploy *unstructured.Unstructured) (*appsv1.Deployment, error) {
	deploy := new(appsv1.Deployment)
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstucrdDeploy.Object, deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func toPod(unstucrdPod *unstructured.Unstructured) (*corev1.Pod, error) {
	pod := new(corev1.Pod)
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstucrdPod.Object, pod)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

func isDaemonSetReady(ctx context.Context, client kubernetes.Interface, object *resource) (bool, error) {
	daemonSet, err := client.AppsV1().DaemonSets(object.namespace).Get(ctx, object.name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if daemonSet.Status.UpdatedNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		return false, nil
	}

	isReady := int(daemonSet.Status.NumberReady) >= expectedReadyDaemonSet
	return isReady, nil
}

func isJobReady(ctx context.Context, client kubernetes.Interface, object *resource) (bool, error) {
	job, err := client.BatchV1().Jobs(object.namespace).Get(ctx, object.name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for _, condition := range job.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			return false, nil
		}
	}
	return true, err
}

func getLatestReplicaSet(ctx context.Context, deployment *appsv1.Deployment, client dynamic.Interface) (*appsv1.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}
	rsGVK := schema.GroupVersionResource{
		Version:  appsv1.SchemeGroupVersion.Version,
		Group:    appsv1.SchemeGroupVersion.Group,
		Resource: "replicasets",
	}
	allReplicaSetsUnstr, err := client.Resource(rsGVK).Namespace(deployment.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	allReplicaSets, err := toReplicaSetList(allReplicaSetsUnstr)
	if err != nil {
		return nil, err
	}

	var ownedReplicaSets []*appsv1.ReplicaSet
	for i := range allReplicaSets.Items {
		if metav1.IsControlledBy(&allReplicaSets.Items[i], deployment) {
			ownedReplicaSets = append(ownedReplicaSets, &allReplicaSets.Items[i])
		}
	}

	if len(ownedReplicaSets) == 0 {
		return nil, nil
	}

	sort.Sort(replicaSetsByCreationTimestamp(ownedReplicaSets))
	return ownedReplicaSets[len(ownedReplicaSets)-1], nil
}

func toReplicaSetList(unstructuredList *unstructured.UnstructuredList) (*appsv1.ReplicaSetList, error) {
	rsList := new(appsv1.ReplicaSetList)
	rsListBytes, err := unstructuredList.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rsListBytes, rsList)
	if err != nil {
		return nil, err
	}
	return rsList, nil
}

type replicaSetsByCreationTimestamp []*appsv1.ReplicaSet

func (o replicaSetsByCreationTimestamp) Len() int      { return len(o) }
func (o replicaSetsByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o replicaSetsByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}
