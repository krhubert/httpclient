package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/krhubert/httpclient"
)

type Config struct {
	Address string
	ApiKey  string
	Mock    bool
}

type Client struct {
	client *httpclient.Client
}

func NewClient(cfg Config) (*Client, error) {
	c, err := httpclient.New(cfg.Address)
	if err != nil {
		return nil, err
	}

	c.SetAuthFunction(func(req *http.Request) {
		q := req.URL.Query()
		q.Add("hapikey", cfg.ApiKey)
		req.URL.RawQuery = q.Encode()
	})

	if cfg.Mock {
		c.SetHTTPClientTransport(httpclient.NewFakeRoundTrip())
	}

	return &Client{client: c}, nil
}

func (c *Client) ReadContact(ctx context.Context, id string) (*Contact, error) {
	contact := &Contact{}
	query := url.Values{
		"archived":             []string{"false"},
		"paginateAssociations": []string{"false"},
	}
	path := fmt.Sprintf("/crm/v3/objects/contacts/%s", id)

	if err := c.client.GetCtx(ctx, path, query, nil, contact, nil); err != nil {
		return nil, err
	}
	return contact, nil
}

func (c *Client) UpdateContact(ctx context.Context, id string, properties map[string]string) error {
	data := Contact{Properties: properties}
	path := fmt.Sprintf("/crm/v3/objects/contacts/%s", id)
	return c.client.PatchCtx(ctx, path, nil, nil, data, nil, nil)
}

func (c *Client) ArchiveContact(ctx context.Context, id string) error {
	path := fmt.Sprintf("/crm/v3/objects/contacts/%s", id)
	return c.client.DeleteCtx(ctx, path, nil, nil, nil, nil, nil)
}

type Contact struct {
	Id         string            `json:"id,omitempty"`
	Archived   bool              `json:"archived,omitempty"`
	CreatedAt  *time.Time        `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time        `json:"updatedAt,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

func main() {}
