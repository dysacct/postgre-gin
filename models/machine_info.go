package models

import "time"

type MachineInfo struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	MachineID    string    `gorm:"column:machine_id;not null;size:36;index" json:"machine_id"`
	ZbxID        string    `gorm:"column:zbx_id;not null;size:50" json:"zbx_id"`
	IPMIIP       string    `gorm:"column:ipmi_ip;not null;size:16" json:"ipmi_ip"`
	SystemType   string    `gorm:"column:system_type;not null;size:100" json:"system_type"`
	Manufacturer string    `gorm:"column:manufacturer;not null;size:100" json:"manufacturer"`
	ServerSN     string    `gorm:"column:server_sn;not null;size:100" json:"server_sn"`
	SystemDisk   string    `gorm:"column:system_disk;not null;size:255" json:"system_disk"`
	SSDCount     string    `gorm:"column:ssd_count;not null;size:255" json:"ssd_count"`
	HDDCount     string    `gorm:"column:hdd_count;not null;size:255" json:"hdd_count"`
	SysHDDCount  string    `gorm:"column:sys_hdd_count;not null;default:'';size:255" json:"sys_hdd_count"`
	MemoryCount  string    `gorm:"column:memory_count;not null;size:255" json:"memory_count"`
	CPUInfo      string    `gorm:"column:cpu_info;type:text;not null" json:"cpu_info"`
	ServerHeight string    `gorm:"column:server_height;not null;size:50" json:"server_height"`
	SwitchPort   string    `gorm:"column:switch_port;not null;default:'';size:100" json:"switch_port"`
	CreatedAt    time.Time `json:"created_at"`
}

func (MachineInfo) TableName() string { return "machine_info" }
