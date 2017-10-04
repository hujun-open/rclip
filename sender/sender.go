// sender
package sender

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"rclip/common"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
)

type RclipSender struct {
	conn *tls.Conn
}

func NewRclipSender(ca_cert_path string, cert_path string, key_path string, svr_ip string, svr_port int, san_check bool) (*RclipSender, error) {
	new_sender := new(RclipSender)
	cer, err := tls.LoadX509KeyPair(cert_path, key_path)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	root_pem, err := ioutil.ReadFile(ca_cert_path)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(root_pem)
	if ok != true {
		return nil, common.MakeErrviaStr("error parsing root CA cert file")
	}
	var cfg *tls.Config
	if san_check == false {
		cfg = &tls.Config{Certificates: []tls.Certificate{cer}, RootCAs: roots, InsecureSkipVerify: false, ServerName: "1.1.1.1", MinVersion: tls.VersionTLS12}
	} else {
		cfg = &tls.Config{Certificates: []tls.Certificate{cer}, RootCAs: roots, InsecureSkipVerify: false, ServerName: svr_ip, MinVersion: tls.VersionTLS12}
	}
	tcp_conn, err := net.Dial("tcp", svr_ip+":"+strconv.Itoa(svr_port))
	if err != nil {
		return nil, common.MakeErr(err)
	}
	err = tcp_conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	new_sender.conn = tls.Client(tcp_conn, cfg)
	//	new_sender.conn, err = tls.Dial("tcp", svr_ip+":"+strconv.Itoa(svr_port), cfg)
	//	if err != nil {
	//		return nil, common.MakeErr(err)
	//	}

	return new_sender, nil
}

func (sender *RclipSender) Receive() {
	//keep receiving from reflector
	common.InfoLog.Printf("ready to receive from %v ... ", sender.conn.RemoteAddr())
	defer sender.conn.Close()
	go sender.sendKeepalive()
	for {
		sender.conn.SetReadDeadline(time.Now().Add(common.MAX_HELLO_LOST * common.HELLO_INTERVAL * time.Second))
		msg, err := common.ReadNextMsg(sender.conn)
		if err != nil {
			common.ErrLog.Printf("error reading from %v, %v", sender.conn.RemoteAddr(), err)
			return
		}
		if string(msg) == common.HELLO_MSG {
			continue
		}
		if err = clipboard.WriteAll(string(msg)); err != nil {
			common.WarnLog.Printf("error copy rcvd msg from %v into clipbaord, %v", sender.conn.RemoteAddr(), err)
			return
		}
		common.InfoLog.Printf("copied %v bytes msg from %v to clipboard", len(msg), sender.conn.RemoteAddr())
	}

}

func (sender *RclipSender) sendKeepalive() {
	for {
		encap_msg := common.EncapMsg([]byte(common.HELLO_MSG))
		sender.conn.SetWriteDeadline(time.Now().Add(common.HELLO_INTERVAL * time.Second))
		_, err := sender.conn.Write(encap_msg)
		if err != nil {
			common.WarnLog.Printf("error sending keepalive to %v", sender.conn.RemoteAddr())
			return
		}
		time.Sleep(common.HELLO_INTERVAL * time.Second)
	}
}
