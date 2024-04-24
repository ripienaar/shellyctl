package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/choria-io/fisk"
	"github.com/dustin/go-humanize"
)

var (
	ip           net.IP
	user         string
	pass         string
	jsonFormat   bool
	choriaFormat bool
	labels       map[string]string
)

func main() {
	app := fisk.New("shellyctl", "Controls Shell Plug / Plug S Smart Plugs")

	labels = make(map[string]string)

	app.Flag("address", "Device IP address").Short('A').Envar("ADDRESS").Required().IPVar(&ip)
	app.Flag("username", "Device username").Short('U').Envar("USERNAME").StringVar(&user)
	app.Flag("password", "Device password").Short('P').Envar("PASSWORD").StringVar(&pass)

	app.Command("on", "Turns the device on").Action(onAction)
	app.Command("off", "Turns the device off").Action(offAction)

	info := app.Command("info", "Shows device information").Action(infoAction)
	info.Flag("json", "Produce JSON output").UnNegatableBoolVar(&jsonFormat)

	energy := app.Command("energy", "Retrieves device energy usage statistics").Action(energyAction)
	energy.Flag("json", "Produce JSON output").UnNegatableBoolVar(&jsonFormat)
	energy.Flag("choria", "Produce Choria Metric output").UnNegatableBoolVar(&choriaFormat)
	energy.Flag("label", "Labels to apply to Choria Metric output").StringMapVar(&labels)

	app.MustParseWithUsage(os.Args[1:])
}

func deviceUrl() url.URL {
	var usr *url.Userinfo
	if user != "" && pass != "" {
		usr = url.UserPassword(user, pass)
	}
	return url.URL{
		Scheme: "http",
		Host:   ip.String(),
		User:   usr,
	}
}

func energyAction(_ *fisk.ParseContext) error {
	plug, err := NewShellyPlug(deviceUrl())
	if err != nil {
		return err
	}

	nfo, err := plug.Status()
	if err != nil {
		return err
	}

	if len(nfo.Meters) != 1 {
		return fmt.Errorf("no meter information received")
	}

	m := nfo.Meters[0]

	reading := map[string]any{
		"power_watt":      m.Power,
		"power_total_kwh": float64(m.Total) * 0.000016666666666666667,
	}

	switch {
	case jsonFormat:
		j, err := json.MarshalIndent(reading, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(j))

	case choriaFormat:
		data := map[string]any{
			"labels": labels,
			"metrics": map[string]any{
				"current_power_watt": reading["power_watt"],
				"today_energy_kwh":   reading["power_total_kwh"],
			}}
		j, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(j))

	default:
		fmt.Println("Meter Information")
		fmt.Println()
		fmt.Printf("               Power: %.2f Watt\n", m.Power)
		fmt.Printf("   Total Consumption: %.2f kWh\n", float64(m.Total)*0.000016666666666666667)
	}

	return nil
}

func infoAction(_ *fisk.ParseContext) error {
	plug, err := NewShellyPlug(deviceUrl())
	if err != nil {
		return err
	}

	nfo, err := plug.Info()
	if err != nil {
		return err
	}
	status, err := plug.Status()
	if err != nil {
		return err
	}

	fmt.Printf("Shelly device information for %s\n", ip.String())
	fmt.Println()
	fmt.Println("Device Information")
	fmt.Println()
	fmt.Printf("         Device Type: %s\n", nfo.Type)
	fmt.Printf("            Firmware: %s\n", nfo.FW)
	fmt.Printf("         MAC Address: %s\n", status.MAC)
	fmt.Println()

	t := time.Unix(status.Unixtime, 0)

	fmt.Println("Device Status")
	fmt.Println()
	fmt.Printf("                Time: %s\n", t)
	fmt.Printf("              Uptime: %v\n", time.Duration(status.Uptime)*time.Second)
	fmt.Printf("         Memory Used: %v\n", humanize.IBytes(uint64(status.RamTotal)))
	fmt.Printf("         Memory Free: %v\n", humanize.IBytes(uint64(status.RamFree)))
	fmt.Printf("       Storage Total: %v\n", humanize.IBytes(uint64(status.FsSize)))
	fmt.Printf("        Storage Free: %v\n", humanize.IBytes(uint64(status.FsFree)))

	fmt.Println()
	fmt.Println("Network Information")
	fmt.Println()
	fmt.Printf("          IP Address: %s\n", status.WiFi.IP)
	fmt.Printf("           WiFi SSID: %s\n", status.WiFi.SSID)
	fmt.Printf("       WiFi Strength: %d\n", status.WiFi.RSSI)
	fmt.Printf("       Cloud Enabled: %t\n", status.Cloud.Enabled)
	if status.Cloud.Enabled {
		fmt.Printf("     Cloud Connected: %t\n", status.Cloud.Connected)
	}
	fmt.Printf("      MQTT Connected: %t\n", status.MQTT.Connected)

	fmt.Println()
	fmt.Println("Updates Information")
	fmt.Println()
	fmt.Printf("          Has Update: %t\n", status.Update.HasUpdate)
	fmt.Printf("    Latest Available: %s\n", status.Update.NewVersion)

	if len(status.Relays) == 1 {
		fmt.Println()
		fmt.Println("Relay Information")
		fmt.Println()
		s := "On"
		if !status.Relays[0].IsOn {
			s = "Off"
		}
		fmt.Printf("        Power Status: %s\n", s)
		fmt.Printf("               Timer: %t\n", status.Relays[0].HasTimer)
		if status.Relays[0].HasTimer {
			t := time.Unix(status.Relays[0].TimerStarted, 0)
			fmt.Printf("             Started: %v\n", t)
			fmt.Printf("            Duration: %v\n", time.Duration(status.Relays[0].TimerDuration)*time.Second)
			fmt.Printf("           Remaining: %v\n", time.Duration(status.Relays[0].TimerRemaining)*time.Second)
		}
	}

	if len(status.Meters) == 1 {
		fmt.Println()
		fmt.Println("Meter Information")
		fmt.Println()
		fmt.Printf("               Power: %.2f Watt\n", status.Meters[0].Power)
		fmt.Printf("   Total Consumption: %.2f kWh\n", float64(status.Meters[0].Total)*0.000016666666666666667)
	}

	return nil
}

func onAction(_ *fisk.ParseContext) error {
	plug, err := NewShellyPlug(deviceUrl())
	if err != nil {
		return err
	}

	_, err = plug.TurnOn()
	if err != nil {
		return err
	}

	fmt.Println("Device turned on")

	return nil
}

func offAction(_ *fisk.ParseContext) error {
	plug, err := NewShellyPlug(deviceUrl())
	if err != nil {
		return err
	}

	_, err = plug.TurnOff()
	if err != nil {
		return err
	}

	fmt.Println("Device turned off")

	return nil
}
