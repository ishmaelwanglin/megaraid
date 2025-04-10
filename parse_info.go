package megaraid

import (
	"encoding/binary"
	"fmt"
	"math"
)

func SizeOfDisk(sectors uint64, unit string) uint64 {
	sizeBytes := sectors * 512

	var divisor uint64
	switch unit {
	case "KB":
		divisor = KB
	case "MB":
		divisor = MB
	case "GB":
		divisor = GB
	case "TB":
		divisor = TB
	default:
		divisor = 1 << 0
	}

	return sizeBytes / divisor
}

func SizeString(sectors uint64) string {
	t := float64(sectors*SectorSz) / TB
	if t > 1 {
		return fmt.Sprintf("%.2f TB", math.Round(t*100)/100)
	}
	g := float64(sectors*SectorSz) / GB
	if g > 1 {
		return fmt.Sprintf("%.2f GB", math.Round(g*100)/100)
	}
	m := float64(sectors*SectorSz) / MB
	if m > 1 {
		return fmt.Sprintf("%.2f MB", math.Round(m*100)/100)
	}
	k := float64(sectors*SectorSz) / KB
	return fmt.Sprintf("%.2f kB", math.Round(k*100)/100)
}

type Uint interface {
	~uint32 | ~uint64 | ~uint16 | ~uint8
}

// zip 2 uint64
func ArrayZip[T Uint](arr []T, offset uint) uint64 {
	n := len(arr)
	if n < 1 {
		panic("input array length must be greate than 1")
	}
	var (
		va uint64
	)

	for i := range n {
		if arr[i] == 0 {
			continue
		}
		if va == 0 {
			va = uint64(arr[i])
		} else {
			va = (uint64(arr[i]) << offset) | va
		}
	}

	return va
}

// jbod的naa正确，raid的需要再看
// ParseVpdPage83Jbod 解析 SCSI VPD Page 0x83 并提取 WWN
func ParseVpdPage83Jbod(data []byte) string {
	if len(data) < 4 {
		return "invalid data"
	}

	offset := 4
	for offset < len(data) {
		if offset+4 > len(data) {
			return "invalid length"
		}

		length := int(data[offset+3])
		if length == 0 || offset+4+length > len(data) {
			return "invalid length"
		}

		identifier := data[offset+4 : offset+4+length]

		// 确保 identifier 长度足够
		if len(identifier) >= 8 {
			naaType := (identifier[0] >> 4) & 0xF
			if naaType == 0x5 || naaType == 0x6 {
				// 只取最后 8 字节作为 WWN
				wwn := binary.BigEndian.Uint64(identifier[len(identifier)-8:])
				return fmt.Sprintf("naa.%016x", wwn)
			}
		}
		offset += 4 + length
	}
	return "no valid WWN found"
}
