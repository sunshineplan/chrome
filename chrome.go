package chrome

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type Chrome struct {
	url              string
	useragent        string
	width            int
	height           int
	proxy            string
	enableExtensions bool
	debugger         *log.Logger

	flags   []chromedp.ExecAllocatorOption
	ctxOpts []chromedp.ContextOption
	actions []chromedp.Action

	mu sync.Mutex

	ctx    context.Context
	cancel chan struct{}
	done   chan struct{}
}

func New(url string) *Chrome {
	c := &Chrome{url: url, debugger: log.New(io.Discard, "", log.LstdFlags), cancel: make(chan struct{})}
	return c.AddContextOptions(chromedp.WithDebugf(c.debugger.Printf))
}

func (c *Chrome) headless() *Chrome {
	return c.AddFlags(
		chromedp.Flag("headless", "new"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
}

func UserAgent() (userAgent string) {
	var errs []error
	for i := range 5 {
		c := New("").headless().NoSandbox()
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()
		if err := chromedp.Run(ctx, chromedp.Evaluate("navigator.userAgent", &userAgent)); err != nil {
			if err == context.Canceled {
				errs = append(errs, context.Cause(ctx))
			}
		}
		c.Close()
		if userAgent != "" {
			break
		} else if i < 4 {
			time.Sleep(5 * time.Second)
		}
	}
	if userAgent == "" {
		panic("failed to get chrome useragent: " + errors.Join(errs...).Error())
	}
	userAgent = strings.ReplaceAll(userAgent, "Headless", "")
	return
}

func Headless() *Chrome {
	return New("").UserAgent(UserAgent()).headless()
}

func Headful() *Chrome {
	return New("")
}

func Remote(url string) *Chrome {
	if url == "" {
		panic("empty url")
	}
	return New(url)
}

func Local(port int) *Chrome {
	if port <= 0 || port > 65535 {
		panic("invalid port number: " + strconv.Itoa(port))
	}
	return Remote(fmt.Sprintf("ws://localhost:%d", port))
}

func (c *Chrome) SetDebuggerOutput(w io.Writer) *Chrome {
	c.debugger.SetOutput(w)
	return c
}

func (c *Chrome) SetDebuggerPrefix(prefix string) *Chrome {
	c.debugger.SetPrefix(prefix)
	return c
}

func (c *Chrome) SetDebuggerFlags(flag int) *Chrome {
	c.debugger.SetFlags(flag)
	return c
}

func (c *Chrome) UserAgent(useragent string) *Chrome {
	c.useragent = useragent
	return c
}

func (c *Chrome) WindowSize(width int, height int) *Chrome {
	c.width, c.height = width, height
	return c
}

func (c *Chrome) Proxy(proxy string) *Chrome {
	c.proxy = proxy
	return c
}

func (c *Chrome) EnableExtensions(enable bool) *Chrome {
	c.enableExtensions = enable
	return c
}

func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
}

func (c *Chrome) AutoOpenDevtools() *Chrome {
	return c.AddFlags(chromedp.Flag("auto-open-devtools-for-tabs", true))
}

func (c *Chrome) Incognito() *Chrome {
	return c.AddFlags(chromedp.Flag("incognito", true))
}

func (c *Chrome) Guest() *Chrome {
	return c.AddFlags(chromedp.Flag("guest", true))
}

func (c *Chrome) NoSandbox() *Chrome {
	return c.AddFlags(chromedp.Flag("no-sandbox", true))
}

func (c *Chrome) DisableUserAgentClientHint() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-features", "UserAgentClientHint"))
}

func (c *Chrome) DisableAutomationControlled() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-blink-features", "AutomationControlled"))
}

func (c *Chrome) AddContextOptions(opts ...chromedp.ContextOption) *Chrome {
	c.ctxOpts = append(c.ctxOpts, opts...)
	return c
}

func (c *Chrome) AddActions(actions ...chromedp.Action) *Chrome {
	c.actions = append(c.actions, actions...)
	return c
}

var DefaultExecAllocatorOptions = [...]chromedp.ExecAllocatorOption{
	// https://github.com/puppeteer/puppeteer/blob/main/packages/puppeteer-core/src/node/ChromeLauncher.ts
	chromedp.Flag("allow-pre-commit-input", true),
	chromedp.Flag("disable-background-networking", true),
	chromedp.Flag("disable-background-timer-throttling", true),
	chromedp.Flag("disable-backgrounding-occluded-windows", true),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-client-side-phishing-detection", true),
	chromedp.Flag("disable-component-extensions-with-background-pages", true),
	chromedp.Flag("disable-crash-reporter", true),
	chromedp.Flag("disable-default-apps", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("disable-hang-monitor", true),
	chromedp.Flag("disable-infobars", true),
	chromedp.Flag("disable-ipc-flooding-protection", true),
	chromedp.Flag("disable-popup-blocking", true),
	chromedp.Flag("disable-prompt-on-repost", true),
	chromedp.Flag("disable-renderer-backgrounding", true),
	chromedp.Flag("disable-search-engine-choice-screen", true),
	chromedp.Flag("disable-sync", true),
	chromedp.Flag("enable-automation", true),
	chromedp.Flag("export-tagged-pdf", true),
	// chromedp.Flag("force-color-profile", "srgb"),
	chromedp.Flag("generate-pdf-document-outline", true),
	chromedp.Flag("metrics-recording-only", true),
	chromedp.Flag("no-first-run", true),
	chromedp.Flag("password-store", "basic"),
	chromedp.Flag("use-mock-keychain", true),
	chromedp.Flag("disable-features", "Translate,AcceptCHFrame,MediaRouter,OptimizationHints,ProcessPerSiteUpToMainFrameThreshold,IsolateSandboxedIframes"),

	chromedp.Flag("start-maximized", true),
}

func (c *Chrome) Run(actions ...chromedp.Action) error {
	return chromedp.Run(c, actions...)
}

func (c *Chrome) WaitNewTarget(fn func(*target.Info) bool) <-chan target.ID {
	return chromedp.WaitNewTarget(c, fn)
}

func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
