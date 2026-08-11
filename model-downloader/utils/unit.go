package utils

import "fmt"

type SizeUnit uint64

const (
	SizeUnitByte SizeUnit = 1
	SizeUnitKB            = SizeUnitByte * 1000
	SizeUnitMB            = SizeUnitKB * 1000
	SizeUnitGB            = SizeUnitMB * 1000
	SizeUnitTB            = SizeUnitGB * 1000
	SizeUnitPB            = SizeUnitTB * 1000
	SizeUnitKiB           = SizeUnitByte << 10
	SizeUnitMiB           = SizeUnitByte << 20
	SizeUnitGiB           = SizeUnitByte << 30
	SizeUnitTiB           = SizeUnitByte << 40
	SizeUnitPiB           = SizeUnitByte << 50
)

func (su SizeUnit) String() string {
	switch su {
	case SizeUnitByte:
		return "byte"
	case SizeUnitKB:
		return "KB"
	case SizeUnitMB:
		return "MB"
	case SizeUnitGB:
		return "GB"
	case SizeUnitTB:
		return "TB"
	case SizeUnitPB:
		return "PB"
	case SizeUnitKiB:
		return "KiB"
	case SizeUnitMiB:
		return "MiB"
	case SizeUnitGiB:
		return "GiB"
	case SizeUnitTiB:
		return "TiB"
	case SizeUnitPiB:
		return "PiB"
	default:
		return "unknown"
	}
}

type Size struct {
	Raw       uint64   // 原始值
	SuitUnit  SizeUnit // 合适的单位
	SuitValue float64  // 合适的值
}

func (s Size) String() string {
	return fmt.Sprintf("%.2f%s", s.SuitValue, s.SuitUnit.String())
}

func ParseUint(size uint64) Size {
	result := Size{
		Raw: size,
	}
	s := []SizeUnit{SizeUnitByte, SizeUnitKB, SizeUnitMB, SizeUnitGB, SizeUnitTB, SizeUnitPB}
	finded := false
	for k, v := range s {
		if size/uint64(v) < 1 {
			t := max(k-1, 0)
			result.SuitUnit = s[t]
			finded = true
			break
		}
	}
	if !finded {
		result.SuitUnit = SizeUnitPB
	}
	result.SuitValue = float64(size) / float64(result.SuitUnit)
	return result
}

func ParseUintI(size uint64) Size {
	result := Size{
		Raw: size,
	}
	s := []SizeUnit{SizeUnitByte, SizeUnitKiB, SizeUnitMiB, SizeUnitGiB, SizeUnitTiB, SizeUnitPiB}
	finded := false
	for k, v := range s {
		if size/uint64(v) < 1 {
			t := max(k-1, 0)
			result.SuitUnit = s[t]
			finded = true
			break
		}
	}
	if !finded {
		result.SuitUnit = SizeUnitPB
	}
	result.SuitValue = float64(size) / float64(result.SuitUnit)
	return result
}
