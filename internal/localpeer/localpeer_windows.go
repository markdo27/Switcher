package localpeer

import (
	"encoding/binary"
	"errors"
	"golang.org/x/sys/windows"
	"net"
	"os"
	"time"
	"unsafe"
)

const lookupsAvailable = true
const checksAtOnce = 32

var tcpTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

type windowsSocket struct {
	local, remote *net.TCPAddr
	pid, state    uint32
}

func windowsSockets(ipv6 bool) ([]windowsSocket, error) {
	family := uintptr(windows.AF_INET)
	width := 24
	if ipv6 {
		family = windows.AF_INET6
		width = 56
	}
	var size uint32
	result, _, _ := tcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, family, 5, 0)
	if result != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
		return nil, windows.Errno(result)
	}
	for attempt := 0; attempt < 3; attempt++ {
		if size < 4 || size > 16<<20 {
			return nil, errors.New("invalid Windows TCP table size")
		}
		data := make([]byte, size)
		result, _, _ = tcpTable.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&size)), 0, family, 5, 0)
		if result == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
			continue
		}
		if result != 0 {
			return nil, windows.Errno(result)
		}
		count := binary.LittleEndian.Uint32(data[:4])
		if uint64(count)*uint64(width)+4 > uint64(len(data)) {
			return nil, errors.New("truncated Windows TCP table")
		}
		sockets := make([]windowsSocket, 0, count)
		for i := 0; i < int(count); i++ {
			row := data[4+i*width : 4+(i+1)*width]
			var s windowsSocket
			if ipv6 {
				s.local = &net.TCPAddr{IP: net.IP(row[:16]), Port: int(binary.BigEndian.Uint16(row[20:22]))}
				s.remote = &net.TCPAddr{IP: net.IP(row[24:40]), Port: int(binary.BigEndian.Uint16(row[44:46]))}
				s.state = binary.LittleEndian.Uint32(row[48:52])
				s.pid = binary.LittleEndian.Uint32(row[52:56])
			} else {
				s.state = binary.LittleEndian.Uint32(row[:4])
				s.local = &net.TCPAddr{IP: net.IP(row[4:8]), Port: int(binary.BigEndian.Uint16(row[8:10]))}
				s.remote = &net.TCPAddr{IP: net.IP(row[12:16]), Port: int(binary.BigEndian.Uint16(row[16:18]))}
				s.pid = binary.LittleEndian.Uint32(row[20:24])
			}
			sockets = append(sockets, s)
		}
		return sockets, nil
	}
	return nil, errTransient
}
func processUID(pid uint32) (uint32, bool) {
	if pid == uint32(os.Getpid()) {
		return ^uint32(0), true
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token) != nil {
		return 0, false
	}
	defer token.Close()
	owner, err := token.GetTokenUser()
	if err != nil {
		return 0, false
	}
	me, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return 0, false
	}
	if owner.User.Sid.String() == me.User.Sid.String() {
		return ^uint32(0), true
	}
	return 1, true // Windows has SIDs rather than Unix UIDs; 1 means a different user.
}
func sameAddress(a, b *net.TCPAddr) bool { return a.Port == b.Port && a.IP.Equal(b.IP) }
func connectedUID(local, remote *net.TCPAddr) (uint32, bool) {
	rows, err := windowsSockets(local.IP.To4() == nil)
	if err != nil {
		return 0, false
	}
	for _, s := range rows {
		if s.state == 5 && sameAddress(s.local, local) && sameAddress(s.remote, remote) {
			return processUID(s.pid)
		}
	}
	return 0, false
}
func listenerUID(addr *net.TCPAddr) (uint32, bool) {
	rows, err := windowsSockets(addr.IP.To4() == nil)
	if err != nil {
		return 0, false
	}
	for _, s := range rows {
		if s.state == 2 && s.local.Port == addr.Port && (s.local.IP.IsUnspecified() || s.local.IP.Equal(addr.IP)) {
			return processUID(s.pid)
		}
	}
	return 0, false
}
func peerOf(_ net.Conn, local, remote *net.TCPAddr, _ time.Time) (uint32, peerKind) {
	uid, ok := connectedUID(remote, local)
	if ok {
		return uid, peerFound
	}
	return 0, peerUnseen
}
func queryFailure(local, remote *net.TCPAddr) error {
	_, err := windowsSockets(local.IP.To4() == nil)
	if err != nil {
		return errTransient
	}
	return errNotFound
}
func otherEndOpen(conn net.Conn) bool {
	local, _ := conn.LocalAddr().(*net.TCPAddr)
	remote, _ := conn.RemoteAddr().(*net.TCPAddr)
	if local == nil || remote == nil {
		return false
	}
	_, ok := connectedUID(remote, local)
	return ok
}
