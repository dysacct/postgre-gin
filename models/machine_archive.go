package models

import "time"

const (
	MachineStatusActive   = "active"
	MachineStatusStale    = "stale"
	MachineStatusArchived = "archived"

	ArchiveStatusArchived = "archived"
	ArchiveStatusRestored = "restored"
)

type MachineSyncState struct {
	IPMIIP             string     `gorm:"column:ipmi_ip;primaryKey;size:16" json:"ipmi_ip"`
	MachineID          string     `gorm:"column:machine_id;size:36;index" json:"machine_id"`
	ZbxID              string     `gorm:"column:zbx_id;size:50;index" json:"zbx_id"`
	Status             string     `gorm:"column:status;not null;size:20;index" json:"status"`
	LastSeenAt         time.Time  `gorm:"column:last_seen_at;not null;index" json:"last_seen_at"`
	FirstStaleAt       *time.Time `gorm:"column:first_stale_at;index" json:"first_stale_at,omitempty"`
	ArchivedAt         *time.Time `gorm:"column:archived_at;index" json:"archived_at,omitempty"`
	LastArchiveBatchID string     `gorm:"column:last_archive_batch_id;size:80;index" json:"last_archive_batch_id,omitempty"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MachineSyncState) TableName() string {
	return "machine_sync_state"
}

type MachineArchiveBatch struct {
	ArchiveBatchID string     `gorm:"column:archive_batch_id;primaryKey;size:80" json:"archive_batch_id"`
	MachineID      string     `gorm:"column:machine_id;not null;size:36;index" json:"machine_id"`
	IPMIIP         string     `gorm:"column:ipmi_ip;not null;size:16;index" json:"ipmi_ip"`
	ZbxID          string     `gorm:"column:zbx_id;size:50;index" json:"zbx_id"`
	IDCCode        string     `gorm:"column:idc_code;size:10;index" json:"idc_code"`
	IDCName        string     `gorm:"column:idc_name;size:50" json:"idc_name"`
	SSHIP          string     `gorm:"column:ssh_ip;size:16" json:"ssh_ip"`
	ArchiveReason  string     `gorm:"column:archive_reason;not null;size:40;index" json:"archive_reason"`
	Status         string     `gorm:"column:status;not null;size:20;index" json:"status"`
	LastSeenAt     *time.Time `gorm:"column:last_seen_at;index" json:"last_seen_at,omitempty"`
	ArchivedAt     time.Time  `gorm:"column:archived_at;not null;index" json:"archived_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at;not null;index" json:"expires_at"`
	RestoredAt     *time.Time `gorm:"column:restored_at;index" json:"restored_at,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MachineArchiveBatch) TableName() string {
	return "machine_archive_batches"
}

type ArchiveMeta struct {
	ArchiveID      uint       `gorm:"primaryKey" json:"archive_id"`
	OriginalID     uint       `gorm:"column:original_id;index" json:"original_id"`
	ArchiveBatchID string     `gorm:"column:archive_batch_id;not null;size:80;index" json:"archive_batch_id"`
	ArchiveReason  string     `gorm:"column:archive_reason;not null;size:40;index" json:"archive_reason"`
	ArchivedAt     time.Time  `gorm:"column:archived_at;not null;index" json:"archived_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at;not null;index" json:"expires_at"`
	IsRestored     bool       `gorm:"column:is_restored;not null;default:false;index" json:"is_restored"`
	RestoredAt     *time.Time `gorm:"column:restored_at;index" json:"restored_at,omitempty"`
}

type ArchivedIDCInfo struct {
	ArchiveMeta
	MachineID string    `gorm:"column:machine_id;not null;size:36;index" json:"machine_id"`
	ZbxID     string    `gorm:"column:zbx_id;not null;size:50;index" json:"zbx_id"`
	IPMIIP    string    `gorm:"column:ipmi_ip;not null;size:16;index" json:"ipmi_ip"`
	IDCCode   string    `gorm:"column:idc_code;not null;size:10;index" json:"idc_code"`
	IDCName   string    `gorm:"column:idc_name;not null;size:50" json:"idc_name"`
	SSHIP     string    `gorm:"column:ssh_ip;not null;size:16" json:"ssh_ip"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ArchivedIDCInfo) TableName() string {
	return "archived_idc_info"
}

type ArchivedMachineInfo struct {
	ArchiveMeta
	MachineID    string    `gorm:"column:machine_id;not null;size:36;index" json:"machine_id"`
	ZbxID        string    `gorm:"column:zbx_id;not null;size:50;index" json:"zbx_id"`
	IPMIIP       string    `gorm:"column:ipmi_ip;not null;size:16;index" json:"ipmi_ip"`
	SystemType   string    `gorm:"column:system_type;not null;size:100" json:"system_type"`
	Manufacturer string    `gorm:"column:manufacturer;not null;size:100;index" json:"manufacturer"`
	ServerSN     string    `gorm:"column:server_sn;not null;size:100;index" json:"server_sn"`
	SystemDisk   string    `gorm:"column:system_disk;not null;size:255" json:"system_disk"`
	SSDCount     string    `gorm:"column:ssd_count;not null;size:255" json:"ssd_count"`
	HDDCount     string    `gorm:"column:hdd_count;not null;size:255" json:"hdd_count"`
	SysHDDCount  string    `gorm:"column:sys_hdd_count;not null;default:'';size:255" json:"sys_hdd_count"`
	MemoryCount  string    `gorm:"column:memory_count;not null;size:255" json:"memory_count"`
	CPUInfo      string    `gorm:"column:cpu_info;type:text;not null" json:"cpu_info"`
	ServerHeight string    `gorm:"column:server_height;not null;size:50" json:"server_height"`
	SwitchPort   string    `gorm:"column:switch_port;not null;default:'';size:100" json:"switch_port"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ArchivedMachineInfo) TableName() string {
	return "archived_machine_info"
}

type ArchivedBusinessInfo struct {
	ArchiveMeta
	MachineID        string    `gorm:"column:machine_id;not null;size:36;index" json:"machine_id"`
	ZbxID            string    `gorm:"column:zbx_id;not null;size:50;index" json:"zbx_id"`
	IPMIIP           string    `gorm:"column:ipmi_ip;not null;size:16;index" json:"ipmi_ip"`
	BusinessName     string    `gorm:"column:business_name;not null;size:100;index" json:"business_name"`
	BusinessID       string    `gorm:"column:business_id;not null;size:50;index" json:"business_id"`
	OldBusinessName  string    `gorm:"column:old_business_name;not null;size:100" json:"old_business_name"`
	OldBusinessID    string    `gorm:"column:old_business_id;not null;size:50" json:"old_business_id"`
	BusinessSpeed    int16     `gorm:"column:business_speed;not null" json:"business_speed"`
	OldBusinessSpeed int16     `gorm:"column:old_business_speed;not null" json:"old_business_speed"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ArchivedBusinessInfo) TableName() string {
	return "archived_business_info"
}

type ArchivedNetworkInfo struct {
	ArchiveMeta
	MachineID    *string   `gorm:"column:machine_id;size:36;index" json:"machine_id,omitempty"`
	IPMIIP       *string   `gorm:"column:ipmi_ip;size:16;index" json:"ipmi_ip,omitempty"`
	IPv4IP       *string   `gorm:"column:ipv4_ip;size:20;index" json:"ipv4_ip,omitempty"`
	ZbxID        *string   `gorm:"column:zbx_id;size:50;index" json:"zbx_id,omitempty"`
	MacAddress   *string   `gorm:"column:mac_address;size:17;index" json:"mac_address,omitempty"`
	EthName      *string   `gorm:"column:eth_name;size:15" json:"eth_name,omitempty"`
	IDCCode      *string   `gorm:"column:idc_code;size:10;index" json:"idc_code,omitempty"`
	NetType      *string   `gorm:"column:net_type;size:20;index" json:"net_type,omitempty"`
	Vlan         *string   `gorm:"column:vlan;size:9" json:"vlan,omitempty"`
	IPv4Gateway  *string   `gorm:"column:ipv4_gateway;size:20" json:"ipv4_gateway,omitempty"`
	IPv6IP       *string   `gorm:"column:ipv6_ip;size:50;index" json:"ipv6_ip,omitempty"`
	IPv6Gateway  *string   `gorm:"column:ipv6_gateway;size:50" json:"ipv6_gateway,omitempty"`
	IPSpeed      *int16    `gorm:"column:ip_speed" json:"ip_speed,omitempty"`
	IPStatus     *string   `gorm:"column:ip_status;size:10;index" json:"ip_status,omitempty"`
	IPNotes      *string   `gorm:"column:ip_notes;size:255" json:"ip_notes,omitempty"`
	SegmentNotes *string   `gorm:"column:segment_notes;size:255" json:"segment_notes,omitempty"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ArchivedNetworkInfo) TableName() string {
	return "archived_network_info"
}
