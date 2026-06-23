package handlers

import (
	"context"
	"errors"
	"fmt"
	"gin-postgre-project/database"
	"gin-postgre-project/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	staleThreshold   = 24 * time.Hour
	archiveGrace     = 12 * time.Hour
	archiveRetention = 30 * 24 * time.Hour
)

func machineArchiveBatchID(machineID string, ipmiIP string, archivedAt time.Time) string {
	identity := machineID
	if identity == "" {
		identity = ipmiIP
	}
	safeIdentity := strings.NewReplacer(".", "-", ":", "-", "_", "-").Replace(identity)
	return fmt.Sprintf("%s-%d", safeIdentity, archivedAt.UnixNano())
}

func touchMachineSyncState(tx *gorm.DB, ipmiIP string, zbxID string, seenAt time.Time) error {
	if ipmiIP == "" {
		return nil
	}

	machineID := ""
	idc, err := resolveCurrentMachineIdentity(tx, ipmiIP)
	if err == nil {
		machineID = idc.MachineID
		if zbxID == "" {
			zbxID = idc.ZbxID
		}
	}

	state := models.MachineSyncState{
		MachineID:  machineID,
		IPMIIP:     ipmiIP,
		ZbxID:      zbxID,
		Status:     models.MachineStatusActive,
		LastSeenAt: seenAt,
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ipmi_ip"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"machine_id":     machineID,
			"zbx_id":         zbxID,
			"status":         models.MachineStatusActive,
			"last_seen_at":   seenAt,
			"first_stale_at": nil,
			"archived_at":    nil,
			"updated_at":     seenAt,
		}),
	}).Create(&state).Error
}

func touchMachineSeen(ipmiIP string, zbxID string) {
	if ipmiIP == "" {
		return
	}

	_ = database.DB.Transaction(func(tx *gorm.DB) error {
		if zbxID == "" {
			var idc models.IDCInfo
			if err := tx.First(&idc, "ipmi_ip = ?", ipmiIP).Error; err == nil {
				zbxID = idc.ZbxID
			}
		}
		return touchMachineSyncState(tx, ipmiIP, zbxID, time.Now())
	})
}

func archiveMachineByIPMI(tx *gorm.DB, ipmiIP string, reason string, archivedAt time.Time) (*models.MachineArchiveBatch, error) {
	idc, err := resolveCurrentMachineIdentity(tx, ipmiIP)
	if err != nil {
		return nil, err
	}

	expiresAt := archivedAt.Add(archiveRetention)
	batchID := machineArchiveBatchID(idc.MachineID, ipmiIP, archivedAt)

	var state models.MachineSyncState
	var lastSeenAt *time.Time
	if err := tx.First(&state, "ipmi_ip = ?", ipmiIP).Error; err == nil {
		lastSeenAt = &state.LastSeenAt
	}

	batch := models.MachineArchiveBatch{
		ArchiveBatchID: batchID,
		MachineID:      idc.MachineID,
		IPMIIP:         idc.IPMIIP,
		ZbxID:          idc.ZbxID,
		IDCCode:        idc.IDCCode,
		IDCName:        idc.IDCName,
		SSHIP:          idc.SSHIP,
		ArchiveReason:  reason,
		Status:         models.ArchiveStatusArchived,
		LastSeenAt:     lastSeenAt,
		ArchivedAt:     archivedAt,
		ExpiresAt:      expiresAt,
	}
	if err := tx.Create(&batch).Error; err != nil {
		return nil, err
	}

	meta := models.ArchiveMeta{
		OriginalID:     idc.ID,
		ArchiveBatchID: batchID,
		ArchiveReason:  reason,
		ArchivedAt:     archivedAt,
		ExpiresAt:      expiresAt,
	}
	if err := tx.Create(&models.ArchivedIDCInfo{
		ArchiveMeta: meta,
		MachineID:   idc.MachineID,
		ZbxID:       idc.ZbxID,
		IPMIIP:      idc.IPMIIP,
		IDCCode:     idc.IDCCode,
		IDCName:     idc.IDCName,
		SSHIP:       idc.SSHIP,
		CreatedAt:   idc.CreatedAt,
	}).Error; err != nil {
		return nil, err
	}

	var machine models.MachineInfo
	if err := tx.First(&machine, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error; err == nil {
		meta.OriginalID = machine.ID
		if err := tx.Create(&models.ArchivedMachineInfo{
			ArchiveMeta:  meta,
			MachineID:    idc.MachineID,
			ZbxID:        machine.ZbxID,
			IPMIIP:       machine.IPMIIP,
			SystemType:   machine.SystemType,
			Manufacturer: machine.Manufacturer,
			ServerSN:     machine.ServerSN,
			SystemDisk:   machine.SystemDisk,
			SSDCount:     machine.SSDCount,
			HDDCount:     machine.HDDCount,
			SysHDDCount:  machine.SysHDDCount,
			MemoryCount:  machine.MemoryCount,
			CPUInfo:      machine.CPUInfo,
			ServerHeight: machine.ServerHeight,
			SwitchPort:   machine.SwitchPort,
			CreatedAt:    machine.CreatedAt,
		}).Error; err != nil {
			return nil, err
		}
	}

	var business models.BusinessInfo
	if err := tx.First(&business, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error; err == nil {
		meta.OriginalID = business.ID
		if err := tx.Create(&models.ArchivedBusinessInfo{
			ArchiveMeta:      meta,
			MachineID:        idc.MachineID,
			ZbxID:            business.ZbxID,
			IPMIIP:           business.IPMIIP,
			BusinessName:     business.BusinessName,
			BusinessID:       business.BusinessID,
			OldBusinessName:  business.OldBusinessName,
			OldBusinessID:    business.OldBusinessID,
			BusinessSpeed:    business.BusinessSpeed,
			OldBusinessSpeed: business.OldBusinessSpeed,
			CreatedAt:        business.CreatedAt,
		}).Error; err != nil {
			return nil, err
		}
	}

	var networks []models.NetworkInfo
	if err := tx.Where("machine_id = ? OR ipmi_ip = ? OR zbx_id = ?", idc.MachineID, ipmiIP, idc.ZbxID).Find(&networks).Error; err != nil {
		return nil, err
	}
	for _, network := range networks {
		networkMachineID := network.MachineID
		if networkMachineID == nil || *networkMachineID == "" {
			networkMachineID = &idc.MachineID
		}
		meta.OriginalID = network.ID
		if err := tx.Create(&models.ArchivedNetworkInfo{
			ArchiveMeta:  meta,
			MachineID:    networkMachineID,
			IPMIIP:       network.IPMIIP,
			IPv4IP:       network.IPv4IP,
			ZbxID:        network.ZbxID,
			MacAddress:   network.MacAddress,
			EthName:      network.EthName,
			IDCCode:      network.IDCCode,
			NetType:      network.NetType,
			Vlan:         network.Vlan,
			IPv4Gateway:  network.IPv4Gateway,
			IPv6IP:       network.IPv6IP,
			IPv6Gateway:  network.IPv6Gateway,
			IPSpeed:      network.IPSpeed,
			IPStatus:     network.IPStatus,
			IPNotes:      network.IPNotes,
			SegmentNotes: network.SegmentNotes,
			CreatedAt:    network.CreatedAt,
		}).Error; err != nil {
			return nil, err
		}
	}

	if err := tx.Delete(&models.NetworkInfo{}, "machine_id = ? OR ipmi_ip = ? OR zbx_id = ?", idc.MachineID, ipmiIP, idc.ZbxID).Error; err != nil {
		return nil, err
	}
	if err := tx.Delete(&models.BusinessInfo{}, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error; err != nil {
		return nil, err
	}
	if err := tx.Delete(&models.MachineInfo{}, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error; err != nil {
		return nil, err
	}
	if err := tx.Delete(&models.IDCInfo{}, "machine_id = ? OR (machine_id = '' AND ipmi_ip = ?)", idc.MachineID, ipmiIP).Error; err != nil {
		return nil, err
	}

	return &batch, tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ipmi_ip"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"machine_id":            idc.MachineID,
			"zbx_id":                idc.ZbxID,
			"status":                models.MachineStatusArchived,
			"archived_at":           archivedAt,
			"last_archive_batch_id": batchID,
			"updated_at":            archivedAt,
		}),
	}).Create(&models.MachineSyncState{
		MachineID:          idc.MachineID,
		IPMIIP:             ipmiIP,
		ZbxID:              idc.ZbxID,
		Status:             models.MachineStatusArchived,
		LastSeenAt:         archivedAt,
		ArchivedAt:         &archivedAt,
		LastArchiveBatchID: batchID,
	}).Error
}

func restoreMachineFromArchive(tx *gorm.DB, ipmiIP string, restoreAt time.Time) error {
	var batch models.MachineArchiveBatch
	err := tx.Where("ipmi_ip = ? AND status = ? AND expires_at > ?", ipmiIP, models.ArchiveStatusArchived, restoreAt).
		Order("archived_at DESC").
		First(&batch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var exists int64
	if err := tx.Model(&models.IDCInfo{}).Where("ipmi_ip = ?", ipmiIP).Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	var archivedIDC models.ArchivedIDCInfo
	if err := tx.First(&archivedIDC, "archive_batch_id = ?", batch.ArchiveBatchID).Error; err != nil {
		return err
	}

	if err := tx.Create(&models.IDCInfo{
		MachineID: archivedIDC.MachineID,
		ZbxID:     archivedIDC.ZbxID,
		IPMIIP:    archivedIDC.IPMIIP,
		IDCCode:   archivedIDC.IDCCode,
		IDCName:   archivedIDC.IDCName,
		SSHIP:     archivedIDC.SSHIP,
		CreatedAt: archivedIDC.CreatedAt,
	}).Error; err != nil {
		return err
	}

	var archivedMachine models.ArchivedMachineInfo
	if err := tx.First(&archivedMachine, "archive_batch_id = ?", batch.ArchiveBatchID).Error; err == nil {
		if err := tx.Create(&models.MachineInfo{
			MachineID:    archivedMachine.MachineID,
			ZbxID:        archivedMachine.ZbxID,
			IPMIIP:       archivedMachine.IPMIIP,
			SystemType:   archivedMachine.SystemType,
			Manufacturer: archivedMachine.Manufacturer,
			ServerSN:     archivedMachine.ServerSN,
			SystemDisk:   archivedMachine.SystemDisk,
			SSDCount:     archivedMachine.SSDCount,
			HDDCount:     archivedMachine.HDDCount,
			SysHDDCount:  archivedMachine.SysHDDCount,
			MemoryCount:  archivedMachine.MemoryCount,
			CPUInfo:      archivedMachine.CPUInfo,
			ServerHeight: archivedMachine.ServerHeight,
			SwitchPort:   archivedMachine.SwitchPort,
			CreatedAt:    archivedMachine.CreatedAt,
		}).Error; err != nil {
			return err
		}
	}

	var archivedBusiness models.ArchivedBusinessInfo
	if err := tx.First(&archivedBusiness, "archive_batch_id = ?", batch.ArchiveBatchID).Error; err == nil {
		if err := tx.Create(&models.BusinessInfo{
			MachineID:        archivedBusiness.MachineID,
			ZbxID:            archivedBusiness.ZbxID,
			IPMIIP:           archivedBusiness.IPMIIP,
			BusinessName:     archivedBusiness.BusinessName,
			BusinessID:       archivedBusiness.BusinessID,
			OldBusinessName:  archivedBusiness.OldBusinessName,
			OldBusinessID:    archivedBusiness.OldBusinessID,
			BusinessSpeed:    archivedBusiness.BusinessSpeed,
			OldBusinessSpeed: archivedBusiness.OldBusinessSpeed,
			CreatedAt:        archivedBusiness.CreatedAt,
		}).Error; err != nil {
			return err
		}
	}

	var archivedNetworks []models.ArchivedNetworkInfo
	if err := tx.Where("archive_batch_id = ?", batch.ArchiveBatchID).Find(&archivedNetworks).Error; err != nil {
		return err
	}
	for _, archivedNetwork := range archivedNetworks {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.NetworkInfo{
			MachineID:    archivedNetwork.MachineID,
			IPMIIP:       archivedNetwork.IPMIIP,
			IPv4IP:       archivedNetwork.IPv4IP,
			ZbxID:        archivedNetwork.ZbxID,
			MacAddress:   archivedNetwork.MacAddress,
			EthName:      archivedNetwork.EthName,
			IDCCode:      archivedNetwork.IDCCode,
			NetType:      archivedNetwork.NetType,
			Vlan:         archivedNetwork.Vlan,
			IPv4Gateway:  archivedNetwork.IPv4Gateway,
			IPv6IP:       archivedNetwork.IPv6IP,
			IPv6Gateway:  archivedNetwork.IPv6Gateway,
			IPSpeed:      archivedNetwork.IPSpeed,
			IPStatus:     archivedNetwork.IPStatus,
			IPNotes:      archivedNetwork.IPNotes,
			SegmentNotes: archivedNetwork.SegmentNotes,
			CreatedAt:    archivedNetwork.CreatedAt,
		}).Error; err != nil {
			return err
		}
	}

	updateFields := map[string]interface{}{"is_restored": true, "restored_at": restoreAt}
	if err := tx.Model(&models.ArchivedIDCInfo{}).Where("archive_batch_id = ?", batch.ArchiveBatchID).Updates(updateFields).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.ArchivedMachineInfo{}).Where("archive_batch_id = ?", batch.ArchiveBatchID).Updates(updateFields).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.ArchivedBusinessInfo{}).Where("archive_batch_id = ?", batch.ArchiveBatchID).Updates(updateFields).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.ArchivedNetworkInfo{}).Where("archive_batch_id = ?", batch.ArchiveBatchID).Updates(updateFields).Error; err != nil {
		return err
	}

	return tx.Model(&batch).Updates(map[string]interface{}{
		"status":      models.ArchiveStatusRestored,
		"restored_at": restoreAt,
	}).Error
}

func MarkStaleAndArchiveMachines() {
	now := time.Now()
	staleBefore := now.Add(-staleThreshold)
	archiveBefore := now.Add(-archiveGrace)

	var activeStates []models.MachineSyncState
	database.DB.Where("status = ? AND last_seen_at < ?", models.MachineStatusActive, staleBefore).Find(&activeStates)
	for _, state := range activeStates {
		firstStaleAt := now
		database.DB.Model(&state).Updates(map[string]interface{}{
			"status":         models.MachineStatusStale,
			"first_stale_at": firstStaleAt,
		})
	}

	var staleStates []models.MachineSyncState
	database.DB.Where("status = ? AND first_stale_at < ?", models.MachineStatusStale, archiveBefore).Find(&staleStates)
	for _, state := range staleStates {
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			_, err := archiveMachineByIPMI(tx, state.IPMIIP, "stale_timeout", now)
			return err
		})
		if err == nil {
			database.CacheDel(context.Background(), cacheKey(state.IPMIIP), networkCacheKey(state.IPMIIP))
		}
	}
}

func CleanupExpiredMachineArchives() {
	now := time.Now()
	tx := database.DB
	tx.Where("expires_at <= ?", now).Delete(&models.ArchivedNetworkInfo{})
	tx.Where("expires_at <= ?", now).Delete(&models.ArchivedBusinessInfo{})
	tx.Where("expires_at <= ?", now).Delete(&models.ArchivedMachineInfo{})
	tx.Where("expires_at <= ?", now).Delete(&models.ArchivedIDCInfo{})
	tx.Where("expires_at <= ?", now).Delete(&models.MachineArchiveBatch{})
	tx.Where("status = ? AND archived_at <= ?", models.MachineStatusArchived, now.Add(-archiveRetention)).Delete(&models.MachineSyncState{})
}

func ListMachineArchives(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.DefaultQuery("status", models.ArchiveStatusArchived))
	idcCode := strings.TrimSpace(c.Query("idc_code"))
	ipmiIP := strings.TrimSpace(c.Query("ipmi_ip"))
	zbxID := strings.TrimSpace(c.Query("zbx_id"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}

	type MachineArchiveListItem struct {
		models.MachineArchiveBatch `gorm:"embedded"`
		SystemDisk                 string `gorm:"column:system_disk" json:"system_disk"`
		SSDCount                   string `gorm:"column:ssd_count" json:"ssd_count"`
		HDDCount                   string `gorm:"column:hdd_count" json:"hdd_count"`
		SysHDDCount                string `gorm:"column:sys_hdd_count" json:"sys_hdd_count"`
	}

	query := database.DB.Table("machine_archive_batches b").
		Select("b.*, m.system_disk, m.ssd_count, m.hdd_count, m.sys_hdd_count").
		Joins("LEFT JOIN archived_machine_info m ON m.archive_batch_id = b.archive_batch_id").
		Where("b.expires_at > ?", time.Now())
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
		query = query.Where("b.ipmi_ip ILIKE ? OR b.zbx_id ILIKE ? OR b.idc_code ILIKE ? OR b.idc_name ILIKE ? OR b.ssh_ip ILIKE ? OR m.system_disk ILIKE ? OR m.ssd_count ILIKE ? OR m.hdd_count ILIKE ? OR m.sys_hdd_count ILIKE ?", like, like, like, like, like, like, like, like, like)
	}

	var total int64
	var list []MachineArchiveListItem
	query.Count(&total)
	query.Order("b.archived_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total": total,
			"page":  page,
			"size":  size,
			"list":  list,
		},
	})
}

func GetMachineArchive(c *gin.Context) {
	batchID := c.Param("batch_id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, models.Response{Code: 400, Message: "archive_batch_id不能为空"})
		return
	}

	var batch models.MachineArchiveBatch
	if err := database.DB.First(&batch, "archive_batch_id = ?", batchID).Error; err != nil {
		c.JSON(http.StatusNotFound, models.Response{Code: 404, Message: "归档批次不存在"})
		return
	}

	var idc models.ArchivedIDCInfo
	var machine models.ArchivedMachineInfo
	var business models.ArchivedBusinessInfo
	var networks []models.ArchivedNetworkInfo
	database.DB.First(&idc, "archive_batch_id = ?", batchID)
	database.DB.First(&machine, "archive_batch_id = ?", batchID)
	database.DB.First(&business, "archive_batch_id = ?", batchID)
	database.DB.Where("archive_batch_id = ?", batchID).Find(&networks)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"batch":         batch,
			"idc_info":      idc,
			"machine_info":  machine,
			"business_info": business,
			"network_info":  networks,
		},
	})
}

func GetMachineSyncStates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	status := strings.TrimSpace(c.Query("status"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}

	query := database.DB.Model(&models.MachineSyncState{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	var list []models.MachineSyncState
	query.Count(&total)
	query.Order("last_seen_at ASC").Offset((page - 1) * size).Limit(size).Find(&list)

	c.JSON(http.StatusOK, models.Response{
		Code:    200,
		Message: "查询成功",
		Data: gin.H{
			"total": total,
			"page":  page,
			"size":  size,
			"list":  list,
		},
	})
}
