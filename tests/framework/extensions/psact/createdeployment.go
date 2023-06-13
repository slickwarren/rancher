package psact

import (
	"fmt"
	"time"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	steveV1 "github.com/rancher/rancher/tests/framework/clients/rancher/v1"
	"github.com/rancher/rancher/tests/framework/extensions/workloads"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	containerName     = "nginx"
	deploymentName    = "nginx"
	imageName         = "nginx"
	namespace         = "default"
	rancherPrivileged = "rancher-privileged"
	rancherRestricted = "rancher-restricted"
	workload          = "workload"
)

// CreateTestDeployment will create an nginx deployment into the default namespace. If the PSACT value is rancher-privileged, then the
// deployment should successfully create. If the PSACT value is rancher-unprivileged, then the deployment should fail to create.
func CreateNginxDeployment(client *rancher.Client, clusterID string, psact string) (*steveV1.SteveAPIObject, error) {
	labels := map[string]string{}
	labels["workload.user.cattle.io/workloadselector"] = fmt.Sprintf("apps.deployment-%v-%v", namespace, workload)

	containerTemplate := workloads.NewContainer(containerName, imageName, v1.PullAlways, []v1.VolumeMount{}, []v1.EnvFromSource{})
	podTemplate := workloads.NewPodTemplate([]v1.Container{containerTemplate}, []v1.Volume{}, []v1.LocalObjectReference{}, labels)
	deploymentTemplate := workloads.NewDeploymentTemplate(deploymentName, namespace, podTemplate, true, labels)

	steveclient, err := client.Steve.ProxyDownstream(clusterID)
	if err != nil {
		return nil, err
	}

	_, err = steveclient.SteveType(workloads.DeploymentSteveType).Create(deploymentTemplate)
	if err != nil {
		return nil, err
	}

	err = kwait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		steveclient, err := client.Steve.ProxyDownstream(clusterID)
		if err != nil {
			return false, err
		}

		deploymentResp, err := steveclient.SteveType(workloads.DeploymentSteveType).ByID(deploymentTemplate.Namespace + "/" + deploymentTemplate.Name)
		if err != nil {
			return false, err
		}

		deployment := &appv1.Deployment{}
		err = steveV1.ConvertToK8sType(deploymentResp.JSONResp, deployment)
		if err != nil {
			return false, err
		}

		if *deployment.Spec.Replicas == deployment.Status.AvailableReplicas && psact == rancherPrivileged {
			logrus.Infof("Deployment successfully created; this is expected for " + psact + "!")
			return true, nil
		} else if *deployment.Spec.Replicas != deployment.Status.AvailableReplicas && psact == rancherRestricted {
			logrus.Infof("Deployment failed to create; this is expected for " + psact + "!")
			return true, nil
		}

		return false, err
	})

	return nil, err
}
