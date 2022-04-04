package bastion // import "moul.io/sshportal/pkg/bastion"

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gliderlabs/ssh"
	oi "github.com/reiver/go-oi"
	telnet "github.com/reiver/go-telnet"
	"moul.io/sshportal/pkg/dbmodels"
	expect "moul.io/sshportal/pkg/expect"
)

type bastionTelnetCaller struct {
	ssh ssh.Session
	loginCmd string
}

func (caller bastionTelnetCaller) CallTELNET(ctx telnet.Context, w telnet.Writer, r telnet.Reader) {
	var last = make(chan byte, 1024)
	var closed = make(chan bool, 1)
	
	go func(writer io.Writer, reader io.Reader, last chan byte, closed chan bool) {
		var buffer [1]byte // Seems like the length of the buffer needs to be small, otherwise will have to wait for buffer to fill up.
		p := buffer[:]
		channelClosed := false

		for {
			// Read 1 byte.
			n, err := reader.Read(p)
			if n <= 0 && err == nil {
				continue
			} else if n <= 0 && err != nil {
				break
			}
			if _, err = oi.LongWrite(writer, p); err != nil {
				log.Printf("telnet longwrite failed: %v", err)
			}
			if !channelClosed {
				select {
				case closed := <-closed:
					channelClosed = closed
					close(last)
					continue
				default:
				}
				last<-p[0]
			}
		}
	}(caller.ssh, r, last, closed)

	var buffer bytes.Buffer
	var p []byte

	var crlfBuffer = [2]byte{'\r', '\n'}
	crlf := crlfBuffer[:]
	

	scanner := bufio.NewScanner(caller.ssh)
	scanner.Split(scannerSplitFunc)

	exp, err := expect.NewExpectModule(caller.loginCmd)
	if caller.loginCmd == "" {
		log.Println("No authentication script provided")
	} else if err != nil {
		log.Printf("WARN: skipping expect module because of error '%v'", err)
	} else {
		maxSteps := 50
		for i:=0;i<maxSteps;i++ {
			lastMsg := []byte{}
			done := false
			for {
				select {
					case readByte := <-last:
						lastMsg = append(lastMsg, readByte)
					default:
						done = true
				}
				if done {
					break
				}
			}
			if nextCmd, send := exp.Next(string(lastMsg)); send {
				buffer.Write([]byte(nextCmd))
				buffer.Write(crlf)
			}
			p = buffer.Bytes()
			if len(p) > 0 {
				n, err := oi.LongWrite(w, p)
				if nil != err {
					// TODO: handle errors
					return
				}
				if expected, actual := int64(len(p)), n; expected != actual {
					err := fmt.Errorf("transmission problem: tried sending %d bytes, but actually only sent %d bytes", expected, actual)
					fmt.Fprint(caller.ssh, err.Error())
					return
				}
				buffer.Reset()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
		
	closed<-true

	log.Println("reached interactive shell")
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
		buffer.Write(crlf)
		p = buffer.Bytes()
		n, err := oi.LongWrite(w, p)
		if nil != err {
			break
		}
		if expected, actual := int64(len(p)), n; expected != actual {
			err := fmt.Errorf("transmission problem: tried sending %d bytes, but actually only sent %d bytes", expected, actual)
			fmt.Fprint(caller.ssh, err.Error())
			return
		}
		buffer.Reset()
	}

	// Wait a bit to receive data from the server (that we would send to io.Stdout).
	time.Sleep(3 * time.Millisecond)
}

func scannerSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		return 0, nil, nil
	}
	return bufio.ScanLines(data, atEOF)
}

func telnetHandler(host *dbmodels.Host) ssh.Handler {
	return func(s ssh.Session) {
		// FIXME: log session in db
		// actx := s.Context().Value(authContextKey).(*authContext)
		caller := bastionTelnetCaller{ssh: s, loginCmd: host.Passwd() }
		if err := telnet.DialToAndCall(host.DialAddr(), caller); err != nil {
			fmt.Fprintf(s, "error: %v", err)
		}
	}
}
