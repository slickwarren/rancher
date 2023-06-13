package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	b64 "encoding/base64"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/pipeline"
	"github.com/rancher/rancher/tests/framework/extensions/token"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

var (
	adminPassword = os.Getenv("ADMIN_PASSWORD")
	host          = os.Getenv("HA_HOST")

	clusterID = "local"

	configFileName       = config.ConfigFileName("cattle-config.yaml")
	environmentsFileName = "environments.groovy"

	tokenEnvironmentKey      = "HA_TOKEN"
	kubeconfigEnvironmentKey = "HA_KUBECONFIG"
)

type wrappedConfig struct {
	Configuration *rancher.Config `yaml:"rancher"`
}

func main() {
	rancherConfig := new(rancher.Config)
	rancherConfig.Host = host
	isCleanupEnabled := false
	rancherConfig.Cleanup = &isCleanupEnabled

	adminUser := &management.User{
		Username: "admin",
		Password: adminPassword,
	}

	//create admin token
	var adminToken *management.Token
	err := kwait.Poll(500*time.Millisecond, 5*time.Minute, func() (done bool, err error) {
		adminToken, err = token.GenerateUserToken(adminUser, host)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		logrus.Fatalf("error creating admin token", err)
	}
	rancherConfig.AdminToken = adminToken.Token

	//create config file
	configWrapped := &wrappedConfig{
		Configuration: rancherConfig,
	}
	configData, err := yaml.Marshal(configWrapped)
	if err != nil {
		logrus.Fatalf("error marshaling", err)
	}
	err = configFileName.NewFile(configData)
	if err != nil {
		logrus.Fatalf("error writing yaml", err)
	}
	err = configFileName.SetEnvironmentKey()
	if err != nil {
		logrus.Fatalf("error while setting environment path", err)
	}

	//generate kubeconfig
	session := session.NewSession()
	client, err := rancher.NewClient("", session)
	if err != nil {
		logrus.Fatalf("error creating client", err)
	}

	err = pipeline.UpdateEULA(client, clusterID)
	if err != nil {
		logrus.Fatalf("error updating EULA", err)
	}

	cluster, err := client.Management.Cluster.ByID(clusterID)
	if err != nil {
		logrus.Fatalf("error getting cluster", err)
	}
	kubeconfig, err := client.Management.Cluster.ActionGenerateKubeconfig(cluster)
	if err != nil {
		logrus.Fatalf("error getting kubeconfig", err)
	}

	//create groovy environments file
	kubeconfigb64 := b64.StdEncoding.EncodeToString([]byte(kubeconfig.Config))
	kubeconfigEnvironment := newGroovyEnvStr(kubeconfigEnvironmentKey, kubeconfigb64)
	tokenEnvironment := newGroovyEnvStr(tokenEnvironmentKey, adminToken.Token)
	environmentsData := strings.Join([]string{tokenEnvironment, kubeconfigEnvironment}, "\n")
	err = os.WriteFile(environmentsFileName, []byte(environmentsData), 0644)
	if err != nil {
		logrus.Fatalf("error writing yaml", err)
	}

}

func newGroovyEnvStr(key, value string) string {
	prefix := "env"
	return fmt.Sprintf("%v.%v='%v'", prefix, key, value)
}
