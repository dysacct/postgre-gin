package handlers

import (
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetIDCInfo 获取IDC信息
// 获取所有IDC信息的ZbxID、IPMIIP、SSHIP字段
func GetIDCInfo(c *gin.Context) {
	var idcInfos []models.IDCInfo

	// 查询所有IDC信息，只选择需要的字段
	if err := database.DB.Select("zbx_id, ipmi_ip, ssh_ip").Find(&idcInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询IDC信息失败",
		})
		return
	}

	// 转换为响应格式
	var responseData []models.IDCInfoResponse
	for _, idc := range idcInfos {
		responseData = append(responseData, models.IDCInfoResponse{
			ZbxID:  idc.ZbxID,
			IPMIIP: idc.IPMIIP,
			SSHIP:  idc.SSHIP,
		})
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data:    responseData,
	})
}

// DeleteIDCByCode 根据 IDC 机房代码删除整个机房的所有数据
// 删除 idc_info、business_info、machine_info、network_info 四个表中该机房的所有数据
// 这是一个危险操作，需要管理员权限
func DeleteIDCByCode(c *gin.Context) {
	idcCode := c.Param("idc_code")
	if idcCode == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IDC机房代码不能为空",
		})
		return
	}

	// 1. 先查询该机房下的所有机器（获取所有 ipmi_ip）
	var idcInfos []models.IDCInfo
	if err := database.DB.Where("idc_code = ?", idcCode).Find(&idcInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询机房信息失败: " + err.Error(),
		})
		return
	}

	if len(idcInfos) == 0 {
		// 添加调试信息，查询所有机房代码
		var allIDCs []models.IDCInfo
		database.DB.Select("DISTINCT idc_code").Find(&allIDCs)
		var codes []string
		for _, idc := range allIDCs {
			codes = append(codes, idc.IDCCode)
		}

		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: fmt.Sprintf("未找到机房代码 '%s' 的任何数据，当前系统中的机房代码有: %v", idcCode, codes),
		})
		return
	}

	// 收集所有的 ipmi_ip
	ipmiIPs := make([]string, 0, len(idcInfos))
	for _, idc := range idcInfos {
		ipmiIPs = append(ipmiIPs, idc.IPMIIP)
	}

	// 开启事务，确保数据一致性
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "开启事务失败",
		})
		return
	}

	// 统计删除数量
	var (
		businessInfoCount int64
		machineInfoCount  int64
		idcInfoCount      int64
		networkInfoCount  int64
	)

	// 2. 删除 business_info 表中的数据
	result := tx.Where("ipmi_ip IN ?", ipmiIPs).Delete(&models.BusinessInfo{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除业务信息失败: " + result.Error.Error(),
		})
		return
	}
	businessInfoCount = result.RowsAffected

	// 3. 删除 machine_info 表中的数据
	result = tx.Where("ipmi_ip IN ?", ipmiIPs).Delete(&models.MachineInfo{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除机器硬件信息失败: " + result.Error.Error(),
		})
		return
	}
	machineInfoCount = result.RowsAffected

	// 4. 删除 idc_info 表中的数据
	result = tx.Where("idc_code = ?", idcCode).Delete(&models.IDCInfo{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除IDC信息失败: " + result.Error.Error(),
		})
		return
	}
	idcInfoCount = result.RowsAffected

	// 5. 删除 network_info 表中的数据
	result = tx.Where("idc_code = ?", idcCode).Delete(&models.NetworkInfo{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + result.Error.Error(),
		})
		return
	}
	networkInfoCount = result.RowsAffected

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "提交事务失败",
		})
		return
	}

	// 清除所有相关机器的缓存
	for _, ipmiIP := range ipmiIPs {
		database.CacheDel(c.Request.Context(), "machine:"+ipmiIP)
		database.CacheDel(c.Request.Context(), "network_info:"+ipmiIP)
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "成功删除机房 " + idcCode + " 的所有数据",
		Data: gin.H{
			"idc_code":             idcCode,
			"deleted_machines":     len(ipmiIPs),
			"deleted_idc_info":     idcInfoCount,
			"deleted_business":     businessInfoCount,
			"deleted_machine_info": machineInfoCount,
			"deleted_network_info": networkInfoCount,
			"total_deleted":        businessInfoCount + machineInfoCount + idcInfoCount + networkInfoCount,
		},
	})
}

