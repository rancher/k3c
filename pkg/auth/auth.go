package auth

import (
	"github.com/rancher/k3c/pkg/client"
	"k8s.io/kubernetes/pkg/credentialprovider"
)

func Lookup(image string) *client.AuthConfig {
	kr := credentialprovider.NewDockerKeyring()
	auth, _ := kr.Lookup(image)
	if len(auth) == 0 {
		return nil
	}

	return &client.AuthConfig{
		Username:      auth[0].Username,
		Password:      auth[0].Password,
		Auth:          auth[0].Auth,
		Email:         auth[0].Email,
		ServerAddress: auth[0].ServerAddress,
		IdentityToken: auth[0].IdentityToken,
		RegistryToken: auth[0].RegistryToken,
	}
}
