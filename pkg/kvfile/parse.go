package kvfile

import (
	"encoding/csv"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/rancher/wrangler/pkg/kv"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func ReadKVEnvStrings(files []string, override []string) ([]string, error) {
	return readKVStrings(files, override, os.Getenv)
}

func ReadEnv(file []string, override []string) (result []*pb.KeyValue, err error) {
	vals, err := ReadKVEnvStrings(file, override)
	if err != nil {
		return nil, err
	}

	for _, val := range vals {
		k, v := kv.Split(val, "=")
		result = append(result, &pb.KeyValue{
			Key:   k,
			Value: v,
		})
	}

	return
}

func readKVStrings(files []string, override []string, emptyFn func(string) string) ([]string, error) {
	var variables []string
	for _, ef := range files {
		parsedVars, err := parseKeyValueFile(ef, emptyFn)
		if err != nil {
			return nil, err
		}
		variables = append(variables, parsedVars...)
	}
	// parse the '-e' and '--env' after, to allow override
	variables = append(variables, override...)

	return variables, nil
}

func ReadKVStrings(files []string, override []string) ([]string, error) {
	return readKVStrings(files, override, nil)
}

func ParseTypeAndKVMap(csvInput string) (string, map[string]string, error) {
	fields, err := csv.NewReader(strings.NewReader(csvInput)).Read()
	if err != nil {
		return "", nil, errors.Wrapf(err, "parsing %s", csvInput)
	}

	m, _ := ReadKVMap(nil, fields)
	t := m["type"]
	delete(m, "type")
	return t, m, nil
}

func ReadKVMap(files []string, override []string) (map[string]string, error) {
	vals, err := ReadKVStrings(files, override)
	if err != nil {
		return nil, err
	}

	return kv.SplitMapFromSlice(vals), nil
}
