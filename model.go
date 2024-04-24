package main

type Plug interface {
	TurnOn() (*Relay, error)
	TurnOff() (*Relay, error)
	Status() (*DeviceStatus, error)
	Info() (*DeviceInfo, error)
}

// DeviceStatus aggregates all status information for the device. Returned from the /status API
type DeviceStatus struct {
	WiFi     WiFiStatus       `json:"wifi_sta" yaml:"wifi_sta"`
	Cloud    CloudStatus      `json:"cloud" yaml:"cloud"`
	MQTT     MQTTStatus       `json:"mqtt" yaml:"mqtt"`
	Time     string           `json:"time" yaml:"time"`         // Current hour and minutes, HH:MM format
	Unixtime int64            `json:"unixtime" yaml:"unixtime"` // Unix timestamp if synced; 0 otherwise
	Serial   int64            `json:"serial" yaml:"serial"`     // Cloud serial number
	MAC      string           `json:"mac" yaml:"mac"`           // MAC address of the device
	Update   UpdateStatus     `json:"update" yaml:"update"`
	FS       FileSystemStatus `json:"fs" yaml:"fs"`
	Uptime   int64            `json:"uptime" yaml:"uptime"`       // Seconds elapsed since boot
	Relays   []Relay          `json:"relays" yaml:"relays"`       // Array of relay statuses
	Meters   []Meter          `json:"meters" yaml:"meters"`       // Array of meter statuses
	RamTotal int64            `json:"ram_total" yaml:"ram_total"` // Total amount of system memory in bytes
	RamFree  int64            `json:"ram_free" yaml:"ram_free"`   // Available amount of system memory in bytes
	FsSize   int64            `json:"fs_size" yaml:"fs_size"`     // Total amount of the file system in bytes
	FsFree   int64            `json:"fs_free" yaml:"fs_free"`     // Available amount of the file system in bytes

}

// DeviceInfo is the response from the /shelly API
type DeviceInfo struct {
	Type   string `json:"type" yaml:"type"`     // Shelly model identifier
	MAC    string `json:"mac" yaml:"mac"`       // MAC address of the device
	Auth   bool   `json:"auth" yaml:"auth"`     // Whether HTTP requests require authentication
	FW     string `json:"fw" yaml:"fw"`         // Current firmware version
	LongID int    `json:"longid" yaml:"longid"` // 1 if the device identifies itself with its full MAC address; 0 if only the last 3 bytes are used
}

// WiFiStatus represents the current status of the WiFi connection.
type WiFiStatus struct {
	Connected bool   `json:"connected" yaml:"connected"` // Status of WiFi connection
	SSID      string `json:"ssid" yaml:"ssid"`           // WiFi SSID
	IP        string `json:"ip" yaml:"ip"`               // IP address assigned by the WiFi router
	RSSI      int    `json:"rssi" yaml:"rssi"`           // Signal strength indicator
}

// CloudStatus represents the current cloud connection status.
type CloudStatus struct {
	Enabled   bool `json:"enabled" yaml:"enabled"`     // Whether cloud functionality is enabled
	Connected bool `json:"connected" yaml:"connected"` // Current cloud connection status
}

// MQTTStatus represents the MQTT connection status when MQTT is enabled.
type MQTTStatus struct {
	Connected bool `json:"connected" yaml:"connected"` // MQTT connection status
}

// UpdateStatus contains information about firmware updates.
type UpdateStatus struct {
	Status     string `json:"status" yaml:"status"`           // Current status of firmware update
	HasUpdate  bool   `json:"has_update" yaml:"has_update"`   // Whether a new firmware version is available
	NewVersion string `json:"new_version" yaml:"new_version"` // New firmware version
	OldVersion string `json:"old_version" yaml:"old_version"` // Old firmware version
}

// FileSystemStatus contains information about the file system storage.
type FileSystemStatus struct {
	Size int64 `json:"size" yaml:"size"` // Total amount of the file system in bytes
	Free int64 `json:"free" yaml:"free"` // Available amount of the file system in bytes
}

// Relay represents the current state of each relay output channel.
type Relay struct {
	IsOn           bool   `json:"ison" yaml:"ison"`                       // Indicates if the relay is on
	HasTimer       bool   `json:"has_timer" yaml:"has_timer"`             // Indicates if a timer is set
	TimerStarted   int64  `json:"timer_started" yaml:"timer_started"`     // Timestamp when the timer was started
	TimerDuration  int64  `json:"timer_duration" yaml:"timer_duration"`   // Duration of the timer
	TimerRemaining int64  `json:"timer_remaining" yaml:"timer_remaining"` // Time remaining on the timer
	Overpower      bool   `json:"overpower" yaml:"overpower"`             // Indicates if the overpower protection is triggered
	Source         string `json:"source" yaml:"source"`                   // Source that caused the last state change
}

// Meter represents the current status of each power meter.
type Meter struct {
	Power     float64   `json:"power" yaml:"power"`         // Current power usage
	Overpower float64   `json:"overpower" yaml:"overpower"` // Overpower value threshold
	IsValid   bool      `json:"is_valid" yaml:"is_valid"`   // Validity of the meter reading
	Timestamp int64     `json:"timestamp" yaml:"timestamp"` // Timestamp of the meter reading
	Counters  []float64 `json:"counters" yaml:"counters"`   // Counters array with meter readings
	Total     int64     `json:"total" yaml:"total"`         // Total consumption
}
