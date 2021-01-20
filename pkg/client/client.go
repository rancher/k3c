package client

import (
	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/apply"
	appsctl "github.com/rancher/wrangler/pkg/generated/controllers/apps"
	appsctlv1 "github.com/rancher/wrangler/pkg/generated/controllers/apps/v1"
	corectl "github.com/rancher/wrangler/pkg/generated/controllers/core"
	corectlv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/kubeconfig"
)

const (
	DefaultNamespace = "k3c"
)

var DefaultConfig = Config{
	Namespace: DefaultNamespace,
}

type Config struct {
	Namespace  string `usage:"namespace" default:"k3c" short:"n" env:"NAMESPACE"`
	Kubeconfig string `usage:"kubeconfig for authentication" short:"k" env:"KUBECONFIG"`
	Context    string `usage:"kubeconfig context for authentication" short:"x" env:"KUBECONTEXT"`
}

func (c *Config) Interface() (*Interface, error) {
	if c == nil {
		return nil, errors.Errorf("client is not configured, please set client config")
	}
	return NewInterface(c.Kubeconfig, c.Context, c.Namespace)
}

type Interface struct {
	Core      corectlv1.Interface
	Apps      appsctlv1.Interface
	Apply     apply.Apply
	Namespace string
}

func NewInterface(kubecfg, kubectx, kubens string) (*Interface, error) {
	cc := kubeconfig.GetNonInteractiveClientConfigWithContext(kubecfg, kubectx)
	ns, _, err := cc.Namespace()
	if err != nil {
		return nil, err
	}

	if kubens != "" {
		ns = kubens
	}

	rc, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}

	c := &Interface{
		Namespace: ns,
	}

	core, err := corectl.NewFactoryFromConfig(rc)
	if err != nil {
		return nil, err
	}
	c.Core = core.Core().V1()

	apps, err := appsctl.NewFactoryFromConfig(rc)
	if err != nil {
		return nil, err
	}
	c.Apps = apps.Apps().V1()

	c.Apply, err = apply.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	if c.Namespace == "" {
		c.Namespace = "k3c"
	}

	c.Apply = c.Apply.
		WithDynamicLookup().
		WithDefaultNamespace(c.Namespace).
		WithListerNamespace(c.Namespace).
		WithRestrictClusterScoped()

	return c, nil
}
