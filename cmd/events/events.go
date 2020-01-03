package events

import (
	"fmt"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/urfave/cli/v2"
)

type Events struct {
	Format string `usage:"Format the output using the given Go template"`
}

func (e *Events) Run(ctx *cli.Context) error {
	c, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	events, err := c.Events(ctx.Context)
	if err != nil {
		return err
	}

	for {
		select {
		case event, done := <-events:
			if done {
				return nil
			}
			fmt.Println(event)
			//t := tables.NewEvents(ctx)
			//t.Write(tables.EventData{
			//	ID:   event.ID,
			//	Name: event.Name,
			//})
			//if err := t.Close(); err != nil {
			//	return err
			//}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
