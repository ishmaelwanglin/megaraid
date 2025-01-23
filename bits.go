package megaraid

func mask(size uint) uint {
	if size == 0 {
		return 0
	}
	size--
	return (1 << size) + mask(size)
}

type uintInterface interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// 获取位域的值
func BitField[T uintInterface](data T, offset uint, size uint) T {

	return (data >> offset) & T(mask(size))
}
