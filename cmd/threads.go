package cmd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/textileio/textile-go/core"
	"github.com/textileio/textile-go/util"
	"github.com/textileio/textile-go/wallet/thread"
	"gopkg.in/abiosoft/ishell.v2"
	libp2pc "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

func ListThreads(c *ishell.Context) {
	threads := core.Node.Wallet.Threads()
	if len(threads) == 0 {
		c.Println("no threads found")
	} else {
		c.Println(fmt.Sprintf("found %v threads", len(threads)))
	}

	blue := color.New(color.FgHiBlue).SprintFunc()
	for _, thrd := range threads {
		c.Println(blue(fmt.Sprintf("name: %s, id: %s", thrd.Name, thrd.Id)))
	}
}

func AddThread(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[0]

	sk, _, err := libp2pc.GenerateEd25519Key(rand.Reader)
	if err != nil {
		c.Err(err)
		return
	}

	if _, err := core.Node.Wallet.AddThread(name, sk); err != nil {
		c.Err(err)
		return
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	c.Println(cyan(fmt.Sprintf("added thread '%s'", name)))
}

func PublishThread(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[0]

	_, thrd := core.Node.Wallet.GetThreadByName(name)
	if thrd == nil {
		c.Err(errors.New(fmt.Sprintf("could not find thread: %s", name)))
		return
	}

	blue := color.New(color.FgHiBlue).SprintFunc()
	head, err := thrd.GetHead()
	if err != nil {
		c.Err(err)
		return
	}
	if head == "" {
		c.Println(blue("nothing to publish"))
		return
	}
	peers := thrd.Peers()
	if len(peers) == 0 {
		c.Println(blue("no peers to publish to"))
		return
	}

	err = thrd.PostHead(peers)
	if err != nil {
		c.Err(err)
		return
	}

	c.Println(blue(fmt.Sprintf("published %s in thread %s to %d peers", head, thrd.Name, len(peers))))
}

func ListThreadPeers(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[0]

	_, thrd := core.Node.Wallet.GetThreadByName(name)
	if thrd == nil {
		c.Err(errors.New(fmt.Sprintf("could not find thread: %s", name)))
		return
	}

	peers := thrd.Peers()
	if len(peers) == 0 {
		c.Println(fmt.Sprintf("no peers found in: %s", name))
	} else {
		c.Println(fmt.Sprintf("found %v peers in: %s", len(peers), name))
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	for _, peer := range peers {
		c.Println(green(peer.Id))
	}
}

func AddThreadInvite(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing peer pub key"))
		return
	}
	pks := c.Args[0]
	if len(c.Args) == 1 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[1]

	_, thrd := core.Node.Wallet.GetThreadByName(name)
	if thrd == nil {
		c.Err(errors.New(fmt.Sprintf("could not find thread: %s", name)))
		return
	}

	pkb, err := libp2pc.ConfigDecodeKey(pks)
	if err != nil {
		c.Err(err)
		return
	}
	pk, err := libp2pc.UnmarshalPublicKey(pkb)
	if err != nil {
		c.Err(err)
		return
	}

	if _, err := thrd.AddInvite(pk); err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green("invite sent!"))
}

func AcceptThreadInvite(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing invite address"))
		return
	}
	blockId := c.Args[0]
	if len(c.Args) == 1 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[1]

	_, err := core.Node.Wallet.AcceptThreadInvite(blockId, name)
	if err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green("ok, accepted"))
}

func AddExternalThreadInvite(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[0]

	_, thrd := core.Node.Wallet.GetThreadByName(name)
	if thrd == nil {
		c.Err(errors.New(fmt.Sprintf("could not find thread: %s", name)))
		return
	}

	added, err := thrd.AddExternalInvite()
	if err != nil {
		c.Err(err)
		return
	}
	link := util.BuildExternalInviteLink(added.Id, string(added.Key), thrd.Name)

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green(link))
}

func AcceptExternalThreadInvite(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing invite link"))
		return
	}
	blockId, key, name, err := util.ParseExternalInviteLink(c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	_, err = core.Node.Wallet.AcceptExternalThreadInvite(blockId, []byte(key), name)
	if err != nil {
		c.Err(err)
		return
	}

	green := color.New(color.FgHiGreen).SprintFunc()
	c.Println(green("ok, accepted"))
}

func RemoveThread(c *ishell.Context) {
	if len(c.Args) == 0 {
		c.Err(errors.New("missing thread name"))
		return
	}
	name := c.Args[0]

	_, err := core.Node.Wallet.RemoveThread(name)
	if err != nil {
		c.Err(err)
		return
	}

	red := color.New(color.FgHiRed).SprintFunc()
	c.Println(red(fmt.Sprintf("removed thread '%s'", name)))
}

func Subscribe(thrd *thread.Thread) {
	cyan := color.New(color.FgCyan).SprintFunc()
	for {
		select {
		case update, ok := <-thrd.Updates():
			if !ok {
				return
			}
			fmt.Printf(cyan(fmt.Sprintf("\nnew block %s in thread %s\n", update.Id, update.ThreadName)))
		}
	}
}
