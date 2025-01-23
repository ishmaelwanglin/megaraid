package megaraid

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"
	"unsafe"

	"github.com/dswarbrick/smart/utils"
)

func TestScanHosts(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	hosts, err := m.ScanHosts()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hosts)
}
func TestMegasasGetPdList(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	devices, err := m.MegasasGetPdList(&Instance{HostNo: 0})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%-10s%-10s%-10s%-10s%-10s%-10s%-30s\n", "DID", "EID", "EIndex", "Slot", "DevType", "Port", "SasAddr")
	for _, v := range devices {

		fmt.Printf("%-10d%-10d%-10d%-10d%-10d%-10d%-30d\n", v.DeviceId, v.EnclosureId, v.EnclosureIndex, v.SlotNumber, v.ScsiDevType, v.ConnectPortBitmap, SasAddrParse(v.SasAddr[:]))
	}
	/*
		DID       EID       EIndex    Slot      DevType   Port      SasAddr
		0         0         1         0         13        0         5768548493399075358
		1         0         1         15        0         0         5768548493399075343
		2         0         1         14        0         0         5768548493399075342
		4         0         1         9         0         0         5768548493399075337
		5         0         1         10        0         0         5768548493399075338
		6         0         1         8         0         0         5768548493399075336
		7         0         1         2         0         0         5768548493399075330
		8         0         1         4         0         0         5768548493399075332
		9         0         1         5         0         0         5768548493399075333
		10        0         1         7         0         0         5768548493399075335
		12        0         1         11        0         0         5768548493399075339
		14        0         1         1         0         0         5768548493399075329
		252       252       1         255       31        0         0
	*/
	// did: 252 , slot: 255, type: 31, 且sasAddr是0说明是坏盘
	// devType: 0 -> scsi 13 -> vd/nvme
}

func TestMegasasGetPdInfo(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	instance := Instance{
		HostNo: 0,
	}
	sdev := ScsiDevice{
		Channel:  0,
		DeviceId: 10,
	}
	// slotnumber不对
	m.MegasasGetPdInfo(&instance, &sdev)
	/*
		did = 10
		 /sys/class/scsi_host/host0/device/target0\:0\:10/0\:0\:10\:0/block/sdg
	*/
}
func TestMegasasGetLdList(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	instance := Instance{
		HostNo: 0,
	}
	m.MegasasGetLdList(&instance)
}
func TestMegasasLdListQuery(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	instance := Instance{
		HostNo: 0,
	}
	m.MegasasLdListQuery(&instance, MR_LD_QUERY_TYPE_EXPOSED_TO_HOST)
}

func TestMegasasGetCtrlInfo(t *testing.T) {
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	instance := Instance{
		HostNo: 0,
	}
	m.MegasasGetCtrlInfo(&instance)
}
func TestMRPDLIST(t *testing.T) {

	// 通过测试
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	ioc := megasas_iocpacket{host_no: 0}
	buf := make([]byte, unsafe.Sizeof(MR_PD_LIST{})*MEGASAS_MAX_PD)
	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))
	dcmd.cmd = uint8(MFI_CMD_DCMD)
	dcmd.opcode = MR_DCMD_PD_LIST_QUERY
	dcmd.data_xfer_len = uint32(len(buf))
	dcmd.sge_count = 1

	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}

	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}

	// size := respBuf[:4]
	respCount := utils.NativeEndian.Uint32(buf[4:])
	if respCount == 0 {
		t.Fatal("null scsi")
	}

	// Create a device array large enough to hold the specified number of devices
	devices := make([]MR_PD_ADDRESS, respCount)
	binary.Read(bytes.NewBuffer(buf[8:]), utils.NativeEndian, &devices)

	for _, v := range devices {
		t.Log(v)
	}
}

func TestPDINFO(t *testing.T) {
	// 测试成功
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	var host uint16
	buf := make([]byte, unsafe.Sizeof(MR_PD_INFO{}))
	ioc := megasas_iocpacket{
		host_no: host,
	}
	eid := 0  // channel
	did := 10 // id
	device_id := eid*MEGASAS_MAX_DEV_PER_CHANNEL + did

	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))

	(*mbox_s)(unsafe.Pointer(&dcmd.mbox[0]))[0] = uint16(device_id)

	dcmd.cmd = MFI_CMD_DCMD
	dcmd.cmd_status = 0xFF
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0
	dcmd.data_xfer_len = uint32(len(buf))
	dcmd.opcode = uint32(MR_DCMD_PD_GET_INFO)

	// ioc set dma
	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}
	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}

	// fmt.Println(buf)
	// fmt.Println("buf len: ", len(buf))
	info := &MR_PD_INFO{}
	// fmt.Println("info size: ", binary.Size(info))
	if err = binary.Read(bytes.NewBuffer(buf), utils.NativeEndian, info); err != nil {
		t.Fatal("binary read failed with: ", err)
	}

	// 将数据格式化为 JSON 字符串
	jsonData, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}
	fmt.Println(string(jsonData))

	// inquiry, err := ParseInquiryData(info.InquiryData[:])
	// if err != nil {
	// 	log.Fatalf("Failed to parse inquiry data: %v", err)
	// }

	// // 打印解析结果
	// fmt.Println(inquiry.String())
	// fmt.Printf("\nDevice Type: %s\n", inquiry.DeviceType())

	// vpd, err := ParseVPDPage83(info.VpdPage83[:])
	// if err != nil {
	// 	log.Fatalf("Failed to parse VPD page 83 data: %v", err)
	// }

	// ParseVpdPage83(info.VpdPage83[:])
	// 打印解析结果
	// fmt.Println(vpd.String())
	// fmt.Println(info.PathInfo.SasAddr)
	// fmt.Println(info.SlotNumber)
}

func TestGetLdList(t *testing.T) {
	// 成功
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	var host uint16
	buf := make([]byte, unsafe.Sizeof(MR_LD_LIST{}))
	ioc := megasas_iocpacket{
		host_no: host,
	}

	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))

	dcmd.cmd = MFI_CMD_DCMD
	dcmd.cmd_status = MFI_STAT_INVALID_STATUS
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0
	dcmd.data_xfer_len = uint32(len(buf))
	dcmd.opcode = uint32(MR_DCMD_LD_GET_LIST)

	// ioc set dma
	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}

	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}

	// Create a device array large enough to hold the specified number of devices
	data := MR_LD_LIST{}
	binary.Read(bytes.NewBuffer(buf[:]), utils.NativeEndian, &data)

	for i := 0; i < int(data.LdCount); i++ {
		// 将数据格式化为 JSON 字符串
		jsonData, err := json.MarshalIndent(data.LdList[0], "", "    ")
		if err != nil {
			fmt.Println("Error formatting JSON:", err)
			return
		}

		fmt.Println(string(jsonData))
	}

}

func TestLdListQuery(t *testing.T) {

	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	var host uint16
	buf := make([]byte, unsafe.Sizeof(MR_LD_TARGETID_LIST{}))
	ioc := megasas_iocpacket{
		host_no: host,
	}

	queryType := MR_LD_QUERY_TYPE_EXPOSED_TO_HOST

	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))

	(*mbox_b)(unsafe.Pointer(&dcmd.mbox[0]))[0] = queryType

	dcmd.cmd = MFI_CMD_DCMD
	dcmd.cmd_status = MFI_STAT_INVALID_STATUS
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0
	dcmd.data_xfer_len = uint32(len(buf))
	dcmd.opcode = uint32(MR_DCMD_LD_LIST_QUERY)

	// ioc set dma
	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}

	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}

	ldInfo := MR_LD_TARGETID_LIST{}
	binary.Read(bytes.NewBuffer(buf[:]), utils.NativeEndian, &ldInfo)

	for i := 0; i < int(ldInfo.Count); i++ {
		fmt.Println(ldInfo.TargetId[i])
	}
	// 将数据格式化为 JSON 字符串
	// jsonData, err := json.MarshalIndent(ldInfo, "", "    ")
	// if err != nil {
	// 	fmt.Println("Error formatting JSON:", err)
	// 	return
	// }
	// fmt.Println(string(jsonData))
}

func TestHostDeviceListQuery(t *testing.T) {
	// 不成功
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, unsafe.Sizeof(MR_HOST_DEVICE_LIST{}))
	ioc := megasas_iocpacket{host_no: 0}

	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))

	(*mbox_b)(unsafe.Pointer(&dcmd.mbox[0]))[0] = 0 //  is_probe ? 0 : 1;

	dcmd.cmd = MFI_CMD_DCMD
	dcmd.cmd_status = MFI_STAT_INVALID_STATUS
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0
	dcmd.data_xfer_len = uint32(len(buf))
	dcmd.opcode = uint32(MR_DCMD_CTRL_DEVICE_LIST_GET)

	// ioc set dma
	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}

	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}
	fmt.Println(buf)
	// Size := utils.NativeEndian.Uint32(buf[:4])
	// count := utils.NativeEndian.Uint32(buf[4:8])
	// fmt.Println(Size, count)
	// list := [1]MR_HOST_DEVICE_LIST_ENTRY{}
	// data := MR_HOST_DEVICE_LIST{}
	// binary.Read(bytes.NewBuffer(buf[:]), utils.NativeEndian, &data)

	// for i := 0; i < int(data.Count); i++ {
	// 	fmt.Println(data.HostDeviceList[i])
	// }
	// // 将数据格式化为 JSON 字符串
	// jsonData, err := json.MarshalIndent(list, "", "    ")
	// if err != nil {
	// 	fmt.Println("Error formatting JSON:", err)
	// 	return
	// }
	// fmt.Println(string(jsonData))
}

func TestCtrlInfo(t *testing.T) {
	// 通过测试
	m, err := CreateMegasasIoctl()
	if err != nil {
		t.Fatal(err)
	}

	ioc := megasas_iocpacket{host_no: 0}
	buf := make([]byte, unsafe.Sizeof(megasas_ctrl_info{}))
	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))
	dcmd.opcode = MR_DCMD_CTRL_GET_INFO
	dcmd.data_xfer_len = uint32(len(buf))
	(*mbox_b)(unsafe.Pointer(&dcmd.mbox[0]))[0] = 1
	dcmd.cmd = MFI_CMD_DCMD
	dcmd.cmd_status = MFI_STAT_INVALID_STATUS
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0

	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&buf[0]))), uint64(len(buf))}

	iocBuf := ioc.PackedBytes()

	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		t.Fatal("ioctl error with:", err)
	}

	// fmt.Println(buf)

	fmt.Printf("unsafe.Sizeof: %d, binary.Sizeof: %d", unsafe.Sizeof(megasas_ctrl_info{}), binary.Size(megasas_ctrl_info{}))
	resp := megasas_ctrl_info{}

	binary.Read(bytes.NewBuffer(buf[:]), utils.NativeEndian, &resp)
	// // 将数据格式化为 JSON 字符串
	jsonData, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}
	fmt.Println(string(jsonData))
}
