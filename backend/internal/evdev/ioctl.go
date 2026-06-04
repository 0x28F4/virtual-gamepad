package evdev

import (
	"encoding/binary"
	"os"
	"syscall"
	"unsafe"
)

const (
	uinputPath = "/dev/uinput"
	uiBase     = 'U'

	iocNone  = 0x0
	iocWrite = 0x1

	iocNrBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14

	iocNrShift   = 0
	iocTypeShift = iocNrShift + iocNrBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits
)

var (
	uiDevCreate  = io(uiBase, 1)
	uiDevDestroy = io(uiBase, 2)
	uiDevSetup   = iow(uiBase, 3, int(unsafe.Sizeof(uinputSetup{})))
	uiAbsSetup   = iow(uiBase, 4, int(unsafe.Sizeof(uinputAbsSetup{})))
	uiSetEVBit   = iow(uiBase, 100, int(unsafe.Sizeof(int32(0))))
	uiSetKeyBit  = iow(uiBase, 101, int(unsafe.Sizeof(int32(0))))
	uiSetAbsBit  = iow(uiBase, 103, int(unsafe.Sizeof(int32(0))))
)

type inputID struct {
	BusType uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type uinputSetup struct {
	ID         inputID
	Name       [80]byte
	EffectsMax uint32
}

type inputAbsInfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

type uinputAbsSetup struct {
	Code    uint16
	_       uint16
	AbsInfo inputAbsInfo
}

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type Device struct {
	file *os.File
}

type DeviceConfig struct {
	Name    string
	Vendor  uint16
	Product uint16
	Version uint16
	Keys    []uint16
	Axes    map[uint16]AbsInfo
}

func OpenDevice(config DeviceConfig) (*Device, error) {
	file, err := os.OpenFile(uinputPath, os.O_WRONLY|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	device := &Device{file: file}

	if err := device.setup(config); err != nil {
		_ = file.Close()
		return nil, err
	}
	return device, nil
}

func (d *Device) Close() error {
	if d.file == nil {
		return nil
	}
	file := d.file
	d.file = nil
	destroyErr := ioctl(file.Fd(), uiDevDestroy, 0)
	closeErr := file.Close()
	if destroyErr != nil {
		return destroyErr
	}
	return closeErr
}

func (d *Device) Emit(events []EventValue) error {
	for _, event := range events {
		if err := d.emit(event.Type, event.Code, event.Value); err != nil {
			return err
		}
	}
	return d.emit(EVSyn, SynReport, 0)
}

func (d *Device) setup(config DeviceConfig) error {
	if err := ioctl(d.file.Fd(), uiSetEVBit, EVKey); err != nil {
		return err
	}
	if err := ioctl(d.file.Fd(), uiSetEVBit, EVAbs); err != nil {
		return err
	}

	for _, key := range config.Keys {
		if err := ioctl(d.file.Fd(), uiSetKeyBit, uintptr(key)); err != nil {
			return err
		}
	}

	for axis, info := range config.Axes {
		if err := ioctl(d.file.Fd(), uiSetAbsBit, uintptr(axis)); err != nil {
			return err
		}
		abs := uinputAbsSetup{
			Code: axis,
			AbsInfo: inputAbsInfo{
				Minimum: info.Min,
				Maximum: info.Max,
				Fuzz:    info.Fuzz,
				Flat:    info.Flat,
			},
		}
		if err := ioctl(d.file.Fd(), uiAbsSetup, unsafe.Pointer(&abs)); err != nil {
			return err
		}
	}

	setup := uinputSetup{
		ID: inputID{
			BusType: BusUSB,
			Vendor:  config.Vendor,
			Product: config.Product,
			Version: config.Version,
		},
	}
	copy(setup.Name[:], config.Name)
	if err := ioctl(d.file.Fd(), uiDevSetup, unsafe.Pointer(&setup)); err != nil {
		return err
	}
	return ioctl(d.file.Fd(), uiDevCreate, 0)
}

func (d *Device) emit(eventType, code uint16, value int32) error {
	event := inputEvent{
		Type:  eventType,
		Code:  code,
		Value: value,
	}
	return binary.Write(d.file, nativeEndian{}, event)
}

func ioctl(fd uintptr, name uintptr, data any) error {
	var value uintptr
	switch typed := data.(type) {
	case int:
		value = uintptr(typed)
	case uint16:
		value = uintptr(typed)
	case uint32:
		value = uintptr(typed)
	case uintptr:
		value = typed
	case unsafe.Pointer:
		value = uintptr(typed)
	default:
		panic("unsupported ioctl argument")
	}

	_, _, errno := syscall.RawSyscall(syscall.SYS_IOCTL, fd, name, value)
	if errno != 0 {
		return errno
	}
	return nil
}

func ioc(dir, typ, nr, size int) uintptr {
	return uintptr((dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNrShift) | (size << iocSizeShift))
}

func io(typ, nr int) uintptr {
	return ioc(iocNone, typ, nr, 0)
}

func iow(typ, nr, size int) uintptr {
	return ioc(iocWrite, typ, nr, size)
}

type nativeEndian struct{}

func (nativeEndian) Uint16(b []byte) uint16       { return binary.LittleEndian.Uint16(b) }
func (nativeEndian) PutUint16(b []byte, v uint16) { binary.LittleEndian.PutUint16(b, v) }
func (nativeEndian) Uint32(b []byte) uint32       { return binary.LittleEndian.Uint32(b) }
func (nativeEndian) PutUint32(b []byte, v uint32) { binary.LittleEndian.PutUint32(b, v) }
func (nativeEndian) Uint64(b []byte) uint64       { return binary.LittleEndian.Uint64(b) }
func (nativeEndian) PutUint64(b []byte, v uint64) { binary.LittleEndian.PutUint64(b, v) }
func (nativeEndian) String() string               { return "native" }
