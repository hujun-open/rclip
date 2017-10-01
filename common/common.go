// common
package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var InfoLog = log.New(os.Stdout, "RCLIP-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
var WarnLog = log.New(os.Stdout, "RCLIP-WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
var ErrLog = log.New(os.Stderr, "RCLIP-ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

const HELLO_MSG = "RCLIPHELLO"
const HELLO_INTERVAL = 40
const MAX_HELLO_LOST = 3
const MAX_MSG_LEN = 640000

func MakeErr(ierr error) error {
	var buf bytes.Buffer
	pc, _, line, _ := runtime.Caller(1)
	logger := log.New(&buf, "", 0)
	logger.Printf("[%s:%d]: %v", runtime.FuncForPC(pc).Name(), line, ierr)
	return errors.New(buf.String())
}

func MakeErrviaStr(errs string) error {
	var buf bytes.Buffer
	pc, _, line, _ := runtime.Caller(1)
	logger := log.New(&buf, "", 0)
	logger.Printf("[%s:%d]: %v", runtime.FuncForPC(pc).Name(), line, errs)
	return errors.New(buf.String())
}

func GetConfDir() string {
	var defDir string
	switch runtime.GOOS {
	case "windows":
		defDir = filepath.Join(os.Getenv("APPDATA"), "rclip")
	case "linux", "darwin":
		defDir = filepath.Join(os.Getenv("HOME"), ".rclip")
	}
	redirectfilename := filepath.Join(defDir, "redirection.conf")
	redir, err := ioutil.ReadFile(redirectfilename)
	if err != nil {
		return defDir
	} else {
		redir_str := strings.TrimRight(string(redir), " 	\n\r")
		return redir_str
	}
	return ""
}

func ReadSpecificLen(msglen uint32, conn net.Conn) ([]byte, error) {

	bs := make([]byte, 0, msglen)
	buf := bytes.NewBuffer(bs)
	_, err := io.CopyN(buf, conn, int64(msglen))
	if err != nil {
		return nil, MakeErr(err)
	}
	return buf.Bytes(), nil
}

func ReadNextMsg(conn net.Conn) (msgval []byte, err error) {
	conn.SetReadDeadline(time.Time{})
	buf, err := ReadSpecificLen(uint32(4), conn)
	if err != nil {
		return nil, MakeErr(err)
	}
	msglen := binary.BigEndian.Uint32(buf)
	if msglen == 0 {
		return nil, MakeErrviaStr("rcvd a zero length msg")
	}
	if msglen > MAX_MSG_LEN {
		return nil, MakeErrviaStr(fmt.Sprintf("the length (%v) of rcvd msg excceed max length %v", msglen, MAX_MSG_LEN))
	}
	valbuf, err := ReadSpecificLen(msglen, conn)
	if err != nil {
		return nil, MakeErr(err)
	}
	return valbuf, nil

}

func EncapMsg(msg []byte) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(msg)))
	return append(buf, msg...)

}
