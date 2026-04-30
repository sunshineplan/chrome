// Package chrome provides a wrapper around chromedp for simplified browser automation.
// It offers fluent API for configuring and running Chrome/Chromium instances.
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

// Chrome represents a Chrome/Chromium browser instance with configuration options.
// It provides a fluent API for setting up and managing browser sessions.
type Chrome struct {
	url              string      // Target URL for the browser instance
	useragent        string      // Custom user agent string
	width            int         // Browser window width
	height           int         // Browser window height
	proxy            string      // Proxy URL for network requests
	enableExtensions bool        // Whether to enable Chrome extensions
	debugger         *log.Logger // Logger for debug output

	flags   []chromedp.ExecAllocatorOption // Chrome execution flags
	ctxOpts []chromedp.ContextOption       // Context options for chromedp
	actions []chromedp.Action              // Actions to execute on browser startup

	mu sync.Mutex // Mutex for thread-safe operations

	ctx    context.Context // Browser context
	cancel chan struct{}   // Channel to signal cancellation
	done   chan struct{}   // Channel to signal completion
}

// New creates a new Chrome instance with the specified URL.
func New(url string) *Chrome {
	c := &Chrome{url: url, debugger: log.New(io.Discard, "", log.LstdFlags), cancel: make(chan struct{})}
	return c.AddContextOptions(chromedp.WithDebugf(c.debugger.Printf))
}

// headless configures Chrome to run in headless mode (no visible UI).
func (c *Chrome) headless() *Chrome {
	return c.AddFlags(
		chromedp.Flag("headless", "new"),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
}

// UserAgent retrieves the user agent string from a running Chrome instance.
// It retries up to 5 times with 5-second intervals if the initial attempt fails.
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
		if len(errs) == 0 {
			errs = append(errs, errors.New("empty chrome useragent string"))
		}
		panic("failed to get chrome useragent: " + errors.Join(errs...).Error())
	}
	userAgent = strings.ReplaceAll(userAgent, "Headless", "")
	return
}

// Headless creates a new headless Chrome instance with the current user agent.
func Headless() *Chrome {
	return New("").UserAgent(UserAgent()).headless()
}

// Headful creates a new Chrome instance with visible UI (headful mode).
func Headful() *Chrome {
	return New("")
}

// NewChrome creates a new Chrome instance in either headless or headful mode.
func NewChrome(headless bool) *Chrome {
	if headless {
		return Headless()
	}
	return Headful()
}

// Remote creates a Chrome instance that connects to a remote Chrome DevTools Protocol endpoint.
// The url parameter should be a WebSocket address (e.g., ws://localhost:9222).
func Remote(url string) *Chrome {
	if url == "" {
		panic("empty url")
	}
	return New(url)
}

// Local creates a Chrome instance that connects to a local Chrome DevTools Protocol server on the specified port.
func Local(port int) *Chrome {
	if port <= 0 || port > 65535 {
		panic("invalid port number: " + strconv.Itoa(port))
	}
	return Remote(fmt.Sprintf("ws://localhost:%d", port))
}

// SetDebuggerOutput sets the output destination for the debugger logger.
func (c *Chrome) SetDebuggerOutput(w io.Writer) *Chrome {
	c.debugger.SetOutput(w)
	return c
}

// SetDebuggerPrefix sets the prefix for the debugger logger output.
func (c *Chrome) SetDebuggerPrefix(prefix string) *Chrome {
	c.debugger.SetPrefix(prefix)
	return c
}

// SetDebuggerFlags sets the flags for the debugger logger (e.g., log.Lshortfile, log.Ldate).
func (c *Chrome) SetDebuggerFlags(flag int) *Chrome {
	c.debugger.SetFlags(flag)
	return c
}

// UserAgent sets a custom user agent string for the browser.
func (c *Chrome) UserAgent(useragent string) *Chrome {
	c.useragent = useragent
	return c
}

// WindowSize sets the browser window dimensions (width and height).
func (c *Chrome) WindowSize(width int, height int) *Chrome {
	c.width, c.height = width, height
	return c
}

// Proxy sets the proxy URL for network requests made by the browser.
func (c *Chrome) Proxy(proxy string) *Chrome {
	c.proxy = proxy
	return c
}

// EnableExtensions enables or disables Chrome extensions in the browser.
func (c *Chrome) EnableExtensions(enable bool) *Chrome {
	c.enableExtensions = enable
	return c
}

// AddFlags appends Chrome execution allocator options/flags.
func (c *Chrome) AddFlags(flags ...chromedp.ExecAllocatorOption) *Chrome {
	c.flags = append(c.flags, flags...)
	return c
}

// AutoOpenDevtools automatically opens Chrome DevTools for each tab.
func (c *Chrome) AutoOpenDevtools() *Chrome {
	return c.AddFlags(chromedp.Flag("auto-open-devtools-for-tabs", true))
}

// Incognito enables incognito (private) browsing mode.
func (c *Chrome) Incognito() *Chrome {
	return c.AddFlags(chromedp.Flag("incognito", true))
}

// Guest enables guest browsing mode.
func (c *Chrome) Guest() *Chrome {
	return c.AddFlags(chromedp.Flag("guest", true))
}

// NoSandbox disables Chrome sandbox (useful in restricted environments).
func (c *Chrome) NoSandbox() *Chrome {
	return c.AddFlags(chromedp.Flag("no-sandbox", true))
}

// DisableUserAgentClientHint disables User-Agent Client Hints feature.
func (c *Chrome) DisableUserAgentClientHint() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-features", "UserAgentClientHint"))
}

// DisableAutomationControlled disables the AutomationControlled feature to hide browser automation detection.
func (c *Chrome) DisableAutomationControlled() *Chrome {
	return c.AddFlags(chromedp.Flag("disable-blink-features", "AutomationControlled"))
}

// AddContextOptions appends chromedp context options.
func (c *Chrome) AddContextOptions(opts ...chromedp.ContextOption) *Chrome {
	c.ctxOpts = append(c.ctxOpts, opts...)
	return c
}

// AddActions appends chromedp actions to be executed on browser startup.
func (c *Chrome) AddActions(actions ...chromedp.Action) *Chrome {
	c.actions = append(c.actions, actions...)
	return c
}

// DefaultExecAllocatorOptions provides default Chrome execution flags optimized for headless automation.
// Based on Puppeteer's default configuration.
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

// Run executes the provided chromedp actions in the browser context.
func (c *Chrome) Run(actions ...chromedp.Action) error {
	return chromedp.Run(c, actions...)
}

// WaitNewTarget waits for a new target (tab/window) matching the provided filter function.
func (c *Chrome) WaitNewTarget(fn func(*target.Info) bool) <-chan target.ID {
	return chromedp.WaitNewTarget(c, fn)
}

// AddScriptToEvaluateOnNewDocument adds a script that will be evaluated on every new document in every frame.
func AddScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return
	})
}
