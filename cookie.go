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

func Cookies(ctx context.Context, u *url.URL) (res []*http.Cookie) {
	var urls []string
	if u != nil {
		urls = append(urls, u.String())
	}

	var cookies []*network.Cookie
	if err := chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			cookies, err = network.GetCookies().WithUrls(urls).Do(ctx)
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
