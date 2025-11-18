## Control for Shelly Plug and Plug S

A small client utility for interacting with the Shelly Plugs.

Tested on a Raspberry PI with Go 1.24

## Usage?

Basic usage information:

```nohighlight
$ shellyctl --help
usage: shellyctl [<flags>] <command> [<args> ...]

Controls Shell Plug / Plug S Smart Plugs

Commands:
  on      Turns the device on
  off     Turns the device off
  info    Shows device information
  energy  Retrieves device energy usage statistics

Global Flags:
      --help               Show context-sensitive help
  -A, --address=ADDRESS    Device IP address ($ADDRESS)
  -U, --username=USERNAME  Device username ($USERNAME)
  -P, --password=PASSWORD  Device password ($PASSWORD)
```

Obtain device info:

```nohighlight
$ shellyctl -A 192.168.1.1 info
Shelly device information for 192.168.1.1

Device Information

         Device Type: SHPLG2-1
            Firmware: 20230913-113610/v1.14.0-gcb84623
         MAC Address: 08F9E04E5CD3

Device Status

                Time: 2024-04-24 21:45:41 +0200 CEST
              Uptime: 6h15m42s
         Memory Used: 51 KiB
         Memory Free: 38 KiB
       Storage Total: 228 KiB
        Storage Free: 162 KiB

Network Information

          IP Address: 192.168.1.1
           WiFi SSID: example
       WiFi Strength: -59
       Cloud Enabled: true
     Cloud Connected: true
      MQTT Connected: false

Updates Information

          Has Update: false
    Latest Available: 20230913-113610/v1.14.0-gcb84623

Relay Information

        Power Status: On
               Timer: false

Meter Information

               Power: 2.42 Watt
   Total Consumption: 0.01 kWh
```

Read energy values:

```nohighlight
$ shellyctl -A 192.168.1.1 energy
Meter Information

               Power: 2.45 Watt
   Total Consumption: 0.01 kWh
```

It supports JSON output:

```nohighlight
$ shellyctl -A 192.168.1.10 energy --json
{
  "power_total_kwh": 0.010216666666666667,
  "power_watt": 2.43
}
```

And also the format required by Choria Metric watchers:

```
$ shellyctl -A 192.168.1.10 energy --choria --label location:office
{
  "labels": {
    "location": "office"
  },
  "metrics": {
    "current_power_watt": 2.42,
    "today_energy_kwh": 0.0103
  }
}
```

## Contact?

R.I. Pienaar / rip@devco.net / [devco.net](https://www.devco.net/)