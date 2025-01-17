package check

import (
	"context"
	"fmt"
	"log"

	"github.com/kubeshark/kubeshark/config"
	"github.com/kubeshark/kubeshark/kubernetes"
	"github.com/kubeshark/kubeshark/utils"
)

func KubernetesResources(ctx context.Context, kubernetesProvider *kubernetes.Provider) bool {
	log.Printf("\nk8s-components\n--------------------")

	exist, err := kubernetesProvider.DoesNamespaceExist(ctx, config.Config.ResourcesNamespace)
	allResourcesExist := checkResourceExist(config.Config.ResourcesNamespace, "namespace", exist, err)

	exist, err = kubernetesProvider.DoesConfigMapExist(ctx, config.Config.ResourcesNamespace, kubernetes.ConfigMapName)
	allResourcesExist = checkResourceExist(kubernetes.ConfigMapName, "config map", exist, err) && allResourcesExist

	exist, err = kubernetesProvider.DoesServiceAccountExist(ctx, config.Config.ResourcesNamespace, kubernetes.ServiceAccountName)
	allResourcesExist = checkResourceExist(kubernetes.ServiceAccountName, "service account", exist, err) && allResourcesExist

	if config.Config.IsNsRestrictedMode() {
		exist, err = kubernetesProvider.DoesRoleExist(ctx, config.Config.ResourcesNamespace, kubernetes.RoleName)
		allResourcesExist = checkResourceExist(kubernetes.RoleName, "role", exist, err) && allResourcesExist

		exist, err = kubernetesProvider.DoesRoleBindingExist(ctx, config.Config.ResourcesNamespace, kubernetes.RoleBindingName)
		allResourcesExist = checkResourceExist(kubernetes.RoleBindingName, "role binding", exist, err) && allResourcesExist
	} else {
		exist, err = kubernetesProvider.DoesClusterRoleExist(ctx, kubernetes.ClusterRoleName)
		allResourcesExist = checkResourceExist(kubernetes.ClusterRoleName, "cluster role", exist, err) && allResourcesExist

		exist, err = kubernetesProvider.DoesClusterRoleBindingExist(ctx, kubernetes.ClusterRoleBindingName)
		allResourcesExist = checkResourceExist(kubernetes.ClusterRoleBindingName, "cluster role binding", exist, err) && allResourcesExist
	}

	exist, err = kubernetesProvider.DoesServiceExist(ctx, config.Config.ResourcesNamespace, kubernetes.HubServiceName)
	allResourcesExist = checkResourceExist(kubernetes.HubServiceName, "service", exist, err) && allResourcesExist

	allResourcesExist = checkPodResourcesExist(ctx, kubernetesProvider) && allResourcesExist

	return allResourcesExist
}

func checkPodResourcesExist(ctx context.Context, kubernetesProvider *kubernetes.Provider) bool {
	if pods, err := kubernetesProvider.ListPodsByAppLabel(ctx, config.Config.ResourcesNamespace, kubernetes.HubPodName); err != nil {
		log.Printf("%v error checking if '%v' pod is running, err: %v", fmt.Sprintf(utils.Red, "✗"), kubernetes.HubPodName, err)
		return false
	} else if len(pods) == 0 {
		log.Printf("%v '%v' pod doesn't exist", fmt.Sprintf(utils.Red, "✗"), kubernetes.HubPodName)
		return false
	} else if !kubernetes.IsPodRunning(&pods[0]) {
		log.Printf("%v '%v' pod not running", fmt.Sprintf(utils.Red, "✗"), kubernetes.HubPodName)
		return false
	}

	log.Printf("%v '%v' pod running", fmt.Sprintf(utils.Green, "√"), kubernetes.HubPodName)

	if pods, err := kubernetesProvider.ListPodsByAppLabel(ctx, config.Config.ResourcesNamespace, kubernetes.TapperPodName); err != nil {
		log.Printf("%v error checking if '%v' pods are running, err: %v", fmt.Sprintf(utils.Red, "✗"), kubernetes.TapperPodName, err)
		return false
	} else {
		tappers := 0
		notRunningTappers := 0

		for _, pod := range pods {
			tappers += 1
			if !kubernetes.IsPodRunning(&pod) {
				notRunningTappers += 1
			}
		}

		if notRunningTappers > 0 {
			log.Printf("%v '%v' %v/%v pods are not running", fmt.Sprintf(utils.Red, "✗"), kubernetes.TapperPodName, notRunningTappers, tappers)
			return false
		}

		log.Printf("%v '%v' %v pods running", fmt.Sprintf(utils.Green, "√"), kubernetes.TapperPodName, tappers)
		return true
	}
}

func checkResourceExist(resourceName string, resourceType string, exist bool, err error) bool {
	if err != nil {
		log.Printf("%v error checking if '%v' %v exists, err: %v", fmt.Sprintf(utils.Red, "✗"), resourceName, resourceType, err)
		return false
	} else if !exist {
		log.Printf("%v '%v' %v doesn't exist", fmt.Sprintf(utils.Red, "✗"), resourceName, resourceType)
		return false
	}

	log.Printf("%v '%v' %v exists", fmt.Sprintf(utils.Green, "√"), resourceName, resourceType)
	return true
}
