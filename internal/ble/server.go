package ble

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"blueowl-ble/internal/hardware"

	"tinygo.org/x/bluetooth"
)

// --- UUID Definitions ---
var (
	// Standard Services
	ServiceBattery   = bluetooth.ServiceUUIDBattery
	CharBatteryLevel = bluetooth.CharacteristicUUIDBatteryLevel

	ServiceDeviceInfo = bluetooth.ServiceUUIDDeviceInformation
	CharManufacturer  = bluetooth.CharacteristicUUIDManufacturerNameString
	CharModel         = bluetooth.CharacteristicUUIDModelNumberString
	CharSerialNumber  = bluetooth.CharacteristicUUIDSerialNumberString

	// Custom Owl Service (Base: A0B4xxxx-926D-4D61-98DF-8C5C62EE53B3)
	ServiceOwlUUID = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x00, 0x92, 0x6D, 0x4D, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})

	// Characteristics
	// 01: Recorder Status (Read/Notify) - Was SystemStatus
	CharRecStatus = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x01, 0x92, 0x6D, 0x4D, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
	// 02: Recorder Control (Write)
	CharRecControl = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x02, 0x92, 0x6D, 0x4d, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
	// 03: Wifi Setup (Write Only)
	CharWifiSetup = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x03, 0x92, 0x6D, 0x4d, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
	// 04: File Browser (Write / Indicate)
	CharBrowser = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x04, 0x92, 0x6D, 0x4d, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
	// 05: Wifi Status (Read/Notify) - NEW
	CharWifiStatus = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x05, 0x92, 0x6D, 0x4d, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
	// 06: Disk Status (Read/Notify) - NEW
	CharDiskStatus = bluetooth.NewUUID([16]byte{0xA0, 0xB4, 0x00, 0x06, 0x92, 0x6D, 0x4d, 0x61, 0x98, 0xDF, 0x8C, 0x5C, 0x62, 0xEE, 0x53, 0xB3})
)

type Server struct {
	Adapter *bluetooth.Adapter
	HW      hardware.Controller

	// Handles
	battHandle      bluetooth.Characteristic
	recStatusHandle bluetooth.Characteristic // Replaces statusHandle
	browserHandle   bluetooth.Characteristic

	// New Status Handles
	wifiStatusHandle bluetooth.Characteristic
	diskStatusHandle bluetooth.Characteristic
}

func NewServer(hw hardware.Controller) *Server {
	return &Server{
		Adapter: bluetooth.DefaultAdapter,
		HW:      hw,
	}
}

func (s *Server) Start() error {
	if err := s.Adapter.Enable(); err != nil {
		return err
	}

	slog.Info("[BLE] Adapter Enabled. Configuring Services...")

	s.addBatteryService()
	s.addDeviceInfoService()
	if err := s.addOwlService(); err != nil {
		return err
	}

	adv := s.Adapter.DefaultAdvertisement()
	err := adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "BlueOWL",
		ServiceUUIDs: []bluetooth.UUID{ServiceOwlUUID, ServiceBattery},
	})
	if err != nil {
		return err
	}

	slog.Info("[BLE] Server Started, Advertising...")
	return adv.Start()
}

func (s *Server) addDeviceInfoService() {
	serialNum := getSerialNumber()
	slog.Info("[BLE] Device Info Configured", "serial", serialNum)

	_ = s.Adapter.AddService(&bluetooth.Service{
		UUID: ServiceDeviceInfo,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				UUID:  CharManufacturer,
				Value: []byte("Augmodo Inc"),
				Flags: bluetooth.CharacteristicReadPermission,
			},
			{
				UUID:  CharModel,
				Value: []byte("BlueOWL v0.1"),
				Flags: bluetooth.CharacteristicReadPermission,
			},
			{
				UUID:  CharSerialNumber,
				Value: []byte(serialNum),
				Flags: bluetooth.CharacteristicReadPermission,
			},
		},
	})
}

func (s *Server) addBatteryService() {
	_ = s.Adapter.AddService(&bluetooth.Service{
		UUID: ServiceBattery,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				UUID:   CharBatteryLevel,
				Value:  []byte{0},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
				Handle: &s.battHandle,
			},
		},
	})

	// Background ticker for periodic updates
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			// Battery
			if status, err := s.HW.GetBatteryStatus(); err == nil {
				s.battHandle.Write([]byte{status.Percentage})
			}
			// Update Disk & Wifi status periodically as well
			s.notifyDiskStatus()
			s.notifyWifiStatus()
		}
	}()
}

func (s *Server) addOwlService() error {
	return s.Adapter.AddService(&bluetooth.Service{
		UUID: ServiceOwlUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			// 1. Recorder Status
			{
				UUID:   CharRecStatus,
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
				Handle: &s.recStatusHandle,
			},
			// 2. Recorder Control
			{
				UUID:       CharRecControl,
				Flags:      bluetooth.CharacteristicWritePermission,
				WriteEvent: s.handleRecorderCommand,
			},
			// 3. Wifi Setup
			{
				UUID:       CharWifiSetup,
				Flags:      bluetooth.CharacteristicWritePermission,
				WriteEvent: s.handleWifiSetup,
			},
			// 4. File Browser
			{
				UUID:       CharBrowser,
				Flags:      bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicIndicatePermission,
				Handle:     &s.browserHandle,
				WriteEvent: s.handleBrowserRequest,
			},
			// 5. Wifi Status (New)
			{
				UUID:   CharWifiStatus,
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
				Handle: &s.wifiStatusHandle,
			},
			// 6. Disk Status (New)
			{
				UUID:   CharDiskStatus,
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
				Handle: &s.diskStatusHandle,
			},
		},
	})
}

// --- Handlers ---

type RecCmd struct {
	Action string                      `json:"action"`
	Tag    string                      `json:"tag,omitempty"`
	Config hardware.RecorderParameters `json:"config,omitempty"`
}

func (s *Server) handleRecorderCommand(client bluetooth.Connection, offset int, value []byte) {
	if offset != 0 {
		return
	}

	var cmd RecCmd
	if err := json.Unmarshal(value, &cmd); err != nil {
		slog.Error("[BLE] Invalid JSON in RecControl", "err", err)
		return
	}

	switch cmd.Action {
	case "start":
		if cmd.Tag == "" {
			cmd.Tag = "Default"
		}
		s.HW.StartRecorder(cmd.Tag)
	case "stop":
		s.HW.StopRecorder()
	case "config":
		s.HW.SetupRecorder(cmd.Config)
	}

	// Update recorder status immediately
	s.notifyRecStatus()
}

func (s *Server) handleWifiSetup(client bluetooth.Connection, offset int, value []byte) {
	var creds hardware.WifiParameters
	if err := json.Unmarshal(value, &creds); err != nil {
		slog.Error("[BLE] Invalid JSON in WifiSetup")
		return
	}
	slog.Info("[BLE] Received Wifi Config", "ssid", creds.SSID)
	s.HW.SetupWifi(creds.SSID, creds.Password)

	go func() {
		s.HW.ConnectToWifi()
		s.notifyWifiStatus() // Update status to show we are connected/connecting
	}()
}

type BrowserRequest struct {
	Type      string `json:"type"`
	TagIndex  uint32 `json:"tag_index"`
	FileIndex uint32 `json:"file_index"`
}

func (s *Server) handleBrowserRequest(client bluetooth.Connection, offset int, value []byte) {
	var req BrowserRequest
	if err := json.Unmarshal(value, &req); err != nil {
		slog.Error("[BLE] Invalid Browser Request")
		return
	}

	go func() {
		switch req.Type {
		case "tags":
			count, _ := s.HW.GetNumOfTags()
			for i := uint32(0); i < count; i++ {
				tag, _ := s.HW.GetTagInfoByIndex(i)
				data, _ := json.Marshal(tag)
				s.browserHandle.Write(data)
				time.Sleep(50 * time.Millisecond)
			}

		case "files":
			tagInfo, _ := s.HW.GetTagInfoByIndex(req.TagIndex)
			if tagInfo != nil {
				for i := uint32(0); i < tagInfo.NumOfRecordings; i++ {
					file, _ := s.HW.GetRecordingDetails(tagInfo.Name, i)
					data, _ := json.Marshal(file)
					s.browserHandle.Write(data)
					time.Sleep(50 * time.Millisecond)
				}
			}

		default:
			slog.Warn("[BLE] Unknown browser request type", "type", req.Type)
			s.browserHandle.Write([]byte(`{"error": "unknown_type"}`))
		}

		eos := []byte("{}")
		s.browserHandle.Write(eos)
	}()
}

// --- Helpers ---

// Split Payloads
type RecStatusPayload struct {
	IsRecording bool   `json:"is_recording"`
	Tag         string `json:"tag"`
	FPS         uint8  `json:"fps"`
	Bitrate     uint32 `json:"bitrate"`
}

type WifiStatusPayload struct {
	SSID      string `json:"ssid"`
	Connected bool   `json:"connected"`
}

func (s *Server) notifyRecStatus() {
	info, err := s.HW.GetRecorderInfo()
	if err != nil {
		return
	}

	isRec := info.FilenameTag != ""
	payload := RecStatusPayload{
		IsRecording: isRec,
		Tag:         info.FilenameTag,
		FPS:         info.FPS,
		Bitrate:     info.Bitrate,
	}

	if data, err := json.Marshal(payload); err == nil {
		s.recStatusHandle.Write(data)
	}
}

func (s *Server) notifyWifiStatus() {
	params, _ := s.HW.GetWifiDetails()
	connected := params.SSID != "" // Simple check for now

	payload := WifiStatusPayload{
		SSID:      params.SSID,
		Connected: connected,
	}
	if data, err := json.Marshal(payload); err == nil {
		s.wifiStatusHandle.Write(data)
	}
}

func (s *Server) notifyDiskStatus() {
	disk, err := s.HW.GetDiskStatus()
	if err != nil {
		return
	}

	if data, err := json.Marshal(disk); err == nil {
		s.diskStatusHandle.Write(data)
	}
}

func getSerialNumber() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "OWL-DEV-SIMULATOR"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Serial") {
			fields := strings.Split(line, ":")
			if len(fields) > 1 {
				return strings.TrimSpace(fields[1])
			}
		}
	}
	return "OWL-UNKNOWN-ID"
}
