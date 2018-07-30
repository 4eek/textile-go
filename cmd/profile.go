package cmd

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/textileio/textile-go/core"
	"gopkg.in/abiosoft/ishell.v2"
)

func PublishProfile(c *ishell.Context) {
	entry, err := core.Node.Wallet.PublishProfile(nil)
	if err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green(fmt.Sprintf("ok, published %s -> %s", entry.Name, entry.Value)))
}

func ResolveProfile(c *ishell.Context) {
	var name string
	if len(c.Args) == 0 {
		id, err := core.Node.Wallet.GetId()
		if err != nil {
			c.Err(err)
			return
		}
		name = id
	} else {
		name = c.Args[0]
	}

	entry, err := core.Node.Wallet.ResolveProfile(name)
	if err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green(entry.String()))
}

func GetProfile(c *ishell.Context) {
	var id string
	if len(c.Args) == 0 {
		var err error
		id, err = core.Node.Wallet.GetId()
		if err != nil {
			c.Err(err)
			return
		}
	} else {
		id = c.Args[0]
	}

	prof, err := core.Node.Wallet.GetProfile(id)
	if err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	if prof.Id != "" {
		c.Println(green(fmt.Sprintf("id:        %s", prof.Id)))
	}
	if prof.Username != "" {
		c.Println(green(fmt.Sprintf("username:  %s", prof.Username)))
	}
	if prof.AvatarId != "" {
		c.Println(green(fmt.Sprintf("avatar_id: %s", prof.AvatarId)))
	}
}

func SetAvatarId(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing photo id"))
		return
	}
	id := c.Args[0]

	if err := core.Node.Wallet.SetAvatarId(id); err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green("ok, updated"))
}
