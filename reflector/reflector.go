// reflector
package reflector

import (
	"crypto/tls"
	"crypto/x509"
	"sync"
	"time"

	"io/ioutil"
	"net"
	"rclip/common"
	"strconv"
)

type Reflector struct {
	send_ln   net.Listener
	rcv_ln    net.Listener
	send_conn net.Conn //send copied text to remote client, long lasting
	send_cfg  tls.Config
}

func NewReflector(ca_cert_path string, cert_path string, key_path string, svr_ip string, send_port int, rcv_port int) (*Reflector, error) {
	cer, err := tls.LoadX509KeyPair(cert_path, key_path)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	new_ref := new(Reflector)
	root_pem, err := ioutil.ReadFile(ca_cert_path)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(root_pem)
	if ok != true {
		return nil, common.MakeErrviaStr("error parsing root CA cert file")
	}

	new_ref.send_cfg = tls.Config{Certificates: []tls.Certificate{cer}, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert}
	new_ref.send_ln, err = tls.Listen("tcp", svr_ip+":"+strconv.Itoa(send_port), &new_ref.send_cfg)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	rcv_cfg := &tls.Config{Certificates: []tls.Certificate{cer}, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS12}
	new_ref.rcv_ln, err = tls.Listen("tcp", svr_ip+":"+strconv.Itoa(rcv_port), rcv_cfg)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	return new_ref, nil

}

func (refl *Reflector) Start() {

	defer refl.send_ln.Close()
	defer refl.rcv_ln.Close()
	common.InfoLog.Println("starting reflector...")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := refl.send_ln.Accept()
			if err != nil {
				common.WarnLog.Printf("error accpeting new connection, %v", err)
				continue
			}
			go refl.handleSendConnection(conn)

		}
	}()
	go func() {
		defer wg.Done()
		for {
			conn, err := refl.rcv_ln.Accept()
			if err != nil {
				common.WarnLog.Printf("error accpeting new connection, %v", err)
				continue
			}
			go refl.handleRcvConnection(conn)

		}
	}()
	wg.Wait()
}

func (refl *Reflector) sendKeepalive() {
	for {

		refl.send_conn.SetWriteDeadline(time.Now().Add(common.HELLO_INTERVAL * common.MAX_HELLO_LOST * time.Second))
		_, err := refl.send_conn.Write(common.EncapMsg([]byte(common.HELLO_MSG)))
		if err != nil {
			common.WarnLog.Printf("error sending keepalive to %v", refl.send_conn.RemoteAddr())
			return
		}
		time.Sleep(common.HELLO_INTERVAL * time.Second)
	}

}

func (refl *Reflector) handleSendConnection(conn net.Conn) {
	//defer conn.Close()
	common.InfoLog.Printf("change sender connection to %v ", conn.RemoteAddr())
	if refl.send_conn != nil {
		refl.send_conn.Close()
	}
	refl.send_conn = conn
	go refl.sendKeepalive()
	go refl.rcvKeepAlive()

}

func (refl *Reflector) handleRcvConnection(conn net.Conn) {
	defer conn.Close()
	common.InfoLog.Printf("got a new receiver connection from %v ", conn.RemoteAddr())
	if refl.send_conn == nil {
		common.WarnLog.Println("there is no sender")
		return
	}
	msg, err := ioutil.ReadAll(conn)
	if err != nil {
		common.ErrLog.Printf("error reading from receiver, %v", err)
		return
	}
	common.InfoLog.Printf("rcvd %v bytes msg from %v", len(msg), conn.RemoteAddr())
	if len(msg) == 0 {
		//ignoring zero length msg
		return
	}
	encaped_msg := common.EncapMsg(msg)
	wrote_len, err := refl.send_conn.Write(encaped_msg)
	if err != nil {
		common.ErrLog.Printf("error sending text to sender, %v", err)
		return
	}
	if wrote_len != len(encaped_msg) {
		common.ErrLog.Printf("only sent %v of %v bytes", wrote_len, len(msg))
	}
	common.InfoLog.Printf("send %v bytes msg to %v", len(msg), refl.send_conn.RemoteAddr())
}

func (refl *Reflector) rcvKeepAlive() {
	for {
		if refl.send_conn == nil {
			continue
		}
		refl.send_conn.SetReadDeadline(time.Now().Add(common.MAX_HELLO_LOST * common.HELLO_INTERVAL * time.Second))
		_, err := common.ReadNextMsg(refl.send_conn)
		if err != nil {
			common.ErrLog.Printf("error getting keepalive from %v, %v", refl.send_conn.RemoteAddr(), err)
			refl.send_conn.Close()
			return
		}
	}
}
