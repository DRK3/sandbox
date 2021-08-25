/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package startcmd

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	ldrest "github.com/hyperledger/aries-framework-go/pkg/controller/rest/ld"
	ldsvc "github.com/hyperledger/aries-framework-go/pkg/ld"
	"github.com/spf13/cobra"
	"github.com/trustbloc/edge-core/pkg/log"
	"github.com/trustbloc/edge-core/pkg/restapi/logspec"
	cmdutils "github.com/trustbloc/edge-core/pkg/utils/cmd"
	tlsutils "github.com/trustbloc/edge-core/pkg/utils/tls"
	"golang.org/x/oauth2"

	"github.com/trustbloc/sandbox/cmd/common"
	"github.com/trustbloc/sandbox/pkg/restapi/healthcheck"
	"github.com/trustbloc/sandbox/pkg/restapi/issuer"
	"github.com/trustbloc/sandbox/pkg/restapi/issuer/operation"
	tokenIssuer "github.com/trustbloc/sandbox/pkg/token/issuer"
	tokenResolver "github.com/trustbloc/sandbox/pkg/token/resolver"
)

const (
	hostURLFlagName      = "host-url"
	hostURLFlagShorthand = "u"
	hostURLFlagUsage     = "URL to run the issuer instance on. Format: HostName:Port."
	hostURLEnvKey        = "ISSUER_HOST_URL"

	// oauth2 endpoint config flags
	endpointAuthURLFlagName      = "auth-url"
	endpointAuthURLFlagShorthand = "a"
	endpointAuthURLFlagUsage     = "Auth URL for auth2 server. Format: HostName:Port."
	endpointAuthURLEnvKey        = "OAUTH2_ENDPOINT_AUTH_URL"

	endpointTokenURLFlagName      = "token-url"
	endpointTokenURLFlagShorthand = "t"
	endpointTokenURLFlagUsage     = "Token URL for auth2 server. Format: HostName:Port." // #nosec
	endpointTokenURLEnvKey        = "OAUTH2_ENDPOINT_TOKEN_URL"                          // #nosec

	// oauth2 client config flags
	clientRedirectURLFlagName      = "redirect-url"
	clientRedirectURLFlagShorthand = "r"
	clientRedirectURLFlagUsage     = "Redirect URL for auth2 client. Format: HostName:Port."
	clientRedirectURLEnvKey        = "OAUTH2_ISSUER_CLIENT_REDIRECT_URL"

	clientIDFlagName      = "client-id"
	clientIDFlagShorthand = "c"
	clientIDFlagUsage     = "Client ID for issuer auth2 client."
	clientIDEnvKey        = "OAUTH2_ISSUER_CLIENT_ID"

	clientSecretFlagName      = "client-secret"
	clientSecretFlagShorthand = "s"
	clientSecretFlagUsage     = "Client secret for issuer auth2 client."
	clientSecretEnvKey        = "OAUTH2_ISSUER_CLIENT_SECRET" // #nosec

	// oauth2 token introspection config flags
	introspectionURLFlagName      = "introspect-url"
	introspectionURLFlagShorthand = "i"
	introspectionURLFlagUsage     = "Token introspection URL for auth2 server. Format: HostName:Port."
	introspectionURLEnvKey        = "OAUTH2_ENDPOINT_TOKEN_INTROSPECTION_URL"

	tlsCertFileFlagName      = "tls-cert-file"
	tlsCertFileFlagShorthand = ""
	tlsCertFileFlagUsage     = "tls certificate file." +
		" Alternatively, this can be set with the following environment variable: " + tlsCertFileEnvKey
	tlsCertFileEnvKey = "ISSUER_TLS_CERT_FILE"

	tlsKeyFileFlagName      = "tls-key-file"
	tlsKeyFileFlagShorthand = ""
	tlsKeyFileFlagUsage     = "tls key file." +
		" Alternatively, this can be set with the following environment variable: " + tlsKeyFileEnvKey
	tlsKeyFileEnvKey = "ISSUER_TLS_KEY_FILE"

	// content management url config flags
	cmsURLFlagName      = "cms-url"
	cmsURLFlagShorthand = "m"
	cmsURLFlagUsage     = "Content management server (CMS) URL. Format: HostName:Port."
	cmsURLEnvKey        = "ISSUER_CMS_URL"

	// vc service url config flags
	vcsURLFlagName      = "vcs-url"
	vcsURLFlagShorthand = "v"
	vcsURLFlagUsage     = "VC Service URL. Format: HostName:Port."
	vcsURLEnvKey        = "ISSUER_VCS_URL"

	tlsSystemCertPoolFlagName  = "tls-systemcertpool"
	tlsSystemCertPoolFlagUsage = "Use system certificate pool." +
		" Possible values [true] [false]. Defaults to false if not set." +
		" Alternatively, this can be set with the following environment variable: " + tlsSystemCertPoolEnvKey
	tlsSystemCertPoolEnvKey = "ISSUER_TLS_SYSTEMCERTPOOL"

	tlsCACertsFlagName  = "tls-cacerts"
	tlsCACertsFlagUsage = "Comma-Separated list of ca certs path." +
		" Alternatively, this can be set with the following environment variable: " + tlsCACertsEnvKey
	tlsCACertsEnvKey = "ISSUER_TLS_CACERTS"

	requestTokensFlagName  = "request-tokens"
	requestTokensEnvKey    = "ISSUER_REQUEST_TOKENS"
	requestTokensFlagUsage = "Tokens used for http request " +
		" Alternatively, this can be set with the following environment variable: " + requestTokensEnvKey

	// issuer adapter url
	issuerAdapterURLFlagName  = "issuer-adapter-url"
	issuerAdapterURLFlagUsage = "Issuer Adapter Service URL. Format: HostName:Port."
	issuerAdapterURLEnvKey    = "ISSUER_ADAPTER_URL"

	// OIDC flags
	oidcProviderURLFlagName  = "oidc-opurl"
	oidcProviderURLFlagUsage = "URL for the OIDC provider." +
		" Alternatively, this can be set with the following environment variable: " + oidcProviderURLEnvKey
	oidcProviderURLEnvKey = "ISSUER_OIDC_OPURL"

	oidcClientIDFlagName  = "oidc-clientid"
	oidcClientIDFlagUsage = "OAuth2 client_id for OIDC." +
		" Alternatively, this can be set with the following environment variable: " + oidcClientIDEnvKey
	oidcClientIDEnvKey = "ISSUER_OIDC_CLIENTID"

	oidcClientSecretFlagName  = "oidc-clientsecret" //nolint:gosec
	oidcClientSecretFlagUsage = "OAuth2 client secret for OIDC." +
		" Alternatively, this can be set with the following environment variable: " + oidcClientSecretEnvKey
	oidcClientSecretEnvKey = "ISSUER_OIDC_CLIENTSECRET" //nolint:gosec

	oidcCallbackURLFlagName  = "oidc-callback"
	oidcCallbackURLFlagUsage = "Base URL for the OAuth2 callback endpoints." +
		" Alternatively, this can be set with the following environment variable: " + oidcCallbackURLEnvKey
	oidcCallbackURLEnvKey = "ISSUER_OIDC_CALLBACK"

	// remote JSON-LD context provider url
	contextProviderFlagName  = "context-provider-url"
	contextProviderEnvKey    = "ISSUER_CONTEXT_PROVIDER_URL"
	contextProviderFlagUsage = "Remote context provider URL to get JSON-LD contexts from." +
		" This flag can be repeated, allowing setting up multiple context providers." +
		" Alternatively, this can be set with the following environment variable (in CSV format): " +
		contextProviderEnvKey

	tokenLength2 = 2
)

var logger = log.New("issuer-rest")

var getOIDCParametersFunc = getOIDCParameters // nolint: gochecknoglobals

type server interface {
	ListenAndServe(host, certFile, keyFile string, router http.Handler) error
}

// HTTPServer represents an actual HTTP server implementation.
type HTTPServer struct{}

// ListenAndServe starts the server using the standard Go HTTP server implementation.
func (s *HTTPServer) ListenAndServe(host, certFile, keyFile string, router http.Handler) error {
	if certFile != "" && keyFile != "" {
		return http.ListenAndServeTLS(host, certFile, keyFile, router)
	}

	return http.ListenAndServe(host, router)
}

type issuerParameters struct {
	srv                   server
	hostURL               string
	oauth2Config          *oauth2.Config
	tokenIntrospectionURL string
	tlsCertFile           string
	tlsKeyFile            string
	cmsURL                string
	vcsURL                string
	tlsSystemCertPool     bool
	tlsCACerts            []string
	requestTokens         map[string]string
	issuerAdapterURL      string
	logLevel              string
	dbParameters          *common.DBParameters
	oidcParameters        *oidcParameters
	contextProviderURLs   []string
}

type tlsConfig struct {
	certFile       string
	keyFile        string
	systemCertPool bool
	caCerts        []string
}

type oidcParameters struct {
	oidcProviderURL  string
	oidcClientID     string
	oidcClientSecret string
	oidcCallbackURL  string
}

// GetStartCmd returns the Cobra start command.
func GetStartCmd(srv server) *cobra.Command {
	startCmd := createStartCmd(srv)

	createFlags(startCmd)

	return startCmd
}

// nolint: funlen
func createStartCmd(srv server) *cobra.Command { // nolint: gocyclo
	return &cobra.Command{
		Use:   "start",
		Short: "Start issuer",
		Long:  "Start issuer",
		RunE: func(cmd *cobra.Command, args []string) error {
			hostURL, err := cmdutils.GetUserSetVarFromString(cmd, hostURLFlagName, hostURLEnvKey, false)
			if err != nil {
				return err
			}

			oauth2Config, err := getOAuth2Config(cmd)
			if err != nil {
				return err
			}

			tokenIntrospectionURL, err := cmdutils.GetUserSetVarFromString(cmd, introspectionURLFlagName,
				introspectionURLEnvKey, false)
			if err != nil {
				return err
			}

			cmsURL, err := cmdutils.GetUserSetVarFromString(cmd, cmsURLFlagName, cmsURLEnvKey, false)
			if err != nil {
				return err
			}

			vcsURL, err := cmdutils.GetUserSetVarFromString(cmd, vcsURLFlagName, vcsURLEnvKey, false)
			if err != nil {
				return err
			}

			tlsConfg, err := getTLS(cmd)
			if err != nil {
				return err
			}

			requestTokens, err := getRequestTokens(cmd)
			if err != nil {
				return err
			}

			issuerAdapterURL, err := cmdutils.GetUserSetVarFromString(cmd, issuerAdapterURLFlagName,
				issuerAdapterURLEnvKey, false)
			if err != nil {
				return err
			}

			loggingLevel, err := cmdutils.GetUserSetVarFromString(cmd, common.LogLevelFlagName, common.LogLevelEnvKey, true)
			if err != nil {
				return err
			}

			dbParams, err := common.DBParams(cmd)
			if err != nil {
				return err
			}

			oidcParams, err := getOIDCParametersFunc(cmd)
			if err != nil {
				return err
			}

			contextProviderURLs, err := cmdutils.GetUserSetVarFromArrayString(cmd, contextProviderFlagName,
				contextProviderEnvKey, true)
			if err != nil {
				return err
			}

			parameters := &issuerParameters{
				srv:                   srv,
				hostURL:               strings.TrimSpace(hostURL),
				oauth2Config:          oauth2Config,
				tokenIntrospectionURL: strings.TrimSpace(tokenIntrospectionURL),
				tlsCertFile:           tlsConfg.certFile,
				tlsKeyFile:            tlsConfg.keyFile,
				cmsURL:                strings.TrimSpace(cmsURL),
				vcsURL:                strings.TrimSpace(vcsURL),
				tlsSystemCertPool:     tlsConfg.systemCertPool,
				tlsCACerts:            tlsConfg.caCerts,
				requestTokens:         requestTokens,
				issuerAdapterURL:      issuerAdapterURL,
				logLevel:              loggingLevel,
				dbParameters:          dbParams,
				oidcParameters:        oidcParams,
				contextProviderURLs:   contextProviderURLs,
			}

			return startIssuer(parameters)
		},
	}
}

func getOIDCParameters(cmd *cobra.Command) (*oidcParameters, error) {
	oidcProviderURL, err := cmdutils.GetUserSetVarFromString(cmd, oidcProviderURLFlagName, oidcProviderURLEnvKey, true)
	if err != nil {
		return nil, err
	}

	oidcClientID, err := cmdutils.GetUserSetVarFromString(cmd, oidcClientIDFlagName, oidcClientIDEnvKey, true)
	if err != nil {
		return nil, err
	}

	oidcClientSecret, err := cmdutils.GetUserSetVarFromString(
		cmd, oidcClientSecretFlagName, oidcClientSecretEnvKey, true)
	if err != nil {
		return nil, err
	}

	oidcCallbackURL, err := cmdutils.GetUserSetVarFromString(cmd, oidcCallbackURLFlagName, oidcCallbackURLEnvKey, true)
	if err != nil {
		return nil, err
	}

	return &oidcParameters{
		oidcProviderURL:  oidcProviderURL,
		oidcClientID:     oidcClientID,
		oidcClientSecret: oidcClientSecret,
		oidcCallbackURL:  oidcCallbackURL,
	}, nil
}

func getRequestTokens(cmd *cobra.Command) (map[string]string, error) {
	requestTokens, err := cmdutils.GetUserSetVarFromArrayString(cmd, requestTokensFlagName,
		requestTokensEnvKey, true)
	if err != nil {
		return nil, err
	}

	tokens := make(map[string]string)

	for _, token := range requestTokens {
		split := strings.Split(token, "=")
		switch len(split) {
		case tokenLength2:
			tokens[split[0]] = split[1]
		default:
			logger.Warnf("invalid token '%s'", token)
		}
	}

	return tokens, nil
}

func getTLS(cmd *cobra.Command) (*tlsConfig, error) {
	tlsCertFile, err := cmdutils.GetUserSetVarFromString(cmd, tlsCertFileFlagName,
		tlsCertFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	tlsKeyFile, err := cmdutils.GetUserSetVarFromString(cmd, tlsKeyFileFlagName,
		tlsKeyFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	tlsSystemCertPoolString, err := cmdutils.GetUserSetVarFromString(cmd, tlsSystemCertPoolFlagName,
		tlsSystemCertPoolEnvKey, true)
	if err != nil {
		return nil, err
	}

	tlsSystemCertPool := false
	if tlsSystemCertPoolString != "" {
		tlsSystemCertPool, err = strconv.ParseBool(tlsSystemCertPoolString)
		if err != nil {
			return nil, err
		}
	}

	tlsCACerts, err := cmdutils.GetUserSetVarFromArrayString(cmd, tlsCACertsFlagName,
		tlsCACertsEnvKey, true)
	if err != nil {
		return nil, err
	}

	return &tlsConfig{certFile: tlsCertFile,
		keyFile: tlsKeyFile, systemCertPool: tlsSystemCertPool, caCerts: tlsCACerts}, nil
}

func createFlags(startCmd *cobra.Command) {
	common.Flags(startCmd)
	startCmd.Flags().StringP(hostURLFlagName, hostURLFlagShorthand, "", hostURLFlagUsage)
	startCmd.Flags().StringP(endpointAuthURLFlagName, endpointAuthURLFlagShorthand, "", endpointAuthURLFlagUsage)
	startCmd.Flags().StringP(endpointTokenURLFlagName, endpointTokenURLFlagShorthand, "", endpointTokenURLFlagUsage)
	startCmd.Flags().StringP(clientRedirectURLFlagName, clientRedirectURLFlagShorthand, "", clientRedirectURLFlagUsage)
	startCmd.Flags().StringP(clientIDFlagName, clientIDFlagShorthand, "", clientIDFlagUsage)
	startCmd.Flags().StringP(clientSecretFlagName, clientSecretFlagShorthand, "", clientSecretFlagUsage)
	startCmd.Flags().StringP(introspectionURLFlagName, introspectionURLFlagShorthand, "",
		introspectionURLFlagUsage)
	startCmd.Flags().StringP(tlsCertFileFlagName, tlsCertFileFlagShorthand, "", tlsCertFileFlagUsage)
	startCmd.Flags().StringP(tlsKeyFileFlagName, tlsKeyFileFlagShorthand, "", tlsKeyFileFlagUsage)
	startCmd.Flags().StringP(cmsURLFlagName, cmsURLFlagShorthand, "", cmsURLFlagUsage)
	startCmd.Flags().StringP(vcsURLFlagName, vcsURLFlagShorthand, "", vcsURLFlagUsage)
	startCmd.Flags().StringP(tlsSystemCertPoolFlagName, "", "",
		tlsSystemCertPoolFlagUsage)
	startCmd.Flags().StringArrayP(tlsCACertsFlagName, "", []string{}, tlsCACertsFlagUsage)
	startCmd.Flags().StringArrayP(requestTokensFlagName, "", []string{}, requestTokensFlagUsage)

	// did-comm
	startCmd.Flags().StringP(issuerAdapterURLFlagName, "", "", issuerAdapterURLFlagUsage)

	// default log level
	startCmd.Flags().StringP(common.LogLevelFlagName, common.LogLevelFlagShorthand, "", common.LogLevelPrefixFlagUsage)

	// OIDC
	startCmd.Flags().StringP(oidcProviderURLFlagName, "", "", oidcProviderURLFlagUsage)
	startCmd.Flags().StringP(oidcClientIDFlagName, "", "", oidcClientIDFlagUsage)
	startCmd.Flags().StringP(oidcClientSecretFlagName, "", "", oidcClientSecretFlagUsage)
	startCmd.Flags().StringP(oidcCallbackURLFlagName, "", "", oidcCallbackURLFlagUsage)

	startCmd.Flags().StringArrayP(contextProviderFlagName, "", []string{}, contextProviderFlagUsage)
}

func startIssuer(parameters *issuerParameters) error { //nolint:funlen,gocyclo
	if parameters.logLevel != "" {
		common.SetDefaultLogLevel(logger, parameters.logLevel)
	}

	rootCAs, err := tlsutils.GetCertPool(parameters.tlsSystemCertPool, parameters.tlsCACerts)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{RootCAs: rootCAs, MinVersion: tls.VersionTLS12}

	storeProvider, err := common.InitEdgeStore(parameters.dbParameters, logger)
	if err != nil {
		return err
	}

	ldStore, err := common.CreateLDStoreProvider(storeProvider)
	if err != nil {
		return err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	documentLoader, err := common.CreateJSONLDDocumentLoader(ldStore, httpClient, parameters.contextProviderURLs)
	if err != nil {
		return err
	}

	cfg := &operation.Config{
		TokenIssuer:      tokenIssuer.New(parameters.oauth2Config, tokenIssuer.WithTLSConfig(tlsConfig)),
		TokenResolver:    tokenResolver.New(parameters.tokenIntrospectionURL, tokenResolver.WithTLSConfig(tlsConfig)),
		DocumentLoader:   documentLoader,
		CMSURL:           parameters.cmsURL,
		VCSURL:           parameters.vcsURL,
		DIDAuthHTML:      "static/didAuth.html",
		ReceiveVCHTML:    "static/receiveVC.html",
		VCHTML:           "static/vc.html",
		DIDCommHTML:      "static/didcomm.html",
		DIDCOMMVPHTML:    "static/didcommvp.html",
		TLSConfig:        tlsConfig,
		RequestTokens:    parameters.requestTokens,
		IssuerAdapterURL: parameters.issuerAdapterURL,
		StoreProvider:    storeProvider,
		OIDCProviderURL:  parameters.oidcParameters.oidcProviderURL,
		OIDCClientID:     parameters.oidcParameters.oidcClientID,
		OIDCClientSecret: parameters.oidcParameters.oidcClientSecret,
		OIDCCallbackURL:  parameters.oidcParameters.oidcCallbackURL,
	}

	issuerService, err := issuer.New(cfg)
	if err != nil {
		return err
	}

	handlers := issuerService.GetOperations()
	router := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))

	router.PathPrefix("/reader/").Handler(fs)
	router.PathPrefix("/view/").Handler(fs)
	router.PathPrefix("/css/").Handler(fs)
	router.PathPrefix("/img/").Handler(fs)
	router.PathPrefix("/js/").Handler(fs)

	router.Handle("/", fs)
	router.PathPrefix("/drivinglicense").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/drivinglicense.html")
	})
	router.PathPrefix("/creditscorenologin").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/creditscorenologin.html")
	})
	router.PathPrefix("/creditscore").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/creditscore.html")
	})
	router.PathPrefix("/flightbooking").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/flightbooking.html")
	})
	router.PathPrefix("/applygreencard").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/applygreencard.html")
	})
	router.PathPrefix("/greencardlookup").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/greencardlookup.html")
	})
	router.PathPrefix("/connectwallet").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/storeprc.html")
	})

	router.PathPrefix("/uploaddrivinglicense").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/uploaddrivinglicense.html")
	})

	for _, handler := range handlers {
		router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())
	}

	for _, handler := range logspec.New().GetOperations() {
		router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())
	}

	// add health check endpoint
	healthCheckService := healthcheck.New()

	healthCheckHandlers := healthCheckService.GetOperations()
	for _, handler := range healthCheckHandlers {
		router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())
	}

	// handlers for JSON-LD context operations
	for _, handler := range ldrest.New(ldsvc.New(ldStore)).GetRESTHandlers() {
		router.HandleFunc(handler.Path(), handler.Handle()).Methods(handler.Method())
	}

	return parameters.srv.ListenAndServe(parameters.hostURL, parameters.tlsCertFile, parameters.tlsKeyFile, router)
}

func getOAuth2Config(cmd *cobra.Command) (*oauth2.Config, error) {
	authURL, err := cmdutils.GetUserSetVarFromString(cmd, endpointAuthURLFlagName, endpointAuthURLEnvKey, false)
	if err != nil {
		return nil, err
	}

	tokenURL, err := cmdutils.GetUserSetVarFromString(cmd, endpointTokenURLFlagName, endpointTokenURLEnvKey, false)
	if err != nil {
		return nil, err
	}

	hydra := oauth2.Endpoint{
		AuthURL:   strings.TrimSpace(authURL),
		TokenURL:  strings.TrimSpace(tokenURL),
		AuthStyle: oauth2.AuthStyleInHeader, // basic
	}

	redirectURL, err := cmdutils.GetUserSetVarFromString(cmd, clientRedirectURLFlagName, clientRedirectURLEnvKey, false)
	if err != nil {
		return nil, err
	}

	clientID, err := cmdutils.GetUserSetVarFromString(cmd, clientIDFlagName, clientIDEnvKey, false)
	if err != nil {
		return nil, err
	}

	secret, err := cmdutils.GetUserSetVarFromString(cmd, clientSecretFlagName, clientSecretEnvKey, false)
	if err != nil {
		return nil, err
	}

	config := &oauth2.Config{
		RedirectURL:  strings.TrimSpace(redirectURL),
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(secret),
		Endpoint:     hydra,
	}

	return config, nil
}
