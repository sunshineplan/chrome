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
	for _, i := range cookies {
		param := network.SetCookieParams{
			Name:     i.Name,
			Value:    i.Value,
			URL:      u.String(),
			Path:     i.Path,
			Domain:   i.Domain,
			Secure:   &i.Secure,
			HTTPOnly: &i.HttpOnly,
		}
		if i.MaxAge != 0 {
			expires := time.Now().Add(time.Duration(i.MaxAge) * time.Second)
			param.Expires = cdp.TimeSinceEpoch(expires.Unix())
		} else if !i.Expires.IsZero() {
			param.Expires = cdp.TimeSinceEpoch(i.Expires.Unix())
		}
		switch i.SameSite {
		case http.SameSiteLaxMode:
			param.SameSite = network.CookieSameSiteLax
		case http.SameSiteStrictMode:
			param.SameSite = network.CookieSameSiteStrict
		case http.SameSiteNoneMode:
			param.SameSite = network.CookieSameSiteNone
		}
		if _, err := chromedp.Call(ctx, network.SetCookie, param); err != nil {
			panic(err)
		}
	}
}

// Cookies retrieves cookies from the browser context for a given URL.
func Cookies(ctx context.Context, u *url.URL) (cookies []*http.Cookie) {
	var urls []string
	if u != nil {
		urls = append(urls, u.String())
	}
	res, err := chromedp.Call(
		ctx,
		network.GetCookies,
		network.GetCookiesParams{URLs: urls},
	)
	if err != nil {
		panic(err)
	}
	for _, i := range res.Cookies {
		cookies = append(cookies, &http.Cookie{Name: i.Name, Value: i.Value})
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
