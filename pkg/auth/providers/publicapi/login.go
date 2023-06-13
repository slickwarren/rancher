package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
	v32 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/auth/providers"
	"github.com/rancher/rancher/pkg/auth/providers/activedirectory"
	"github.com/rancher/rancher/pkg/auth/providers/azure"
	"github.com/rancher/rancher/pkg/auth/providers/github"
	"github.com/rancher/rancher/pkg/auth/providers/googleoauth"
	"github.com/rancher/rancher/pkg/auth/providers/keycloakoidc"
	"github.com/rancher/rancher/pkg/auth/providers/ldap"
	"github.com/rancher/rancher/pkg/auth/providers/local"
	"github.com/rancher/rancher/pkg/auth/providers/oidc"
	"github.com/rancher/rancher/pkg/auth/providers/saml"
	"github.com/rancher/rancher/pkg/auth/settings"
	"github.com/rancher/rancher/pkg/auth/tokens"
	"github.com/rancher/rancher/pkg/auth/util"
	client "github.com/rancher/rancher/pkg/client/generated/management/v3public"
	"github.com/rancher/rancher/pkg/clustermanager"
	"github.com/rancher/rancher/pkg/controllers/managementuser/clusterauthtoken/common"
	v1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	schema "github.com/rancher/rancher/pkg/schemas/management.cattle.io/v3public"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/rancher/rancher/pkg/user"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	CookieName = "R_SESS"
)

func newLoginHandler(ctx context.Context, mgmt *config.ScaledContext) *loginHandler {
	return &loginHandler{
		scaledContext: mgmt,
		userMGR:       mgmt.UserManager,
		tokenMGR:      tokens.NewManager(ctx, mgmt),
		clusterLister: mgmt.Management.Clusters("").Controller().Lister(),
		secretLister:  mgmt.Core.Secrets("").Controller().Lister(),
	}
}

type loginHandler struct {
	scaledContext *config.ScaledContext
	userMGR       user.Manager
	tokenMGR      *tokens.Manager
	clusterLister v3.ClusterLister
	secretLister  v1.SecretLister
}

func (h *loginHandler) login(actionName string, action *types.Action, request *types.APIContext) error {
	if actionName != "login" {
		return httperror.NewAPIError(httperror.ActionNotAvailable, "")
	}

	w := request.Response

	token, unhashedTokenKey, responseType, err := h.createLoginToken(request)
	if err != nil {
		// if user fails to authenticate, hide the details of the exact error. bad credentials will already be APIErrors
		// otherwise, return a generic error message
		if httperror.IsAPIError(err) {
			return err
		}
		return httperror.WrapAPIError(err, httperror.ServerError, "Server error while authenticating")
	}

	if responseType == "cookie" {
		tokenCookie := &http.Cookie{
			Name:     CookieName,
			Value:    token.ObjectMeta.Name + ":" + unhashedTokenKey,
			Secure:   true,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, tokenCookie)
	} else if responseType == "saml" {
		return nil
	} else {
		tokenData, err := tokens.ConvertTokenResource(request.Schemas.Schema(&schema.PublicVersion, client.TokenType), token)
		if err != nil {
			return httperror.WrapAPIError(err, httperror.ServerError, "Server error while authenticating")
		}
		tokenData["token"] = token.ObjectMeta.Name + ":" + unhashedTokenKey
		request.WriteResponse(http.StatusCreated, tokenData)
	}

	return nil
}

// createLoginToken returns token, unhashed token key (where applicable), responseType and error
func (h *loginHandler) createLoginToken(request *types.APIContext) (v3.Token, string, string, error) {
	var userPrincipal v3.Principal
	var groupPrincipals []v3.Principal
	var providerToken string
	logrus.Debugf("Create Token Invoked")

	bytes, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		logrus.Errorf("login failed with error: %v", err)
		return v3.Token{}, "", "", httperror.NewAPIError(httperror.InvalidBodyContent, "")
	}

	generic := &v32.GenericLogin{}
	err = json.Unmarshal(bytes, generic)
	if err != nil {
		logrus.Errorf("unmarshal failed with error: %v", err)
		return v3.Token{}, "", "", httperror.NewAPIError(httperror.InvalidBodyContent, "")
	}
	responseType := generic.ResponseType
	description := generic.Description
	ttl := generic.TTLMillis

	authTimeout := settings.AuthUserSessionTTLMinutes.Get()
	if minutes, err := strconv.ParseInt(authTimeout, 10, 64); err == nil {
		ttl = minutes * 60 * 1000
	}

	var input interface{}
	var providerName string
	switch request.Type {
	case client.LocalProviderType:
		input = &v32.BasicLogin{}
		providerName = local.Name
	case client.GithubProviderType:
		input = &v32.GithubLogin{}
		providerName = github.Name
	case client.ActiveDirectoryProviderType:
		input = &v32.BasicLogin{}
		providerName = activedirectory.Name
	case client.AzureADProviderType:
		input = &v32.AzureADLogin{}
		providerName = azure.Name
	case client.OpenLdapProviderType:
		input = &v32.BasicLogin{}
		providerName = ldap.OpenLdapName
	case client.FreeIpaProviderType:
		input = &v32.BasicLogin{}
		providerName = ldap.FreeIpaName
	case client.PingProviderType:
		input = &v32.SamlLoginInput{}
		providerName = saml.PingName
	case client.ADFSProviderType:
		input = &v32.SamlLoginInput{}
		providerName = saml.ADFSName
	case client.KeyCloakProviderType:
		input = &v32.SamlLoginInput{}
		providerName = saml.KeyCloakName
	case client.OKTAProviderType:
		input = &v32.SamlLoginInput{}
		providerName = saml.OKTAName
	case client.ShibbolethProviderType:
		input = &v32.SamlLoginInput{}
		providerName = saml.ShibbolethName
	case client.GoogleOAuthProviderType:
		input = &v32.GoogleOauthLogin{}
		providerName = googleoauth.Name
	case client.OIDCProviderType:
		input = &v32.OIDCLogin{}
		providerName = oidc.Name
	case client.KeyCloakOIDCProviderType:
		input = &v32.OIDCLogin{}
		providerName = keycloakoidc.Name
	default:
		return v3.Token{}, "", "", httperror.NewAPIError(httperror.ServerError, "unknown authentication provider")
	}

	err = json.Unmarshal(bytes, input)
	if err != nil {
		logrus.Errorf("unmarshal failed with error: %v", err)
		return v3.Token{}, "", "", httperror.NewAPIError(httperror.InvalidBodyContent, "")
	}

	// Authenticate User
	// SAML's login flow is different from the other providers. Unlike the other providers, it gets the logged in user's data via a POST from
	// the identity provider on a separate endpoint specifically for that.

	if providerName == saml.PingName || providerName == saml.ADFSName || providerName == saml.KeyCloakName ||
		providerName == saml.OKTAName || providerName == saml.ShibbolethName {
		err = saml.PerformSamlLogin(providerName, request, input)
		return v3.Token{}, "", "saml", err
	}

	ctx := context.WithValue(request.Request.Context(), util.RequestKey, request.Request)
	userPrincipal, groupPrincipals, providerToken, err = providers.AuthenticateUser(ctx, input, providerName)
	if err != nil {
		return v3.Token{}, "", "", err
	}

	displayName := userPrincipal.DisplayName
	if displayName == "" {
		displayName = userPrincipal.LoginName
	}
	currUser, err := h.userMGR.EnsureUser(userPrincipal.Name, displayName)
	if err != nil {
		return v3.Token{}, "", "", err
	}

	if currUser.Enabled != nil && !*currUser.Enabled {
		return v3.Token{}, "", "", httperror.NewAPIError(httperror.PermissionDenied, "Permission Denied")
	}

	if strings.HasPrefix(responseType, tokens.KubeconfigResponseType) {
		token, tokenValue, err := tokens.GetKubeConfigToken(currUser.Name, responseType, h.userMGR, userPrincipal)
		if err != nil {
			return v3.Token{}, "", "", err
		}

		if err := h.createClusterAuthTokenIfNeeded(token, tokenValue); err != nil {
			return v3.Token{}, "", "", httperror.NewAPIError(httperror.ServerError,
				fmt.Sprintf("Failed to create cluster auth token for cluster [%s], cluster auth endpoint may fail: %v", token.ClusterName, err))
		}
		return *token, tokenValue, responseType, nil
	}

	userExtraInfo := providers.GetUserExtraAttributes(providerName, userPrincipal)

	rToken, unhashedTokenKey, err := h.tokenMGR.NewLoginToken(currUser.Name, userPrincipal, groupPrincipals, providerToken, ttl, description, userExtraInfo)
	return rToken, unhashedTokenKey, responseType, err
}

// createClusterAuthTokenIfNeeded checks if local cluster auth endpoint is enabled. If it is, a cluster auth token
// is created.
func (h *loginHandler) createClusterAuthTokenIfNeeded(token *v3.Token, tokenValue string) error {
	clusterID := token.ClusterName
	if clusterID == "" {
		// Cluster ID being empty is likely because cluster auth endpoint is not being used.
		// Custer auth token is not necessary in the aforementioned scenario.
		return nil
	}
	cluster, err := h.clusterLister.Get("", clusterID)
	if err != nil {
		return err
	}

	if !cluster.Spec.LocalClusterAuthEndpoint.Enabled {
		return nil
	}
	clusterConfig, err := clustermanager.ToRESTConfig(cluster, h.scaledContext, h.secretLister)
	if err != nil {
		return err
	}

	clusterContext, err := config.NewUserContext(h.scaledContext, *clusterConfig, clusterID)
	if err != nil {
		return err
	}

	clusterAuthToken, err := common.NewClusterAuthToken(token, tokenValue)
	if err != nil {
		return err
	}

	_, err = clusterContext.Cluster.ClusterAuthTokens("").Controller().Lister().Get("cattle-system", clusterAuthToken.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err == nil {
		err = clusterContext.Cluster.ClusterAuthTokens("cattle-system").Delete(clusterAuthToken.Name, &metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	var backoff = wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   1,
		Jitter:   0,
		Steps:    7,
	}

	err = wait.ExponentialBackoff(backoff, func() (bool, error) {
		if _, err := clusterContext.Cluster.ClusterAuthTokens("cattle-system").Create(clusterAuthToken); err != nil {
			if errors.IsAlreadyExists(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})

	return err
}
