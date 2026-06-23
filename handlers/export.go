package handlers

import (
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func ExportMachines(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	businessName := strings.TrimSpace(c.Query("business_name"))

	query := database.DB.Table("idc_info i").
		Select(`i.zbx_id, i.ipmi_ip, i.idc_code, i.idc_name, i.ssh_ip,
			m.system_type, m.manufacturer, m.server_sn, m.system_disk,
			m.ssd_count, m.hdd_count, m.sys_hdd_count, m.memory_count, m.cpu_info, m.server_height, m.switch_port,
			b.business_name, b.business_id, b.business_speed,
			b.old_business_name, b.old_business_id, b.old_business_speed`).
		Joins("LEFT JOIN machine_info m ON m.machine_id = i.machine_id OR (m.machine_id = '' AND m.ipmi_ip = i.ipmi_ip)").
		Joins("LEFT JOIN business_info b ON b.machine_id = i.machine_id OR (b.machine_id = '' AND b.ipmi_ip = i.ipmi_ip)")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where(`i.zbx_id ILIKE ? OR i.idc_name ILIKE ? OR i.ipmi_ip ILIKE ? OR
			i.ssh_ip ILIKE ? OR b.business_name ILIKE ?`, like, like, like, like, like)
	}
	if idcCode != "" {
		query = query.Where("i.idc_code ILIKE ?", "%"+idcCode+"%")
	}
	if businessName != "" {
		query = query.Where("b.business_name ILIKE ?", "%"+businessName+"%")
	}

	type Row struct {
		ZbxID            string `gorm:"column:zbx_id"`
		IPMIIP           string `gorm:"column:ipmi_ip"`
		IDCCode          string `gorm:"column:idc_code"`
		IDCName          string `gorm:"column:idc_name"`
		SSHIP            string `gorm:"column:ssh_ip"`
		SystemType       string `gorm:"column:system_type"`
		Manufacturer     string `gorm:"column:manufacturer"`
		ServerSN         string `gorm:"column:server_sn"`
		SystemDisk       string `gorm:"column:system_disk"`
		SSDCount         string `gorm:"column:ssd_count"`
		HDDCount         string `gorm:"column:hdd_count"`
		SysHDDCount      string `gorm:"column:sys_hdd_count"`
		MemoryCount      string `gorm:"column:memory_count"`
		CPUInfo          string `gorm:"column:cpu_info"`
		ServerHeight     string `gorm:"column:server_height"`
		SwitchPort       string `gorm:"column:switch_port"`
		BusinessName     string `gorm:"column:business_name"`
		BusinessID       string `gorm:"column:business_id"`
		BusinessSpeed    int16  `gorm:"column:business_speed"`
		OldBusinessName  string `gorm:"column:old_business_name"`
		OldBusinessID    string `gorm:"column:old_business_id"`
		OldBusinessSpeed int16  `gorm:"column:old_business_speed"`
	}
	var rows []Row
	query.Order("i.created_at DESC").Find(&rows)

	f := excelize.NewFile()
	sheet := "机器信息"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ZbxID", "IPMI_IP", "机房编码", "机房名称", "SSH_IP", "系统类型", "厂商", "序列号", "系统盘", "SSD", "HDD", "系统直通HDD", "内存", "CPU", "高度", "交换机端口", "业务名称", "业务ID", "带宽(M)", "旧业务名称", "旧业务ID", "旧带宽(M)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ZbxID, r.IPMIIP, r.IDCCode, r.IDCName, r.SSHIP, r.SystemType, r.Manufacturer, r.ServerSN, r.SystemDisk, r.SSDCount, r.HDDCount, r.SysHDDCount, r.MemoryCount, r.CPUInfo, r.ServerHeight, r.SwitchPort, r.BusinessName, r.BusinessID, r.BusinessSpeed, r.OldBusinessName, r.OldBusinessID, r.OldBusinessSpeed}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	setExportDimension(f, sheet, len(headers), len(rows)+1)

	streamExcel(c, f, "machines.xlsx")
}

func ExportMachineArchives(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.DefaultQuery("status", models.ArchiveStatusArchived))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	ipmiIP := strings.TrimSpace(c.Query("ipmi_ip"))
	zbxID := strings.TrimSpace(c.Query("zbx_id"))
	now := time.Now()

	applyFilters := func(query *gorm.DB) *gorm.DB {
		query = query.Where("b.expires_at > ?", now)
		if status != "" && status != "all" {
			query = query.Where("b.status = ?", status)
		}
		if idcCode != "" {
			query = query.Where("b.idc_code ILIKE ?", "%"+idcCode+"%")
		}
		if ipmiIP != "" {
			query = query.Where("b.ipmi_ip ILIKE ?", "%"+ipmiIP+"%")
		}
		if zbxID != "" {
			query = query.Where("b.zbx_id ILIKE ?", "%"+zbxID+"%")
		}
		if search != "" {
			like := "%" + search + "%"
			query = query.Where("b.ipmi_ip ILIKE ? OR b.zbx_id ILIKE ? OR b.idc_code ILIKE ? OR b.idc_name ILIKE ? OR b.ssh_ip ILIKE ?", like, like, like, like, like)
		}
		return query
	}

	type ArchiveRow struct {
		ArchiveBatchID   string     `gorm:"column:archive_batch_id"`
		IPMIIP           string     `gorm:"column:ipmi_ip"`
		ZbxID            string     `gorm:"column:zbx_id"`
		IDCCode          string     `gorm:"column:idc_code"`
		IDCName          string     `gorm:"column:idc_name"`
		SSHIP            string     `gorm:"column:ssh_ip"`
		ArchiveReason    string     `gorm:"column:archive_reason"`
		Status           string     `gorm:"column:status"`
		LastSeenAt       *time.Time `gorm:"column:last_seen_at"`
		ArchivedAt       time.Time  `gorm:"column:archived_at"`
		ExpiresAt        time.Time  `gorm:"column:expires_at"`
		RestoredAt       *time.Time `gorm:"column:restored_at"`
		SystemType       string     `gorm:"column:system_type"`
		Manufacturer     string     `gorm:"column:manufacturer"`
		ServerSN         string     `gorm:"column:server_sn"`
		SystemDisk       string     `gorm:"column:system_disk"`
		SSDCount         string     `gorm:"column:ssd_count"`
		HDDCount         string     `gorm:"column:hdd_count"`
		SysHDDCount      string     `gorm:"column:sys_hdd_count"`
		MemoryCount      string     `gorm:"column:memory_count"`
		CPUInfo          string     `gorm:"column:cpu_info"`
		ServerHeight     string     `gorm:"column:server_height"`
		SwitchPort       string     `gorm:"column:switch_port"`
		BusinessName     string     `gorm:"column:business_name"`
		BusinessID       string     `gorm:"column:business_id"`
		BusinessSpeed    int16      `gorm:"column:business_speed"`
		OldBusinessName  string     `gorm:"column:old_business_name"`
		OldBusinessID    string     `gorm:"column:old_business_id"`
		OldBusinessSpeed int16      `gorm:"column:old_business_speed"`
	}

	var archiveRows []ArchiveRow
	archiveQuery := database.DB.Table("machine_archive_batches b").
		Select(`b.archive_batch_id, b.ipmi_ip, b.zbx_id, b.idc_code, b.idc_name, b.ssh_ip,
			b.archive_reason, b.status, b.last_seen_at, b.archived_at, b.expires_at, b.restored_at,
			m.system_type, m.manufacturer, m.server_sn, m.system_disk, m.ssd_count, m.hdd_count, m.sys_hdd_count,
			m.memory_count, m.cpu_info, m.server_height, m.switch_port,
			bi.business_name, bi.business_id, bi.business_speed,
			bi.old_business_name, bi.old_business_id, bi.old_business_speed`).
		Joins("LEFT JOIN archived_machine_info m ON m.archive_batch_id = b.archive_batch_id").
		Joins("LEFT JOIN archived_business_info bi ON bi.archive_batch_id = b.archive_batch_id")
	applyFilters(archiveQuery).Order("b.archived_at DESC").Find(&archiveRows)

	type NetworkRow struct {
		ArchiveBatchID string    `gorm:"column:archive_batch_id"`
		IPMIIP         string    `gorm:"column:ipmi_ip"`
		ZbxID          string    `gorm:"column:zbx_id"`
		IDCCode        string    `gorm:"column:idc_code"`
		IPv4IP         *string   `gorm:"column:ipv4_ip"`
		IPv6IP         *string   `gorm:"column:ipv6_ip"`
		MacAddress     *string   `gorm:"column:mac_address"`
		EthName        *string   `gorm:"column:eth_name"`
		NetType        *string   `gorm:"column:net_type"`
		Vlan           *string   `gorm:"column:vlan"`
		IPv4Gateway    *string   `gorm:"column:ipv4_gateway"`
		IPv6Gateway    *string   `gorm:"column:ipv6_gateway"`
		IPSpeed        *int16    `gorm:"column:ip_speed"`
		IPStatus       *string   `gorm:"column:ip_status"`
		IPNotes        *string   `gorm:"column:ip_notes"`
		SegmentNotes   *string   `gorm:"column:segment_notes"`
		ArchivedAt     time.Time `gorm:"column:archived_at"`
		ExpiresAt      time.Time `gorm:"column:expires_at"`
	}

	var networkRows []NetworkRow
	networkQuery := database.DB.Table("machine_archive_batches b").
		Select(`b.archive_batch_id, b.ipmi_ip, b.zbx_id, b.idc_code, n.ipv4_ip, n.ipv6_ip,
			n.mac_address, n.eth_name, n.net_type, n.vlan, n.ipv4_gateway, n.ipv6_gateway,
			n.ip_speed, n.ip_status, n.ip_notes, n.segment_notes, b.archived_at, b.expires_at`).
		Joins("JOIN archived_network_info n ON n.archive_batch_id = b.archive_batch_id")
	applyFilters(networkQuery).Order("b.archived_at DESC, n.archive_id ASC").Find(&networkRows)

	f := excelize.NewFile()
	summarySheet := "归档机器"
	f.SetSheetName("Sheet1", summarySheet)

	headers := []string{"归档批次", "IPMI_IP", "ZbxID", "机房编码", "机房名称", "SSH_IP", "归档原因", "状态", "最后更新", "归档时间", "过期时间", "恢复时间", "系统类型", "厂商", "序列号", "系统盘", "SSD", "HDD", "系统直通HDD", "内存", "CPU", "高度", "交换机端口", "业务名称", "业务ID", "带宽(M)", "旧业务名称", "旧业务ID", "旧带宽(M)"}
	writeHeader(f, summarySheet, headers)
	for i, r := range archiveRows {
		vals := []interface{}{
			r.ArchiveBatchID, r.IPMIIP, r.ZbxID, r.IDCCode, r.IDCName, r.SSHIP,
			r.ArchiveReason, r.Status, formatTimePtr(r.LastSeenAt), formatTime(r.ArchivedAt),
			formatTime(r.ExpiresAt), formatTimePtr(r.RestoredAt), r.SystemType, r.Manufacturer,
			r.ServerSN, r.SystemDisk, r.SSDCount, r.HDDCount, r.SysHDDCount, r.MemoryCount, r.CPUInfo,
			r.ServerHeight, r.SwitchPort, r.BusinessName, r.BusinessID, r.BusinessSpeed,
			r.OldBusinessName, r.OldBusinessID, r.OldBusinessSpeed,
		}
		writeRow(f, summarySheet, i+2, vals)
	}
	setExportDimension(f, summarySheet, len(headers), len(archiveRows)+1)

	networkSheet := "归档网络"
	_, _ = f.NewSheet(networkSheet)
	networkHeaders := []string{"归档批次", "IPMI_IP", "ZbxID", "机房编码", "IPv4", "IPv6", "MAC", "网卡", "网络类型", "VLAN", "IPv4网关", "IPv6网关", "速率", "状态", "备注", "网段备注", "归档时间", "过期时间"}
	writeHeader(f, networkSheet, networkHeaders)
	for i, r := range networkRows {
		vals := []interface{}{
			r.ArchiveBatchID, r.IPMIIP, r.ZbxID, r.IDCCode, strPtr(r.IPv4IP), strPtr(r.IPv6IP),
			strPtr(r.MacAddress), strPtr(r.EthName), strPtr(r.NetType), strPtr(r.Vlan),
			strPtr(r.IPv4Gateway), strPtr(r.IPv6Gateway), int16Ptr(r.IPSpeed), strPtr(r.IPStatus),
			strPtr(r.IPNotes), strPtr(r.SegmentNotes), formatTime(r.ArchivedAt), formatTime(r.ExpiresAt),
		}
		writeRow(f, networkSheet, i+2, vals)
	}
	setExportDimension(f, networkSheet, len(networkHeaders), len(networkRows)+1)

	f.SetActiveSheet(0)
	streamExcel(c, f, "machine_archives.xlsx")
}

func ExportNetworkInfo(c *gin.Context) {
	idcCode := c.Query("idc_code")
	ipStatus := c.Query("ip_status")
	netType := c.Query("net_type")

	query := database.DB.Model(&models.NetworkInfo{})
	if idcCode != "" {
		query = query.Where("idc_code = ?", idcCode)
	}
	if ipStatus != "" {
		query = query.Where("ip_status = ?", ipStatus)
	}
	if netType != "" {
		query = query.Where("net_type = ?", netType)
	}

	var rows []models.NetworkInfo
	query.Order("id DESC").Find(&rows)

	f := excelize.NewFile()
	sheet := "网络信息"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "IPMI_IP", "IPv4", "IPv6", "MAC", "网卡", "机房", "网络类型", "VLAN", "网关", "速率", "状态", "备注"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ID, strPtr(r.IPMIIP), strPtr(r.IPv4IP), strPtr(r.IPv6IP), strPtr(r.MacAddress), strPtr(r.EthName), strPtr(r.IDCCode), strPtr(r.NetType), strPtr(r.Vlan), strPtr(r.IPv4Gateway),
			int16Ptr(r.IPSpeed), strPtr(r.IPStatus), strPtr(r.IPNotes)}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	setExportDimension(f, sheet, len(headers), len(rows)+1)

	streamExcel(c, f, "network_info.xlsx")
}

func ExportIDCInfo(c *gin.Context) {
	var rows []models.IDCInfo
	database.DB.Select("zbx_id, ipmi_ip, ssh_ip, idc_code, idc_name").Order("idc_code").Find(&rows)

	f := excelize.NewFile()
	sheet := "SSH信息"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ZbxID", "IPMI_IP", "SSH_IP", "机房编码", "机房名称"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ZbxID, r.IPMIIP, r.SSHIP, r.IDCCode, r.IDCName}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	setExportDimension(f, sheet, len(headers), len(rows)+1)

	streamExcel(c, f, "idc_info.xlsx")
}

func ExportDeletedRecords(c *gin.Context) {
	recordType := c.Query("record_type")

	query := database.DB.Model(&models.DeletedRecord{}).Where("expires_at > ?", time.Now())
	if recordType != "" {
		query = query.Where("record_type = ?", recordType)
	}

	var rows []models.DeletedRecord
	query.Order("deleted_at DESC").Find(&rows)

	f := excelize.NewFile()
	sheet := "删除记录"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "类型", "IPMI_IP", "机房", "来源表", "原始数据(JSON)", "操作人", "删除时间", "过期时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ID, r.RecordType, r.IPMIIP, r.IDCCode, r.SourceTable, r.RecordData, r.DeletedBy, r.DeletedAt.Format("2006-01-02 15:04:05"), r.ExpiresAt.Format("2006-01-02 15:04:05")}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	setExportDimension(f, sheet, len(headers), len(rows)+1)

	streamExcel(c, f, "deleted_records.xlsx")
}

func ExportBusinessInfo(c *gin.Context) {
	businessName := strings.TrimSpace(c.Query("business_name"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))

	query := database.DB.Table("idc_info i").
		Select("i.zbx_id, i.ipmi_ip, i.idc_code, b.business_name, b.business_id, b.business_speed, b.old_business_name, b.old_business_id, b.old_business_speed").
		Joins("LEFT JOIN business_info b ON b.machine_id = i.machine_id OR (b.machine_id = '' AND b.ipmi_ip = i.ipmi_ip)")

	if businessName != "" {
		query = query.Where("b.business_name ILIKE ?", "%"+businessName+"%")
	}
	if idcCode != "" {
		query = query.Where("i.idc_code ILIKE ?", "%"+idcCode+"%")
	}

	type Row struct {
		ZbxID            string `gorm:"column:zbx_id"`
		IPMIIP           string `gorm:"column:ipmi_ip"`
		IDCCode          string `gorm:"column:idc_code"`
		BusinessName     string `gorm:"column:business_name"`
		BusinessID       string `gorm:"column:business_id"`
		BusinessSpeed    int16  `gorm:"column:business_speed"`
		OldBusinessName  string `gorm:"column:old_business_name"`
		OldBusinessID    string `gorm:"column:old_business_id"`
		OldBusinessSpeed int16  `gorm:"column:old_business_speed"`
	}
	var rows []Row
	query.Order("i.idc_code").Find(&rows)

	f := excelize.NewFile()
	sheet := "业务信息"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ZbxID", "IPMI_IP", "机房编码", "业务名称", "业务ID", "带宽(M)", "旧业务名称", "旧业务ID", "旧带宽(M)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ZbxID, r.IPMIIP, r.IDCCode, r.BusinessName, r.BusinessID, r.BusinessSpeed, r.OldBusinessName, r.OldBusinessID, r.OldBusinessSpeed}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
	setExportDimension(f, sheet, len(headers), len(rows)+1)

	streamExcel(c, f, "business_info.xlsx")
}

func setExportDimension(f *excelize.File, sheet string, colCount, rowCount int) {
	if colCount <= 0 || rowCount <= 0 {
		return
	}
	_ = f.SetSheetDimension(sheet, fmt.Sprintf("A1:%s", cellName(colCount, rowCount)))
}

func writeHeader(f *excelize.File, sheet string, headers []string) {
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)
}

func writeRow(f *excelize.File, sheet string, row int, vals []interface{}) {
	for i, v := range vals {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, v)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int16Ptr(i *int16) int16 {
	if i == nil {
		return 0
	}
	return *i
}

func streamExcel(c *gin.Context, f *excelize.File, filename string) {
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Transfer-Encoding", "binary")

	buf, err := f.WriteToBuffer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "导出 Excel 失败: " + err.Error(),
		})
		return
	}
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
