package chrome

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// SetCookies sets cookies in the browser context for a given URL.
func SetCookies(ctx context.Context, u *url.URL, cookies []*http.Cookie) {
	var actions []chromedp.Action
	for _, i := range cookies {
		params := network.SetCookie(i.Name, i.Value).
			WithURL(u.String()).
			WithPath(i.Path).
			WithDomain(i.Domain).
			WithSecure(i.Secure).
			WithHTTPOnly(i.HttpOnly)
		if i.MaxAge != 0 {
			expires := time.Now().Add(time.Duration(i.MaxAge) * time.Second)
			params.WithExpires((*cdp.TimeSinceEpoch)(&expires))
		} else if !i.Expires.IsZero() {
			params.WithExpires((*cdp.TimeSinceEpoch)(&i.Expires))
		}
		switch i.SameSite {
		case http.SameSiteLaxMode:
			params.WithSameSite(network.CookieSameSiteLax)
		case http.SameSiteStrictMode:
			params.WithSameSite(network.CookieSameSiteStrict)
		case http.SameSiteNoneMode:
			params.WithSameSite(network.CookieSameSiteNone)
		}
		actions = append(actions, params)
	}
	if err := chromedp.Run(ctx, actions...); err != nil {
		panic(err)
	}
}

// Cookies retrieves cookies from the browser context for a given URL.
func Cookies(ctx context.Context, u *url.URL) (res []*http.Cookie) {
	var urls []string
	if u != nil {
		urls = append(urls, u.String())
	}

	var cookies []*network.Cookie
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			cookies, err = network.GetCookies().WithURLs(urls).Do(ctx)
			return
		}),
	); err != nil {
		panic(err)
	}
	for _, i := range cookies {
		res = append(res, &http.Cookie{Name: i.Name, Value: i.Value})
	}
	return
}

// Ensure Chrome implements http.CookieJar interface.
var _ http.CookieJar = &Chrome{}

// SetCookies sets cookies in the browser for the given URL.
func (c *Chrome) SetCookies(u *url.URL, cookies []*http.Cookie) {
	SetCookies(c, u, cookies)
}

// Cookies retrieves cookies from the browser for the given URL.
func (c *Chrome) Cookies(u *url.URL) []*http.Cookie {
	return Cookies(c, u)
}
