package serviceaccounts

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	ServiceAccountSteveType = "serviceaccount"
)

func IsServiceAccountReady(rancherClient *rancher.Client, clusterId, namespace, serviceAccountName string) error {
	userAccountID := fmt.Sprintf("%s/%s", namespace, serviceAccountName)
	steveClient, err := rancherClient.Steve.ProxyDownstream(clusterId)
	if err != nil {
		return err
	}

	err = kwait.Poll(500*time.Millisecond, 2*time.Minute, func() (done bool, err error) {
		serviceAccount, err := steveClient.SteveType(ServiceAccountSteveType).ByID(userAccountID)
		if err != nil {
			if strings.Contains(err.Error(), "Status [404 Not Found]") {
				return false, nil
			} else {
				return false, err
			}
		} else if serviceAccount.State.Name == "active" {
			return true, nil
		}

		return false, nil
	})

	return err
}
