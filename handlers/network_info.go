package handlers

import (
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateNetworkInfo 创建网络信息
// 创建新的网络配置信息，ipv4_ip 和 ipv6_ip 必须唯一
func CreateNetworkInfo(c *gin.Context) {
	var networkInfo models.NetworkInfo
	if err := c.ShouldBindJSON(&networkInfo); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	// 检查至少提供一个 IP 地址
	if (networkInfo.IPv4IP == nil || *networkInfo.IPv4IP == "") &&
		(networkInfo.IPv6IP == nil || *networkInfo.IPv6IP == "") {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "至少需要提供 ipv4_ip 或 ipv6_ip",
		})
		return
	}

	if err := database.DB.Create(&networkInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "创建网络信息失败: " + err.Error(),
		})
		return
	}

	// 清除相关缓存
	if networkInfo.IPMIIP != nil {
		database.CacheDel(c.Request.Context(), networkCacheKey(*networkInfo.IPMIIP))
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "创建网络信息成功",
		Data:    networkInfo,
	})
}

// GetNetworkInfoByID 根据 ID 获取单个网络信息
func GetNetworkInfoByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ID不能为空",
		})
		return
	}

	var networkInfo models.NetworkInfo
	if err := database.DB.First(&networkInfo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "网络信息不存在",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data:    networkInfo,
	})
}

// GetNetworkInfoByIPMI 根据 IPMI IP 获取网络信息列表
func GetNetworkInfoByIPMI(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IPMI IP不能为空",
		})
		return
	}

	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("ipmi_ip = ?", ipmiIP).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("查询成功，找到 %d 条记录", len(networkInfos)),
		Data:    networkInfos,
	})
}

// GetNetworkInfoByZbxID 根据 ZBX ID 获取网络信息列表
func GetNetworkInfoByZbxID(c *gin.Context) {
	zbxID := c.Param("zbx_id")
	if zbxID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ZBX ID不能为空",
		})
		return
	}

	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("zbx_id = ?", zbxID).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("查询成功，找到 %d 条记录", len(networkInfos)),
		Data:    networkInfos,
	})
}

// UpdateNetworkInfo 更新网络信息
// 根据 ID 局部更新网络信息，只更新提供的字段
func UpdateNetworkInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ID不能为空",
		})
		return
	}

	var existingInfo models.NetworkInfo
	if err := database.DB.First(&existingInfo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "网络信息不存在",
		})
		return
	}

	// 使用 map 接收请求数据，支持局部更新
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	// 验证和过滤允许更新的字段（不允许更新 ID）
	allowedFields := map[string]bool{
		"ipmi_ip":       true,
		"ipv4_ip":       true,
		"zbx_id":        true,
		"mac_address":   true,
		"eth_name":      true,
		"idc_code":      true,
		"net_type":      true,
		"vlan":          true,
		"ipv4_gateway":  true,
		"ipv6_ip":       true,
		"ipv6_gateway":  true,
		"ip_speed":      true,
		"ip_status":     true,
		"ip_notes":      true,
		"segment_notes": true,
	}

	// 构建更新数据，只包含允许更新的字段
	updateFields := make(map[string]interface{})
	for field, value := range updateData {
		if allowedFields[field] {
			updateFields[field] = value
		}
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "没有提供有效的更新字段",
		})
		return
	}

	// 使用 GORM 的 Updates 方法进行局部更新
	if err := database.DB.Model(&existingInfo).Updates(updateFields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新网络信息失败: " + err.Error(),
		})
		return
	}

	// 重新查询更新后的数据
	var updatedInfo models.NetworkInfo
	database.DB.First(&updatedInfo, id)

	// 清除相关缓存
	if updatedInfo.IPMIIP != nil {
		database.CacheDel(c.Request.Context(), networkCacheKey(*updatedInfo.IPMIIP))
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "更新网络信息成功",
		Data:    updatedInfo,
	})
}

// DeleteNetworkInfo 删除网络信息
// 根据 ID 删除网络信息
func DeleteNetworkInfo(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ID不能为空",
		})
		return
	}

	// 检查记录是否存在
	var networkInfo models.NetworkInfo
	if err := database.DB.First(&networkInfo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "网络信息不存在",
		})
		return
	}

	if err := database.DB.Delete(&networkInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + err.Error(),
		})
		return
	}

	// 清除相关缓存
	if networkInfo.IPMIIP != nil {
		database.CacheDel(c.Request.Context(), networkCacheKey(*networkInfo.IPMIIP))
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "删除网络信息成功",
	})
}

// DeleteNetworkInfoByIPMI 根据 IPMI IP 批量删除网络信息
// 删除指定 IPMI IP 的所有网卡信息
func DeleteNetworkInfoByIPMI(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IPMI IP不能为空",
		})
		return
	}

	// 先查询要删除的记录
	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("ipmi_ip = ?", ipmiIP).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	// 批量删除
	result := database.DB.Where("ipmi_ip = ?", ipmiIP).Delete(&models.NetworkInfo{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + result.Error.Error(),
		})
		return
	}

	// 清除相关缓存
	database.CacheDel(c.Request.Context(), networkCacheKey(ipmiIP))

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功删除 %d 条网络信息", result.RowsAffected),
		Data: gin.H{
			"deleted_count": result.RowsAffected,
			"ipmi_ip":       ipmiIP,
		},
	})
}

// DeleteNetworkInfoByZbxID 根据 ZBX ID 批量删除网络信息
// 删除指定 ZBX ID 的所有网卡信息
func DeleteNetworkInfoByZbxID(c *gin.Context) {
	zbxID := c.Param("zbx_id")
	if zbxID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ZBX ID不能为空",
		})
		return
	}

	// 先查询要删除的记录
	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("zbx_id = ?", zbxID).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	// 批量删除
	result := database.DB.Where("zbx_id = ?", zbxID).Delete(&models.NetworkInfo{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + result.Error.Error(),
		})
		return
	}

	// 清除相关缓存（可能有多个 ipmi_ip）
	for _, info := range networkInfos {
		if info.IPMIIP != nil {
			database.CacheDel(c.Request.Context(), networkCacheKey(*info.IPMIIP))
		}
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功删除 %d 条网络信息", result.RowsAffected),
		Data: gin.H{
			"deleted_count": result.RowsAffected,
			"zbx_id":        zbxID,
		},
	})
}

// DeleteNetworkInfoByIPv4 根据 IPv4 地址删除网络信息
// 删除指定 IPv4 地址的网络信息（IPv4 唯一，只会删除一条）
// 支持两种方式：1. 路径参数 2. 查询参数（推荐用于包含特殊字符的IP）
func DeleteNetworkInfoByIPv4(c *gin.Context) {
	// 优先从查询参数获取，如果没有则从路径参数获取
	ipv4IP := c.Query("ip")
	if ipv4IP == "" {
		ipv4IP = c.Param("ipv4_ip")
	}

	if ipv4IP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IPv4 地址不能为空",
		})
		return
	}

	// 检查记录是否存在
	var networkInfo models.NetworkInfo
	if err := database.DB.Where("ipv4_ip = ?", ipv4IP).First(&networkInfo).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到该 IPv4 地址的网络信息",
		})
		return
	}

	// 删除记录
	if err := database.DB.Delete(&networkInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + err.Error(),
		})
		return
	}

	// 清除相关缓存
	if networkInfo.IPMIIP != nil {
		database.CacheDel(c.Request.Context(), networkCacheKey(*networkInfo.IPMIIP))
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "删除网络信息成功",
		Data: gin.H{
			"ipv4_ip": ipv4IP,
			"id":      networkInfo.ID,
		},
	})
}

// DeleteNetworkInfoByIDC 根据 IDC 机房代码批量删除网络信息
// 删除指定机房的所有网络信息（危险操作）
func DeleteNetworkInfoByIDC(c *gin.Context) {
	idcCode := c.Param("idc_code")
	if idcCode == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IDC机房代码不能为空",
		})
		return
	}

	// 先查询要删除的记录
	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("idc_code = ?", idcCode).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: fmt.Sprintf("未找到机房代码 %s 的网络信息", idcCode),
		})
		return
	}

	// 批量删除
	result := database.DB.Where("idc_code = ?", idcCode).Delete(&models.NetworkInfo{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "删除网络信息失败: " + result.Error.Error(),
		})
		return
	}

	// 清除相关缓存（可能有多个 ipmi_ip）
	for _, info := range networkInfos {
		if info.IPMIIP != nil {
			database.CacheDel(c.Request.Context(), networkCacheKey(*info.IPMIIP))
		}
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功删除机房 %s 的 %d 条网络信息", idcCode, result.RowsAffected),
		Data: gin.H{
			"deleted_count": result.RowsAffected,
			"idc_code":      idcCode,
		},
	})
}

// ListNetworkInfo 获取网络信息列表
// 分页获取所有网络信息，支持按 IDCCode、IP 状态、网络类型过滤
func ListNetworkInfo(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))
	idcCode := c.Query("idc_code")
	ipStatus := c.Query("ip_status")
	netType := c.Query("net_type")
	ipmiIP := c.Query("ipmi_ip")
	zbxID := c.Query("zbx_id")
	ipv4IP := c.Query("ipv4_ip")
	ipv6IP := c.Query("ipv6_ip")
	macAddress := c.Query("mac_address")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 1000 {
		size = 300
	}

	var total int64
	var networkInfos []models.NetworkInfo

	query := database.DB.Model(&models.NetworkInfo{})

	// 添加过滤条件
	if idcCode != "" {
		query = query.Where("idc_code = ?", idcCode)
	}
	if ipStatus != "" {
		query = query.Where("ip_status = ?", ipStatus)
	}
	if netType != "" {
		query = query.Where("net_type = ?", netType)
	}
	if ipmiIP != "" {
		query = query.Where("ipmi_ip = ?", ipmiIP)
	}
	if zbxID != "" {
		query = query.Where("zbx_id = ?", zbxID)
	}
	if ipv4IP != "" {
		query = query.Where("ipv4_ip = ?", ipv4IP)
	}
	if ipv6IP != "" {
		query = query.Where("ipv6_ip = ?", ipv6IP)
	}
	if macAddress != "" {
		query = query.Where("mac_address = ?", macAddress)
	}

	// 查询总数
	query.Count(&total)

	// 分页查询，按 ID 排序
	query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&networkInfos)

	result := gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"list":  networkInfos,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data:    result,
	})
}

// GetNetworkInfoByIDC 根据 IDC 机房代码获取网络信息
// 获取指定机房的所有网络信息
func GetNetworkInfoByIDC(c *gin.Context) {
	idcCode := c.Param("idc_code")
	if idcCode == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IDC代码不能为空",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 1000 {
		size = 300
	}

	var total int64
	var networkInfos []models.NetworkInfo

	query := database.DB.Model(&models.NetworkInfo{}).Where("idc_code = ?", idcCode)

	// 查询总数
	query.Count(&total)

	// 分页查询
	query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&networkInfos)

	result := gin.H{
		"idc_code": idcCode,
		"total":    total,
		"page":     page,
		"size":     size,
		"list":     networkInfos,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("查询机房 %s 的网络信息成功", idcCode),
		Data:    result,
	})
}

// SearchNetworkInfo 模糊搜索网络信息
// 支持按 IPMIIP、IPv4IP、IPv6IP、MAC 地址、网卡名称、IDC 代码等多字段模糊查询
func SearchNetworkInfo(c *gin.Context) {
	// 获取所有可能的搜索参数
	ipmiIP := strings.TrimSpace(c.Query("ipmi_ip"))
	ipv4IP := strings.TrimSpace(c.Query("ipv4_ip"))
	ipv6IP := strings.TrimSpace(c.Query("ipv6_ip"))
	macAddress := strings.TrimSpace(c.Query("mac_address"))
	ethName := strings.TrimSpace(c.Query("eth_name"))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	zbxID := strings.TrimSpace(c.Query("zbx_id"))
	netType := strings.TrimSpace(c.Query("net_type"))
	ipStatus := strings.TrimSpace(c.Query("ip_status"))

	// 至少需要一个搜索条件
	if ipmiIP == "" && ipv4IP == "" && ipv6IP == "" && macAddress == "" && ethName == "" && idcCode == "" && zbxID == "" && netType == "" && ipStatus == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "至少需要提供一个搜索条件",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 1000 {
		size = 300
	}

	var total int64
	var networkInfos []models.NetworkInfo

	// 构建动态查询条件
	query := database.DB.Model(&models.NetworkInfo{})
	var conditions []string
	var args []interface{}

	if ipmiIP != "" {
		conditions = append(conditions, "ipmi_ip ILIKE ?")
		args = append(args, "%"+ipmiIP+"%")
	}
	if ipv4IP != "" {
		conditions = append(conditions, "ipv4_ip ILIKE ?")
		args = append(args, "%"+ipv4IP+"%")
	}
	if ipv6IP != "" {
		conditions = append(conditions, "ipv6_ip ILIKE ?")
		args = append(args, "%"+ipv6IP+"%")
	}
	if macAddress != "" {
		conditions = append(conditions, "mac_address ILIKE ?")
		args = append(args, "%"+macAddress+"%")
	}
	if ethName != "" {
		conditions = append(conditions, "eth_name ILIKE ?")
		args = append(args, "%"+ethName+"%")
	}
	if idcCode != "" {
		conditions = append(conditions, "idc_code ILIKE ?")
		args = append(args, "%"+idcCode+"%")
	}
	if zbxID != "" {
		conditions = append(conditions, "zbx_id ILIKE ?")
		args = append(args, "%"+zbxID+"%")
	}
	if netType != "" {
		conditions = append(conditions, "net_type ILIKE ?")
		args = append(args, "%"+netType+"%")
	}
	if ipStatus != "" {
		conditions = append(conditions, "ip_status ILIKE ?")
		args = append(args, "%"+ipStatus+"%")
	}

	// 使用 OR 连接所有条件（任意一个匹配即可）
	whereClause := strings.Join(conditions, " OR ")
	query = query.Where(whereClause, args...)

	// 查询总数
	query.Count(&total)

	// 分页查询
	query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&networkInfos)

	// 构建搜索条件描述
	var searchDesc []string
	if ipmiIP != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IPMI_IP:'%s'", ipmiIP))
	}
	if ipv4IP != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IPv4_IP:'%s'", ipv4IP))
	}
	if ipv6IP != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IPv6_IP:'%s'", ipv6IP))
	}
	if macAddress != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("MAC地址:'%s'", macAddress))
	}
	if ethName != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("网卡名称:'%s'", ethName))
	}
	if idcCode != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IDC代码:'%s'", idcCode))
	}
	if zbxID != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("ZBX_ID:'%s'", zbxID))
	}
	if netType != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("网络类型:'%s'", netType))
	}
	if ipStatus != "" {
		searchDesc = append(searchDesc, fmt.Sprintf("IP状态:'%s'", ipStatus))
	}

	result := gin.H{
		"total":           total,
		"page":            page,
		"size":            size,
		"search_criteria": strings.Join(searchDesc, ", "),
		"list":            networkInfos,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("搜索成功，找到%d条记录", total),
		Data:    result,
	})
}

// GetNetworkInfoStats 获取网络信息统计
// 按 IDC 机房统计网络信息数量
func GetNetworkInfoStats(c *gin.Context) {
	type IDCStats struct {
		IDCCode string `json:"idc_code"`
		Count   int64  `json:"count"`
	}

	var stats []IDCStats
	if err := database.DB.Model(&models.NetworkInfo{}).
		Select("idc_code, count(*) as count").
		Where("idc_code IS NOT NULL AND idc_code != ''").
		Group("idc_code").
		Order("count DESC").
		Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询统计信息失败: " + err.Error(),
		})
		return
	}

	// 获取总数
	var total int64
	database.DB.Model(&models.NetworkInfo{}).Count(&total)

	result := gin.H{
		"total_count": total,
		"idc_stats":   stats,
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询统计信息成功",
		Data:    result,
	})
}

// ClearAllNetworkInfo 清空所有网络信息
// 删除 network_info 表中的所有数据（危险操作，需要管理员权限）
func ClearAllNetworkInfo(c *gin.Context) {
	// 执行删除操作
	result := database.DB.Exec("DELETE FROM network_info")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "清空网络信息失败: " + result.Error.Error(),
		})
		return
	}

	// 清除所有相关缓存
	database.CacheFlushDB(c.Request.Context())

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功清空所有网络信息，共删除 %d 条记录", result.RowsAffected),
		Data: gin.H{
			"deleted_count": result.RowsAffected,
		},
	})
}

// UpdateNetworkInfoByIPv4 根据 IPv4 地址更新网络信息
// IPv4 地址唯一，只会更新一条记录
// 支持两种方式：1. 路径参数 2. 查询参数（推荐用于包含特殊字符的IP）
func UpdateNetworkInfoByIPv4(c *gin.Context) {
	// 优先从查询参数获取，如果没有则从路径参数获取
	ipv4IP := c.Query("ip")
	if ipv4IP == "" {
		ipv4IP = c.Param("ipv4_ip")
	}

	if ipv4IP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IPv4 地址不能为空",
		})
		return
	}

	// 检查记录是否存在
	var existingInfo models.NetworkInfo
	if err := database.DB.Where("ipv4_ip = ?", ipv4IP).First(&existingInfo).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到该 IPv4 地址的网络信息",
		})
		return
	}

	// 使用 map 接收请求数据，支持局部更新
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	// 验证和过滤允许更新的字段
	allowedFields := map[string]bool{
		"ipmi_ip":       true,
		"ipv4_ip":       true,
		"zbx_id":        true,
		"mac_address":   true,
		"eth_name":      true,
		"idc_code":      true,
		"net_type":      true,
		"vlan":          true,
		"ipv4_gateway":  true,
		"ipv6_ip":       true,
		"ipv6_gateway":  true,
		"ip_speed":      true,
		"ip_status":     true,
		"ip_notes":      true,
		"segment_notes": true,
	}

	// 构建更新数据
	updateFields := make(map[string]interface{})
	for field, value := range updateData {
		if allowedFields[field] {
			updateFields[field] = value
		}
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "没有提供有效的更新字段",
		})
		return
	}

	// 执行更新
	if err := database.DB.Model(&existingInfo).Updates(updateFields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "更新网络信息失败: " + err.Error(),
		})
		return
	}

	// 重新查询更新后的数据
	var updatedInfo models.NetworkInfo
	database.DB.Where("ipv4_ip = ?", ipv4IP).First(&updatedInfo)

	// 清除相关缓存
	if updatedInfo.IPMIIP != nil {
		database.CacheDel(c.Request.Context(), networkCacheKey(*updatedInfo.IPMIIP))
	}

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "更新网络信息成功",
		Data:    updatedInfo,
	})
}

// UpdateNetworkInfoByIPMI 根据 IPMI IP 批量更新网络信息
// 更新指定 IPMI IP 的所有网卡信息（批量操作）
func UpdateNetworkInfoByIPMI(c *gin.Context) {
	ipmiIP := c.Param("ipmi_ip")
	if ipmiIP == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "IPMI IP不能为空",
		})
		return
	}

	// 先查询要更新的记录
	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("ipmi_ip = ?", ipmiIP).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	// 使用 map 接收请求数据
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	// 验证和过滤允许更新的字段（不允许更新 ipmi_ip 本身，防止误操作）
	allowedFields := map[string]bool{
		"zbx_id":        true,
		"idc_code":      true,
		"net_type":      true,
		"ip_status":     true,
		"segment_notes": true,
	}

	// 构建更新数据
	updateFields := make(map[string]interface{})
	for field, value := range updateData {
		if allowedFields[field] {
			updateFields[field] = value
		}
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "没有提供有效的更新字段。批量更新只允许更新: zbx_id, idc_code, net_type, ip_status, segment_notes",
		})
		return
	}

	// 批量更新
	result := database.DB.Model(&models.NetworkInfo{}).
		Where("ipmi_ip = ?", ipmiIP).
		Updates(updateFields)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "批量更新失败: " + result.Error.Error(),
		})
		return
	}

	// 清除相关缓存
	database.CacheDel(c.Request.Context(), networkCacheKey(ipmiIP))

	// 重新查询更新后的数据
	var updatedInfos []models.NetworkInfo
	database.DB.Where("ipmi_ip = ?", ipmiIP).Find(&updatedInfos)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功批量更新 %d 条网络信息", result.RowsAffected),
		Data: gin.H{
			"updated_count": result.RowsAffected,
			"ipmi_ip":       ipmiIP,
			"list":          updatedInfos,
		},
	})
}

// UpdateNetworkInfoByZbxID 根据 ZBX ID 批量更新网络信息
// 更新指定 ZBX ID 的所有网卡信息（批量操作）
func UpdateNetworkInfoByZbxID(c *gin.Context) {
	zbxID := c.Param("zbx_id")
	if zbxID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "ZBX ID不能为空",
		})
		return
	}

	// 先查询要更新的记录
	var networkInfos []models.NetworkInfo
	if err := database.DB.Where("zbx_id = ?", zbxID).Find(&networkInfos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	if len(networkInfos) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code:    404,
			Message: "未找到相关网络信息",
		})
		return
	}

	// 使用 map 接收请求数据
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "请求数据格式错误: " + err.Error(),
		})
		return
	}

	// 验证和过滤允许更新的字段（不允许更新 zbx_id 本身，防止误操作）
	allowedFields := map[string]bool{
		"ipmi_ip":       true,
		"idc_code":      true,
		"net_type":      true,
		"ip_status":     true,
		"segment_notes": true,
	}

	// 构建更新数据
	updateFields := make(map[string]interface{})
	for field, value := range updateData {
		if allowedFields[field] {
			updateFields[field] = value
		}
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Code:    400,
			Message: "没有提供有效的更新字段。批量更新只允许更新: ipmi_ip, idc_code, net_type, ip_status, segment_notes",
		})
		return
	}

	// 批量更新
	result := database.DB.Model(&models.NetworkInfo{}).
		Where("zbx_id = ?", zbxID).
		Updates(updateFields)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code:    500,
			Message: "批量更新失败: " + result.Error.Error(),
		})
		return
	}

	// 清除相关缓存
	for _, info := range networkInfos {
		if info.IPMIIP != nil {
			database.CacheDel(c.Request.Context(), networkCacheKey(*info.IPMIIP))
		}
	}

	// 重新查询更新后的数据
	var updatedInfos []models.NetworkInfo
	database.DB.Where("zbx_id = ?", zbxID).Find(&updatedInfos)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: fmt.Sprintf("成功批量更新 %d 条网络信息", result.RowsAffected),
		Data: gin.H{
			"updated_count": result.RowsAffected,
			"zbx_id":        zbxID,
			"list":          updatedInfos,
		},
	})
}

// networkCacheKey 生成网络信息缓存键
func networkCacheKey(ipmiIP string) string {
	return fmt.Sprintf("network_info:%s", ipmiIP)
}

