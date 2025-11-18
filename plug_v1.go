package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
)

// deviceStatusV1 aggregates all status information for the device. Returned from the /status API
type deviceStatusV1 struct {
	WiFi     wiFiStatusV1       `json:"wifi_sta" yaml:"wifi_sta"`
	Cloud    cloudStatusV1      `json:"cloud" yaml:"cloud"`
	MQTT     mQTTStatusV1       `json:"mqtt" yaml:"mqtt"`
	Time     string             `json:"time" yaml:"time"`         // Current hour and minutes, HH:MM format
	Unixtime int64              `json:"unixtime" yaml:"unixtime"` // Unix timestamp if synced; 0 otherwise
	Serial   int64              `json:"serial" yaml:"serial"`     // Cloud serial number
	MAC      string             `json:"mac" yaml:"mac"`           // MAC address of the device
	Update   updateStatusV1     `json:"update" yaml:"update"`
	FS       fileSystemStatusV1 `json:"fs" yaml:"fs"`
	Uptime   int64              `json:"uptime" yaml:"uptime"`       // Seconds have elapsed since boot
	Relays   []relayV1          `json:"relays" yaml:"relays"`       // Array of relay statuses
	Meters   []meterV1          `json:"meters" yaml:"meters"`       // Array of meter statuses
	RamTotal int64              `json:"ram_total" yaml:"ram_total"` // Total amount of system memory in bytes
	RamFree  int64              `json:"ram_free" yaml:"ram_free"`   // Available amount of system memory in bytes
	FsSize   int64              `json:"fs_size" yaml:"fs_size"`     // Total amount of the file system in bytes
	FsFree   int64              `json:"fs_free" yaml:"fs_free"`     // Available amount of the file system in bytes

}

// deviceInfoV1 is the response from the /shelly API
type deviceInfoV1 struct {
	Type   string `json:"type" yaml:"type"`     // Shelly model identifier
	MAC    string `json:"mac" yaml:"mac"`       // MAC address of the device
	Auth   bool   `json:"auth" yaml:"auth"`     // Whether HTTP requests require authentication
	FW     string `json:"fw" yaml:"fw"`         // Current firmware version
	LongID int    `json:"longid" yaml:"longid"` // 1 if the device identifies itself with its full MAC address; 0 if only the last 3 bytes are used
}

// wiFiStatusV1 represents the current status of the WiFi connection.
type wiFiStatusV1 struct {
	Connected bool   `json:"connected" yaml:"connected"` // Status of WiFi connection
	SSID      string `json:"ssid" yaml:"ssid"`           // WiFi SSID
	IP        string `json:"ip" yaml:"ip"`               // IP address assigned by the WiFi router
	RSSI      int    `json:"rssi" yaml:"rssi"`           // Signal strength indicator
}

// cloudStatusV1 represents the current cloud connection status.
type cloudStatusV1 struct {
	Enabled   bool `json:"enabled" yaml:"enabled"`     // Whether cloud functionality is enabled
	Connected bool `json:"connected" yaml:"connected"` // Current cloud connection status
}

// mQTTStatusV1 represents the MQTT connection status when MQTT is enabled.
type mQTTStatusV1 struct {
	Connected bool `json:"connected" yaml:"connected"` // MQTT connection status
}

// updateStatusV1 contains information about firmware updates.
type updateStatusV1 struct {
	Status     string `json:"status" yaml:"status"`           // Current status of firmware update
	HasUpdate  bool   `json:"has_update" yaml:"has_update"`   // Whether a new firmware version is available
	NewVersion string `json:"new_version" yaml:"new_version"` // New firmware version
	OldVersion string `json:"old_version" yaml:"old_version"` // Old firmware version
}

// fileSystemStatusV1 contains information about the file system storage.
type fileSystemStatusV1 struct {
	Size int64 `json:"size" yaml:"size"` // Total amount of the file system in bytes
	Free int64 `json:"free" yaml:"free"` // Available amount of the file system in bytes
}

// relayV1 represents the current state of each relay output channel.
type relayV1 struct {
	IsOn           bool   `json:"ison" yaml:"ison"`                       // Indicates if the relay is on
	HasTimer       bool   `json:"has_timer" yaml:"has_timer"`             // Indicates if a timer is set
	TimerStarted   int64  `json:"timer_started" yaml:"timer_started"`     // Timestamp when the timer was started
	TimerDuration  int64  `json:"timer_duration" yaml:"timer_duration"`   // Duration of the timer
	TimerRemaining int64  `json:"timer_remaining" yaml:"timer_remaining"` // Time remaining on the timer
	Overpower      bool   `json:"overpower" yaml:"overpower"`             // Indicates if the overpower protection is triggered
	Source         string `json:"source" yaml:"source"`                   // Source that caused the last state change
}

// meterV1 represents the current status of each power meter.
type meterV1 struct {
	Power     float64   `json:"power" yaml:"power"`         // Current power usage
	Overpower float64   `json:"overpower" yaml:"overpower"` // Overpower value threshold
	IsValid   bool      `json:"is_valid" yaml:"is_valid"`   // Validity of the meter reading
	Timestamp int64     `json:"timestamp" yaml:"timestamp"` // Timestamp of the meter reading
	Counters  []float64 `json:"counters" yaml:"counters"`   // Counters array with meter readings
	Total     int64     `json:"total" yaml:"total"`         // Total consumption
}

func newShellyV1Plug(address url.URL) (Plug, error) {
	if address.Host == "" {
		return nil, fmt.Errorf("invalid address")
	}

	return &shellyPlugV1{
		address: &address,
	}, nil
}

type shellyPlugV1 struct {
	address *url.URL
}

func (s *shellyPlugV1) RenderInfo(w io.Writer) error {
	nfo, err := s.Info()
	if err != nil {
		return err
	}
	status, err := s.Status()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Shelly device information for %s\n", ip.String())
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Device Information")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "         Device Type: %s\n", nfo.Type)
	fmt.Fprintf(w, "            Firmware: %s\n", nfo.FW)
	fmt.Fprintf(w, "         MAC Address: %s\n", status.MAC)
	fmt.Fprintln(w)

	t := time.Unix(status.Unixtime, 0)

	fmt.Fprintln(w, "Device Status")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "                Time: %s\n", t)
	fmt.Fprintf(w, "              Uptime: %v\n", time.Duration(status.Uptime)*time.Second)
	fmt.Fprintf(w, "        Memory Total: %v\n", humanize.IBytes(uint64(status.RamTotal)))
	fmt.Fprintf(w, "         Memory Free: %v\n", humanize.IBytes(uint64(status.RamFree)))
	fmt.Fprintf(w, "       Storage Total: %v\n", humanize.IBytes(uint64(status.FsSize)))
	fmt.Fprintf(w, "        Storage Free: %v\n", humanize.IBytes(uint64(status.FsFree)))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Network Information")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "          IP Address: %s\n", status.WiFi.IP)
	fmt.Fprintf(w, "           WiFi SSID: %s\n", status.WiFi.SSID)
	fmt.Fprintf(w, "       WiFi Strength: %d\n", status.WiFi.RSSI)
	fmt.Fprintf(w, "       Cloud Enabled: %t\n", status.Cloud.Enabled)
	if status.Cloud.Enabled {
		fmt.Fprintf(w, "     Cloud Connected: %t\n", status.Cloud.Connected)
	}
	fmt.Fprintf(w, "      MQTT Connected: %t\n", status.MQTT.Connected)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Updates Information")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "          Has Update: %t\n", status.Update.HasUpdate)
	fmt.Fprintf(w, "    Latest Available: %s\n", status.Update.NewVersion)

	if len(status.Relays) == 1 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Relay Information")
		fmt.Fprintln(w)
		s := "On"
		if !status.Relays[0].IsOn {
			s = "Off"
		}
		fmt.Fprintf(w, "        Power Status: %s\n", s)
		fmt.Fprintf(w, "               Timer: %t\n", status.Relays[0].HasTimer)
		if status.Relays[0].HasTimer {
			t := time.Unix(status.Relays[0].TimerStarted, 0)
			fmt.Fprintf(w, "             Started: %v\n", t)
			fmt.Fprintf(w, "            Duration: %v\n", time.Duration(status.Relays[0].TimerDuration)*time.Second)
			fmt.Fprintf(w, "           Remaining: %v\n", time.Duration(status.Relays[0].TimerRemaining)*time.Second)
		}
	}

	if len(status.Meters) == 1 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Meter Information")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "               Power: %.2f Watt\n", status.Meters[0].Power)
		fmt.Fprintf(w, "   Total Consumption: %.2f kWh\n", float64(status.Meters[0].Total)*0.000016666666666666667)
	}

	return nil
}

func (s *shellyPlugV1) RenderEnergy(w io.Writer) error {
	status, err := s.Status()
	if err != nil {
		return err
	}

	if len(status.Meters) != 1 {
		return fmt.Errorf("no meter information received")
	}
	if len(status.Relays) != len(status.Meters) {
		return fmt.Errorf("invalid relay information received")
	}

	m := status.Meters[0]
	r := status.Relays[0]

	isOn := float64(0)
	if r.IsOn {
		isOn = 1
	}

	reading := map[string]any{
		"power_watt":      m.Power,
		"power_total_kwh": float64(m.Total) * 0.000016666666666666667,
		"is_on":           isOn,
	}

	switch {
	case jsonFormat:
		j, err := json.MarshalIndent(reading, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(j))

	case choriaFormat:
		data := map[string]any{
			"labels": labels,
			"metrics": map[string]any{
				"current_power_watt": reading["power_watt"],
				"today_energy_kwh":   reading["power_total_kwh"],
				"relay_on":           reading["is_on"],
			}}
		j, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(j))

	default:
		fmt.Fprintln(w, "Meter Information")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "          Powered On: %t\n", isOn == 1)
		fmt.Fprintf(w, "               Power: %.2f Watt\n", m.Power)
		fmt.Fprintf(w, "   Total Consumption: %.2f kWh\n", float64(m.Total)*0.000016666666666666667)
	}

	return nil
}

func (s *shellyPlugV1) get(path string, queries map[string]string, response any) error {
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

func (s *shellyPlugV1) TurnOn() (bool, error) {
	var res relayV1

	err := s.get("relay/0", map[string]string{"turn": "on"}, &res)
	if err != nil {
		return false, err
	}

	if !res.IsOn {
		return false, fmt.Errorf("relay is not on")
	}
	return res.IsOn, nil
}

func (s *shellyPlugV1) TurnOff() (bool, error) {
	var res relayV1

	err := s.get("relay/0", map[string]string{"turn": "off"}, &res)
	if err != nil {
		return false, err
	}

	if res.IsOn {
		return res.IsOn, fmt.Errorf("relay is on")
	}

	return res.IsOn, nil
}

func (s *shellyPlugV1) Status() (*deviceStatusV1, error) {
	var res deviceStatusV1

	err := s.get("status", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *shellyPlugV1) Info() (*deviceInfoV1, error) {
	var res deviceInfoV1

	err := s.get("shelly", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
