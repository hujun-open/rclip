// receiver
package receiver

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"rclip/common"
	"strconv"
)

type RclipRcv struct {
	conn *tls.Conn
}

func NewRclipRcv(ca_cert_path string, cert_path string, key_path string, svr_ip string, svr_port int, svn_check bool) (*RclipRcv, error) {
	new_rcv := new(RclipRcv)
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
	if svn_check == false {
		cfg = &tls.Config{Certificates: []tls.Certificate{cer}, RootCAs: roots, InsecureSkipVerify: false, ServerName: "1.1.1.1", MinVersion: tls.VersionTLS12}
	} else {
		cfg = &tls.Config{Certificates: []tls.Certificate{cer}, RootCAs: roots, InsecureSkipVerify: false, MinVersion: tls.VersionTLS12}
	}
	new_rcv.conn, err = tls.Dial("tcp", svr_ip+":"+strconv.Itoa(svr_port), cfg)
	if err != nil {
		return nil, common.MakeErr(err)
	}
	return new_rcv, nil
}

func (clnt *RclipRcv) ReadandSend(reader io.Reader) {
	//read from stdin and send to the peer
	msg, err := ioutil.ReadAll(reader)
	if err != nil {
		common.ErrLog.Printf("error reading, %v", err)
		return
	}
	if len(msg) == 0 {
		common.InfoLog.Printf("ignoring zero length input")
		return
	}
	_, err = clnt.conn.Write(msg)
	if err != nil {
		common.ErrLog.Printf("error sending, %v", err)
		return
	}

}
