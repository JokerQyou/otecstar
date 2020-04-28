package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/getlantern/systray"
	"golang.org/x/net/html"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"otecstar/icons"
	"strings"
	"time"
)

// State represents a captured state (snapshot) from the router
type State struct {
	wlanState string
	linkState string
	linkLoss  string
	upWidth   string
	upSNR     string
	downWidth string
	downSNR   string
}

// OTECStarApp embeds all necessary data to start up our application
type OTECStarApp struct {
	wlanState *systray.MenuItem
	linkState *systray.MenuItem
	linkLoss  *systray.MenuItem
	upWidth   *systray.MenuItem
	upSNR     *systray.MenuItem
	downWidth *systray.MenuItem
	downSNR   *systray.MenuItem
	updateCh  chan *State
	stopCh    chan int
	auth      AuthContainer
}

// AuthContainer embeds all data specific to router authentication
type AuthContainer struct {
	routerIP      string
	loginUrl      string
	stateUrl      string
	sysauthCookie *http.Cookie
	username      string
	password      string
}

// Clicked connects a given MenuItem's clicked event to given function
func (*OTECStarApp) Clicked(which *systray.MenuItem, callback func()) {
	go func() {
		<-which.ClickedCh
		callback()
	}()
}

// login authenticates against the router, and set necessary data to `app.auth` for future consumption
func (o *OTECStarApp) login() error {
	/**
	Login: POST http://{ROUTER_IP}/cgi-bin/luci/customer/ with {username, password, login_in=登录}
	Login returns cookies: sysauth={SYS_AUTH}; path=/cgi-bin/luci/;stok={STOCK}
	Get state from this page: http://{ROUTER_IP}/cgi-bin/luci/;stok={STOCK}/customer/status/wan/
	If form#sysauth[name="sysauth"] is presented in responding HTML, it means we need to login again.
	*/
	data := url.Values{}
	data.Set("username", o.auth.username)
	data.Set("password", o.auth.password)
	data.Set("login_in", "登录")

	postData := bytes.Buffer{}
	postData.WriteString(data.Encode())

	logger.Debug().Msg("login")
	resp, err := http.Post(
		fmt.Sprintf(o.auth.loginUrl, o.auth.routerIP),
		"application/x-www-form-urlencoded",
		&postData,
	)
	if err != nil {
		return err
	}

	// Golang http does not honor `;` in cookie values, so we must do this manually
	for _, header := range resp.Header.Values("set-cookie") {
		for _, cookieStr := range strings.Split(header, "; ") {
			logger.Debug().Str("cookieStr", cookieStr).Msg("Processing...")
			_c := strings.SplitN(cookieStr, "=", 2)
			if len(_c) != 2 {
				logger.Debug().Str("cookie", cookieStr).Msg("Skipped bad cookie string")
				continue
			}
			name, value := _c[0], _c[1]
			if name == `path` {
				o.auth.stateUrl = fmt.Sprintf(`http://%%s/%s/customer/status/wan/`, value)
				logger.Debug().Str("stateUrl", o.auth.stateUrl).Msg("stateUrl updated")
				continue
			}
			if name == `sysauth` {
				o.auth.sysauthCookie = &http.Cookie{Name: name, Value: value}
				logger.Debug().Msg("Got sysauth cookie")
			}
		}
	}

	if o.auth.sysauthCookie == nil {
		return fmt.Errorf(`failed to login`)
	}
	logger.Debug().Interface("sysauthCookie", o.auth.sysauthCookie).Msg("login OK")
	return nil
}

// getState captures a state from the router
func (o *OTECStarApp) getState() *State {
	/**
	Login: POST http://{ROUTER_IP}/cgi-bin/luci/customer/ with {username, password, login_in=登录}
	Login returns cookies: sysauth={SYS_AUTH}; path=/cgi-bin/luci/;stok={STOCK}
	Get state from this page: http://{ROUTER_IP}/cgi-bin/luci/;stok={STOCK}/customer/status/wan/
	If form#sysauth[name="sysauth"] is presented in responding HTML, it means we need to login again.
	*/
	state := State{
		wlanState: "-",
		linkState: "-",
		linkLoss:  "-",
		upWidth:   "-",
		upSNR:     "-",
		downWidth: "-",
		downSNR:   "-",
	}

	logger.Debug().Msg("getState")
	// Login for the first time
	if o.auth.sysauthCookie == nil {
		if err := o.login(); err != nil {
			logger.Error().Err(err).Msg("Failed to login")
			state.wlanState = "ERROR: " + err.Error()
			return &state
		}
	}

	// Get state
	jar, _ := cookiejar.New(nil)
	client := http.Client{
		Jar:     jar,
		Timeout: time.Second * 5,
	}
	stateUrl := fmt.Sprintf(o.auth.stateUrl, o.auth.routerIP)
	u, _ := url.Parse(stateUrl)
	jar.SetCookies(u, []*http.Cookie{o.auth.sysauthCookie})

	resp, err := client.Get(stateUrl)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to access WAN state page")
		state.wlanState = "ERROR: " + err.Error()
		return &state
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to parse WAN page response")
		state.wlanState = "ERROR: " + err.Error()
		return &state
	}

	// Check state response, if our authentication expired, do another login
	if doc.Find(`form#sysauth`).Length() > 0 {
		logger.Info().Msg("Login expired, retrying")
		o.auth.sysauthCookie = nil
		return o.getState()
	}

	dataTables := doc.Find(`table.cbi-table-list`)
	if dataTables.Length() != 4 {
		logger.Error().Msg("Unexpected data tables")
		state.wlanState = "ERROR: 未预期的数据表格式"
		return &state
	}
	dataTables.Each(func(index int, table *goquery.Selection) {
		if index == 2 {
			state.wlanState = table.Find(`td.cbi-table-field`).Last().Text()
		} else if index == 3 {
			linkTable := table.Find(`td.cbi-table-field`)
			state.linkState = getText(linkTable.Get(0))
			state.linkLoss = getText(linkTable.Get(1))
			state.upWidth = getText(linkTable.Get(2))
			state.upSNR = getText(linkTable.Get(3))
			state.downWidth = getText(linkTable.Get(4))
			state.downSNR = getText(linkTable.Get(5))
		}
	})
	return &state
}

func (o *OTECStarApp) renderState(state *State) {
	o.wlanState.SetTitle("宽带: " + state.wlanState)
	if state.wlanState == `连接上` {
		if !o.wlanState.Checked() {
			o.wlanState.Checked()
			systray.SetIcon(icons.OK)
		}
	} else {
		if o.wlanState.Checked() {
			o.wlanState.Uncheck()
			systray.SetIcon(icons.ERROR)
		}
	}

	o.linkState.SetTitle("链路: " + state.linkState)
	if state.linkState == `连接上` {
		if !o.linkState.Checked() {
			o.linkState.Check()
			systray.SetIcon(icons.OK)
		}
	} else {
		if o.linkState.Checked() {
			o.linkState.Uncheck()
			systray.SetIcon(icons.ERROR)
		}
	}
	o.linkLoss.SetTitle("链路衰减: " + state.linkLoss + " dB")
	o.upWidth.SetTitle("上行速率: " + state.upWidth + " Mbps")
	o.upSNR.SetTitle("上行信噪比: " + state.upSNR + " dB")
	o.downWidth.SetTitle("下行速率: " + state.downWidth + " Mbps")
	o.downSNR.SetTitle("下行信噪比: " + state.downSNR + " dB")

}

func getText(node *html.Node) (t string) {
	if node.Type == html.TextNode {
		t = node.Data
		return
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		t += getText(c)
	}
	return
}

// NewOTECStarApp constructs a new OTECStarApp instance that is ready to run
func NewOTECStarApp(config *Config) *OTECStarApp {
	systray.SetIcon(icons.OK)
	systray.SetTooltip("OTECStar network status")

	app := OTECStarApp{
		wlanState: systray.AddMenuItem("宽带: -", ""),
		linkState: systray.AddMenuItem("链路: -", ""),
		linkLoss:  systray.AddMenuItem("链路衰减: -", ""),
		upWidth:   systray.AddMenuItem("上行速率: -", ""),
		upSNR:     systray.AddMenuItem("上行信噪比: -", ""),
		downWidth: systray.AddMenuItem("下行速率: -", ""),
		downSNR:   systray.AddMenuItem("下行信噪比: -", ""),
		updateCh:  make(chan *State),
		stopCh:    make(chan int),
		auth: AuthContainer{
			routerIP:      config.RouterIP,
			loginUrl:      "http://%s/cgi-bin/luci/customer/",
			stateUrl:      "",
			sysauthCookie: nil,
			username:      config.Username,
			password:      config.Password,
		},
	}
	systray.AddSeparator()
	systray.AddMenuItem(VERSION, "").Disable()
	systray.AddSeparator()

	if config.Interval < time.Second {
		logger.Warn().Dur("interval", config.Interval).
			Dur("actualInterval", time.Second).
			Msg("Interval should be at least 1 second")
		config.Interval = time.Second
	}
	ticker := time.NewTicker(config.Interval)
	app.Clicked(systray.AddMenuItem("Quit", ""), func() {
		ticker.Stop()
		close(app.stopCh)
		close(app.updateCh)
		systray.Quit()
	})

	// This goroutine triggers state capturing at an interval, the captured state is then sent through `updateCh`
	go func() {
		for {
			select {
			case <-app.stopCh:
				return
			case _, ok := <-ticker.C:
				if !ok { // Closed?
					return
				}
				app.updateCh <- app.getState()
			}
		}
	}()

	// This goroutine takes captured state from `updateCh`, and renders it to menu items
	go func() {
		for {
			select {
			case <-app.stopCh:
				// Drain update channel upon exit
				for range app.updateCh {
				}
				return
			case state, ok := <-app.updateCh:
				if !ok { // Closed?
					return
				}
				app.renderState(state)
			}
		}
	}()

	return &app
}
