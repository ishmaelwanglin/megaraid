package megaraid

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const SectorSz = 512 // Bytes
const (
	KB = 1 << 10
	MB = 1 << 20
	GB = 1 << 30
	TB = 1 << 40
)

/*
 * =====================================
 * MegaRAID SAS MFI firmware definitions
 * =====================================
 */

/*
 * MFI frame flags
 */
const (
	MFI_FRAME_POST_IN_REPLY_QUEUE      = 0x0000
	MFI_FRAME_DONT_POST_IN_REPLY_QUEUE = 0x0001
	MFI_FRAME_SGL32                    = 0x0000
	MFI_FRAME_SGL64                    = 0x0002
	MFI_FRAME_SENSE32                  = 0x0000
	MFI_FRAME_SENSE64                  = 0x0004
	MFI_FRAME_DIR_NONE                 = 0x0000 // 无数据传输
	MFI_FRAME_DIR_WRITE                = 0x0008 // 写操作
	MFI_FRAME_DIR_READ                 = 0x0010 // 读操作
	MFI_FRAME_DIR_BOTH                 = 0x0018 // 双向传输
	MFI_FRAME_IEEE                     = 0x0020
)

// MFI_CMD_OP 说明
const (
	// 功能：初始化 MFI 命令接口，通常在控制器启动或驱动加载时使用。
	// 说明：初始化后，控制器可以处理其他命令。
	MFI_CMD_INIT uint8 = iota
	// 功能：从逻辑盘（Logical Drive LD）执行读取操作。
	// 说明：用于从 RAID 配置的逻辑卷中读取数据。
	MFI_CMD_LD_READ
	// 	功能：向逻辑盘执行写入操作。
	// 说明：用于向 RAID 配置的逻辑卷写入数据。
	MFI_CMD_LD_WRITE
	// 功能：对逻辑盘执行 SCSI I/O 操作。
	// 说明：逻辑盘的高级 I/O 操作，通过 SCSI 命令完成。
	MFI_CMD_LD_SCSI_IO
	// 功能：对物理盘（Physical Drive PD）执行 SCSI I/O 操作。
	// 说明：直接操作物理磁盘，例如非 RAID 配置或诊断操作。
	MFI_CMD_PD_SCSI_IO
	// 	功能：执行 Direct Command（直接命令）。
	// 说明：这是一个通用命令，用于执行控制器的管理任务（如配置、状态查询、日志获取等）。
	// 子命令：DCMD 通常与子命令一起使用，以完成特定任务。
	MFI_CMD_DCMD
	// 	功能：中止当前的命令。
	// 说明：用于取消正在执行的 I/O 操作。
	MFI_CMD_ABORT
	// 	功能：执行 SMP（Serial Management Protocol）命令。
	// 说明：用于通过 SAS 控制器管理和诊断连接的设备。
	MFI_CMD_SMP
	// 	功能：执行 STP（Serial ATA Tunneling Protocol）命令。
	// 说明：用于通过 SAS 控制器对 SATA 设备进行操作。
	MFI_CMD_STP
	// 	功能：执行 NVMe 相关操作。
	// 说明：用于支持 NVMe 驱动器的命令传输。
	MFI_CMD_NVME
	// 	功能：用于特定的工具箱命令。
	// 说明：可能包含调试或扩展功能，具体用途需要参考具体的驱动文档。
	MFI_CMD_TOOLBOX
	// 功能：命令操作码计数。
	// 说明：用作标志，表示支持的命令总数。
	MFI_CMD_OP_COUNT
	// 	功能：表示无效命令。
	// 说明：用于占位或标记不支持的操作。
	MFI_CMD_INVALID = 0xff
)

/*
 * MFI command completion codes
 */
const (
	MFI_STAT_OK uint8 = iota
	MFI_STAT_INVALID_CMD
	MFI_STAT_INVALID_DCMD
	MFI_STAT_INVALID_PARAMETER
	MFI_STAT_INVALID_SEQUENCE_NUMBER
	MFI_STAT_ABORT_NOT_POSSIBLE
	MFI_STAT_APP_HOST_CODE_NOT_FOUND
	MFI_STAT_APP_IN_USE
	MFI_STAT_APP_NOT_INITIALIZED
	MFI_STAT_ARRAY_INDEX_INVALID
	MFI_STAT_ARRAY_ROW_NOT_EMPTY
	MFI_STAT_CONFIG_RESOURCE_CONFLICT
	MFI_STAT_DEVICE_NOT_FOUND
	MFI_STAT_DRIVE_TOO_SMALL
	MFI_STAT_FLASH_ALLOC_FAIL
	MFI_STAT_FLASH_BUSY
	MFI_STAT_FLASH_ERROR
	MFI_STAT_FLASH_IMAGE_BAD
	MFI_STAT_FLASH_IMAGE_INCOMPLETE
	MFI_STAT_FLASH_NOT_OPEN
	MFI_STAT_FLASH_NOT_STARTED
	MFI_STAT_FLUSH_FAILED
	MFI_STAT_HOST_CODE_NOT_FOUNT
	MFI_STAT_LD_CC_IN_PROGRESS
	MFI_STAT_LD_INIT_IN_PROGRESS
	MFI_STAT_LD_LBA_OUT_OF_RANGE
	MFI_STAT_LD_MAX_CONFIGURED
	MFI_STAT_LD_NOT_OPTIMAL
	MFI_STAT_LD_RBLD_IN_PROGRESS
	MFI_STAT_LD_RECON_IN_PROGRESS
	MFI_STAT_LD_WRONG_RAID_LEVEL
	MFI_STAT_MAX_SPARES_EXCEEDED
	MFI_STAT_MEMORY_NOT_AVAILABLE
	MFI_STAT_MFC_HW_ERROR
	MFI_STAT_NO_HW_PRESENT
	MFI_STAT_NOT_FOUND
	MFI_STAT_NOT_IN_ENCL
	MFI_STAT_PD_CLEAR_IN_PROGRESS
	MFI_STAT_PD_TYPE_WRONG
	MFI_STAT_PR_DISABLED
	MFI_STAT_ROW_INDEX_INVALID
	MFI_STAT_SAS_CONFIG_INVALID_ACTION
	MFI_STAT_SAS_CONFIG_INVALID_DATA
	MFI_STAT_SAS_CONFIG_INVALID_PAGE
	MFI_STAT_SAS_CONFIG_INVALID_TYPE
	MFI_STAT_SCSI_DONE_WITH_ERROR
	MFI_STAT_SCSI_IO_FAILED
	MFI_STAT_SCSI_RESERVATION_CONFLICT
	MFI_STAT_SHUTDOWN_FAILED
	MFI_STAT_TIME_NOT_SET
	MFI_STAT_WRONG_STATE
	MFI_STAT_LD_OFFLINE
	MFI_STAT_PEER_NOTIFICATION_REJECTED
	MFI_STAT_PEER_NOTIFICATION_FAILED
	MFI_STAT_RESERVATION_IN_PROGRESS
	MFI_STAT_I2C_ERRORS_DETECTED
	MFI_STAT_PCI_ERRORS_DETECTED

	MFI_STAT_CONFIG_SEQ_MISMATCH uint8 = 0x67
	MFI_STAT_INVALID_STATUS      uint8 = 0xff
)

const (
	/*
		命令结构
		这些命令码的高 16 位通常表示主命令类别，低 16 位表示子命令。例如：
		0x01xxxxxx：控制器相关命令。
		0x02xxxxxx：物理磁盘相关命令。
		0x03xxxxxx：逻辑磁盘相关命令。
		0x08xxxxxx：集群相关命令。
	*/
	MR_DCMD_CTRL_GET_INFO = 0x01010000 //	获取控制器信息, 查询 MegaRAID 控制器的详细信息（如固件版本、缓存大小等）。

	MR_DCMD_LD_GET_LIST = 0x03010000 // 	获取逻辑磁盘（Logical Drive LD）列表。

	MR_DCMD_LD_LIST_QUERY = 0x03010100 // 	查询特定逻辑盘列表信息, 用于筛选或特定查询逻辑盘信息。

	MR_DCMD_LD_GET_PROPERTIES = 0x03030000 //	获取逻辑盘的属性, 返回逻辑盘的详细配置，例如 RAID 级别、大小等。

	MR_DCMD_PD_LIST_QUERY = 0x02010100 //	查询物理磁盘(Physical Drive PD)列表, 返回当前控制器管理的所有物理磁盘。

	MR_DCMD_PD_GET_LIST = 0x02010000 //	一条被废弃，但仍可以使用的命令,同MR_DCMD_PD_LIST_QUERY

	MR_DCMD_PD_GET_INFO = 0x02020000 //	获取物理磁盘信息, 返回物理磁盘的详细信息，例如容量、状态、序列号等。

	MR_DCMD_CTRL_EVENT_GET_INFO = 0x01040100 //	获取控制器事件的统计信息,例如，获取控制器已记录的事件数量。

	MR_DCMD_CTRL_EVENT_GET = 0x01040300 //	获取控制器事件日志,返回事件详细信息，例如错误或状态更改。

	MR_DCMD_CTRL_EVENT_WAIT = 0x01040500 //	等待特定事件发生,常用于监控控制器运行状态。
)

const (
	SGE_BUFFER_SIZE         = 4096
	MEGASAS_CLUSTER_ID_SIZE = 16
)

// MR_PD_STATE
const (
	MR_PD_STATE_UNCONFIGURED_GOOD uint8 = iota
	MR_PD_STATE_UNCONFIGURED_BAD
	MR_PD_STATE_HOT_SPARE
	MR_PD_STATE_OFFLINE  uint8 = 0x10
	MR_PD_STATE_FAILED   uint8 = 0x11
	MR_PD_STATE_REBUILD  uint8 = 0x14
	MR_PD_STATE_ONLINE   uint8 = 0x18
	MR_PD_STATE_COPYBACK uint8 = 0x20
	MR_PD_STATE_SYSTEM   uint8 = 0x40
)

type mrProgress struct {
	Progress    uint16
	ElapsedSecs uint16 // union: elapsedSecsForLastPercent
}

// union
type MR_PROGRESS struct {
	Mrprogress mrProgress
}

// 56
type MR_PD_PROGRESS struct {
	Active   [4]byte
	Rbld     MR_PROGRESS
	Patrol   MR_PROGRESS
	Clear    MR_PROGRESS
	Pause    [4]byte
	Reserved [3]MR_PROGRESS
}

// union MR_PD_REF

type MR_PD_DDF_TYPE struct {
	PdType uint16
	// u16	 forcedPDGUID:1;
	// u16	 inVD:1;
	// u16	 isGlobalSpare:1;
	// u16	 isSpare:1;
	// u16	 isForeign:1;
	// u16	 reserved:7;
	// u16	 intf:4;
	_ uint16
} // __packed

type MR_PD_INFO struct {
	Ref struct {
		DeviceId uint16
		SeqNum   uint16
	}
	InquiryData         [96]uint8 // serial vendor可以在这里解析
	VpdPage83           [64]uint8 // naa(wwn)在这里可以解析到
	NotSupported        uint8
	ScsiDevType         uint8
	ConnectedPortBitmap uint8

	DeviceSpeed             uint8
	MediaErrCount           uint32
	OtherErrCount           uint32
	PredFailCount           uint32
	LastPredFailEventSeqNum uint32

	FwState            uint16
	DisabledForRemoval uint8
	LinkSpeed          uint8
	State              MR_PD_DDF_TYPE // vd 有intf = 11

	PathInfo struct {
		Count uint8
		Bits  uint8
		/*
			u8 isPathBroken:4;
			u8 reserved3:3;
			u8 widePortCapable:1;
		*/
		ConnectorIndex [2]uint8
		_              [4]uint8
		SasAddr        [4]uint32 // [2]uint64
		_              [16]uint8
	}
	RawSize        [2]uint32 //	uint64
	NonCoercedSize [2]uint32 //	uint64
	CoercedSize    [2]uint32 // 	uint64
	EnclDeviceId   uint16    // eid
	EnclIndex      uint8
	SlotNumber     uint8 // slot

	ProgInfo               MR_PD_PROGRESS // 重建过程
	BadBlockTableFull      uint8
	NusableInCurrentConfig uint8
	VpdPage83Ext           [64]uint8
	PowerState             uint8
	EnclPosition           uint8
	AllowedOps             uint32
	CopyBackPartnerId      uint16
	EnclPartnerDeviceId    uint16

	Security uint16
	/*
		security struct:
		u16 fdeCapable:1;
		u16 fdeEnabled:1;
		u16 secured:1;
		u16 locked:1;
		u16 foreign:1;
		u16 needsEKM:1;
		u16 reserved:10;
	*/
	MediaType                   uint8 // hdd ssd
	NotCertified                uint8
	DorbridgeVen                [8]uint8
	BridgeProductIdentification [16]uint8
	BridgeProductRevisionLevel  [4]uint8
	SatBridgeExists             uint8

	InterfaceType     uint8
	Temperature       uint8
	EmulatedBlockSize uint8
	UserDataBlockSize uint16
	_                 uint16

	Properties struct {
		Bits uint32
		/*
			u32 piType:3;
			u32 piFormatted:1;
			u32 piEligible:1;
			u32 NCQ:1;
			u32 WCE:1;
			u32 commissionedSpare:1;
			u32 emergencySpare:1;
			u32 ineligibleForSSCD:1;
			u32 ineligibleForLd:1;
			u32 useSSEraseType:1;
			u32 wceUnchanged:1;
			u32 supportScsiUnmap:1;
			u32 reserved:18;
		*/
	}

	ShieldDiagCompletionTime [2]uint32 // uint64
	ShieldCounter            uint8

	LinkSpeedOther uint8
	_              [2]uint8
	BbmErr         struct {
		Bits uint32
		/*
			u32 bbmErrCountSupported:1;
			u32 bbmErrCount:31;
		*/
	}
	_ [512 - 428]uint8
} // __packed

func (info *MR_PD_INFO) GetSize() string {
	return SizeString(ArrayZip(info.RawSize[:], 32))
}

func (info *MR_PD_INFO) GetMediaType() string {
	// 0: HDD 1: SSD
	var mediaType string
	switch info.MediaType {
	case 0:
		mediaType = "HDD"
	case 1:
		mediaType = "SSD"
	default:
		mediaType = "Unknown"
	}

	return mediaType
}

func (info *MR_PD_INFO) GetFwState() string {
	var status string
	switch uint8(info.FwState) {
	case MR_PD_STATE_UNCONFIGURED_GOOD:
		status = "UGood"
	case MR_PD_STATE_UNCONFIGURED_BAD:
		status = "UBad"
	case MR_PD_STATE_HOT_SPARE:
		status = "HotSpare"
	case MR_PD_STATE_OFFLINE:
		status = "Offline"
	case MR_PD_STATE_FAILED:
		status = "Failed"
	case MR_PD_STATE_REBUILD:
		status = "Rebuild"
	case MR_PD_STATE_ONLINE:
		status = "Online"
	case MR_PD_STATE_COPYBACK:
		status = "Copyback"
	case MR_PD_STATE_SYSTEM:
		status = "Jbod"
	default:
		status = "Unkown"
	}

	return status
}

func (info *MR_PD_INFO) NCQ() bool {
	return BitField(info.Properties.Bits, 5, 1) == 1
}

// // InquiryData represents the parsed SCSI inquiry data structure
type InquiryData struct {
	PeripheralQualifier   uint8
	PeripheralDeviceType  uint8
	RemovableMedia        bool
	Version               uint8
	ResponseDataFormat    uint8
	VendorIdentification  string
	ProductIdentification string
	FirmwareRevision      string
	SerialNumber          string
}

// trimString removes trailing spaces and null bytes from byte slice and converts to string
func trimString(b []byte) string {
	return strings.TrimSpace(string(bytes.TrimRight(b, "\x00")))
}

// String returns a formatted string representation of the inquiry data
func (id *InquiryData) String() string {
	return fmt.Sprintf(
		"SCSI Inquiry Data:\n"+
			"  Peripheral Qualifier:    %d\n"+
			"  Peripheral Device Type:  %d\n"+
			"  Removable Media:         %t\n"+
			"  Version:                 %d\n"+
			"  Response Data Format:    %d\n"+
			"  Vendor Identification:   %s\n"+
			"  Product Identification:  %s\n"+
			"  Firmware Revision:  		%s\n"+
			"  Serial Number:           %s",
		id.PeripheralQualifier,
		id.PeripheralDeviceType,
		id.RemovableMedia,
		id.Version,
		id.ResponseDataFormat,
		id.VendorIdentification,
		id.ProductIdentification,
		id.FirmwareRevision,
		id.SerialNumber,
	)
}

// DeviceType returns a human-readable string for the peripheral device type
func (id *InquiryData) DeviceType() string {
	deviceTypes := map[uint8]string{
		0x00: "Direct access block device", // 硬盘、SSD等
		0x01: "Sequential access device",   // 磁带机
		0x02: "Printer device",
		0x03: "Processor device",
		0x04: "Write-once device",
		0x05: "CD/DVD device", // 光驱
		0x06: "Scanner device",
		0x07: "Optical memory device",
		0x08: "Medium changer device",
		0x09: "Communications device",
		0x0A: "Storage array controller device",
		0x0B: "Enclosure services device",
		0x0C: "Simplified direct-access device",
		0x0D: "Optical card reader/writer device",
		0x0E: "Bridge controller device",
		0x0F: "Object-based storage device",
		0x10: "Automation/Drive interface",
		0x14: "Storage Array Controller Device", // 存储阵列控制器
	}

	if deviceType, ok := deviceTypes[id.PeripheralDeviceType]; ok {
		return deviceType
	}
	return "Unknown device type"
}

func (info *MR_PD_INFO) GetInquiryData() (*InquiryData, error) {
	data := info.InquiryData[:]
	if len(info.InquiryData[:]) < 96 {
		return nil, fmt.Errorf("inquiry data too short: expected 96 bytes, got %d", len(data))
	}

	inquiry := &InquiryData{
		PeripheralQualifier:   (data[0] & 0xE0) >> 5,
		PeripheralDeviceType:  data[0] & 0x1F,
		RemovableMedia:        (data[1] & 0x80) == 0x80,
		Version:               data[2],
		ResponseDataFormat:    data[3] & 0x0F,
		VendorIdentification:  trimString(data[8:16]),
		ProductIdentification: trimString(data[16:32]),
		FirmwareRevision:      trimString(data[32:36]),
		SerialNumber:          trimString(data[36:56]),
	}

	return inquiry, nil
}

const (
	SYSFS_SCSI_HOST_DIR = "/sys/class/scsi_host"
	MAX_MGMT_ADAPTERS   = 1024
	// 最大 SGE (Scatter Gather Element) 数量
	MAX_IOCTL_SGE = 16
)

type megasas_sge64 struct {
	phys_addr uint32
	length    uint32
	_         uint32
} // __packed

type Iovec struct {
	IovBase uint64
	IovLen  uint64
}

type megasas_dcmd_frame struct {
	cmd           uint8
	reserved_0    uint8
	cmd_status    uint8
	reserved_1    [4]uint8
	sge_count     uint8
	context       uint32
	pad_0         uint32
	flags         uint16
	timeout       uint16
	data_xfer_len uint32
	opcode        uint32
	mbox          [12]byte      //	union of [12]uint8 / [6]uint16 / [3]uint32
	sgl           megasas_sge64 //	union of megasas_sge64 / megasas_sge32
}

type mbox_b [12]uint8
type mbox_s [6]uint16
type mbox_w [3]uint32

// megasas_iocpacket struct - caution: megasas driver expects packet struct
type megasas_iocpacket struct {
	host_no   uint16
	__pad1    uint16
	sgl_off   uint32
	sge_count uint32
	sense_off uint32
	sense_len uint32
	frame     [128]byte // union of megasas_frame
	sgl       [MAX_IOCTL_SGE]Iovec
} // __packed

/*
 * defines the physical drive address structure
 */
type MR_PD_ADDRESS struct {
	DeviceId          uint16
	EnclosureId       uint16
	EnclosureIndex    uint8
	SlotNumber        uint8
	ScsiDevType       uint8
	ConnectPortBitmap uint8
	SasAddr           [4]uint32
}

func (a *MR_PD_ADDRESS) IsScsiDev() bool {
	return a.ScsiDevType == 0
}

func (a *MR_PD_ADDRESS) GetSasAddr() uint64 {
	return ArrayZip(a.SasAddr[:], 32)
}
func (a *MR_PD_ADDRESS) GetSasAddrs() string {
	return fmt.Sprintf("0x%s", strconv.FormatUint(a.GetSasAddr(), 16))
}

// Holder for megaraid_sas ioctl device
type MegasasIoctl struct {
	DeviceMajor uint32
	fd          int
}

/*
 * defines the physical drive list structure
 */
type MR_PD_LIST struct {
	Size  uint32
	Count uint32
	Addr  [1]MR_PD_ADDRESS
}

/*
 * ===============================
 * MegaRAID SAS driver definitions
 * ===============================
 */
const (
	MEGASAS_MAX_PD_CHANNELS = 2

	MEGASAS_MAX_DEV_PER_CHANNEL = 128

	MEGASAS_MAX_PD = MEGASAS_MAX_PD_CHANNELS * MEGASAS_MAX_DEV_PER_CHANNEL
)

const (
	MR_PD_QUERY_TYPE_ALL uint8 = iota
	MR_PD_QUERY_TYPE_STATE
	MR_PD_QUERY_TYPE_POWER_STATE
	MR_PD_QUERY_TYPE_MEDIA_TYPE
	MR_PD_QUERY_TYPE_SPEED
	MR_PD_QUERY_TYPE_EXPOSED_TO_HOST
)

const (
	MR_LD_QUERY_TYPE_ALL uint8 = iota
	MR_LD_QUERY_TYPE_EXPOSED_TO_HOST
	MR_LD_QUERY_TYPE_USED_TGT_IDS
	MR_LD_QUERY_TYPE_CLUSTER_ACCESS
	MR_LD_QUERY_TYPE_CLUSTER_LOCALE
)

const MAX_LOGICAL_DRIVES_EXT = 256

const MR_DCMD_CTRL_DEVICE_LIST_GET = 0x01190600

/*
 * defines the logical drive list structure
 */
type MR_LD_LIST struct {
	LdCount uint32
	_       uint32
	LdList  [MAX_LOGICAL_DRIVES_EXT]LD_INFO
}

type LD_INFO struct {
	Ref struct {
		TargetId uint8
		_        uint8
		SeqNum   uint16
	}
	State uint8
	_     [3]uint8
	Size  uint64 // unit: sector
}

func (ld *LD_INFO) GetState() string {
	var status string
	switch ld.State {
	case 2:
		status = "Degraded"
	case 3:
		status = "Optimal"
	default:
		status = "Unknown"
	}

	return status
}

func (ld *LD_INFO) GetSize() string {
	return SizeString(ld.Size)
}

type MR_LD_TARGETID_LIST struct {
	Size     uint32
	Count    uint32
	_        [3]uint8
	TargetId [MAX_LOGICAL_DRIVES_EXT]uint8
}

type MR_HOST_DEVICE_LIST struct {
	Size           uint32
	Count          uint32
	_              [2]uint32
	HostDeviceList [1]MR_HOST_DEVICE_LIST_ENTRY
}

type MR_HOST_DEVICE_LIST_ENTRY struct {
	Flags struct {
		Bits uint8
		// u8 is_sys_pd:1;
		// u8 reserved:7;
	}
	ScsiType uint8
	TargetId uint16
	_        [4]uint8
	SasAddr  [4]uint32 //	[2]uint64
} // __packed

/*
 * SAS controller information
 */
type megasas_ctrl_info struct {
	/*
	 * PCI device information
	 */
	Pci struct {
		VendorId    uint16
		DeviceId    uint16
		SubVendorId uint16
		SubDeviceId uint16
		_           [24]uint8
	} // __packed
	/*
	 * Host interface information
	 */
	HostInterface struct {
		Bits uint8
		// u8 PCIX:1;
		// 	u8 PCIE:1;
		// 	u8 iSCSI:1;
		// 	u8 SAS_3G:1;
		// 	u8 SRIOV:1;
		// 	u8 reserved_0:3;
		_         [6]uint8
		PortCount uint8
		PortAddr  [16]uint32 //	[8]uint64
	} // __packed

	/*
	 * Device (backend) interface information
	 */
	DeviceInterface struct {
		Bits uint8
		// u8 SPI:1;
		// u8 SAS_3G:1;
		// u8 SATA_1_5G:1;
		// u8 SATA_3G:1;
		// u8 reserved_0:4;
		_         [6]uint8
		PortCount uint8
		PortAddr  [16]uint32 // [8]uint64
	} // __packed
	/*
	 * List of components residing in flash. All str are null terminated
	 */
	ImageCheckWord      uint32
	ImageComponentCount uint32

	ImageComponent [8]struct {
		Name      [8]byte // string
		Version   [32]byte
		BuildDate [16]byte
		BuiltTime [16]byte
	} // __packed
	/*
	 * List of flash components that have been flashed on the card, but
	 * are not in use, pending reset of the adapter. This list will be
	 * empty if a flash operation has not occurred. All stings are null
	 * terminated
	 */
	PendingImageComponentCount uint32

	PendingImageComponent [8]struct {
		Name      [8]byte
		Version   [32]byte
		BuildDate [16]byte
		BuiltTime [16]byte
	} // __packed

	MaxArms   uint8
	MaxSpans  uint8
	MaxArrays uint8
	MaxLds    uint8

	ProductName [80]byte // string
	SerialNo    [32]byte // string

	/*
	 * Other physical/controller/operation information. Indicates the
	 * presence of the hardware
	 */
	HwPresent struct {
		Bits uint32
		// u32 bbu:1;
		// u32 alarm:1;
		// u32 nvram:1;
		// u32 uart:1;
		// u32 reserved:28;

	}

	CurrentFwTime uint32

	/*
	 * Maximum data transfer sizes
	 */
	MaxConcurrentCmds uint16
	MaxSgeCount       uint16
	MaxRequestSize    uint32

	/*
	 * Logical and physical device counts
	 */
	LdPresentCount  uint16
	LdDegradedCount uint16
	LdOfflineCount  uint16

	PdPresentCount         uint16
	PdDiskPresentCount     uint16
	PdDiskPredFailureCount uint16
	PdDiskFailedCount      uint16

	/*
	 * Memory size information
	 */
	NvramSize  uint16
	MemorySize uint16
	FlashSize  uint16

	/*
	 * Error counters
	 */
	MemCorrectableErrorCount   uint16
	MemUncorrectableErrorCount uint16

	/*
	 * Cluster information
	 */
	ClusterPermitted uint8
	ClusterActive    uint8

	/*
	 * Additional max data transfer sizes
	 */
	MaxStripsPerIo uint16

	/*
	 * Controller capabilities structures
	 */
	RaidLevels struct {
		Bits uint32
		// u32 raid_level_0:1;
		// u32 raid_level_1:1;
		// u32 raid_level_5:1;
		// u32 raid_level_1E:1;
		// u32 raid_level_6:1;
		// u32 reserved:27;
	}

	AdapterOperations struct {
		Bits uint32
		// u32 rbld_rate:1;
		// u32 cc_rate:1;
		// u32 bgi_rate:1;
		// u32 recon_rate:1;
		// u32 patrol_rate:1;
		// u32 alarm_control:1;
		// u32 cluster_supported:1;
		// u32 bbu:1;
		// u32 spanning_allowed:1;
		// u32 dedicated_hotspares:1;
		// u32 revertible_hotspares:1;
		// u32 foreign_config_import:1;
		// u32 self_diagnostic:1;
		// u32 mixed_redundancy_arr:1;
		// u32 global_hot_spares:1;
		// u32 reserved:17;
	}

	LdOperations struct {
		Bits uint32
		// u32 read_policy:1;
		// u32 write_policy:1;
		// u32 io_policy:1;
		// u32 access_policy:1;
		// u32 disk_cache_policy:1;
		// u32 reserved:27;
	}

	StripeSzOps struct {
		Min uint8
		Max uint8
		_   [2]uint8
	}

	PdOperations struct {
		Bits uint32
		// u32 force_online:1;
		// u32 force_offline:1;
		// u32 force_rebuild:1;
		// u32 reserved:29;
	}

	PdMixSupport struct {
		Bits uint32
		// u32 ctrl_supports_sas:1;
		// u32 ctrl_supports_sata:1;
		// u32 allow_mix_in_encl:1;
		// u32 allow_mix_in_ld:1;
		// u32 allow_sata_in_cluster:1;
		// u32 reserved:27;
	}

	/*
	 * Define ECC single-bit-error bucket information
	 */
	EccBucketCount uint8
	_              [11]uint8

	/*
	 * Include the controller properties (changeable items)
	 */
	Properties megasas_ctrl_prop

	/*
	 * Define FW pkg version (set in envt v'bles on OEM basis)
	 */
	PackageVersion [0x60]byte

	/*
	* If adapterOperations.supportMoreThan8Phys is set,
	* and deviceInterface.portCount is greater than 8,
	* SAS Addrs for first 8 ports shall be populated in
	* deviceInterface.portAddr, and the rest shall be
	* populated in deviceInterfacePortAddr2.
	 */
	DeviceInterfacePortAddr2 [16]uint32 // [8]uint64
	_                        [128]uint8 /*6e0h 1760 */

	PdsForRaidLevels struct { /*760h */
		// u16 minPdRaidLevel_0:4;
		// u16 maxPdRaidLevel_0:12;
		Bits1 uint16
		// u16 minPdRaidLevel_1:4;
		// u16 maxPdRaidLevel_1:12;
		Bits2 uint16
		// u16 minPdRaidLevel_5:4;
		// u16 maxPdRaidLevel_5:12;
		Bits3 uint16
		// u16 minPdRaidLevel_1E:4;
		// u16 maxPdRaidLevel_1E:12;
		Bits4 uint16
		// u16 minPdRaidLevel_6:4;
		// u16 maxPdRaidLevel_6:12;
		Bits5 uint16
		// u16 minPdRaidLevel_10:4;
		// u16 maxPdRaidLevel_10:12;
		Bits6 uint16
		// u16 minPdRaidLevel_50:4;
		// u16 maxPdRaidLevel_50:12;
		Bits7 uint16
		// u16 minPdRaidLevel_60:4;
		// u16 maxPdRaidLevel_60:12;
		Bits8 uint16
		// u16 minPdRaidLevel_1E_RLQ0:4;
		// u16 maxPdRaidLevel_1E_RLQ0:12;
		Bits9 uint16
		// u16 minPdRaidLevel_1E0_RLQ0:4;
		// u16 maxPdRaidLevel_1E0_RLQ0:12;
		Bits10 uint16
		_      [6]uint16
	}
	MaxPds          uint16 /*780h */
	MaxDedHSPs      uint16 /*782h */
	MaxGlobalHSP    uint16 /*784h */
	DdfSize         uint16 /*786h */
	MaxLdsPerArray  uint8  /*788h */
	PartitionsInDDF uint8  /*789h */
	LockKeyBinding  uint8  /*78ah */
	MaxPITsPerLd    uint8  /*78bh */
	MaxViewsPerLd   uint8  /*78ch */
	MaxTargetId     uint8  /*78dh */
	MaxBvlVdSize    uint16 /*78eh */

	MaxConfigurableSSCSize uint16 /*790h */
	CurrentSSCsize         uint16 /*792h */

	ExpanderFwVersion [12]byte /*794h */

	PFKTrialTimeRemaining uint16 /*7A0h */

	CacheMemorySize uint16 /*7A2h */

	AdapterOperations2 struct { /*7A4h */
		Bits uint32
		// u32     supportPIcontroller:1;
		// u32     supportLdPIType1:1;
		// u32     supportLdPIType2:1;
		// u32     supportLdPIType3:1;
		// u32     supportLdBBMInfo:1;
		// u32     supportShieldState:1;
		// u32     blockSSDWriteCacheChange:1;
		// u32     supportSuspendResumeBGops:1;
		// u32     supportEmergencySpares:1;
		// u32     supportSetLinkSpeed:1;
		// u32     supportBootTimePFKChange:1;
		// u32     supportJBOD:1;
		// u32     disableOnlinePFKChange:1;
		// u32     supportPerfTuning:1;
		// u32     supportSSDPatrolRead:1;
		// u32     realTimeScheduler:1;

		// u32     supportResetNow:1;
		// u32     supportEmulatedDrives:1;
		// u32     headlessMode:1;
		// u32     dedicatedHotSparesLimited:1;

		// u32     supportUnevenSpans:1;
		// u32	supportPointInTimeProgress:1;
		// u32	supportDataLDonSSCArray:1;
		// u32	mpio:1;
		// u32	supportConfigAutoBalance:1;
		// u32	activePassive:2;
		// u32     reserved:5;
	} // __packed

	DriverVersion        [32]uint8 /*7A8h */
	MaxDAPdCountSpinup60 uint8     /*7C8h */
	TemperatureROC       uint8     /*7C9h */
	TemperatureCtrl      uint8     /*7CAh */
	_                    uint8     /*7CBh */
	MaxConfigurablePds   uint16    /*7CCh */

	_ [2]uint8 /*0x7CDh */

	/*
	* HA cluster information
	 */
	Cluster struct {
		Bits uint32
		// u32     peerIsPresent:1;
		// u32     peerIsIncompatible:1;
		// u32     hwIncompatible:1;
		// u32     fwVersionMismatch:1;
		// u32     ctrlPropIncompatible:1;
		// u32     premiumFeatureMismatch:1;
		// u32     passive:1;
		// u32     reserved:25;
	}
	ClusterId [MEGASAS_CLUSTER_ID_SIZE]byte /*0x7D4 */
	Iov       struct {
		MaxVFsSupported uint8 /*0x7E4*/
		NumVFsEnabled   uint8 /*0x7E5*/
		RequestorId     uint8 /*0x7E6 0:PF, 1:VF1, 2:VF2*/
		Reserved        uint8 /*0x7E7*/
	}

	AdapterOperations3 struct {
		Bits uint32
		// u32     supportPersonalityChange:2;
		// u32     supportThermalPollInterval:1;
		// u32     supportDisableImmediateIO:1;
		// u32     supportT10RebuildAssist:1;
		// u32		supportMaxExtLDs:1;
		// u32		supportCrashDump:1;
		// u32     supportSwZone:1;
		// u32     supportDebugQueue:1;
		// u32     supportNVCacheErase:1;
		// u32     supportForceTo512e:1;
		// u32     supportHOQRebuild:1;
		// u32     supportAllowedOpsforDrvRemoval:1;
		// u32     supportDrvActivityLEDSetting:1;
		// u32     supportNVDRAM:1;
		// u32     supportForceFlash:1;
		// u32     supportDisableSESMonitoring:1;
		// u32     supportCacheBypassModes:1;
		// u32     supportSecurityonJBOD:1;
		// u32     discardCacheDuringLDDelete:1;
		// u32     supportTTYLogCompression:1;
		// u32     supportCPLDUpdate:1;
		// u32     supportDiskCacheSettingForSysPDs:1;
		// u32     supportExtendedSSCSize:1;
		// u32     useSeqNumJbodFP:1;
		// u32     reserved:7;
	} // _packed

	Cpld struct {
		Bits uint8
		// u8 cpld_in_flash:1;
		// u8 reserved:7;
		_ [3]uint8
		/* Null terminated string. Has the version
		 *  information if cpld_in_flash = FALSE
		 */
		UserCodeDefinition [12]uint8
	} /* Valid only if upgradableCPLD is TRUE */

	AdapterOperations4 struct {
		Bits uint16
		// u16 ctrl_info_ext_supported:1;
		// u16 support_ibutton_less:1;
		// u16 supported_enc_algo:1;
		// u16 support_encrypted_mfc:1;
		// u16 image_upload_supported:1;
		/* FW supports LUN based association and target port based */
		// u16 support_ses_ctrl_in_multipathcfg:1;
		/* association for the SES device connected in multipath mode */
		/* FW defines Jbod target Id within MR_PD_CFG_SEQ */
		// u16 support_pd_map_target_id:1;
		/* FW swaps relevant fields in MR_BBU_VPD_INFO_FIXED to
		*  provide the data in little endian order
		 */
		// u16 fw_swaps_bbu_vpd_info:1;
		// u16 support_ssc_rev3:1;
		/* FW supports CacheCade 3.0, only one SSCD creation allowed */
		// u16 support_dual_fw_update:1;
		/* FW supports dual firmware update feature */
		// u16 support_host_info:1;
		/* FW supports MR_DCMD_CTRL_HOST_INFO_SET/GET */
		// u16 support_flash_comp_info:1;
		/* FW supports MR_DCMD_CTRL_FLASH_COMP_INFO_GET */
		// u16 support_pl_debug_info:1;
		/* FW supports retrieval of PL debug information through apps */
		// u16 support_nvme_passthru:1;
		/* FW supports NVMe passthru commands */
		// u16 reserved:2;
	}

	_ [0x800 - 0x7FE]uint8 /* 0x7FE pad to 2K for expansion  2048 - 2046 */

	Size uint32
	_    uint32
	_    [64]uint8

	AdapterOperations5 struct {
		Bits uint32
		// u32 mr_config_ext2_supported:1;
		// u32 support_profile_change:2;
		// u32 support_cvhealth_info:1;
		// u32 support_pcie:1;
		// u32 support_ext_mfg_vpd:1;
		// u32 support_oce_only:1;
		// u32 support_nvme_tm:1;
		// u32 support_snap_dump:1;
		// u32 support_fde_type_mix:1;
		// u32 support_force_personality_change:1;
		// u32 support_psoc_update:1;
		// u32 support_pci_lane_margining: 1;
		// u32 reserved:19;
	}

	RsvdForAdptOp [63]uint32

	_ [3]uint8

	TaskAbortTO uint8 /* Timeout value in seconds used by Abort Task TM Request. */
	MaxResetTO  uint8 /* Max Supported Reset timeout in seconds. */
	_           [3]uint8
} // __packed

/*
 * SAS controller properties
 */
type megasas_ctrl_prop struct {
	SeqNum                        uint16
	PredFailPollInterval          uint16
	IntrThrottleCount             uint16
	IntrThrottleTimeouts          uint16
	RebuildRate                   uint8
	PatrolReadRate                uint8
	BgiRate                       uint8
	CcRate                        uint8
	ReconRate                     uint8
	CacheFlushInterval            uint8
	SpinupDrvCount                uint8
	SpinupDelay                   uint8
	ClusterEnable                 uint8
	CoercionMode                  uint8
	AlarmEnable                   uint8
	DisableAutoRebuild            uint8
	DisableBatteryWarn            uint8
	EccBucketSize                 uint8
	EccBucketLeakRate             uint16
	RestoreHotspareOnInsertion    uint8
	ExposeEnclDevices             uint8
	MaintainPdFailHistory         uint8
	DisallowHostRequestReordering uint8
	AbortCCOnError                uint8
	LoadBalanceMode               uint8
	DisableAutoDetectBackplane    uint8

	SnapVDSpace uint8
	/*
	* Add properties that can be controlled by
	* a bit in the following structure.
	 */
	OnOffProperties struct {
		Bits uint32
		// u32     copyBackDisabled:1;
		// u32     SMARTerEnabled:1;
		// u32     prCorrectUnconfiguredAreas:1;
		// u32     useFdeOnly:1;
		// u32     disableNCQ:1;
		// u32     SSDSMARTerEnabled:1;
		// u32     SSDPatrolReadEnabled:1;
		// u32     enableSpinDownUnconfigured:1;
		// u32     autoEnhancedImport:1;
		// u32     enableSecretKeyControl:1;
		// u32     disableOnlineCtrlReset:1;
		// u32     allowBootWithPinnedCache:1;
		// u32     disableSpinDownHS:1;
		// u32     enableJBOD:1;
		// u32     reserved:18;

	}
	OnOffProperties2 struct {
		Bits uint16
		// u16 reserved1:4;
		// u16 enable_snap_dump:1;
		// u16 reserved2:1;
		// u16 enable_fw_dev_list:1;
		// u16 reserved3:9;
	}

	SpinDownTime uint16
	Reserved     [24]uint8
} // __packed

func (ctrl *megasas_ctrl_info) GetProductName() string {
	return string(ctrl.ProductName[:])
}

func (ctrl *megasas_ctrl_info) SerialNumber() string {
	return string(ctrl.SerialNo[:])
}
func (ctrl *megasas_ctrl_info) JbodEnabled() bool {
	return BitField(ctrl.Properties.OnOffProperties.Bits, 13, 1) == 1
}
