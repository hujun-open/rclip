# rclip
rclip is a tool copy text from remote host to local system cliboard; 

it is specifically designed for copy seletected text in tmux copy-mode on a remote host to local system cliboard over network; which is a big pain point of using tmux;

**Note: You don't need rclip if your terminal supports [OSC 52](https://github.com/tmux/tmux/wiki/Clipboard#how-it-works)**

## how does it work?
first run rclip as reflector role on remote host, which is processing listening on two ports over TLS:
* rcv_port: default 8891
* send_port: default 8890

the ports could be changed by commandline parameters

any text received on rcv_port will be forwarded to send_port   

run rclip as sender role on local system, it connects to the send_port of reflector, and copy the received text to local system clipboard

configure tmux so that the tmux copy-pipe command invoke rclip as receiver role, which connects to rcv_port of reflector, and send selected text to reflector;

## why this 3 party design?
this is to get around firewall between remote host and local system; remote host is typicall a server, while local system is typically a client device like a laptop;

if you use VPN to remote access the server, there typically will be a firewall between server and local system, and that firewall will likely to block any connection initiated by server to the local system; so a simple client and server design won't work

rclip also send keepalive message every 40 seconds between reflector and sender to avoid firewall idle timeout the TCP connection;



## security

rclip use TLS to secure communication between reflector, sender and receiver; it also uses client authentication with its own root CA to prevent spoof;

by default, rclip sender and receiver will check SAN(Subject Alternative Name) of reflector's certificate, to see the reflector's IP or FQDN match its certificate SAN; however this check could be skipped by using a hard coded "1.1.1.1" in SAN of reflector certificate when parameter "-loose" is specified; this is to avoid hassle to create a different certificate for each remote host; this of course decrease the overall security, but personally think it is acceptable trade-off given user typically specify the reflector address on local system directly;

## build
rclip is coded with golang ver1.9, just use "go build" in the source directory to build the binary;
golang pkg required:
* github.com/atotto/clipboard

## install
since rclip use TLS and its own CA, so following key and certificates are needed to generated before installation:
* root CA cert/key
* reflector key/cert
* sender key/cert
* receiver key/cert

notes:
  * there are many opensource tools could generate key and certs like [openssl](https://www.openssl.org/) or [XCA](http://xca.sourceforge.net/)
  * If you want to skip SAN check, make sure there is SubjectAltName extension with "1.1.1.1" as ip addresss in the certificate of reflector


rclip expect above key/certs located in following directory:
* Windows: [windows_user_dir]\appdata\AppData\Roaming\rclip
* Linux/OSX: $HOME/.rclip/


on remote host where tmux is running, following cert/keys with expected file name are needed:
* root CA cert: ca_cert
* reflector cert/key: refl_cert/refl_key
* receiver cert/key: rcv_cert/rcv_key

on local system, following cert/keys are needed:
* root CA cert: ca_cert
* sender key/cert: sender_cert/sender_key

all cert/key files's permission should be set that only owner could read

add following line in your .tmux.conf on remote host:
```
bind-key -T copy-mode-vi _your_key_of_choice_ send-keys -X copy-pipe "_rclip_install_path_/rclip"
```
this is an example:
```
bind-key -T copy-mode-vi y send-keys -X copy-pipe "/usr/local/bin/rclip"
```
note: if you want to skip SAN check, add "-loose"; e.g. 
```
bind-key -T copy-mode-vi _your_key_of_choice_ send-keys -X copy-pipe "_rclip_install_path_/rclip -loose"
```
note: invoke rclip without any parameter, it will run as receiver, and connects to reflector on the same host, using default recv_port

## usage
on remote host:
* start reflector process: 
```
rclip -role refl
``` 

on local system:
* start sender process: 
```
rclip -role sender -refl_ip <remote_host_ip>
```
note: if you want to skip SAN check, add "-loose"; e.g.
```
rclip -role sender -refl_ip <remote_host_ip> -loose
```

To make a remote copy: go into tmux copy mode on remote host, select some text, press the specified key, voila, the text are copied into local clipboard, secury & easy;

## CLI Parameter
```
remote clipboard, version 1.1
flag provided but not defined: -?
Usage of rclip:
  -loose
        use hard coded address for reflector certificate SAN check
  -rcv_port uint
        specify the port for receiver on reflector (default 8891)
  -refl_ip string
        reflector listening ip address
  -role string
        specify the role, rcv/refl/sender (default "rcv")
  -send_port uint
        specify the port for sender on reflector (default 8890)
```


## license
MIT; https://opensource.org/licenses/MIT
