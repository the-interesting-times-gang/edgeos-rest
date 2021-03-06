package edgeos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type Scenario string

const (
	PortForwarding Scenario = ".Port_Forwarding"
)

// A Client can interact with the EdgeOS REST API
type Client struct {
	Username, Password, Address string

	Path, Suffix, LoginEndpoint string

	cli *http.Client
}

func (c *Client) endpoint(s string) string {
	return fmt.Sprintf("%s/%s/%s%s", c.Address, c.Path, s, c.Suffix)
}

// Login sets up an http session with the EdgeOS device using the supplied
// endpoint and credentials
func (c *Client) Login() error {
	v := url.Values{
		"username": []string{c.Username},
		"password": []string{c.Password},
	}

	res, err := c.cli.PostForm(c.Address+"/"+c.LoginEndpoint, v)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

type Resp map[string]interface{}

func (c *Client) GetJSON(endpoint string, data interface{}) (Resp, error) {
	var m map[string]interface{}

	err := c.JSONFor(endpoint, data, &m)

	return m, err
}

func (c *Client) JSONFor(endpoint string, data interface{}, out interface{}) error {
	var (
		res *http.Response
		err error
	)
	if data == nil {
		res, err = c.cli.Get(c.endpoint(endpoint))
	} else {
		bs, _ := json.Marshal(map[string]interface{}{"data": data})
		res, err = c.cli.Post(c.endpoint(endpoint), "application/json", bytes.NewReader(bs))
	}

	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(out)
}

// Get returns some standard configuration information from EdgeOS
func (c *Client) Get() (Resp, error) {
	return c.GetJSON("get", nil)
}

func (c *Client) Feature(s Scenario) (Resp, error) {
	f, err := c.GetJSON("feature", map[string]string{
		"action":   "load",
		"scenario": string(s),
	})

	if err != nil {
		return Resp{}, err
	}

	return Resp(f["FEATURE"].(map[string]interface{})), nil
}

func (c *Client) FeatureFor(s Scenario, out interface{}) error {
	return c.JSONFor("feature", map[string]string{
		"action":   "load",
		"scenario": string(s),
	}, out)
}

// SetFeature allows users to programmatically update features
func (c *Client) SetFeature(s Scenario, data interface{}) (Resp, error) {
	f, err := c.GetJSON("feature", map[string]interface{}{
		"action":   "apply",
		"apply":    data,
		"scenario": string(s),
	})
	if err != nil {
		return Resp{}, err
	}

	return Resp(f["FEATURE"].(map[string]interface{})), nil
}

// SetFeatureFor takes a "Scenario", some data to send, and a pointer to an
// interface into which the JSON response will be decoded.
func (c *Client) SetFeatureFor(s Scenario, data interface{}, out interface{}) error {
	return c.JSONFor("feature", map[string]interface{}{
		"action":   "apply",
		"apply":    data,
		"scenario": string(s),
	}, out)
}

// NewClient returns an initialized Client for interacting with an EdgeOS device
func NewClient(addr, username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Username: username,
		Password: password,
		Path:     "api/edge",
		Suffix:   ".json",
		Address:  addr,
		cli: &http.Client{
			Transport: &csrfTransport{
				Referrer: addr,
			},
			Jar: jar,
		},
	}
	return c, nil
}
