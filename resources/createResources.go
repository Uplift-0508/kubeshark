package resources

import (
	"context"
	"fmt"
	"log"

	"github.com/kubeshark/kubeshark/config"
	"github.com/kubeshark/kubeshark/errormessage"
	"github.com/kubeshark/kubeshark/kubernetes"
	"github.com/kubeshark/kubeshark/kubeshark"
	"github.com/kubeshark/kubeshark/utils"
	"github.com/kubeshark/worker/models"
	"github.com/op/go-logging"
	core "k8s.io/api/core/v1"
)

func CreateTapKubesharkResources(ctx context.Context, kubernetesProvider *kubernetes.Provider, serializedKubesharkConfig string, isNsRestrictedMode bool, kubesharkResourcesNamespace string, maxEntriesDBSizeBytes int64, hubResources models.Resources, imagePullPolicy core.PullPolicy, logLevel logging.Level, profiler bool) (bool, error) {
	if !isNsRestrictedMode {
		if err := createKubesharkNamespace(ctx, kubernetesProvider, kubesharkResourcesNamespace); err != nil {
			return false, err
		}
	}

	if err := createKubesharkConfigmap(ctx, kubernetesProvider, serializedKubesharkConfig, kubesharkResourcesNamespace); err != nil {
		return false, err
	}

	kubesharkServiceAccountExists, err := createRBACIfNecessary(ctx, kubernetesProvider, isNsRestrictedMode, kubesharkResourcesNamespace, []string{"pods", "services", "endpoints"})
	if err != nil {
		log.Printf(utils.Warning, fmt.Sprintf("Failed to ensure the resources required for IP resolving. Kubeshark will not resolve target IPs to names. error: %v", errormessage.FormatError(err)))
	}

	var serviceAccountName string
	if kubesharkServiceAccountExists {
		serviceAccountName = kubernetes.ServiceAccountName
	} else {
		serviceAccountName = ""
	}

	opts := &kubernetes.HubOptions{
		Namespace:             kubesharkResourcesNamespace,
		PodName:               kubernetes.HubPodName,
		PodImage:              "kubeshark/hub:latest",
		KratosImage:           "",
		KetoImage:             "",
		ServiceAccountName:    serviceAccountName,
		IsNamespaceRestricted: isNsRestrictedMode,
		MaxEntriesDBSizeBytes: maxEntriesDBSizeBytes,
		Resources:             hubResources,
		ImagePullPolicy:       imagePullPolicy,
		LogLevel:              logLevel,
		Profiler:              profiler,
	}

	frontOpts := &kubernetes.HubOptions{
		Namespace:             kubesharkResourcesNamespace,
		PodName:               kubernetes.FrontPodName,
		PodImage:              "kubeshark/worker:latest",
		KratosImage:           "",
		KetoImage:             "",
		ServiceAccountName:    serviceAccountName,
		IsNamespaceRestricted: isNsRestrictedMode,
		MaxEntriesDBSizeBytes: maxEntriesDBSizeBytes,
		Resources:             hubResources,
		ImagePullPolicy:       imagePullPolicy,
		LogLevel:              logLevel,
		Profiler:              profiler,
	}

	if err := createKubesharkHubPod(ctx, kubernetesProvider, opts); err != nil {
		return kubesharkServiceAccountExists, err
	}

	if err := createFrontPod(ctx, kubernetesProvider, frontOpts); err != nil {
		return kubesharkServiceAccountExists, err
	}

	_, err = kubernetesProvider.CreateService(ctx, kubesharkResourcesNamespace, kubernetes.HubServiceName, kubernetes.HubServiceName, 80, int32(config.Config.Hub.PortForward.DstPort), int32(config.Config.Hub.PortForward.SrcPort))
	if err != nil {
		return kubesharkServiceAccountExists, err
	}

	log.Printf("Successfully created service: %s", kubernetes.HubServiceName)

	_, err = kubernetesProvider.CreateService(ctx, kubesharkResourcesNamespace, kubernetes.FrontServiceName, kubernetes.FrontServiceName, 80, int32(config.Config.Front.PortForward.DstPort), int32(config.Config.Front.PortForward.SrcPort))
	if err != nil {
		return kubesharkServiceAccountExists, err
	}

	log.Printf("Successfully created service: %s", kubernetes.FrontServiceName)

	return kubesharkServiceAccountExists, nil
}

func createKubesharkNamespace(ctx context.Context, kubernetesProvider *kubernetes.Provider, kubesharkResourcesNamespace string) error {
	_, err := kubernetesProvider.CreateNamespace(ctx, kubesharkResourcesNamespace)
	return err
}

func createKubesharkConfigmap(ctx context.Context, kubernetesProvider *kubernetes.Provider, serializedKubesharkConfig string, kubesharkResourcesNamespace string) error {
	err := kubernetesProvider.CreateConfigMap(ctx, kubesharkResourcesNamespace, kubernetes.ConfigMapName, serializedKubesharkConfig)
	return err
}

func createRBACIfNecessary(ctx context.Context, kubernetesProvider *kubernetes.Provider, isNsRestrictedMode bool, kubesharkResourcesNamespace string, resources []string) (bool, error) {
	if !isNsRestrictedMode {
		if err := kubernetesProvider.CreateKubesharkRBAC(ctx, kubesharkResourcesNamespace, kubernetes.ServiceAccountName, kubernetes.ClusterRoleName, kubernetes.ClusterRoleBindingName, kubeshark.RBACVersion, resources); err != nil {
			return false, err
		}
	} else {
		if err := kubernetesProvider.CreateKubesharkRBACNamespaceRestricted(ctx, kubesharkResourcesNamespace, kubernetes.ServiceAccountName, kubernetes.RoleName, kubernetes.RoleBindingName, kubeshark.RBACVersion); err != nil {
			return false, err
		}
	}

	return true, nil
}

func createKubesharkHubPod(ctx context.Context, kubernetesProvider *kubernetes.Provider, opts *kubernetes.HubOptions) error {
	pod, err := kubernetesProvider.BuildHubPod(opts, false, "", false)
	if err != nil {
		return err
	}
	if _, err = kubernetesProvider.CreatePod(ctx, opts.Namespace, pod); err != nil {
		return err
	}
	log.Printf("Successfully created pod: [%s]", pod.Name)
	return nil
}

func createFrontPod(ctx context.Context, kubernetesProvider *kubernetes.Provider, opts *kubernetes.HubOptions) error {
	pod, err := kubernetesProvider.BuildFrontPod(opts, false, "", false)
	if err != nil {
		return err
	}
	if _, err = kubernetesProvider.CreatePod(ctx, opts.Namespace, pod); err != nil {
		return err
	}
	log.Printf("Successfully created pod: [%s]", pod.Name)
	return nil
}
