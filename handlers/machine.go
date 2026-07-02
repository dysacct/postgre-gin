package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const cacheTTL = 30 * time.Minute // 缓存过期时间30分钟

// 统一的redis缓存key格式
func cacheKey(ipmiIP string) string {
	return "cache:machine:" + ipmiIP
}

func buildIDCUpdateData(idc models.IDCInfo) map[string]interface{} {
	updateData := make(map[string]interface{})
	if idc.ZbxID != "" {
		updateData["zbx_id"] = idc.ZbxID
	}
	if idc.IDCCode != "" {
		updateData["idc_code"] = idc.IDCCode
	}
	if idc.IDCName != "" {
		updateData["idc_name"] = idc.IDCName
	}
	if idc.SSHIP != "" {
		updateData["ssh_ip"] = idc.SSHIP
	}
	return updateData
}

func syncRelatedZbxID(tx *gorm.DB, ipmiIP string, oldZbxID string, newZbxID string) error {
	if newZbxID == "" {
		return nil
	}

	idc, err := resolveCurrentMachineIdentity(tx, ipmiIP)
	machineID := ""
	if err == nil {
		machineID = idc.MachineID
	}

	relatedQuery := "ipmi_ip = ?"
	relatedArgs := []interface{}{ipmiIP}
	if machineID != "" {
		relatedQuery = "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)"
		relatedArgs = []interface{}{machineID, ipmiIP}
	}

	if err := tx.Model(&models.MachineInfo{}).Where(relatedQuery, relatedArgs...).Update("zbx_id", newZbxID).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.BusinessInfo{}).Where(relatedQuery, relatedArgs...).Update("zbx_id", newZbxID).Error; err != nil {
		return err
	}

	networkQuery := tx.Model(&models.NetworkInfo{}).Where("ipmi_ip = ?", ipmiIP)
	if oldZbxID != "" && oldZbxID != newZbxID {
		networkQuery = tx.Model(&models.NetworkInfo{}).Where("ipmi_ip = ? OR zbx_id = ?", ipmiIP, oldZbxID)
	}
	if err := networkQuery.Update("zbx_id", newZbxID).Error; err != nil {
		return err
	}
	if machineID != "" {
		if err := tx.Model(&models.NetworkInfo{}).Where("ipmi_ip = ? OR zbx_id = ?", ipmiIP, newZbxID).Update("machine_id", machineID).Error; err != nil {
			return err
		}
	}
	return nil
}

func updateIDCAndRelatedZbxID(tx *gorm.DB, existingIDC *models.IDCInfo, updateIDC models.IDCInfo) error {
	updateData := buildIDCUpdateData(updateIDC)
	if len(updateData) == 0 {
		return nil
	}

	oldZbxID := existingIDC.ZbxID
	if err := tx.Model(existingIDC).Updates(updateData).Error; err != nil {
		return err
	}

	return syncRelatedZbxID(tx, existingIDC.IPMIIP, oldZbxID, updateIDC.ZbxID)
}

func loadMachineRelations(idc models.IDCInfo) (models.MachineInfo, models.BusinessInfo, []models.NetworkInfo) {
	var machine models.MachineInfo
	var business models.BusinessInfo
	var networks []models.NetworkInfo

	if idc.MachineID != "" {
		database.DB.First(&machine, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, idc.IPMIIP)
		database.DB.First(&business, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, idc.IPMIIP)
		database.DB.Find(&networks, "machine_id = ? OR ipmi_ip = ?", idc.MachineID, idc.IPMIIP)
		return machine, business, networks
	}

	database.DB.First(&machine, "ipmi_ip = ?", idc.IPMIIP)
	database.DB.First(&business, "ipmi_ip = ?", idc.IPMIIP)
	database.DB.Find(&networks, "ipmi_ip = ?", idc.IPMIIP)
	return machine, business, networks
}

// GetMachine 获取单个机器信息
// 根据ipmi_ip获取机器的详细信息，包括IDC、机器、业务、网络信息
func GetMachine(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip") // 从URL路径参数获取ipmi_ip
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "ipmi_ip不能为空"})
		return
	}

	// Step 1 : 先查Redis缓存
	ctx := c.Request.Context() // Request是Gin框架的请求对象, Context()是获取请求的上下文
	cacheResult := database.CacheGet(ctx, cacheKey(ipmiIP))

	if cacheResult.Err() == nil { // .Err() 是redis.Nil错误, 表示缓存不存在
		// 缓存命中
		var result map[string]interface{}
		json.Unmarshal([]byte(cacheResult.Val()), &result)
		c.JSON(http.StatusOK, models.Response{
			Code:    200,
			Message: "缓存命中",
			Data:    result,
		})
		return
	}

	// redis.Nil 是redis的错误, 表示缓存不存在
	if cacheResult.Err() != redis.Nil {
		slog.Warn("Redis 查询失败, 将直接查库", "err", cacheResult.Err()) // slog.Warn 是日志记录器, 用于记录警告信息
	}

	// Step 2 : 缓存 miss，查 PostgreSQL (联表查询)
	var idc models.IDCInfo
	err := database.DB.First(&idc, "ipmi_ip = ?", ipmiIP).Error
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "IDC信息不存在",
		})
		return
	}
	machine, business, networks := loadMachineRelations(idc)

	// 组装最终返回数据
	result := gin.H{
		"idc_info":      idc,
		"machine_info":  machine,
		"business_info": business,
		"network_info":  networks,
	}

	// Step 3 : 写入Redis缓存
	jsonData, _ := json.Marshal(result)
	database.CacheSet(ctx, cacheKey(ipmiIP), jsonData, cacheTTL)
	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data:    result,
	})
	slog.Info("数据库查询并已经缓存", "ipmi_ip", ipmiIP)
}

// CreateMachine 创建机器
// 创建新的机器记录
func CreateMachine(c *gin.Context) {
	var idc models.IDCInfo
	// ShouldBindJSON 是Gin框架的函数, 用于将请求体中的JSON数据绑定到结构体中
	if err := c.ShouldBindJSON(&idc); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	var responseIDC models.IDCInfo
	created := false
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := ensureIDCInfoMachineID(&idc); err != nil {
			return err
		}

		var existingIDC models.IDCInfo
		err := tx.First(&existingIDC, "ipmi_ip = ?", idc.IPMIIP).Error
		if err == nil {
			if existingIDC.MachineID == "" {
				if err := ensureIDCInfoMachineID(&existingIDC); err != nil {
					return err
				}
				if err := tx.Model(&existingIDC).Update("machine_id", existingIDC.MachineID).Error; err != nil {
					return err
				}
			}
			if err := updateIDCAndRelatedZbxID(tx, &existingIDC, idc); err != nil {
				return err
			}
			if err := tx.First(&responseIDC, existingIDC.ID).Error; err != nil {
				return err
			}
			if err := touchMachineSyncState(tx, responseIDC.IPMIIP, responseIDC.ZbxID, now); err != nil {
				return err
			}
			return reconcileRestoredArchiveForCurrentMachine(tx, responseIDC.IPMIIP, now)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := restoreMachineFromArchive(tx, idc.IPMIIP, now); err != nil {
			return err
		}
		err = tx.First(&existingIDC, "ipmi_ip = ?", idc.IPMIIP).Error
		if err == nil {
			if err := updateIDCAndRelatedZbxID(tx, &existingIDC, idc); err != nil {
				return err
			}
			if err := tx.First(&responseIDC, existingIDC.ID).Error; err != nil {
				return err
			}
			return touchMachineSyncState(tx, responseIDC.IPMIIP, responseIDC.ZbxID, now)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(&idc).Error; err != nil {
			return err
		}
		created = true
		responseIDC = idc
		if err := touchMachineSyncState(tx, responseIDC.IPMIIP, responseIDC.ZbxID, now); err != nil {
			return err
		}
		return reconcileRestoredArchiveForCurrentMachine(tx, responseIDC.IPMIIP, now)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "保存机器失败: " + err.Error(),
		})
		return
	}

	database.CacheDel(c.Request.Context(), cacheKey(responseIDC.IPMIIP), networkCacheKey(responseIDC.IPMIIP))

	if !created {
		c.JSON(http.StatusOK, models.Response{
			Code:    200,
			Message: "机器已存在，已按ipmi_ip更新",
			Data:    responseIDC,
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "创建成功",
		Data:    responseIDC,
	})
}

// UpdateMachine 更新机器信息
// 根据ipmi_ip更新机器信息
func UpdateMachine(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip") // 从URL中获取ipmi_ip

	// 查找现有的IDC信息
	var existingIDC models.IDCInfo
	if err := database.DB.First(&existingIDC, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "机器不存在" + err.Error(),
		})
		return
	}

	// 绑定请求数据
	var updateIDC models.IDCInfo
	if err := c.ShouldBindJSON(&updateIDC); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	updateData := buildIDCUpdateData(updateIDC)

	// 如果没有任何字段需要更新
	if len(updateData) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "没有提供需要更新的字段",
		})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		idc, err := resolveCurrentMachineIdentity(tx, ipmiIP)
		if err != nil {
			return err
		}
		existingIDC = idc
		return updateIDCAndRelatedZbxID(tx, &existingIDC, updateIDC)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新IDC信息失败: " + err.Error(),
		})
		return
	}

	// 重新查询更新后的数据用于返回
	database.DB.First(&updateIDC, existingIDC.ID)

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))
	database.CacheDel(c.Request.Context(), networkCacheKey(ipmiIP))
	touchMachineSeen(ipmiIP, updateIDC.ZbxID)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "更新成功",
		Data:    updateIDC,
	})
}

// UpdateBusinessInfo 更新机器业务信息
// 根据ipmi_ip更新机器的业务信息
func UpdateBusinessInfo(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ipmi_ip不能为空",
		})
		return
	}

	// 绑定请求数据
	var businessInfo models.BusinessInfo
	if err := c.ShouldBindJSON(&businessInfo); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	var idc models.IDCInfo
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		idc, err = resolveCurrentMachineIdentity(tx, ipmiIP)
		if err != nil {
			return err
		}

		// 设置ipmi_ip和zbx_id
		businessInfo.IPMIIP = ipmiIP
		businessInfo.ZbxID = idc.ZbxID
		businessInfo.MachineID = idc.MachineID

		// 检查业务信息是否已存在
		var existingBusiness models.BusinessInfo
		err = tx.First(&existingBusiness, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error

		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// 业务信息不存在，创建新记录
			return tx.Create(&businessInfo).Error
		}

		// 业务信息已存在，使用部分更新
		// 只更新非零值字段，避免覆盖未传入的字段
		updateData := map[string]interface{}{
			"machine_id": idc.MachineID,
			"ipmi_ip":    ipmiIP,
			"zbx_id":     idc.ZbxID,
		}

		// 检查每个字段是否需要更新（非零值才更新）
		if businessInfo.BusinessName != "" {
			updateData["business_name"] = businessInfo.BusinessName
		}
		if businessInfo.BusinessID != "" {
			updateData["business_id"] = businessInfo.BusinessID
		}
		if businessInfo.OldBusinessName != "" {
			updateData["old_business_name"] = businessInfo.OldBusinessName
		}
		if businessInfo.OldBusinessID != "" {
			updateData["old_business_id"] = businessInfo.OldBusinessID
		}
		if businessInfo.BusinessSpeed != 0 {
			updateData["business_speed"] = businessInfo.BusinessSpeed
		}
		if businessInfo.OldBusinessSpeed != 0 {
			updateData["old_business_speed"] = businessInfo.OldBusinessSpeed
		}

		// 执行部分更新
		if err := tx.Model(&existingBusiness).Updates(updateData).Error; err != nil {
			return err
		}

		// 重新查询更新后的数据用于返回
		return tx.First(&businessInfo, existingBusiness.ID).Error
	}); err != nil {
		status := http.StatusInternalServerError
		message := "保存业务信息失败: " + err.Error()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			message = "机器不存在"
		}
		c.JSON(status, models.Response{
			Code:    status,
			Message: message,
		})
		return
	}

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))
	touchMachineSeen(ipmiIP, idc.ZbxID)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "业务信息更新成功",
		Data:    businessInfo,
	})
}

// UpdateMachineInfo 更新机器硬件信息
// 根据ipmi_ip更新机器的硬件信息
func UpdateMachineInfo(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ipmi_ip不能为空",
		})
		return
	}

	// 绑定请求数据
	var machineInfo models.MachineInfo
	if err := c.ShouldBindJSON(&machineInfo); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	var idc models.IDCInfo
	replaced := false
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		idc, err = resolveCurrentMachineIdentity(tx, ipmiIP)
		if err != nil {
			return err
		}

		// 设置ipmi_ip和zbx_id
		machineInfo.IPMIIP = ipmiIP
		machineInfo.ZbxID = idc.ZbxID
		machineInfo.MachineID = idc.MachineID

		// 检查机器硬件信息是否已存在
		var existingMachine models.MachineInfo
		err = tx.First(&existingMachine, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error

		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// 机器硬件信息不存在，创建新记录
			if err := tx.Create(&machineInfo).Error; err != nil {
				return err
			}
			now := time.Now()
			if err := touchMachineSyncState(tx, ipmiIP, idc.ZbxID, now); err != nil {
				return err
			}
			return reconcileRestoredArchiveForCurrentMachine(tx, ipmiIP, now)
		}

		if existingMachine.ServerSN != "" && machineInfo.ServerSN != "" && existingMachine.ServerSN != machineInfo.ServerSN {
			now := time.Now()
			if _, err := archiveMachineByIPMI(tx, ipmiIP, "replacement", now); err != nil {
				return err
			}
			newID, err := newMachineID()
			if err != nil {
				return err
			}
			idc.ID = 0
			idc.MachineID = newID
			idc.CreatedAt = time.Time{}
			if err := tx.Create(&idc).Error; err != nil {
				return err
			}
			machineInfo.ID = 0
			machineInfo.MachineID = newID
			machineInfo.ZbxID = idc.ZbxID
			machineInfo.CreatedAt = time.Time{}
			if err := tx.Create(&machineInfo).Error; err != nil {
				return err
			}
			replaced = true
			return touchMachineSyncState(tx, ipmiIP, idc.ZbxID, now)
		}

		// 机器硬件信息已存在，使用部分更新
		// 只更新非零值字段，避免覆盖未传入的字段
		updateData := map[string]interface{}{
			"machine_id": idc.MachineID,
			"ipmi_ip":    ipmiIP,
			"zbx_id":     idc.ZbxID,
		}

		// 检查每个字段是否需要更新（非零值才更新）
		if machineInfo.SystemType != "" {
			updateData["system_type"] = machineInfo.SystemType
		}
		if machineInfo.Manufacturer != "" {
			updateData["manufacturer"] = machineInfo.Manufacturer
		}
		if machineInfo.ServerSN != "" {
			updateData["server_sn"] = machineInfo.ServerSN
		}
		if machineInfo.SystemDisk != "" {
			updateData["system_disk"] = machineInfo.SystemDisk
		}
		if machineInfo.SSDCount != "" {
			updateData["ssd_count"] = machineInfo.SSDCount
		}
		if machineInfo.HDDCount != "" {
			updateData["hdd_count"] = machineInfo.HDDCount
		}
		if machineInfo.SysHDDCount != "" {
			updateData["sys_hdd_count"] = machineInfo.SysHDDCount
		}
		if machineInfo.MemoryCount != "" {
			updateData["memory_count"] = machineInfo.MemoryCount
		}
		if machineInfo.CPUInfo != "" {
			updateData["cpu_info"] = machineInfo.CPUInfo
		}
		if machineInfo.ServerHeight != "" {
			updateData["server_height"] = machineInfo.ServerHeight
		}
		if machineInfo.SwitchPort != "" {
			updateData["switch_port"] = machineInfo.SwitchPort
		}

		// 执行部分更新
		if err := tx.Model(&existingMachine).Updates(updateData).Error; err != nil {
			return err
		}

		// 重新查询更新后的数据用于返回
		if err := tx.First(&machineInfo, existingMachine.ID).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := touchMachineSyncState(tx, ipmiIP, idc.ZbxID, now); err != nil {
			return err
		}
		return reconcileRestoredArchiveForCurrentMachine(tx, ipmiIP, now)
	}); err != nil {
		status := http.StatusInternalServerError
		message := "保存机器硬件信息失败: " + err.Error()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			message = "机器不存在"
		}
		c.JSON(status, models.Response{
			Code:    status,
			Message: message,
		})
		return
	}

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))
	database.CacheDel(c.Request.Context(), networkCacheKey(ipmiIP))

	message := "机器硬件信息更新成功"
	if replaced {
		message = "检测到序列号变化，旧机器已归档，新机器实例已创建"
	}
	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: message,
		Data:    machineInfo,
	})
}

// DeleteMachine 删除机器
// 根据ipmi_ip删除机器记录，会同时删除idc_info、machine_info、business_info表中的对应数据，但保留network_info表数据
func DeleteMachine(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ipmi_ip不能为空",
		})
		return
	}

	// 检查机器是否存在
	var idc models.IDCInfo
	if err := database.DB.First(&idc, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "机器不存在",
		})
		return
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

	// 1. 删除 machine_info 表中的数据（如果存在）
	if err := tx.Delete(&models.MachineInfo{}, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除机器硬件信息失败",
		})
		return
	}

	// 2. 删除 business_info 表中的数据（如果存在）
	if err := tx.Delete(&models.BusinessInfo{}, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除业务信息失败",
		})
		return
	}

	// 3. 最后删除 idc_info 表中的数据（主表）
	if err := tx.Delete(&models.IDCInfo{}, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除IDC信息失败",
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "提交事务失败",
		})
		return
	}

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "删除成功",
	})
}

// DeleteMachineInfo 删除机器硬件信息
// 根据ipmi_ip删除机器硬件信息，不影响IDC信息
func DeleteMachineInfo(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ipmi_ip不能为空",
		})
		return
	}

	// 检查机器硬件信息是否存在
	var machineInfo models.MachineInfo
	if err := database.DB.First(&machineInfo, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "机器硬件信息不存在",
		})
		return
	}

	// 删除机器硬件信息
	if err := database.DB.Delete(&machineInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除机器硬件信息失败",
		})
		return
	}

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "删除机器硬件信息成功",
	})
}

// DeleteBusinessInfo 删除业务信息
// 根据ipmi_ip删除业务信息，不影响IDC信息
func DeleteBusinessInfo(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ipmi_ip不能为空",
		})
		return
	}

	// 检查业务信息是否存在
	var businessInfo models.BusinessInfo
	if err := database.DB.First(&businessInfo, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "业务信息不存在",
		})
		return
	}

	// 删除业务信息
	if err := database.DB.Delete(&businessInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除业务信息失败",
		})
		return
	}

	// 清除缓存
	database.CacheDel(c.Request.Context(), cacheKey(ipmiIP))

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "删除业务信息成功",
	})
}

// SearchMachinesByIDC 多字段搜索机器
// 支持按机房名称、机房编码、ZBX_ID、IP地址等多字段模糊查询机器信息
func SearchMachinesByIDC(c *gin.Context) {
	// 获取所有可能的搜索参数
	idcName := strings.TrimSpace(c.Query("idc_name"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	zbxID := strings.TrimSpace(c.Query("zbx_id"))
	ipmiIP := strings.TrimSpace(c.Query("ipmi_ip"))
	sshIP := strings.TrimSpace(c.Query("ssh_ip"))

	// 支持通过业务ID查询（兼容 business_id 和 business_Id 两种写法）
	businessID := strings.TrimSpace(c.Query("business_id"))
	if businessID == "" {
		businessID = strings.TrimSpace(c.Query("business_Id"))
	}

	// 至少需要一个搜索条件
	if idcName == "" && idcCode == "" && zbxID == "" && ipmiIP == "" && sshIP == "" && businessID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "至少需要提供一个搜索条件（idc_name、idc_code、zbx_id、ipmi_ip、ssh_ip、business_id）",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 301 {
		size = 300
	}

	var total int64
	var results []gin.H

	// 构建动态查询条件
	query := database.DB.Model(&models.IDCInfo{})
	var conditions []string
	var args []interface{}

	if idcName != "" {
		conditions = append(conditions, "idc_name ILIKE ?")
		args = append(args, "%"+idcName+"%")
	}
	if idcCode != "" {
		conditions = append(conditions, "idc_code ILIKE ?")
		args = append(args, "%"+idcCode+"%")
	}
	if zbxID != "" {
		conditions = append(conditions, "zbx_id ILIKE ?")
		args = append(args, "%"+zbxID+"%")
	}
	if ipmiIP != "" {
		conditions = append(conditions, "ipmi_ip ILIKE ?")
		args = append(args, "%"+ipmiIP+"%")
	}
	if sshIP != "" {
		conditions = append(conditions, "ssh_ip ILIKE ?")
		args = append(args, "%"+sshIP+"%")
	}

	// 通过业务ID关联 business_info 表查询对应机器
	if businessID != "" {
		conditions = append(conditions,
			"EXISTS (SELECT 1 FROM business_info b WHERE (b.machine_id = idc_info.machine_id OR (b.machine_id = '' AND b.ipmi_ip = idc_info.ipmi_ip)) AND b.business_id ILIKE ?)")
		args = append(args, "%"+businessID+"%")
	}

	// 使用OR连接所有条件（任意一个匹配即可）
	whereClause := strings.Join(conditions, " OR ")
	query = query.Where(whereClause, args...)

	// 查询匹配的IDC信息
	var idcInfos []models.IDCInfo

	// 查询总数
	query.Count(&total)

	// 分页查询
	query.Offset((page - 1) * size).Limit(size).Find(&idcInfos)

	// 为每个IDC信息查询关联的机器和业务信息
	for _, idc := range idcInfos {
		machine, business, networks := loadMachineRelations(idc)

		result := gin.H{
			"idc_info":      idc,
			"machine_info":  machine,
			"business_info": business,
			"network_info":  networks,
		}
		results = append(results, result)
	}

	// 构建搜索条件描述
	var searchDesc []string
	if idcName != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("机房名称:'%s'", idcName))
	}
	if idcCode != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("机房编码:'%s'", idcCode))
	}
	if zbxID != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("ZBX_ID:'%s'", zbxID))
	}
	if ipmiIP != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IPMI_IP:'%s'", ipmiIP))
	}
	if sshIP != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("SSH_IP:'%s'", sshIP))
	}
	if businessID != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("业务ID:'%s'", businessID))
	}

	response := gin.H{
		"total":           total,
		"page":            page,
		"size":            size,
		"search_criteria": strings.Join(searchDesc, ", "),
		"list":            results,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("搜索成功，找到%d条记录", total),
		Data:    response,
	})
}

// SearchMachinesByManufacturer 根据厂商名称模糊搜索机器
// 支持按制造商（Manufacturer）字段模糊查询机器硬件信息
func SearchMachinesByManufacturer(c *gin.Context) {
	// 获取查询参数
	manufacturer := strings.TrimSpace(c.Query("manufacturer"))

	// 检查参数是否为空
	if manufacturer == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "manufacturer参数不能为空",
		})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 301 {
		size = 300
	}

	var total int64
	var results []gin.H

	// 先查询 machine_info 表，使用 ILIKE 进行模糊匹配（不区分大小写）
	var machineInfos []models.MachineInfo
	query := database.DB.Model(&models.MachineInfo{}).Where("manufacturer ILIKE ?", "%"+manufacturer+"%")

	// 查询总数
	query.Count(&total)

	// 分页查询
	query.Offset((page - 1) * size).Limit(size).Find(&machineInfos)

	// 为每个机器硬件信息查询关联的IDC、业务和网络信息
	for _, machine := range machineInfos {
		var idc models.IDCInfo

		if machine.MachineID != "" {
			database.DB.First(&idc, "machine_id = ?", machine.MachineID)
		} else {
			database.DB.First(&idc, "ipmi_ip = ?", machine.IPMIIP)
		}
		_, business, networks := loadMachineRelations(idc)

		result := gin.H{
			"idc_info":      idc,
			"machine_info":  machine,
			"business_info": business,
			"network_info":  networks,
		}
		results = append(results, result)
	}

	response := gin.H{
		"total":           total,
		"page":            page,
		"size":            size,
		"search_criteria": fmt.Sprintf("制造商:'%s'", manufacturer),
		"list":            results,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("搜索成功，找到%d条记录", total),
		Data:    response,
	})
}
