package models

import "time"

type DeletedRecord struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RecordType string    `gorm:"column:record_type;not null;size:20" json:"record_type"`
	IPMIIP     string    `gorm:"column:ipmi_ip;size:16;index" json:"ipmi_ip"`
	IDCCode    string    `gorm:"column:idc_code;size:10;index" json:"idc_code"`
	SourceTable string    `gorm:"column:source_table;not null;size:30" json:"source_table"`
	RecordData string    `gorm:"column:record_data;type:jsonb" json:"record_data"`
	DeletedBy  string    `gorm:"column:deleted_by;not null;size:50" json:"deleted_by"`
	DeletedAt  time.Time `gorm:"column:deleted_at;autoCreateTime" json:"deleted_at"`
	ExpiresAt  time.Time `gorm:"column:expires_at" json:"expires_at"`
}

func (DeletedRecord) TableName() string {
	return "deleted_records"
}
