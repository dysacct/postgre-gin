package models

import "time"

type BusinessInfo struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ZbxID            string    `gorm:"column:zbx_id;not null;size:50" json:"zbx_id"`
	IPMIIP           string    `gorm:"column:ipmi_ip;not null;size:16" json:"ipmi_ip"`
	BusinessName     string    `gorm:"column:business_name;not null;size:100" json:"business_name"`
	BusinessID       string    `gorm:"column:business_id;not null;size:50" json:"business_id"`
	OldBusinessName  string    `gorm:"column:old_business_name;not null;size:100" json:"old_business_name"`
	OldBusinessID    string    `gorm:"column:old_business_id;not null;size:50" json:"old_business_id"`
	BusinessSpeed    int16     `gorm:"column:business_speed;not null" json:"business_speed"`
	OldBusinessSpeed int16     `gorm:"column:old_business_speed;not null" json:"old_business_speed"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (BusinessInfo) TableName() string { return "business_info" }
