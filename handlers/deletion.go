package handlers

import (
	"encoding/json"
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeleteMachineRequest struct {
	IDCCode string   `json:"idc_code"`
	IPMIIPs []string `json:"ipmi_ips"`
}

type DeleteNetworkRequest struct {
	IDCCode string   `json:"idc_code"`
	IPMIIPs []string `json:"ipmi_ips"`
}

func idcCodeOfIPMI(tx *gorm.DB, ipmiIP string) string {
	var idc models.IDCInfo
	tx.Where("ipmi_ip = ?", ipmiIP).First(&idc)
	return idc.IDCCode
}

// archiveAndDeleteMachine 归档并删除机器信息
func archiveAndDeleteMachine(c *gin.Context, ipmiIPs []string, idcCode string, username string) (int, error) {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var deletedCount int
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	if idcCode != "" {
		var idcInfos []models.IDCInfo
		if err := tx.Where("idc_code = ?", idcCode).Find(&idcInfos).Error; err != nil {
			tx.Rollback()
			return 0, err
		}
		for _, idc := range idcInfos {
			ipmiIPs = append(ipmiIPs, idc.IPMIIP)
		}
		if len(ipmiIPs) == 0 {
			tx.Rollback()
			return 0, fmt.Errorf("机房 %s 没有机器", idcCode)
		}
	}

	for _, ipmiIP := range ipmiIPs {
		ipmiIP = strings.TrimSpace(ipmiIP)
		if ipmiIP == "" {
			continue
		}

		// 归档 idc_info
		var idc models.IDCInfo
		if err := tx.First(&idc, "ipmi_ip = ?", ipmiIP).Error; err == nil {
			jsonData, _ := json.Marshal(idc)
			tx.Create(&models.DeletedRecord{
				RecordType:  "machine",
				IPMIIP:      ipmiIP,
				IDCCode:     idc.IDCCode,
				SourceTable: "idc_info",
				RecordData:  string(jsonData),
				DeletedBy:   username,
				ExpiresAt:   expiresAt,
			})
			tx.Delete(&idc)
		}

		// 归档 machine_info
		var machine models.MachineInfo
		if err := tx.First(&machine, "ipmi_ip = ?", ipmiIP).Error; err == nil {
			jsonData, _ := json.Marshal(machine)
			tx.Create(&models.DeletedRecord{
				RecordType:  "machine",
				IPMIIP:      ipmiIP,
				IDCCode:     idcCodeOfIPMI(tx, ipmiIP),
				SourceTable: "machine_info",
				RecordData:  string(jsonData),
				DeletedBy:   username,
				ExpiresAt:   expiresAt,
			})
			tx.Delete(&machine)
		}

		// 归档 business_info
		var business models.BusinessInfo
		if err := tx.First(&business, "ipmi_ip = ?", ipmiIP).Error; err == nil {
			jsonData, _ := json.Marshal(business)
			tx.Create(&models.DeletedRecord{
				RecordType:  "machine",
				IPMIIP:      ipmiIP,
				IDCCode:     idcCodeOfIPMI(tx, ipmiIP),
				SourceTable: "business_info",
				RecordData:  string(jsonData),
				DeletedBy:   username,
				ExpiresAt:   expiresAt,
			})
			tx.Delete(&business)
		}

		deletedCount++
		database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}
	return deletedCount, nil
}

// archiveAndDeleteNetwork 归档并删除网络信息
func archiveAndDeleteNetwork(c *gin.Context, ipmiIPs []string, idcCode string, username string) (int64, error) {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var networks []models.NetworkInfo
	if idcCode != "" {
		tx.Where("idc_code = ?", idcCode).Find(&networks)
	} else if len(ipmiIPs) > 0 {
		tx.Where("ipmi_ip IN ?", ipmiIPs).Find(&networks)
	} else {
		tx.Rollback()
		return 0, fmt.Errorf("必须提供 idc_code 或 ipmi_ips")
	}

	if len(networks) == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("未找到匹配的网络信息")
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	for _, net := range networks {
		jsonData, _ := json.Marshal(net)
		ipmi := ""
		idc := ""
		if net.IPMIIP != nil {
			ipmi = *net.IPMIIP
		}
		if net.IDCCode != nil {
			idc = *net.IDCCode
		}
		tx.Create(&models.DeletedRecord{
			RecordType:  "network",
			IPMIIP:      ipmi,
			IDCCode:     idc,
			SourceTable: "network_info",
			RecordData:  string(jsonData),
			DeletedBy:   username,
			ExpiresAt:   expiresAt,
		})
	}

	var rowsAffected int64
	if idcCode != "" {
		result := tx.Where("idc_code = ?", idcCode).Delete(&models.NetworkInfo{})
		if result.Error != nil {
			tx.Rollback()
			return 0, result.Error
		}
		rowsAffected = result.RowsAffected
	} else {
		result := tx.Where("ipmi_ip IN ?", ipmiIPs).Delete(&models.NetworkInfo{})
		if result.Error != nil {
			tx.Rollback()
			return 0, result.Error
		}
		rowsAffected = result.RowsAffected
	}

	for _, net := range networks {
		if net.IPMIIP != nil {
			database.CacheDel(c.Request.Context(), networkCacheKey(*net.IPMIIP))
		}
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// DeleteMachines 删除机器信息（归档30天）
func DeleteMachines(c *gin.Context) {
	var req DeleteMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	if req.IDCCode == "" && len(req.IPMIIPs) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "至少需要提供 idc_code 或 ipmi_ips"})
		return
	}

	username, _ := c.Get("username")
	count, err := archiveAndDeleteMachine(c, req.IPMIIPs, req.IDCCode, username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功删除 %d 台机器的信息（三表），已归档保留30天", count),
		Data:    gin.H{"deleted_count": count},
	})
}

// DeleteNetworks 删除网络信息（归档30天）
func DeleteNetworks(c *gin.Context) {
	var req DeleteNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "参数错误: " + err.Error()})
		return
	}

	if req.IDCCode == "" && len(req.IPMIIPs) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "至少需要提供 idc_code 或 ipmi_ips"})
		return
	}

	username, _ := c.Get("username")
	count, err := archiveAndDeleteNetwork(c, req.IPMIIPs, req.IDCCode, username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{Code: 500, Message: "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功删除 %d 条网络信息，已归档保留30天", count),
		Data:    gin.H{"deleted_count": count},
	})
}

// GetDeletedRecords 查询已删除记录
func GetDeletedRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	recordType := c.Query("record_type")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 2000 {
		size = 50
	}

	var total int64
	var records []models.DeletedRecord

	query := database.DB.Model(&models.DeletedRecord{}).
		Where("expires_at > ?", time.Now())

	if recordType != "" {
		query = query.Where("record_type = ?", recordType)
	}

	query.Count(&total)
	query.Order("deleted_at DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&records)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total": total,
			"page":  page,
			"size":  size,
			"list":  records,
		},
	})
}

// CleanupExpiredRecords 清理过期记录
func CleanupExpiredRecords() {
	result := database.DB.Where("expires_at <= ?", time.Now()).Delete(&models.DeletedRecord{})
	if result.RowsAffected > 0 {
		fmt.Printf("已清理 %d 条过期的删除记录\n", result.RowsAffected)
	}
}
