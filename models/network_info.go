package models

import "time"

type NetworkInfo struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	IPMIIP       *string   `gorm:"column:ipmi_ip;size:16;index" json:"ipmi_ip,omitempty"`
	IPv4IP       *string   `gorm:"column:ipv4_ip;size:20;uniqueIndex:idx_ipv4_unique,where:ipv4_ip IS NOT NULL" json:"ipv4_ip,omitempty"`
	ZbxID        *string   `gorm:"column:zbx_id;size:50;index" json:"zbx_id,omitempty"`
	MacAddress   *string   `gorm:"column:mac_address;size:17;index" json:"mac_address,omitempty"`
	EthName      *string   `gorm:"column:eth_name;size:15" json:"eth_name,omitempty"`
	IDCCode      *string   `gorm:"column:idc_code;size:10;index" json:"idc_code,omitempty"`
	NetType      *string   `gorm:"column:net_type;size:20" json:"net_type,omitempty"`
	Vlan         *string   `gorm:"column:vlan;size:9" json:"vlan,omitempty"`
	IPv4Gateway  *string   `gorm:"column:ipv4_gateway;size:20" json:"ipv4_gateway,omitempty"`
	IPv6IP       *string   `gorm:"column:ipv6_ip;size:50;uniqueIndex:idx_ipv6_unique,where:ipv6_ip IS NOT NULL" json:"ipv6_ip,omitempty"`
	IPv6Gateway  *string   `gorm:"column:ipv6_gateway;size:50" json:"ipv6_gateway,omitempty"`
	IPSpeed      *int16    `gorm:"column:ip_speed" json:"ip_speed,omitempty"`
	IPStatus     *string   `gorm:"column:ip_status;size:10;index" json:"ip_status,omitempty"`
	IPNotes      *string   `gorm:"column:ip_notes;size:255" json:"ip_notes,omitempty"`
	SegmentNotes *string   `gorm:"column:segment_notes;size:255" json:"segment_notes,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (NetworkInfo) TableName() string { return "network_info" }

