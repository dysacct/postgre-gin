package models

import "time"

type IDCInfo struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ZbxID     string    `gorm:"column:zbx_id;not null;size:50" json:"zbx_id"`
	IPMIIP    string    `gorm:"column:ipmi_ip;unique;not null;size:16" json:"ipmi_ip"`
	IDCCode   string    `gorm:"column:idc_code;not null;size:10" json:"idc_code"`
	IDCName   string    `gorm:"column:idc_name;not null;size:50" json:"idc_name"`
	SSHIP     string    `gorm:"column:ssh_ip;not null;size:16" json:"ssh_ip"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (IDCInfo) TableName() string { return "idc_info" }

// IDCInfoResponse 用于返回IDC信息的特定字段
type IDCInfoResponse struct {
	ZbxID  string `json:"zbx_id"`
	IPMIIP string `json:"ipmi_ip"`
	SSHIP  string `json:"ssh_ip"`
}

