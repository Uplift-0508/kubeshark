package check

import (
	"fmt"
	"log"

	"github.com/kubeshark/kubeshark/config"
	"github.com/kubeshark/kubeshark/kubernetes"
	"github.com/kubeshark/kubeshark/semver"
	"github.com/kubeshark/kubeshark/utils"
)

func KubernetesApi() (*kubernetes.Provider, *semver.SemVersion, bool) {
	log.Printf("\nkubernetes-api\n--------------------")

	kubernetesProvider, err := kubernetes.NewProvider(config.Config.KubeConfigPath(), config.Config.KubeContext)
	if err != nil {
		log.Printf("%v can't initialize the client, err: %v", fmt.Sprintf(utils.Red, "✗"), err)
		return nil, nil, false
	}
	log.Printf("%v can initialize the client", fmt.Sprintf(utils.Green, "√"))

	kubernetesVersion, err := kubernetesProvider.GetKubernetesVersion()
	if err != nil {
		log.Printf("%v can't query the Kubernetes API, err: %v", fmt.Sprintf(utils.Red, "✗"), err)
		return nil, nil, false
	}
	log.Printf("%v can query the Kubernetes API", fmt.Sprintf(utils.Green, "√"))

	return kubernetesProvider, kubernetesVersion, true
}
