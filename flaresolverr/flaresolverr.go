package flaresolverr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type FlareSolverr struct {
	URL string
}

type getOptions struct {
	Cmd               string   `json:"cmd"`
	URL               string   `json:"url"`
	MaxTimeout        int      `json:"maxTimeout"`
	Cookies           []Cookie `json:"cookies"`
	ReturnOnlyCookies bool     `json:"returnOnlyCookies"`
	WaitInSeconds     int      `json:"waitInSeconds"`
	DisableMedia      bool     `json:"disableMedia"`
	// TODO: proxy, session, tabsTillVerify
}

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type GetResponse struct {
	Solution struct {
		Url       string            `json:"url"`
		Status    int               `json:"status"`
		Cookies   []Cookie          `json:"cookies"`
		UserAgent string            `json:"userAgent"`
		Headers   map[string]string `json:"headers"`
		Response  string            `json:"response"`
	} `json:"solution"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	Session        string `json:"session,omitempty"`
	StartTimestamp int64  `json:"startTimestamp"`
	EndTimestamp   int64  `json:"endTimestamp"`
	Version        string `json:"version"`
}

type GetOption = func(*getOptions)

func WithDisabledMedia() func(o *getOptions) {
	return func(o *getOptions) {
		o.DisableMedia = true
	}
}

func (f FlareSolverr) Get(url string, opts ...GetOption) (*GetResponse, error) {
	reqOptions := &getOptions{
		Cmd: "request.get",
		URL: url,
	}

	for _, optFunc := range opts {
		optFunc(reqOptions)
	}

	payload, err := json.Marshal(&reqOptions)
	if err != nil {
		return nil, fmt.Errorf("unable marshal getoptions. %w", err)
	}

	r, err := http.Post(f.URL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("unable to make FlareSolverr get request. %w", err)
	}
	defer r.Body.Close()

	res := GetResponse{}
	d := json.NewDecoder(r.Body)
	err = d.Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("unable to decode FlareSolverr get response. %w", err)
	}

	return &res, nil
}
