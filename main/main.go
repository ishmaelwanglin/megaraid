package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ishmaelwanglin/megaraid"
)

func main() {
	m, err := megaraid.CreateMegasasIoctl()
	if err != nil {

	}
	defer m.Close()

	hosts, err := m.ScanHosts()
	if err != nil {
		log.Fatal(err)
	}

	for _, host := range hosts {
		instance := megaraid.Instance{HostNo: host}
		devices, err := m.MegasasGetPdList(&instance)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", strings.Repeat("-", 160))
		fmt.Printf("%-10s%-10s%-20s%-20s%-20s%-20s%-10s%-10s%-20s%-20s\n", "Eid:Slt", "DID", "MediaType", "Size", "Serial", "Product", "Vendor", "FwState", "State", "Properties")
		fmt.Printf("%s\n", strings.Repeat("-", 160))
		for _, v := range devices {
			if !v.IsScsiDev() {
				continue
			}
			sdev := megaraid.ScsiDevice{
				Channel:  v.EnclosureId,
				DeviceId: v.DeviceId,
			}

			pdInfo, err := m.MegasasGetPdInfo(&instance, &sdev)
			if err != nil {
				continue
			}
			inq, err := pdInfo.GetInquiryData()
			if err != nil {
				continue
			}

			fmt.Printf("%-10s%-10d%-20s%-20s%-20s%-20s%-10s%-10s%-20b%-20b\n", fmt.Sprintf("%d:%d", pdInfo.EnclDeviceId, pdInfo.SlotNumber),
				pdInfo.Ref.DeviceId, pdInfo.GetMediaType(),
				pdInfo.GetSize(), inq.SerialNumber, inq.ProductIdentification, inq.VendorIdentification, pdInfo.GetFwState(),
				pdInfo.State.PdType, pdInfo.Properties.Bits)
		}
		fmt.Printf("%s\n", strings.Repeat("-", 160))

		fmt.Printf("\n\n")
		ldList, err := m.MegasasGetLdList(&instance)
		if err != nil {
			continue
		}
		if ldList.LdCount == 0 {
			continue
		}

		fmt.Printf("%s\n", strings.Repeat("-", 30))
		fmt.Printf("%-10s%-10s%-10s\n", "TargetId", "State", "Size")
		fmt.Printf("%s\n", strings.Repeat("-", 30))
		for i := 0; i < int(ldList.LdCount); i++ {
			fmt.Printf("%-10d%-10s%-10s\n", ldList.LdList[i].Ref.TargetId, ldList.LdList[i].GetState(), ldList.LdList[i].GetSize())
		}
		fmt.Printf("%s\n", strings.Repeat("-", 30))
		fmt.Printf("\n\n")
		m.MegasasLdListQuery(&instance, megaraid.MR_LD_QUERY_TYPE_EXPOSED_TO_HOST)
		fmt.Printf("\n\n")
		ctrlInfo := m.MegasasGetCtrlInfo(&instance)

		var devInterface string
		if megaraid.BitField(ctrlInfo.DeviceInterface.Bits, 0, 4) == 10 {
			devInterface = "SAS-12G"
		} else {
			devInterface = "Unknown"
		}

		fmt.Printf("ProductName: %s\nVendorId: %#x\nSerial: %s\nDeviceInterface: %s\nJbodEnabled: %t\n",
			ctrlInfo.GetProductName(), ctrlInfo.Pci.VendorId, ctrlInfo.SerialNumber(), devInterface, ctrlInfo.JbodEnabled())
		fmt.Printf("\n\n")
		fmt.Printf("%b\n", megaraid.BitField(ctrlInfo.DeviceInterface.Bits, 5, 1))
		fmt.Printf("%b\n", megaraid.BitField(ctrlInfo.DeviceInterface.Bits, 0, 8))
	}

}

// devType: 0 -> scsi 13 -> vd/nvme

// /sys/class/scsi_host/host0/device/target0\:0\:10/0\:0\:10\:0/block/sdg
// /sys/class/scsi_device
