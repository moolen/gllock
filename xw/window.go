package xw

import (
	"fmt"
	"image"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/BurntSushi/xgbutil/mousebind"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/moolen/glitchlock/pam"
	log "github.com/sirupsen/logrus"
)

type XW struct {
	X  *xgb.Conn
	Xu *xgbutil.XUtil
}

func New() (*XW, error) {
	X, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	Xu, err := xgbutil.NewConnXgb(X)
	if err != nil {
		return nil, err
	}
	return &XW{X, Xu}, nil
}

func (x *XW) GrabInput() error {
	keybind.Initialize(x.Xu)
	xscreen := xproto.Setup(x.X).DefaultScreen(x.X)
	grabc := xproto.GrabKeyboard(x.X, false, xscreen.Root, xproto.TimeCurrentTime,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
	)
	repk, err := grabc.Reply()
	if err != nil {
		return fmt.Errorf("error grabbing Keyboard")
	}
	if repk.Status != xproto.GrabStatusSuccess {
		return fmt.Errorf("could not grab keyboard")
	}
	grabp := xproto.GrabPointer(x.X, false, xscreen.Root, (xproto.EventMaskKeyPress|xproto.EventMaskKeyRelease)&0,
		xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.CursorNone, xproto.TimeCurrentTime)
	repp, err := grabp.Reply()
	if err != nil {
		return fmt.Errorf("error grabbing pointer")
	}
	if repp.Status != xproto.GrabStatusSuccess {
		return fmt.Errorf("could not grab pointer")
	}
	return nil
}

func (x *XW) FindWindow(name string) (xproto.Window, error) {
	clientids, err := ewmh.ClientListGet(x.Xu)
	if err != nil {
		return 0, err
	}
	for _, clientid := range clientids {
		winName, err := ewmh.WmNameGet(x.Xu, clientid)
		if err != nil || len(winName) == 0 {
			winName, err = icccm.WmNameGet(x.Xu, clientid)
			if err != nil || len(winName) == 0 {
				winName = "N/A"
			}
		}
		if winName == name {
			return clientid, nil
		}
	}
	return 0, fmt.Errorf("X Window not found")
}

func (x *XW) Fullscreen(name string) error {
	win, err := x.FindWindow(name)
	if err != nil {
		return err
	}
	err = ewmh.WmStateReq(x.Xu, win, ewmh.StateToggle,
		"_NET_WM_STATE_FULLSCREEN")
	if err != nil {
		return err
	}
	return nil
}

func (x *XW) PasswordMatch() <-chan struct{} {
	lastInput := time.Now()
	done := make(chan struct{}, 1)

	go func() {
		var password string
		for {
			ev, err := x.X.WaitForEvent()
			if ev == nil && err == nil {
				log.Error(fmt.Errorf("Both event and error are nil. Exiting"))
				done <- struct{}{}
				return
			}
			if err != nil {
				log.Error(fmt.Errorf("Both event and error are nil. Exiting"))
				done <- struct{}{}
				return
			}
			if time.Now().Sub(lastInput) > time.Second*2 {
				log.Debugf("timeout reached. clearing password")
				password = ""
			}
			switch e := ev.(type) {
			case xproto.KeyPressEvent:
				key := keybind.LookupString(x.Xu, e.State, e.Detail)
				log.Debugf("keypress: %s %v ", key, e)
				lastInput = time.Now()
				if len(key) == 1 {
					password += key
				}
				if keybind.KeyMatch(x.Xu, "BackSpace", e.State, e.Detail) && len(password) > 0 {
					password = password[:len(password)-1]
				}
				log.Debugf("current password: %s", password)
				if keybind.KeyMatch(x.Xu, "Return", e.State, e.Detail) {
					log.Debugf("...checking password")
					if pam.AuthenticateCurrentUser(password) {
						done <- struct{}{}
						return
					}
					log.Debugf("password does not match")
				}
			}
		}
	}()
	return done
}

func (xw *XW) Overlay(x, y, width, height int) error {
	log.Debugf("overlay size: %d %d %d %d", x, y, x+width, y+height)
	rgba := image.NewRGBA(image.Rect(x, y, x+width, y+height))
	ximg := xgraphics.NewConvert(xw.Xu, rgba)
	win, err := xwindow.Generate(xw.Xu)
	if err != nil {
		return err
	}
	win.Create(xw.Xu.RootWin(), x, y, width, height, 0)
	win.WMGracefulClose(func(w *xwindow.Window) {
		xevent.Detach(w.X, w.Id)
		keybind.Detach(w.X, w.Id)
		mousebind.Detach(w.X, w.Id)
		w.Destroy()
	})
	err = icccm.WmStateSet(xw.Xu, win.Id, &icccm.WmState{
		State: icccm.StateNormal,
	})
	if err != nil {
		return err
	}
	err = ewmh.WmStateSet(xw.Xu, win.Id, []string{"_NET_WM_STATE_FULLSCREEN", "_NET_WM_STATE_ABOVE"})
	if err != nil {
		return err
	}
	err = icccm.WmNormalHintsSet(xw.Xu, win.Id, &icccm.NormalHints{
		Flags:     icccm.SizeHintPMinSize | icccm.SizeHintPMaxSize,
		MinWidth:  uint(width),
		MinHeight: uint(height),
		MaxWidth:  uint(width),
		MaxHeight: uint(height),
	})
	if err != nil {
		return err
	}

	// Paint our image before mapping.
	ximg.XSurfaceSet(win.Id)
	ximg.XDraw()
	ximg.XPaint(win.Id)
	win.Map()

	// some WM override this position after mapping
	win.Move(x, y)
	return nil
}
