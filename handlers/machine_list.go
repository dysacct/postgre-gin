// ListMachines 资产机器分页列表
// 支持全局搜索、机房过滤、业务优先级排序，带Redis缓存

package handlers

import (
	"encoding/json"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 列表项结构体
type MachineListItem struct {
	IDCInfo      models.IDCInfo       `json:"idc_info"`               // 不用指针是值拷贝，指针是引用拷贝
	MachineInfo  *models.MachineInfo  `json:"machine_info,omitempty"` // 用指针是可选字段，不传就不返回
	BusinessInfo *models.BusinessInfo `json:"business_info,omitempty"`
	NetworkInfo  []models.NetworkInfo `json:"network_info,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
}

// 列表响应结构体
type ListResponse struct {
	models.Response
	Data struct {
		Total int               `json:"total"`
		Page  int               `json:"page"`
		Size  int               `json:"size"`
		List  []MachineListItem `json:"list"`
	} `json:"data"`
}

// 固态， 机械， 业务 模糊查询
// 用于Scan的中间结构体
type ScanResult struct {
	// IDC信息
	ID        uint      `gorm:"column:id"`
	ZbxID     string    `gorm:"column:zbx_id"`
	IDCCode   string    `gorm:"column:idc_code"`
	IDCName   string    `gorm:"column:idc_name"`
	IPMIIP    string    `gorm:"column:ipmi_ip"`
	SSHIP     string    `gorm:"column:ssh_ip"`
	CreatedAt time.Time `gorm:"column:created_at"`

	// Machine信息 (可能为空)
	MachineID        *uint      `gorm:"column:m.id"`
	SystemType       *string    `gorm:"column:system_type"`
	Manufacturer     *string    `gorm:"column:manufacturer"`
	ServerSN         *string    `gorm:"column:server_sn"`
	SystemDisk       *string    `gorm:"column:system_disk"`
	SSDCount         *string    `gorm:"column:ssd_count"`
	HDDCount         *string    `gorm:"column:hdd_count"`
	MemoryCount      *string    `gorm:"column:memory_count"`
	CPUInfo          *string    `gorm:"column:cpu_info"`
	ServerHeight     *string    `gorm:"column:server_height"`
	MachineCreatedAt *time.Time `gorm:"column:m.created_at"`

	// Business信息 (可能为空)
	BusinessID        *uint      `gorm:"column:b.id"`
	BusinessName      *string    `gorm:"column:business_name"`
	BusinessIDField   *string    `gorm:"column:business_id"`
	OldBusinessName   *string    `gorm:"column:old_business_name"`
	OldBusinessID     *string    `gorm:"column:old_business_id"`
	BusinessSpeed     *int16     `gorm:"column:business_speed"`
	OldBusinessSpeed  *int16     `gorm:"column:old_business_speed"`
	BusinessCreatedAt *time.Time `gorm:"column:b.created_at"`

	// Network信息 (JSON字符串)
	NetworkInfo interface{} `gorm:"column:network_info"`
}

// 缓存键生成 (包含所有查询参数，保证不同条件不同缓存)
func listCacheKey(page, size int, c *gin.Context) string {
	// 构建标准化的查询字符串，确保默认值也包含在缓存键中
	params := c.Request.URL.Query()
	params.Set("page", strconv.Itoa(page))
	params.Set("size", strconv.Itoa(size))
	return "cache:machine:list:" + params.Encode()
}

func ListMachines(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "300"))
	search := strings.TrimSpace(c.Query("search")) // strings.TrimSpace 是去除字符串两端的空格
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 301 {
		size = 300
	}

	ctx := c.Request.Context()
	cacheKey := listCacheKey(page, size, c)

	// Step 1: 尝试读 Redis 缓存
	if cached := database.CacheGet(ctx, cacheKey); cached.Err() == nil {
		var resp ListResponse
		json.Unmarshal([]byte(cached.Val()), &resp)
		resp.Code = 200
		resp.Message = "缓存命中"
		// 确保返回当前请求的分页参数
		resp.Data.Page = page
		resp.Data.Size = size
		c.JSON(200, resp)
		return
	}

	// Step 2: 缓存miss -> 查数据库 (多表 JSON)
	var total int64
	var scanResults []ScanResult

	query := database.DB.Table("idc_info i").
		Select(`i.id, i.zbx_id, i.idc_code, i.idc_name, i.ipmi_ip, i.ssh_ip, i.created_at,
						m.id as "m.id", m.system_type, m.manufacturer, m.server_sn, m.system_disk, 
						m.ssd_count, m.hdd_count, m.memory_count, m.cpu_info, m.server_height, m.created_at as "m.created_at",
						b.id as "b.id", b.business_name, b.business_id, b.old_business_name, b.old_business_id, 
						b.business_speed, b.old_business_speed, b.created_at as "b.created_at",
						COALESCE(n.networks, '[]') as network_info`).
		Joins("LEFT JOIN machine_info m ON m.ipmi_ip = i.ipmi_ip").
		Joins("LEFT JOIN business_info b ON b.ipmi_ip = i.ipmi_ip").
		Joins(`LEFT JOIN (
			    SELECT zbx_id, json_agg(network_info.*) as networks
					FROM network_info GROUP BY zbx_id
			) n ON n.zbx_id = i.zbx_id`)

	// 全局搜索 (任意字段模糊匹配)
	if search != "" {
		like := "%" + search + "%"
		// i.zbx_id 到 b.business_name 这个i和b.是连接表的别名，i是连接idc_info表，b是连接business_info表，还可以n.是连接network_info表。
		query = query.Where(`i.zbx_id ILIKE ? OR
												 i.idc_name ILIKE ? OR
												 i.ipmi_ip ILIKE ? OR
												 i.ssh_ip ILIKE ? OR
												 m.ssd_count ILIKE ? OR
												 m.hdd_count ILIKE ? OR
												 b.business_name ILIKE ? OR
												 EXISTS (SELECT 1 FROM network_info n WHERE n.zbx_id = i.zbx_id AND (
												 	   n.ipv4_ip ILIKE ? OR n.mac_address ILIKE ?
												 ))`, like, like, like, like, like, like, like, like, like)
	}

	// 其他过滤条件
	if idc := strings.TrimSpace(c.Query("idc_code")); idc != "" {
		like := "%" + idc + "%"
		query = query.Where("i.idc_code ILIKE ?", like)
	}
	// 支持通过机房名称查询 (改为模糊匹配)
	if idcName := strings.TrimSpace(c.Query("idc_name")); idcName != "" {
		like := "%" + idcName + "%"
		query = query.Where("i.idc_name ILIKE ?", like)
	}
	if speed := c.Query("business_speed"); speed != "" {
		query = query.Where("b.business_speed = ?", speed)
	}

	// 新增模糊查询条件
	// 业务名称模糊查询 (支持通过业务名查询关联的机房信息)
	if businessName := strings.TrimSpace(c.Query("business_name")); businessName != "" {
		like := "%" + businessName + "%"
		query = query.Where("b.business_name ILIKE ?", like)
	}

	// 固态硬盘数量模糊查询
	if ssdCount := strings.TrimSpace(c.Query("ssd_count")); ssdCount != "" {
		like := "%" + ssdCount + "%"
		query = query.Where("m.ssd_count ILIKE ?", like)
	}

	// 机械硬盘数量模糊查询
	if hddCount := strings.TrimSpace(c.Query("hdd_count")); hddCount != "" {
		like := "%" + hddCount + "%"
		query = query.Where("m.hdd_count ILIKE ?", like)
	}

	// 服务器高度模糊查询 (如: 1U, 2U, 4U等)
	if serverHeight := strings.TrimSpace(c.Query("server_height")); serverHeight != "" {
		like := "%" + serverHeight + "%"
		query = query.Where("m.server_height ILIKE ?", like)
	}

	// 排序
	order := c.DefaultQuery("order", "i.created_at desc")
	query = query.Order(order)

	// 先查总数
	query.Count(&total)

	// 再查分页数据
	query.Offset((page - 1) * size).Limit(size).Scan(&scanResults)

	// 手动映射到最终结构 (GORM 多表复杂查询需要)
	var result []MachineListItem
	for _, row := range scanResults {
		item := MachineListItem{
			IDCInfo: models.IDCInfo{
				ID:        row.ID,
				ZbxID:     row.ZbxID,
				IDCCode:   row.IDCCode,
				IDCName:   row.IDCName,
				IPMIIP:    row.IPMIIP,
				SSHIP:     row.SSHIP,
				CreatedAt: row.CreatedAt,
			},
			CreatedAt: row.CreatedAt,
		}

		// 构建Machine信息 (如果存在)
		if row.MachineID != nil {
			item.MachineInfo = &models.MachineInfo{
				ID:           *row.MachineID,
				ZbxID:        row.ZbxID,
				IPMIIP:       row.IPMIIP, // 添加缺失的IPMI IP字段
				SystemType:   *row.SystemType,
				Manufacturer: *row.Manufacturer,
				ServerSN:     *row.ServerSN,
				SystemDisk:   *row.SystemDisk,
				SSDCount:     *row.SSDCount,
				HDDCount:     *row.HDDCount,
				MemoryCount:  *row.MemoryCount,
				CPUInfo:      *row.CPUInfo,
				ServerHeight: *row.ServerHeight,
				CreatedAt:    *row.MachineCreatedAt,
			}
		}

		// 构建Business信息 (如果存在)
		if row.BusinessID != nil {
			item.BusinessInfo = &models.BusinessInfo{
				ID:               *row.BusinessID,
				ZbxID:            row.ZbxID,
				IPMIIP:           row.IPMIIP, // 添加缺失的IPMI IP字段
				BusinessName:     *row.BusinessName,
				BusinessID:       *row.BusinessIDField,
				OldBusinessName:  *row.OldBusinessName,
				OldBusinessID:    *row.OldBusinessID,
				BusinessSpeed:    *row.BusinessSpeed,
				OldBusinessSpeed: *row.OldBusinessSpeed,
				CreatedAt:        *row.BusinessCreatedAt,
			}
		}

		// 解析Network信息
		var networks []models.NetworkInfo
		if row.NetworkInfo != nil {
			// 处理不同类型的返回值
			if netJSON, ok := row.NetworkInfo.(string); ok && netJSON != "[]" && netJSON != "" {
				json.Unmarshal([]byte(netJSON), &networks)
			} else if netBytes, ok := row.NetworkInfo.([]byte); ok && len(netBytes) > 0 {
				json.Unmarshal(netBytes, &networks)
			}
		}
		item.NetworkInfo = networks

		result = append(result, item)
	}

	// Step 3: 写入 Redis 缓存 5分钟
	resp := ListResponse{
		Response: models.Response{
			Code:    200,
			Message: "查询成功",
		},
	}
	resp.Data.Total = int(total)
	resp.Data.Page = page
	resp.Data.Size = size
	resp.Data.List = result

	jsonData, _ := json.Marshal(resp)
	database.CacheSet(ctx, cacheKey, jsonData, 5*time.Minute)
	c.JSON(200, resp)
}

