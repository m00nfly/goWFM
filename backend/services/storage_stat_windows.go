//go:build windows

package services

import (
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

func diskSpace(path string) (available uint64, total uint64, err error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeForCaller uint64
	var totalBytes uint64
	var totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &freeForCaller, &totalBytes, &totalFree); err != nil {
		return 0, 0, err
	}
	return freeForCaller, totalBytes, nil
}

type fileStandardInfo struct {
	AllocationSize int64
	EndOfFile      int64
	NumberOfLinks  uint32
	DeletePending  byte
	Directory      byte
}

func fileDiskUsage(path string, info os.FileInfo) (int64, string) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return info.Size(), ""
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return info.Size(), ""
	}
	defer windows.CloseHandle(handle)

	size := info.Size()
	var standard fileStandardInfo
	if err := windows.GetFileInformationByHandleEx(
		handle,
		windows.FileStandardInfo,
		(*byte)(unsafe.Pointer(&standard)),
		uint32(unsafe.Sizeof(standard)),
	); err == nil && standard.AllocationSize >= 0 {
		size = standard.AllocationSize
	}

	identity := ""
	var byHandle windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &byHandle); err == nil {
		fileIndex := uint64(byHandle.FileIndexHigh)<<32 | uint64(byHandle.FileIndexLow)
		identity = strconv.FormatUint(uint64(byHandle.VolumeSerialNumber), 10) + ":" + strconv.FormatUint(fileIndex, 10)
	}
	return size, identity
}
