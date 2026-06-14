package models

type VersionInfo struct {
	VersionNum string `gorm:"not null;size:20;column:version_num"`
}

func (VersionInfo) TableName() string {
	return "version_info"
}
