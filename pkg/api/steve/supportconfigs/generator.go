// Package supportconfigs provides a HTTPHandler to serve supportconfigs. This handler should be registered at Endpoint
package supportconfigs

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/rancher/rancher/pkg/auth/util"
	v1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	"github.com/rancher/rancher/pkg/managedcharts/cspadapter"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/sirupsen/logrus"
	authzv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
	authv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

const (
	// Endpoint The endpoint that this URL is accessible at - used for routing and for SARs
	Endpoint            = "/v1/generateSUSERancherSupportConfig"
	cspAdapterConfigmap = "csp-config"
	tarContentType      = "application/x-tar"
	logPrefix           = "support-config-generator"
)

var errNotFound = errors.New("not implemented")

// Handler implements http.Handler - and serves supportconfigs (tar file which contains support relevant information)
type Handler struct {
	ConfigMaps           v1.ConfigMapInterface
	SubjectAccessReviews authv1.SubjectAccessReviewInterface
	adapterUtil          *cspadapter.ChartUtil
}

// NewHandler creates a handler using the clients defined in scaledContext
func NewHandler(scaledContext *config.ScaledContext) Handler {
	return Handler{
		ConfigMaps:           scaledContext.Core.ConfigMaps(cspadapter.ChartNamespace),
		SubjectAccessReviews: scaledContext.K8sClient.AuthorizationV1().SubjectAccessReviews(),
		adapterUtil:          cspadapter.NewChartUtil(scaledContext.Wrangler.RESTClientGetter),
	}
}

// ServerHTTP implements http.Handler - attempts to authenticate/authorize the user. Returns a tar of support information
// if the user can get the backing configmap in the adapter namespace
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	authorized, err := h.authorize(request)
	if err != nil {
		util.ReturnHTTPError(writer, request, http.StatusForbidden, http.StatusText(http.StatusForbidden))
		logrus.Errorf("[%s] Failed to authorize user with error: %s", logPrefix, err.Error())
		return
	}
	if !authorized {
		util.ReturnHTTPError(writer, request, http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}
	_, err = h.adapterUtil.GetRelease()
	if err != nil {
		if errors.Is(err, cspadapter.ErrNotFound) {
			// if the adapter is not installed, return a 501 so the user knows to install the adapter
			util.ReturnHTTPError(writer, request, http.StatusNotImplemented, "csp-adapter must be installed in order to to generate supportconfigs")
			return
		}
		logrus.Warnf("[%s] Error when attempting to determine if adapter is installed, %s", logPrefix, err)
		util.ReturnHTTPError(writer, request, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	logrus.Infof("[%s] Generating supportconfig", logPrefix)
	archive, err := h.generateSupportConfig()
	logrus.Infof("[%s] Done Generating supportconfig", logPrefix)
	if err != nil {
		if errors.Is(err, errNotFound) {
			util.ReturnHTTPError(writer, request, http.StatusServiceUnavailable, "supportconfig not yet generated, try again later")
			return
		}
		logrus.Errorf("[%s] Error when generating supportconfig %v", logPrefix, err)
		util.ReturnHTTPError(writer, request, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	writer.Header().Set("Content-Type", tarContentType)
	writer.Header().Set("Content-Disposition", "attachment; filename=\"supportconfig_rancher.tar\"")
	n, err := io.Copy(writer, archive)
	if err != nil {
		logrus.Warnf("set archive on http response writer: %v", err)
		util.ReturnHTTPError(writer, request, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	logrus.Debugf("[%s] wrote %v bytes in archive response", logPrefix, n)
}

// authorize checks to see if the user can get the csp adapter configmap. Returns a bool (if the user is authorized)
// and optionally an error
func (h *Handler) authorize(r *http.Request) (bool, error) {
	userInfo, ok := request.UserFrom(r.Context())
	if !ok {
		return false, fmt.Errorf("unable to extract user info from context")
	}
	response, err := h.SubjectAccessReviews.Create(r.Context(), &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Resource:  "configmap",
				Verb:      "get",
				Name:      cspAdapterConfigmap,
				Namespace: cspadapter.ChartNamespace,
			},
			User:   userInfo.GetName(),
			Groups: userInfo.GetGroups(),
			Extra:  convertExtra(userInfo.GetExtra()),
			UID:    userInfo.GetUID(),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to create sar %s", err)
	}
	if !response.Status.Allowed {
		return false, nil
	}
	return true, nil
}

// generateSupportConfig produces an io.Reader with the supportconfig ready to be returned using a http.ResponseWriter
// Returns an err if it can't get the support config
func (h *Handler) generateSupportConfig() (io.Reader, error) {
	cspConfig, err := h.getCSPConfig()
	// csp adapter won't always be installed
	if err != nil {
		return nil, err
	}
	configData, err := json.MarshalIndent(cspConfig, "", "  ")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	files := map[string][]byte{
		"rancher/config.json": configData,
	}

	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(body); err != nil {
			return nil, err
		}
	}

	err = tw.Close()
	return &buf, err
}

// getCSPConfig gets the configmap produced by the csp-adapter returns an error if not able to produce the map. Will return
// an errNotFound if the map is not found at all
func (h *Handler) getCSPConfig() (map[string]interface{}, error) {
	cspConfigMap, err := h.ConfigMaps.Get(cspAdapterConfigmap, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errNotFound
		}
		return nil, err
	}
	retVal := map[string]interface{}{}
	err = json.Unmarshal([]byte(cspConfigMap.Data["data"]), &retVal)
	if err != nil {
		return nil, err
	}

	return retVal, nil
}

func convertExtra(extra map[string][]string) map[string]authzv1.ExtraValue {
	result := map[string]authzv1.ExtraValue{}
	for k, v := range extra {
		result[k] = authzv1.ExtraValue(v)
	}
	return result
}
