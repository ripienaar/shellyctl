package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
)

func NewShellyPlug(address url.URL) (Plug, error) {
	if address.Host == "" {
		return nil, fmt.Errorf("invalid address")
	}

	return &shellyPlug{
		address: &address,
	}, nil
}

type shellyPlug struct {
	address *url.URL
}

func (s *shellyPlug) get(path string, queries map[string]string, response any) error {
	rc := resty.New()
	client := rc.R()

	if s.address.User != nil {
		password, _ := s.address.User.Password()
		client.SetBasicAuth(s.address.User.Username(), password)
		rc.SetDisableWarn(true)
	}

	client.SetQueryParams(queries)

	resp, err := client.Get(fmt.Sprintf("http://%s/%s", s.address.Hostname(), path))
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("%s: %s", resp.Request.URL, resp.String())
	}

	err = json.Unmarshal(resp.Body(), response)
	if err != nil {
		return fmt.Errorf("invalid response body: %v", err)
	}

	return nil
}

func (s *shellyPlug) TurnOn() (*Relay, error) {
	var res Relay

	err := s.get("relay/0", map[string]string{"turn": "on"}, &res)
	if err != nil {
		return nil, err
	}

	if !res.IsOn {
		return nil, fmt.Errorf("relay is not on")
	}
	return &res, nil
}

func (s *shellyPlug) TurnOff() (*Relay, error) {
	var res Relay

	err := s.get("relay/0", map[string]string{"turn": "off"}, &res)
	if err != nil {
		return nil, err
	}

	if res.IsOn {
		return &res, fmt.Errorf("relay is on")
	}

	return &res, nil
}

func (s *shellyPlug) Status() (*DeviceStatus, error) {
	var res DeviceStatus

	err := s.get("status", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *shellyPlug) Info() (*DeviceInfo, error) {
	var res DeviceInfo

	err := s.get("shelly", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
