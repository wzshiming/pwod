package pwod

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/howardjohn/pilot-load/pkg/kube"
	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"github.com/howardjohn/pilot-load/pkg/simulation/security"
	"github.com/spf13/cobra"
	"istio.io/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/wzshiming/pwod/pkg/pwod/controllers"
)

var (
	pilotAddress = defaultAddress()
	xdsMetadata  = map[string]string{}
	auth         = string(security.AuthTypeDefault)
	delta        = false
	kubeconfig   = os.Getenv("KUBECONFIG")

	authTrustDomain   = ""
	authClusterUrl    = ""
	authProjectNumber = ""

	qps = 10000
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&pilotAddress, "pilot-address", "p", pilotAddress, "address to pilot")
	RootCmd.PersistentFlags().StringVarP(&auth, "auth", "a", auth, fmt.Sprintf("auth type use. If not set, default based on port number. Supported options: %v", security.AuthTypeOptions()))
	RootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "kubeconfig")
	RootCmd.PersistentFlags().IntVar(&qps, "qps", qps, "qps for kube client")
	RootCmd.PersistentFlags().StringToStringVarP(&xdsMetadata, "metadata", "m", xdsMetadata, "xds metadata")

	RootCmd.PersistentFlags().BoolVar(&delta, "delta", delta, "use delta XDS")

	RootCmd.PersistentFlags().StringVar(&authClusterUrl, "clusterURL", authClusterUrl, "cluster URL (for google auth)")
	RootCmd.PersistentFlags().StringVar(&authTrustDomain, "trustDomain", authTrustDomain, "trust domain (for google auth)")
	RootCmd.PersistentFlags().StringVar(&authProjectNumber, "projectNumber", authProjectNumber, "project number (for google auth)")
}

func defaultAddress() string {
	_, inCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if inCluster {
		return "istiod.istio-system.svc:15010"
	}
	return "localhost:15010"
}

func defaultLogOptions() *log.Options {
	o := log.DefaultOptions()

	// These scopes are, at the default "INFO" level, too chatty for command line use
	o.SetOutputLevel("dump", log.WarnLevel)
	o.SetOutputLevel("token", log.ErrorLevel)

	return o
}

func GetArgs() (model.Args, error) {
	var err error
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), "/.kube/config")
	}
	cl, err := kube.NewClient(kubeconfig, qps)
	if err != nil {
		return model.Args{}, err
	}
	auth := security.AuthType(auth)
	if auth == "" {
		auth = security.DefaultAuthForAddress(pilotAddress)
	}
	authOpts := &security.AuthOptions{
		Type:          auth,
		Client:        cl,
		TrustDomain:   authTrustDomain,
		ProjectNumber: authProjectNumber,
		ClusterURL:    authClusterUrl,
	}
	args := model.Args{
		PilotAddress: pilotAddress,
		DeltaXDS:     delta,
		Metadata:     xdsMetadata,
		Client:       cl,
		Auth:         authOpts,
	}
	args, err = setDefaultArgs(args)
	if err != nil {
		return model.Args{}, err
	}
	return args, nil
}

const CLOUDRUN_ADDR = "CLOUDRUN_ADDR"

func setDefaultArgs(args model.Args) (model.Args, error) {
	if err := args.Auth.AutoPopulate(); err != nil {
		return model.Args{}, err
	}
	if _, f := xdsMetadata[CLOUDRUN_ADDR]; !f && args.Auth.Type == security.AuthTypeGoogle {
		mwh, err := args.Client.Kubernetes.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), "istiod-asm-managed", metav1.GetOptions{})
		if err != nil {
			return model.Args{}, fmt.Errorf("failed to default CLOUDRUN_ADDR: %v", err)
		}
		if len(mwh.Webhooks) == 0 {
			return args, nil
		}
		wh := mwh.Webhooks[0]
		if wh.ClientConfig.URL == nil {
			return model.Args{}, fmt.Errorf("failed to default CLOUDRUN_ADDR: clientConfig is not a URL")
		}
		addr, _ := url.Parse(*wh.ClientConfig.URL)
		log.Infof("defaulted CLOUDRUNN_ADDR to %v", addr.Host)
		xdsMetadata[CLOUDRUN_ADDR] = addr.Host
	}
	return args, nil
}

var adscConfig = model.AdscConfig{
	Delay:     time.Millisecond * 10,
	Count:     1,
	Namespace: "default",
}

var RootCmd = &cobra.Command{
	Short: "open simple ADS connection to Istiod",
	RunE: func(cmd *cobra.Command, _ []string) error {
		err := log.Configure(defaultLogOptions())
		if err != nil {
			return err
		}

		args, err := GetArgs()
		if err != nil {
			return err
		}
		args.AdsConfig = adscConfig
		ctr := controllers.NewController(args)
		return ctr.Run(cmd.Context())
	},
}
