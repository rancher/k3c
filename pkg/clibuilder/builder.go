package clibuilder

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/urfave/cli/v2"
)

var (
	caseRegexp = regexp.MustCompile("([a-z])([A-Z])")
)

type Runnable interface {
	Run(app *cli.Context) error
}

type customizer interface {
	Customize(cmd *cli.Command)
}

type fieldInfo struct {
	FieldType  reflect.StructField
	FieldValue reflect.Value
}

func fields(obj interface{}) []fieldInfo {
	ptrValue := reflect.ValueOf(obj)
	objValue := ptrValue.Elem()

	var result []fieldInfo

	for i := 0; i < objValue.NumField(); i++ {
		fieldType := objValue.Type().Field(i)
		if fieldType.Anonymous && fieldType.Type.Kind() == reflect.Struct {
			result = append(result, fields(objValue.Field(i).Addr().Interface())...)
		} else if !fieldType.Anonymous {
			result = append(result, fieldInfo{
				FieldValue: objValue.Field(i),
				FieldType:  objValue.Type().Field(i),
			})
		}
	}

	return result
}

func CommandName(obj Runnable) string {
	ptrValue := reflect.ValueOf(obj)
	objValue := ptrValue.Elem()
	return strings.ToLower(strings.Replace(objValue.Type().Name(), "Command", "", 1))
}

func Command(obj Runnable, cmd cli.Command) *cli.Command {
	slices := map[string]reflect.Value{}
	maps := map[string]reflect.Value{}
	ptrValue := reflect.ValueOf(obj)
	objValue := ptrValue.Elem()

	c := cmd
	if c.Name == "" {
		c.Name = CommandName(obj)
	}
	c.UseShortOptionHandling = true

	for _, info := range fields(obj) {
		defMessage := ""
		fieldType := info.FieldType
		v := info.FieldValue

		name, alias := name(fieldType.Name, fieldType.Tag.Get("name"))
		usage := fieldType.Tag.Get("usage")
		env := strings.Split(fieldType.Tag.Get("env"), ",")
		if len(env) == 1 && env[0] == "" {
			env = nil
		}

		switch fieldType.Type.Kind() {
		case reflect.Int:
			flag := &cli.IntFlag{
				Name:        name,
				Aliases:     alias,
				Usage:       usage,
				EnvVars:     env,
				Destination: (*int)(unsafe.Pointer(v.Addr().Pointer())),
			}
			defValue := fieldType.Tag.Get("default")
			if defValue != "" {
				n, err := strconv.Atoi(defValue)
				if err != nil {
					panic("bad default " + defValue + " on field " + fieldType.Name)
				}
				flag.Value = n
			}
			c.Flags = append(c.Flags, flag)
		case reflect.String:
			flag := &cli.StringFlag{
				Name:        name,
				Aliases:     alias,
				Usage:       usage,
				Value:       fieldType.Tag.Get("default"),
				EnvVars:     env,
				Destination: (*string)(unsafe.Pointer(v.Addr().Pointer())),
			}
			c.Flags = append(c.Flags, flag)
		case reflect.Slice:
			slices[name] = v
			defMessage = " "
			fallthrough
		case reflect.Map:
			if defMessage == "" {
				maps[name] = v
				defMessage = " "
			}
			flag := &cli.StringSliceFlag{
				Name:    name,
				Aliases: alias,
				Usage:   usage + defMessage,
				EnvVars: env,
				//Value:   &cli.StringSlice{},
			}
			c.Flags = append(c.Flags, flag)
		case reflect.Bool:
			flag := &cli.BoolFlag{
				Name:        name,
				Aliases:     alias,
				Usage:       usage,
				EnvVars:     env,
				Destination: (*bool)(unsafe.Pointer(v.Addr().Pointer())),
			}
			if fieldType.Tag.Get("default") == "true" {
				flag.Value = true
			}
			c.Flags = append(c.Flags, flag)

		default:
			panic("Unknown kind on field " + fieldType.Name + " on " + objValue.Type().Name())
		}
	}

	c.Action = func(ctx *cli.Context) error {
		assignSlices(ctx, slices)
		assignMaps(ctx, maps)
		return obj.Run(ctx)
	}

	cust, ok := obj.(customizer)
	if ok {
		cust.Customize(&c)
	}

	return &c
}

func assignMaps(app *cli.Context, maps map[string]reflect.Value) {
	for k, v := range maps {
		k = contextKey(k)
		s := app.StringSlice(k)
		if s != nil {
			values := map[string]string{}
			for _, part := range s {
				parts := strings.SplitN(part, "=", 2)
				if len(parts) == 1 {
					values[parts[0]] = ""
				} else {
					values[parts[0]] = parts[1]
				}
			}
			v.Set(reflect.ValueOf(values))
		}
	}
}

func assignSlices(app *cli.Context, slices map[string]reflect.Value) {
	for k, v := range slices {
		k = contextKey(k)
		s := app.StringSlice(k)
		if s != nil {
			// TODO: weird BUG?
			if len(s) > 1 {
				s = s[:len(s)/2]
			}
			v.Set(reflect.ValueOf(s))
		}
	}
}

func contextKey(name string) string {
	parts := strings.Split(name, ",")
	return parts[len(parts)-1]
}

func name(name, setName string) (string, []string) {
	if setName != "" {
		return setName, nil
	}
	parts := strings.Split(name, "_")
	i := len(parts) - 1
	name = caseRegexp.ReplaceAllString(parts[i], "$1-$2")
	name = strings.ToLower(name)
	result := append([]string{name}, parts[0:i]...)
	for i := 0; i < len(result); i++ {
		result[i] = strings.ToLower(result[i])
	}
	return result[0], result[1:]
}
