package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0xThiebaut/mdeproxy/internal/cookies"
	"github.com/0xThiebaut/mdeproxy/internal/times"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type Client interface {
	Timeline(ctx context.Context, from, to time.Time, device string) <-chan interface{}
	Error() error
}

const (
	keyFrom         = "fromDate"
	keyTo           = "toDate"
	headerUserAgent = "MDE-Proxy (+https://github.com/0xThiebaut/mdeproxy)"
	proxyHost       = "security.microsoft.com"
)

var proxy = &url.URL{
	Scheme: "https",
	Host:   proxyHost,
	Path:   "/apiproxy/mtp/",
}

func New(cookie string, csrf string) (Client, error) {
	// Create a new cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	v, err := cookies.Parse(cookie)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(proxy, v)

	// Create a hooked client
	c := &client{xsrf: csrf, retries: 3}
	c.client = &http.Client{
		Transport: c,
		Jar:       jar,
	}

	return c, nil
}

type client struct {
	err     error
	xsrf    string
	retries int
	client  *http.Client
}

// RoundTrip provides a hooking mechanism to inject headers for all requests
func (c *client) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Accept-Language", "en-us")
	if r.URL.Host == proxyHost {
		r.Header.Set("X-XSRF-TOKEN", c.xsrf)
	}
	r.Header.Set("User-Agent", headerUserAgent)
	return http.DefaultTransport.RoundTrip(r)
}

func (c *client) Timeline(ctx context.Context, from, to time.Time, id string) <-chan interface{} {
	// Build the base URL
	base := proxy.JoinPath("mdeTimelineExperience")

	// Build the first query
	uri := base.JoinPath("machines", id, "events")
	query := uri.Query()
	query.Set("fromDate", from.Format(times.Layout))
	query.Set("toDate", times.Min(from.AddDate(0, 0, 7), to).Format(times.Layout))
	query.Set("generateIdentityEvents", "true")
	query.Set("includeIdentityEvents", "true")
	query.Set("supportMdiOnlyEvents", "true")
	query.Set("includeSentinelEvents", "false")
	query.Set("doNotUseCache", "false")
	query.Set("forceUseCache", "false")
	query.Set("pageSize", "1000")
	uri.RawQuery = query.Encode()

	result := make(chan interface{})
	go func() {
		defer close(result)

		// Perform an initial query
		var data *events
		if data, c.err = c.get(ctx, uri.String()); c.err != nil {
			return
		}

		// Stream the data
		for _, item := range data.Items {
			result <- item
		}

		// Abort on partial errors
		if len(data.PartialResponseReasons) > 0 {
			c.err = fmt.Errorf("partial data: %#v", data.PartialResponseReasons)
			return
		}

		// Get the linked queries
		prev, next := data.Prev, data.Next
		var overlap time.Time

		// Loop backwards
		for len(prev) > 0 && ctx.Err() == nil {
			// Ensure the back-walking is time-boxed
			if _, overlap, c.err = parse(prev); c.err != nil {
				return
			} else if overlap.Before(from) {
				break
			}

			// Perform an initial query
			if data, c.err = c.get(ctx, base.String()+prev); c.err != nil {
				return
			}

			// Stream the data
			for _, item := range data.Items {
				result <- item
			}

			// Abort on partial errors
			if len(data.PartialResponseReasons) > 0 {
				c.err = fmt.Errorf("partial data: %#v", data.PartialResponseReasons)
				return
			}

			prev = data.Prev
		}

		// Loop forwards
		for len(next) > 0 && ctx.Err() == nil {
			// Ensure the forward-walking is time-boxed

			if overlap, _, c.err = parse(next); c.err != nil {
				return
			} else if overlap.After(to) {
				break
			}

			// Perform an initial query

			if data, c.err = c.get(ctx, base.String()+next); c.err != nil {
				return
			}

			// Stream the data
			for _, item := range data.Items {
				result <- item
			}

			// Abort on partial errors
			if len(data.PartialResponseReasons) > 0 {
				c.err = fmt.Errorf("partial data: %#v", data.PartialResponseReasons)
				return
			}

			next = data.Next
		}
	}()
	return result
}

type events struct {
	Items                  []interface{} `json:"Items"`
	PartialResponseReasons []interface{} `json:"PartialResponseReasons"`
	Prev                   string        `json:"Prev"`
	Next                   string        `json:"Next"`
}

func (c *client) do(req *http.Request) (resp *http.Response, err error) {
	// Execute the query
	for i := 0; i < c.retries; i++ {
		if resp, err = c.client.Do(req); err == nil {
			break
		}
		log.Printf("Retrying (%d/%d): %s", i+1, c.retries, err)
		time.Sleep(5 * time.Second)
	}
	return resp, err
}

func (c *client) get(ctx context.Context, url string) (*events, error) {
	// Build the query
	log.Println(url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Execute the query
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Abort for errors
	if resp.StatusCode != http.StatusOK {
		m, merr := io.ReadAll(resp.Body)
		if merr == nil {
			merr = errors.New(string(m))
		}
		return nil, fmt.Errorf("bad status: %s (%s): %w", resp.Status, http.StatusText(resp.StatusCode), merr)
	}

	// Decode the response
	v := &events{}
	d := json.NewDecoder(resp.Body)
	return v, d.Decode(v)
}

func (c *client) Error() error {
	return c.err
}

func parse(path string) (from, to time.Time, err error) {
	u, err := url.ParseRequestURI(path)
	if err != nil {
		return
	}
	v := u.Query()
	if !v.Has(keyFrom) || !v.Has(keyTo) {
		return from, to, errors.New("missing time range query parameter")
	}
	if from, err = time.Parse(times.Layout, v.Get(keyFrom)); err == nil {
		to, err = time.Parse(times.Layout, v.Get(keyTo))
	}
	return from, to, err
}
