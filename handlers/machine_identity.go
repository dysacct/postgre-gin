package handlers

import (
	"crypto/rand"
	"fmt"
	"gin-postgre-project/models"

	"gorm.io/gorm"
)

func newMachineID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func ensureIDCInfoMachineID(idc *models.IDCInfo) error {
	if idc.MachineID != "" {
		return nil
	}
	machineID, err := newMachineID()
	if err != nil {
		return err
	}
	idc.MachineID = machineID
	return nil
}

func resolveCurrentMachineIdentity(tx *gorm.DB, ipmiIP string) (models.IDCInfo, error) {
	var idc models.IDCInfo
	if err := tx.First(&idc, "ipmi_ip = ?", ipmiIP).Error; err != nil {
		return idc, err
	}
	if idc.MachineID != "" {
		return idc, nil
	}
	if err := ensureIDCInfoMachineID(&idc); err != nil {
		return idc, err
	}
	if err := tx.Model(&idc).Update("machine_id", idc.MachineID).Error; err != nil {
		return idc, err
	}
	return idc, nil
}

func applyMachineIdentityToNetwork(tx *gorm.DB, networkInfo *models.NetworkInfo) error {
	if networkInfo.IPMIIP == nil || *networkInfo.IPMIIP == "" {
		return nil
	}
	idc, err := resolveCurrentMachineIdentity(tx, *networkInfo.IPMIIP)
	if err != nil {
		return nil
	}
	networkInfo.MachineID = &idc.MachineID
	if networkInfo.ZbxID == nil || *networkInfo.ZbxID == "" {
		networkInfo.ZbxID = &idc.ZbxID
	}
	if networkInfo.IDCCode == nil || *networkInfo.IDCCode == "" {
		networkInfo.IDCCode = &idc.IDCCode
	}
	return nil
}
