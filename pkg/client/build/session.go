package build

import (
	"strings"

	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/session/sshforward/sshprovider"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/kvfile"
	"github.com/rancher/wrangler/pkg/kv"
)

func (b *Opts) Session() (result []session.Attachable, err error) {
	if len(b.Secrets) > 0 {
		attach, err := parseSecrets(b.Secrets)
		if err != nil {
			return nil, err
		}
		result = append(result, attach)
	}

	if len(b.SSH) > 0 {
		attach, err := parseSSHs(b.SSH)
		if err != nil {
			return nil, err
		}
		result = append(result, attach)
	}

	return
}

func parseSSHs(sshs []string) (session.Attachable, error) {
	var agentConfigs []sshprovider.AgentConfig

	for _, ssh := range sshs {
		agentConfigs = append(agentConfigs, parseSSH(ssh))
	}

	return sshprovider.NewSSHAgentProvider(agentConfigs)
}

func parseSSH(value string) sshprovider.AgentConfig {
	id, paths := kv.Split(value, "=")
	return sshprovider.AgentConfig{
		ID:    id,
		Paths: splitCheckEmpty(paths),
	}
}

func splitCheckEmpty(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	return strings.Split(val, ",")
}

func parseSecrets(secrets []string) (session.Attachable, error) {
	var sources []secretsprovider.FileSource

	for _, v := range secrets {
		s, err := parseSecret(v)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}

	store, err := secretsprovider.NewFileStore(sources)
	if err != nil {
		return nil, err
	}

	return secretsprovider.NewSecretProvider(store), nil
}

func firstNonEmpty(m map[string]string, key ...string) string {
	for _, key := range key {
		v := m[key]
		if v != "" {
			return v
		}
	}
	return ""
}

func parseSecret(value string) (secretsprovider.FileSource, error) {
	typeName, attr, err := kvfile.ParseTypeAndKVMap(value)
	if err != nil {
		return secretsprovider.FileSource{}, err
	}

	if typeName != "" && typeName != "file" {
		return secretsprovider.FileSource{}, errors.Errorf("unsupported secret type %q", typeName)
	}

	return secretsprovider.FileSource{
		ID:       attr["id"],
		FilePath: firstNonEmpty(attr, "source", "src"),
	}, nil
}
