package hardware

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MockController simulates the hardware for local development.
type MockController struct {
	FileBrowser // Embeds GetNumOfTags, GetTagInfoByIndex, etc.

	mu          sync.Mutex
	isRecording bool

	// Configuration State
	recConfig  RecorderParameters
	wifiConfig WifiParameters
}

func NewController() Controller {
	// Setup a local folder for testing
	cwd, _ := os.Getwd()
	localTestPath := filepath.Join(cwd, "test_recordings")

	// Ensure the root exists
	_ = os.MkdirAll(localTestPath, 0755)

	return &MockController{
		FileBrowser: FileBrowser{
			RootPath: localTestPath,
		},
		recConfig: RecorderParameters{
			FPS:         30,
			Bitrate:     5000000,
			ChunkSecs:   300,
			FilenameTag: "",
		},
		// Default dummy wifi
		wifiConfig: WifiParameters{
			SSID:     "",
			Password: "",
		},
	}
}

// --- Lifecycle ---

func (m *MockController) Init() error {
	slog.Info("[MOCK] Hardware Initialized", "root_path", m.RootPath)
	return nil
}

func (m *MockController) Close() {
	slog.Info("[MOCK] Hardware Shutdown")
}

// --- Connectivity ---

func (m *MockController) SetupWifi(ssid, pwd string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.wifiConfig.SSID = ssid
	m.wifiConfig.Password = pwd

	slog.Info("[MOCK] Wifi Credentials Saved", "ssid", ssid)
	return nil
}

func (m *MockController) ConnectToWifi() error {
	slog.Info("[MOCK] Connecting to Wifi...")
	time.Sleep(500 * time.Millisecond) // Simulate delay

	m.mu.Lock()
	ssid := m.wifiConfig.SSID
	m.mu.Unlock()

	if ssid == "" {
		return fmt.Errorf("no wifi credentials configured")
	}

	slog.Info("[MOCK] Wifi Connected", "ssid", ssid)
	return nil
}

func (m *MockController) GetWifiDetails() (*WifiParameters, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid race conditions
	return &WifiParameters{
		SSID:     m.wifiConfig.SSID,
		Password: m.wifiConfig.Password,
	}, nil
}

// --- Battery & Storage ---

func (m *MockController) GetBatteryStatus() (*BatteryStatus, error) {
	return &BatteryStatus{
		Percentage:    88,
		IsCharging:    false,
		EstimatedMins: 145,
	}, nil
}

func (m *MockController) GetDiskStatus() (*DiskStatus, error) {
	return &DiskStatus{
		TotalMB: 64000,
		UsedMB:  12500,
		FreeMB:  51500,
	}, nil
}

// --- Recorder Controls ---

func (m *MockController) SetupRecorder(params RecorderParameters) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recConfig = params
	slog.Info("[MOCK] Recorder Configured",
		"fps", params.FPS,
		"bitrate", params.Bitrate,
		"chunk_secs", params.ChunkSecs)
	return nil
}

func (m *MockController) StartRecorder(folderTag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRecording {
		return fmt.Errorf("already recording")
	}

	m.isRecording = true
	m.recConfig.FilenameTag = folderTag

	// Create physical folder
	fullPath := filepath.Join(m.RootPath, folderTag)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return err
	}

	slog.Info("[MOCK] Recording STARTED", "tag", folderTag)
	return nil
}

func (m *MockController) StopRecorder() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRecording {
		return fmt.Errorf("not recording")
	}

	tag := m.recConfig.FilenameTag
	timestamp := time.Now().Format("150405")
	baseName := fmt.Sprintf("vid_%s", timestamp)
	folderPath := filepath.Join(m.RootPath, tag)

	// 1. Create Dummy Video
	videoPath := filepath.Join(folderPath, baseName+".mp4")
	if err := os.WriteFile(videoPath, []byte("mock-header"), 0644); err != nil {
		return err
	}

	// Sparse file trick for realistic size
	fakeSize := int64(rand.Intn(100)+20) * 1024 * 1024
	_ = os.Truncate(videoPath, fakeSize)

	// 2. Create Dummy IMU
	imuPath := filepath.Join(folderPath, baseName+".imu")
	_ = os.WriteFile(imuPath, []byte("ts,x,y,z\n"), 0644)

	// 3. Create Dummy Thumbnail
	thumbPath := filepath.Join(folderPath, baseName+".jpg")
	_ = os.WriteFile(thumbPath, []byte("fake-jpg"), 0644)

	m.isRecording = false
	m.recConfig.FilenameTag = ""

	slog.Info("[MOCK] Recording STOPPED", "file", videoPath)
	return nil
}

func (m *MockController) GetRecorderInfo() (*RecorderParameters, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.recConfig
	return &c, nil
}
