package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/mitchellh/go-homedir"
	"github.com/op/go-logging"
	"github.com/skip2/go-qrcode"
	"github.com/textileio/textile-go/core"
	"github.com/textileio/textile-go/repo"
	rconfig "github.com/textileio/textile-go/repo/config"
	"github.com/textileio/textile-go/wallet"
	"github.com/textileio/textile-go/wallet/thread"
	"os"
	"path/filepath"
	"time"
)

var (
	appName  = "Textile"
	builtAt  string
	debug    = flag.Bool("d", false, "enables the debug mode")
	window   *astilectron.Window
	gateway  string
	expanded bool
)

const (
	SetupSize       = 384
	QRCodeSize      = 256
	InitialWidth    = 1024
	InitialHeight   = 633
	SleepOnLoad     = time.Second * 1
	SleepOnPreReady = time.Millisecond * 200
	SleepOnExpand   = time.Millisecond * 200
)

func main() {
	flag.Parse()
	astilog.FlagInit()
	bootstrapApp()
}

func start(_ *astilectron.Astilectron, w *astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
	window = w
	window.Show()

	// get homedir
	home, err := homedir.Dir()
	if err != nil {
		astilog.Fatal(fmt.Errorf("get homedir failed: %s", err))
	}

	// ensure app support folder is created
	appDir := filepath.Join(home, "Library/Application Support/Textile")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	// create a desktop textile node
	config := core.NodeConfig{
		LogLevel: logging.DEBUG,
		LogFiles: true,
		WalletConfig: wallet.Config{
			RepoPath: filepath.Join(appDir, "repo"),
		},
	}
	core.Node, _, err = core.NewNode(config)
	if err != nil {
		return err
	}

	// bring the node online and startup the gateway
	if err := core.Node.StartWallet(); err != nil {
		return err
	}
	<-core.Node.Wallet.Online()

	// subscribe to wallet updates
	go func() {
		for {
			select {
			case update, ok := <-core.Node.Wallet.Updates():
				if !ok {
					return
				}
				payload := map[string]interface{}{
					"update": update,
				}
				switch update.Type {
				case wallet.ThreadAdded:
					if _, thrd := core.Node.Wallet.GetThread(update.Id); thrd != nil {
						go subscribe(thrd)
					}
					if expanded {
						sendData("wallet.update", payload)
					} else {
						sendPreReady()
						window.Hide()
						expandWindow()
						sendData("wallet.update", payload)
						window.Show()
						window.Focus()
					}
				case wallet.ThreadRemoved:
					break
				case wallet.DeviceAdded:
					break
				case wallet.DeviceRemoved:
					break
				}
			}
		}
	}()

	// subscribe to thread updates
	for _, thrd := range core.Node.Wallet.Threads() {
		go func(t *thread.Thread) {
			subscribe(t)
		}(thrd)
	}

	// start the gateway
	core.Node.StartGateway(fmt.Sprintf("127.0.0.1:%d", rconfig.GetRandomPort()))

	// save off the server address
	gateway = fmt.Sprintf("http://%s", core.Node.GetGatewayAddr())

	// sleep for a bit on the landing screen, it feels better
	time.Sleep(SleepOnLoad)

	// send cookie info to front-end
	sendData("login", map[string]interface{}{
		"name":    "SessionId",
		"value":   "not used",
		"gateway": gateway,
	})

	// check if we're configured yet
	threads := core.Node.Wallet.Threads()
	if len(threads) > 0 {
		// load threads for UI
		var threadsJSON []map[string]interface{}
		for _, thrd := range threads {
			threadsJSON = append(threadsJSON, map[string]interface{}{
				"id":   thrd.Id,
				"name": thrd.Name,
			})
		}

		// reveal
		sendPreReady()
		window.Hide()
		expandWindow()
		sendData("ready", map[string]interface{}{
			"threads": threadsJSON,
		})
		window.Show()
		window.Focus()

	} else {
		// get qr code for setup
		qr, pk, err := getQRCode()
		if err != nil {
			astilog.Error(err)
			return err
		}
		sendData("setup", map[string]interface{}{
			"qr": qr,
			"pk": pk,
		})
	}

	return nil
}

func sendMessage(name string) {
	window.SendMessage(map[string]string{"name": name})
}

func sendData(name string, data map[string]interface{}) {
	data["name"] = name
	window.SendMessage(data)
}

func handleMessage(_ *astilectron.Window, m bootstrap.MessageIn) (interface{}, error) {
	switch m.Name {
	case "refresh":
		if err := core.Node.Wallet.RefreshMessages(); err != nil {
			return nil, err
		}
		return map[string]interface{}{}, nil
	case "thread.load":
		var threadId string
		if err := json.Unmarshal(m.Payload, &threadId); err != nil {
			return nil, err
		}
		html, err := getThreadPhotos(threadId)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"html": html,
		}, nil
	default:
		return map[string]interface{}{}, nil
	}
}

func subscribe(thrd *thread.Thread) {
	for {
		select {
		case update, ok := <-thrd.Updates():
			if !ok {
				return
			}
			sendData("thread.update", map[string]interface{}{
				"update": update,
			})
		}
	}
}

func getQRCode() (string, string, error) {
	// get our own public key
	pk, err := core.Node.Wallet.GetPubKeyString()
	if err != nil {
		return "", "", err
	}

	// create a qr code
	url := fmt.Sprintf("https://www.textile.photos/invites/device#key=%s", pk)
	png, err := qrcode.Encode(url, qrcode.Medium, QRCodeSize)
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(png), pk, nil
}

func getThreadPhotos(id string) (string, error) {
	_, thrd := core.Node.Wallet.GetThread(id)
	if thrd == nil {
		return "", errors.New("thread not found")
	}
	var html string
	for _, block := range thrd.Blocks("", -1, repo.PhotoBlock) {
		photo := fmt.Sprintf("%s/ipfs/%s/photo?block=%s", gateway, block.DataId, block.Id)
		thumb := fmt.Sprintf("%s/ipfs/%s/thumb?block=%s", gateway, block.DataId, block.Id)
		meta := fmt.Sprintf("%s/ipfs/%s/meta?block=%s", gateway, block.DataId, block.Id)
		img := fmt.Sprintf("<img src=\"%s\" />", thumb)
		html += fmt.Sprintf(
			"<div id=\"%s\" class=\"grid-item\" ondragstart=\"imageDragStart(event);\" draggable=\"true\" data-url=\"%s\" data-meta=\"%s\">%s</div>",
			block.Id, photo, meta, img)
	}
	return html, nil
}

func sendPreReady() {
	sendMessage("preready")
	time.Sleep(SleepOnPreReady)
}

func expandWindow() {
	expanded = true
	go window.Resize(InitialWidth, InitialHeight)
	go window.Center()
	time.Sleep(SleepOnExpand)
}
