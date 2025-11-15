package main

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/choria-io/fisk"
)

var (
	ip           net.IP
	user         string
	pass         string
	jsonFormat   bool
	choriaFormat bool
	labels       map[string]string
	v2Plus       bool
)

func main() {
	app := fisk.New("shellyctl", "Controls Shell Plug / Plug S Smart Plugs")
	app.Flag("v2", "Enable support for Shelly V2+ devices").BoolVar(&v2Plus)
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

func newPlug() (Plug, error) {
	if v2Plus {
		return nil, fmt.Errorf("shelly V2+ devices are not supported yet")
	} else {
		return newShellyV1Plug(deviceUrl())
	}
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
	plug, err := newPlug()
	if err != nil {
		return err
	}

	return plug.RenderEnergy(os.Stdout)
}

func infoAction(_ *fisk.ParseContext) error {
	plug, err := newPlug()
	if err != nil {
		return err
	}

	return plug.RenderInfo(os.Stdout)
}

func onAction(_ *fisk.ParseContext) error {
	plug, err := newPlug()
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
	plug, err := newPlug()
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
