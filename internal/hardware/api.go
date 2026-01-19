package hardware

type Controller interface {
	// Lifecycle
	Init() error
	Close()

	// Wifi Connectivity
	SetupWifi(ssid, pwd string) error
	ConnectToWifi() error
	GetWifiDetails() (*WifiParameters, error)

	// Battery and Storage
	GetBatteryStatus() (*BatteryStatus, error)
	GetDiskStatus() (*DiskStatus, error)

	// Camera controls
	// StartRecorder creates a new videos inside the specified 'folderTag'.
	// e.g. StartRecorder("BestBuyDublin") --> /mnt/sdcard/BestBuyDublin/video_001.mp4
	StartRecorder(folderTag string) error
	StopRecorder() error
	SetupRecorder(params RecorderParameters) error
	GetRecorderInfo() (*RecorderParameters, error)

	// Recording filesystem browser
	// 1. Top Level: Returns how many folders/tags do we have
	GetNumOfTags() (uint32, error)

	// 2. Discovery: Get tag name & file count by global index
	GetTagInfoByIndex(idx uint32) (*TagInfo, error)

	// 3. Drill down: Get specific file details
	// "fileindex" is a 0-based index within that specific tag/folder.
	// Returns the file metadata.
	GetRecordingDetails(tag string, fileIndex uint32) (*RecordingFileInfo, error)
}

type WifiParameters struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type BatteryStatus struct {
	Percentage    uint8  `json:"percentage"`
	IsCharging    bool   `json:"is_charging"`
	EstimatedMins uint16 `json:"estimated_mins"`
}

type DiskStatus struct {
	TotalMB uint32 `json:"total_mb"`
	UsedMB  uint32 `json:"used_mb"`
	FreeMB  uint32 `json:"free_mb"`
}

type RecorderParameters struct {
	FPS         uint8  `json:"fps"`
	Bitrate     uint32 `json:"bitrate"`
	ChunkSecs   uint16 `json:"chunk_secs"`
	FilenameTag string `json:"filename_tag"`
}

type TagInfo struct {
	Name            string `json:"name"`
	NumOfRecordings uint32 `json:"num_recordings"`
}

type RecordingFileInfo struct {
	ID            uint16 `json:"id"`
	FileName      string `json:"filename"`
	Path          string `json:"path"`
	SizeMB        uint32 `json:"size_mb"`
	IMUFilePath   string `json:"imu_filepath"`
	ThumbnailPath string `json:"thumbnail_path"`
}
