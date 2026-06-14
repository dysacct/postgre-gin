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
)

func ExportMachines(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	businessName := strings.TrimSpace(c.Query("business_name"))

	query := database.DB.Table("idc_info i").
		Select(`i.zbx_id, i.ipmi_ip, i.idc_code, i.idc_name, i.ssh_ip,
			m.system_type, m.manufacturer, m.server_sn, m.system_disk,
			m.ssd_count, m.hdd_count, m.memory_count, m.cpu_info, m.server_height,
			b.business_name, b.business_id, b.business_speed,
			b.old_business_name, b.old_business_id, b.old_business_speed`).
		Joins("LEFT JOIN machine_info m ON m.ipmi_ip = i.ipmi_ip").
		Joins("LEFT JOIN business_info b ON b.ipmi_ip = i.ipmi_ip")

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
		ZbxID, IPMIIP, IDCCode, IDCName, SSHIP, SystemType, Manufacturer, ServerSN, SystemDisk, SSDCount, HDDCount, MemoryCount, CPUInfo, ServerHeight, BusinessName, BusinessID, OldBusinessName, OldBusinessID string
		BusinessSpeed, OldBusinessSpeed int16
	}
	var rows []Row
	query.Order("i.created_at DESC").Find(&rows)

	f := excelize.NewFile()
	sheet := "机器信息"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ZbxID", "IPMI_IP", "机房编码", "机房名称", "SSH_IP", "系统类型", "厂商", "序列号", "系统盘", "SSD", "HDD", "内存", "CPU", "高度", "业务名称", "业务ID", "带宽(M)", "旧业务名称", "旧业务ID", "旧带宽(M)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), headerStyle)

	for i, r := range rows {
		vals := []interface{}{r.ZbxID, r.IPMIIP, r.IDCCode, r.IDCName, r.SSHIP, r.SystemType, r.Manufacturer, r.ServerSN, r.SystemDisk, r.SSDCount, r.HDDCount, r.MemoryCount, r.CPUInfo, r.ServerHeight, r.BusinessName, r.BusinessID, r.BusinessSpeed, r.OldBusinessName, r.OldBusinessID, r.OldBusinessSpeed}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	streamExcel(c, f, "machines.xlsx")
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

	streamExcel(c, f, "deleted_records.xlsx")
}

func ExportBusinessInfo(c *gin.Context) {
	businessName := strings.TrimSpace(c.Query("business_name"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))

	query := database.DB.Table("idc_info i").
		Select("i.zbx_id, i.ipmi_ip, i.idc_code, b.business_name, b.business_id, b.business_speed, b.old_business_name, b.old_business_id, b.old_business_speed").
		Joins("LEFT JOIN business_info b ON b.ipmi_ip = i.ipmi_ip")

	if businessName != "" {
		query = query.Where("b.business_name ILIKE ?", "%"+businessName+"%")
	}
	if idcCode != "" {
		query = query.Where("i.idc_code ILIKE ?", "%"+idcCode+"%")
	}

	type Row struct {
		ZbxID, IPMIIP, IDCCode, BusinessName, BusinessID, OldBusinessName, OldBusinessID string
		BusinessSpeed, OldBusinessSpeed int16
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

	streamExcel(c, f, "business_info.xlsx")
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

	buf, _ := f.WriteToBuffer()
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
