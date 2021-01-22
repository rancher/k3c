package auth

import (
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Parse parses AuthConfig and returns username and password/secret required by containerd.
func Parse(auth *criv1.AuthConfig, host string) (string, string, error) {
	if auth == nil {
		return "", "", nil
	}
	if auth.ServerAddress != "" {
		// Do not return the auth info when server address doesn't match.
		u, err := url.Parse(auth.ServerAddress)
		if err != nil {
			return "", "", errors.Wrap(err, "parse server address")
		}
		if host != u.Host {
			return "", "", nil
		}
	}
	if auth.Username != "" {
		return auth.Username, auth.Password, nil
	}
	if auth.IdentityToken != "" {
		return "", auth.IdentityToken, nil
	}
	if auth.Auth != "" {
		decLen := base64.StdEncoding.DecodedLen(len(auth.Auth))
		decoded := make([]byte, decLen)
		_, err := base64.StdEncoding.Decode(decoded, []byte(auth.Auth))
		if err != nil {
			return "", "", err
		}
		fields := strings.SplitN(string(decoded), ":", 2)
		if len(fields) != 2 {
			return "", "", errors.Errorf("invalid decoded auth: %q", decoded)
		}
		user, passwd := fields[0], fields[1]
		return user, strings.Trim(passwd, "\x00"), nil
	}
	// TODO(random-liu): Support RegistryToken.
	// An empty auth config is valid for anonymous registry
	return "", "", nil
}
