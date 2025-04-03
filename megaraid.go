package megaraid

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	// Beware: cannot use unsafe.Sizeof(megasas_iocpacket{}) due to Go struct padding!
	MEGASAS_IOC_FIRMWARE = Iowr('M', 1, uintptr(binary.Size(megasas_iocpacket{})))
)

// PackedBytes is a convenience method that will pack a megasas_iocpacket struct in little-endian
// format and return it as a byte slice
func (ioc *megasas_iocpacket) PackedBytes() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, ioc)
	return b.Bytes()
}

// CreateMegasasIoctl determines the device ID for the MegaRAID SAS ioctl device, creates it
// if necessary, and returns a MegasasIoctl struct to interact with the megaraid_sas driver.
func CreateMegasasIoctl() (*MegasasIoctl, error) {
	var (
		m   MegasasIoctl
		err error
	)

	// megaraid_sas driver does not automatically create ioctl device node, so find out the device
	// major number and create it.
	file, err := os.Open("/proc/devices")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasSuffix(scanner.Text(), "megaraid_sas_ioctl") {
			if _, err := fmt.Sscanf(scanner.Text(), "%d", &m.DeviceMajor); err != nil {
				break
			}
		}
	}

	if m.DeviceMajor == 0 {
		return nil, fmt.Errorf("could not determine megaraid major number")
	}

	if _, err := os.Stat("/dev/megaraid_sas_ioctl_node"); err != nil {
		unix.Mknod("/dev/megaraid_sas_ioctl_node", unix.S_IFCHR, int(unix.Mkdev(m.DeviceMajor, 0)))
	}

	if m.fd, err = unix.Open("/dev/megaraid_sas_ioctl_node", unix.O_RDWR, 0600); err != nil {
		return nil, err
	}
	return &m, nil
}

// Close closes the file descriptor of the MegasasIoctl instance
func (m *MegasasIoctl) Close() {
	unix.Close(m.fd)
}

// ScanHosts scans system for megaraid_sas controllers and returns a slice of host numbers
func (m *MegasasIoctl) ScanHosts() ([]uint16, error) {
	var hosts []uint16
	fEntrys, err := os.ReadDir(SYSFS_SCSI_HOST_DIR)
	if err != nil {
		return hosts, err
	}

	for _, entry := range fEntrys {
		if entry.Type()&os.ModeSymlink != 0 {
			// symlink
			b, err := os.ReadFile(filepath.Join(SYSFS_SCSI_HOST_DIR, entry.Name(), "proc_name"))
			if err != nil {
				continue
			}

			if string(bytes.TrimSpace(b)) == "megaraid_sas" {
				var hostNum uint16

				if _, err := fmt.Sscanf(entry.Name(), "host%d", &hostNum); err == nil {
					hosts = append(hosts, hostNum)
				}
			}
		}
	}

	return hosts, nil
}

func (m *MegasasIoctl) MFI_READ(instance *Instance, sdev ...*ScsiDevice) error {
	ioc := megasas_iocpacket{host_no: instance.HostNo}

	// Approximation of C union behaviour
	dcmd := (*megasas_dcmd_frame)(unsafe.Pointer(&ioc.frame))

	if len(sdev) > 0 {
		device_id := sdev[0].Channel*MEGASAS_MAX_DEV_PER_CHANNEL + sdev[0].DeviceId
		(*mbox_s)(unsafe.Pointer(&dcmd.mbox[0]))[0] = device_id
	} else {
		if instance.Dcmd.MboxB[0] > 0 {
			(*mbox_b)(unsafe.Pointer(&dcmd.mbox[0]))[0] = instance.Dcmd.MboxB[0]
		}
	}

	dcmd.cmd = uint8(MFI_CMD_DCMD)
	dcmd.cmd_status = MFI_STAT_INVALID_STATUS
	dcmd.opcode = instance.Cmd.OpCode
	dcmd.data_xfer_len = uint32(len(instance.Buf))
	dcmd.sge_count = 1
	dcmd.flags = MFI_FRAME_DIR_READ
	dcmd.timeout = 0
	dcmd.pad_0 = 0

	// ioc set dma
	ioc.sge_count = 1
	ioc.sgl_off = uint32(unsafe.Offsetof(dcmd.sgl))
	ioc.sgl[0] = Iovec{uint64(uintptr(unsafe.Pointer(&instance.Buf[0]))), uint64(len(instance.Buf))}

	iocBuf := ioc.PackedBytes()
	// Note pointer to first item in iocBuf buffer
	if err := Ioctl(uintptr(m.fd), MEGASAS_IOC_FIRMWARE, uintptr(unsafe.Pointer(&iocBuf[0]))); err != nil {
		return err
	}

	return nil
}

type Instance struct {
	HostNo uint16
	Buf    []byte
	Cmd    struct {
		OpCode uint32
	}
	Dcmd struct {
		MboxB [12]uint8
	}
}

func (m *MegasasIoctl) MegasasGetPdList(instance *Instance) ([]MR_PD_ADDRESS, error) {
	instance.Buf = make([]byte, unsafe.Sizeof(MR_PD_LIST{})*MEGASAS_MAX_PD)
	instance.Cmd.OpCode = MR_DCMD_PD_LIST_QUERY

	m.MFI_READ(instance)

	respCount := binary.LittleEndian.Uint32(instance.Buf[4:])
	if respCount == 0 {
		return nil, fmt.Errorf("null scsi")
	}

	// Create a device array large enough to hold the specified number of devices
	devices := make([]MR_PD_ADDRESS, respCount)
	binary.Read(bytes.NewBuffer(instance.Buf[8:]), binary.LittleEndian, &devices)

	return devices, nil
}

type ScsiDevice struct {
	Channel  uint16 // eid
	DeviceId uint16 // did
}

func (m *MegasasIoctl) MegasasGetPdInfo(instance *Instance, sdev *ScsiDevice) (*MR_PD_INFO, error) {
	// 测试成功
	instance.Buf = make([]byte, unsafe.Sizeof(MR_PD_INFO{}))
	instance.Cmd.OpCode = MR_DCMD_PD_GET_INFO
	m.MFI_READ(instance, sdev)

	data := &MR_PD_INFO{}
	if err := binary.Read(bytes.NewBuffer(instance.Buf), binary.LittleEndian, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (m *MegasasIoctl) MegasasGetLdList(instance *Instance) (*MR_LD_LIST, error) {
	instance.Buf = make([]byte, unsafe.Sizeof(MR_LD_LIST{}))
	instance.Cmd.OpCode = MR_DCMD_LD_GET_LIST

	if err := m.MFI_READ(instance); err != nil {
		return nil, err
	}

	// Create a device array large enough to hold the specified number of devices
	data := MR_LD_LIST{}

	if err := binary.Read(bytes.NewBuffer(instance.Buf[:]), binary.LittleEndian, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// 只能获取所有导出到OS的targetID，无关联关系，似乎没啥用
func (m *MegasasIoctl) MegasasLdListQuery(instance *Instance, queryType uint8) error {

	instance.Buf = make([]byte, unsafe.Sizeof(MR_LD_TARGETID_LIST{}))
	instance.Cmd.OpCode = MR_DCMD_LD_LIST_QUERY
	instance.Dcmd.MboxB[0] = MR_LD_QUERY_TYPE_EXPOSED_TO_HOST
	m.MFI_READ(instance)

	ldInfo := MR_LD_TARGETID_LIST{}
	binary.Read(bytes.NewBuffer(instance.Buf[:]), binary.LittleEndian, &ldInfo)
	fmt.Printf("export os targetID: ")
	for i := 0; i < int(ldInfo.Count); i++ {
		fmt.Printf("%d", ldInfo.TargetId[i])
		if i != int(ldInfo.Count)-1 {
			fmt.Printf(",")
		}
	}
	fmt.Printf("\n")
	return nil
}

func (m *MegasasIoctl) MegasasGetCtrlInfo(instance *Instance) *megasas_ctrl_info {
	instance.Buf = make([]byte, unsafe.Sizeof(megasas_ctrl_info{}))
	instance.Cmd.OpCode = MR_DCMD_CTRL_GET_INFO
	instance.Dcmd.MboxB[0] = 1
	m.MFI_READ(instance)
	data := megasas_ctrl_info{}
	binary.Read(bytes.NewBuffer(instance.Buf), binary.LittleEndian, &data)

	return &data
}

func PdSasAddr(array []uint32, n uint8) (string, error) {
	if n != 2 {
		return "", fmt.Errorf("n must be 2")
	}
	return fmt.Sprintf("0x%s", strconv.FormatUint(uint64(array[1])<<32|uint64(array[0]), 16)), nil
}
