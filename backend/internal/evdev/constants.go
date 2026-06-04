package evdev

const (
	BusUSB = 0x03

	EVSyn = 0x00
	EVKey = 0x01
	EVAbs = 0x03

	SynReport = 0

	BtnSouth  = 0x130
	BtnEast   = 0x131
	BtnNorth  = 0x133
	BtnWest   = 0x134
	BtnTL     = 0x136
	BtnTR     = 0x137
	BtnTL2    = 0x138
	BtnTR2    = 0x139
	BtnSelect = 0x13a
	BtnStart  = 0x13b
	BtnMode   = 0x13c
	BtnThumbL = 0x13d
	BtnThumbR = 0x13e

	AbsX     = 0x00
	AbsY     = 0x01
	AbsZ     = 0x02
	AbsRX    = 0x03
	AbsRY    = 0x04
	AbsRZ    = 0x05
	AbsHat0X = 0x10
	AbsHat0Y = 0x11
)

type AbsInfo struct {
	Min  int32
	Max  int32
	Fuzz int32
	Flat int32
}

type EventValue struct {
	Type  uint16
	Code  uint16
	Value int32
}
