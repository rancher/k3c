package validate

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func NArgs(cli *cli.Context, name string, min, max int) error {
	if min == max && cli.NArg() != min {
		return fmt.Errorf("\"%s\" requires exactly %d argument%s, see --help", name, min, plural(min))
	}

	if cli.NArg() < min {
		return fmt.Errorf("\"%s\" requires at least %d argument%s, see --help", name, min, plural(min))
	}

	if max > -1 && cli.NArg() > max {
		return fmt.Errorf("\"%s\" requires no more than %d argument%s, see --help", name, max, plural(max))
	}

	return nil
}

func plural(count int) string {
	if count > 1 {
		return "s"
	}
	return ""
}
