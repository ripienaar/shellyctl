package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
)

type DeviceStatusV2 struct {
	System  *SystemStatusV2 `json:"sys"`
	Switch1 *SwitchStatusV2 `json:"switch:0"`
	WiFi    *WifiStatusV2   `json:"wifi"`
}

type DeviceInfoV2 struct {
	ID    string `json:"id"`    // Device ID
	MAC   string `json:"mac"`   // MAC address
	Model string `json:"model"` // Device model
	Gen   int    `json:"gen"`   // Hardware generation

	FWID    string `json:"fw_id"`             // Firmware ID
	Ver     string `json:"ver"`               // Firmware version
	App     string `json:"app"`               // Application name
	Profile string `json:"profile,omitempty"` // Device profile (multi-profile devices only)

	AuthEn     bool   `json:"auth_en"`     // Authentication enabled
	AuthDomain string `json:"auth_domain"` // Domain name or null when authentication disabled

	Discoverable bool `json:"discoverable,omitempty"` // Present only when false

	Key     string `json:"key,omitempty"`      // Cloud key (only when ident=true)
	Batch   string `json:"batch,omitempty"`    // Provisioning batch (when ident=true)
	FWSBits string `json:"fw_sbits,omitempty"` // Internal flags (when ident=true)
}

// SystemStatusV2 represents the top-level status payload.
type SystemStatusV2 struct {
	MAC             string `json:"mac"`              // Mac address of the device
	RestartRequired bool   `json:"restart_required"` // True if restart is required
	Time            string `json:"time"`             // "HH:MM" in local TZ, nil when not synced
	UnixTime        int64  `json:"unixtime"`         // Unix timestamp (UTC), nil when not synced
	LastSyncTS      int64  `json:"last_sync_ts"`     // Last NTP sync time (UTC), nil when not synced
	Uptime          int64  `json:"uptime"`           // Seconds since last reboot
	RAMSize         int64  `json:"ram_size"`         // Total RAM bytes
	RAMFree         int64  `json:"ram_free"`         // Free RAM bytes
	FSSize          int64  `json:"fs_size"`          // Total filesystem size bytes
	FSFree          int64  `json:"fs_free"`          // Free filesystem size bytes
	CfgRev          int64  `json:"cfg_rev"`          // Configuration revision
	KVSRev          int64  `json:"kvs_rev"`          // KVS revision

	ScheduleRev int64 `json:"schedule_rev,omitempty"` // Schedules revision (if schedules enabled)
	WebhookRev  int64 `json:"webhook_rev,omitempty"`  // Webhooks revision (if webhooks enabled)
	KNXRev      int64 `json:"knx_rev,omitempty"`      // KNX config revision (if KNX enabled)
	BTRelayRev  int64 `json:"btrelay_rev,omitempty"`  // BLE cloud relay config revision
	BTHCRev     int64 `json:"bthc_rev,omitempty"`     // BTHomeControl config revision

	AvailableUpdates AvailableUpdatesV2 `json:"available_updates"` // Info about available updates

	WakeupReason *WakeupReasonV2 `json:"wakeup_reason,omitempty"` // Only for battery devices
	WakeupPeriod *int64          `json:"wakeup_period,omitempty"` // Seconds; keep-alive interval (battery devices)

	UTCOffset int64 `json:"utc_offset"` // Local time offset from UTC in seconds
}

// AvailableUpdatesV2 describes available firmware updates.
type AvailableUpdatesV2 struct {
	Beta   *UpdateInfoV2 `json:"beta,omitempty"`   // Present if beta update is available
	Stable *UpdateInfoV2 `json:"stable,omitempty"` // Present if stable update is available
}

// UpdateInfoV2 describes a single firmware update.
type UpdateInfoV2 struct {
	Version string `json:"version"` // Version of the new firmware
}

// WakeupReasonV2 contains info about boot type and cause (battery devices).
type WakeupReasonV2 struct {
	Boot  string `json:"boot"`  // poweron, software_restart, deepsleep_wake, internal, unknown
	Cause string `json:"cause"` // button, usb, periodic, status_update, alarm, alarm_test, undefined
}

// WifiScanResultV2 represents the top-level result for a WiFi scan.
type WifiScanResultV2 struct {
	Results []WifiNetworkV2 `json:"results"` // List of discovered networks
}

// WifiNetworkV2 represents a single WiFi network entry.
type WifiNetworkV2 struct {
	SSID    string `json:"ssid"`    // SSID or null for hidden network
	BSSID   string `json:"bssid"`   // BSSID of the network
	Auth    int    `json:"auth"`    // Authentication method (0–5)
	Channel int    `json:"channel"` // Network channel
	RSSI    int    `json:"rssi"`    // Signal strength in dBm
}

// SwitchStatusV2 represents the full status of a Switch component instance.
type SwitchStatusV2 struct {
	ID     int    `json:"id"`     // Id of the Switch component instance
	Source string `json:"source"` // Source of the last command
	Output bool   `json:"output"` // true if output is on

	TimerStartedAt int64 `json:"timer_started_at,omitempty"` // UTC timestamp (when timer triggered)
	TimerDuration  int64 `json:"timer_duration,omitempty"`   // Duration in seconds

	APower  float64 `json:"apower,omitempty"`  // Active power in Watts
	Voltage float64 `json:"voltage,omitempty"` // Voltage in Volts
	Current float64 `json:"current,omitempty"` // Current in Amperes
	PF      float64 `json:"pf,omitempty"`      // Power factor
	Freq    float64 `json:"freq,omitempty"`    // Frequency in Hz

	AEnergy    *ActiveEnergyV2   `json:"aenergy,omitempty"`     // Active energy counter
	RetAEnergy *ReturnedEnergyV2 `json:"ret_aenergy,omitempty"` // Returned (reverse-flow) energy

	Temperature *TemperatureStatusV2 `json:"temperature,omitempty"` // Temperature info

	Errors []string `json:"errors,omitempty"` // Error conditions
}

// ActiveEnergyV2 contains active energy counters.
type ActiveEnergyV2 struct {
	Total    float64   `json:"total"`               // Total energy (Wh)
	ByMinute []float64 `json:"by_minute,omitempty"` // mWh for last 3 complete minutes
	MinuteTS int64     `json:"minute_ts,omitempty"` // UTC timestamp of current minute start
}

// ReturnedEnergyV2 contains returned active energy counters.
type ReturnedEnergyV2 struct {
	Total    float64   `json:"total"`               // Total returned energy (Wh)
	ByMinute []float64 `json:"by_minute,omitempty"` // Returned mWh for last 3 complete minutes
	MinuteTS int64     `json:"minute_ts,omitempty"` // UTC timestamp of current minute start
}

// TemperatureStatusV2 contains temperature readings.
type TemperatureStatusV2 struct {
	TC float64 `json:"tC"` // Temperature in Celsius or null
	TF float64 `json:"tF"` // Temperature in Fahrenheit or null
}

type SwitchStateV2 struct {
	IsOn           bool    `json:"ison"`             // True if the switch is on
	HasTimer       bool    `json:"has_timer"`        // True if a timer is active
	TimerStartedAt int64   `json:"timer_started_at"` // UTC timestamp when timer started
	TimerDuration  float64 `json:"timer_duration"`   // Duration of the timer in seconds
	TimerRemaining float64 `json:"timer_remaining"`  // Seconds remaining until execution

	Overpower bool   `json:"overpower,omitempty"` // Present if overpower condition tracking is applicable
	Source    string `json:"source"`              // Source of the last command (init, WS_in, http, ...)
}

type WifiStatusV2 struct {
	StaIP         string `json:"sta_ip"`                    // IP address or null if disconnected
	Status        string `json:"status"`                    // disconnected, connecting, connected, got ip
	SSID          string `json:"ssid"`                      // SSID or null if disconnected
	BSSID         string `json:"bssid,omitempty"`           // BSSID of AP (only when connected)
	RSSI          int    `json:"rssi"`                      // Signal strength in dBm
	APClientCount int    `json:"ap_client_count,omitempty"` // Only present when AP is enabled with extender mode
}
type shellyPlugV2 struct {
	address *url.URL
}

func newShellyV2Plug(address url.URL) (Plug, error) {
	if address.Host == "" {
		return nil, fmt.Errorf("invalid address")
	}

	return &shellyPlugV2{
		address: &address,
	}, nil
}

func (s *shellyPlugV2) get(path string, queries map[string]string, response any) error {
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

	body := resp.Body()
	log.Printf("body: %s\n", string(body))
	err = json.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("invalid response body: %v", err)
	}

	return nil
}

func (s *shellyPlugV2) Status() (*DeviceStatusV2, error) {
	var res DeviceStatusV2

	err := s.get("rpc/Shelly.GetStatus", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *shellyPlugV2) Info() (*DeviceInfoV2, error) {
	var res DeviceInfoV2

	err := s.get("shelly", nil, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *shellyPlugV2) RenderInfo(w io.Writer) error {
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
	fmt.Fprintf(w, "         Device Type: %s\n", nfo.Model)
	fmt.Fprintf(w, "            Firmware: %s\n", nfo.Ver)
	fmt.Fprintf(w, "         MAC Address: %s\n", status.System.MAC)
	fmt.Fprintln(w)

	t := time.Unix(status.System.UnixTime, 0)

	fmt.Fprintln(w, "Device Status")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "                Time: %s\n", t)
	fmt.Fprintf(w, "              Uptime: %v\n", time.Duration(status.System.Uptime)*time.Second)
	fmt.Fprintf(w, "        Memory Total: %v\n", humanize.IBytes(uint64(status.System.RAMSize)))
	fmt.Fprintf(w, "         Memory Free: %v\n", humanize.IBytes(uint64(status.System.RAMFree)))
	fmt.Fprintf(w, "       Storage Total: %v\n", humanize.IBytes(uint64(status.System.FSSize)))
	fmt.Fprintf(w, "        Storage Free: %v\n", humanize.IBytes(uint64(status.System.FSFree)))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Network Information")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "          IP Address: %s\n", status.WiFi.StaIP)
	fmt.Fprintf(w, "           WiFi SSID: %s\n", status.WiFi.SSID)
	fmt.Fprintf(w, "       WiFi Strength: %d\n", status.WiFi.RSSI)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Updates Information")
	fmt.Fprintln(w)
	hasUpdates := status.System.AvailableUpdates.Stable != nil
	fmt.Fprintf(w, "          Has Update: %t\n", hasUpdates)
	if hasUpdates {
		fmt.Fprintf(w, "    Latest Available: %s\n", status.System.AvailableUpdates.Stable.Version)
	}

	if status.Switch1 != nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Relay Information")
		fmt.Fprintln(w)
		s := "On"
		if !status.Switch1.Output {
			s = "Off"
		}
		fmt.Fprintf(w, "        Power Status: %s\n", s)

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Meter Information")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "                   Power: %.2f Watt\n", status.Switch1.APower)
		fmt.Fprintf(w, "                 Voltage: %.2f V\n", status.Switch1.Voltage)
		fmt.Fprintf(w, "                 Current: %.2f Amp\n", status.Switch1.Current)
		fmt.Fprintf(w, "               Frequency: %.2f Hz\n", status.Switch1.Freq)
		fmt.Fprintf(w, "       Total Consumption: %.2f kWh\n", float64(status.Switch1.AEnergy.Total)/1000)
		fmt.Fprintf(w, "           Internal Temp: %.2f C\n", status.Switch1.Temperature.TC)
	}

	return nil

}

func (s *shellyPlugV2) RenderEnergy(w io.Writer) error {
	status, err := s.Status()
	if err != nil {
		return err
	}

	if status.Switch1 == nil {
		return fmt.Errorf("no switch information received")
	}
	if status.Switch1.AEnergy == nil {
		return fmt.Errorf("no meter information received")
	}

	isOn := float64(0)
	if status.Switch1.Output {
		isOn = 1
	}

	reading := map[string]any{
		"power_watt":       status.Switch1.APower,
		"power_volt":       status.Switch1.Voltage,
		"power_ampere":     status.Switch1.Current,
		"power_frequency":  status.Switch1.Freq,
		"power_total_kwh":  float64(status.Switch1.AEnergy.Total) / 1000,
		"internal_celsius": status.Switch1.Temperature.TC,
		"is_on":            isOn,
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
				"current_power_watt":      reading["power_watt"],
				"current_power_volt":      reading["power_volt"],
				"current_power_ampere":    reading["power_ampere"],
				"current_power_frequency": reading["power_frequency"],
				"internal_celsius":        reading["internal_celsius"],
				"total_energy_kwh":        reading["power_total_kwh"],
				"relay_on":                reading["is_on"],
			}}
		j, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(j))
	default:
		fmt.Fprintln(w, "Meter Information")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "              Powered On: %t\n", isOn == 1)
		fmt.Fprintf(w, "                   Power: %.2f Watt\n", status.Switch1.APower)
		fmt.Fprintf(w, "                 Voltage: %.2f V\n", status.Switch1.Voltage)
		fmt.Fprintf(w, "                 Current: %.2f Amp\n", status.Switch1.Current)
		fmt.Fprintf(w, "               Frequency: %.2f Hz\n", status.Switch1.Freq)
		fmt.Fprintf(w, "       Total Consumption: %.2f kWh\n", float64(status.Switch1.AEnergy.Total)/1000)
		fmt.Fprintf(w, "           Internal Temp: %.2f C\n", status.Switch1.Temperature.TC)

	}

	return nil
}

func (s *shellyPlugV2) TurnOn() (bool, error) {
	var res SwitchStateV2

	err := s.get("relay/0", map[string]string{"turn": "on"}, &res)
	if err != nil {
		return false, err
	}

	if !res.IsOn {
		return false, fmt.Errorf("relay is not on")
	}
	return res.IsOn, nil
}

func (s *shellyPlugV2) TurnOff() (bool, error) {
	var res SwitchStateV2

	err := s.get("relay/0", map[string]string{"turn": "off"}, &res)
	if err != nil {
		return false, err
	}

	if res.IsOn {
		return res.IsOn, fmt.Errorf("relay is on")
	}

	return res.IsOn, nil
}
