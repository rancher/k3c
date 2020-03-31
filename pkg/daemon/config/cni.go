package config

func DefaultCniConf(config *K3Config) map[string]interface{} {
	type obj map[string]interface{}

	cidr := config.BridgeCIDR
	if cidr == "" {
		cidr = DefaultBridgeCIDR
	}
	bridge := config.BridgeName
	if bridge == "" {
		bridge = DefaultBridgeName
	}
	return map[string]interface{}{
		"cniVersion":  "0.3.1",
		"type":        "bridge",
		"name":        "k3c-net",
		"bridge":      bridge,
		"isGateway":   true,
		"ipMasq":      true,
		"promiscMode": true,
		"ipam": obj{
			"type":   "host-local",
			"subnet": cidr,
			"routes": []obj{
				{
					"dst": "0.0.0.0/0",
				},
			},
		},
	}
}

func DefaultCniConflist(config *K3Config) map[string]interface{} {
	type obj map[string]interface{}

	return obj{
		"cniVersion": "0.3.1",
		"name":       "k3c-net",
		"plugins": []obj{
			DefaultCniConf(config),
			{
				"type": "portmap",
				"capabilities": obj{
					"portMappings": true,
				},
			},
		},
	}
}
