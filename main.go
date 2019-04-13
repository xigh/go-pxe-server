package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
	"unicode"

	"github.com/pin/tftp"
)

var (
	logp       = flag.Bool("log", false, "show log information")
	debug      = flag.Bool("debug", false, "show debug information")
	pxeClassID = "PXEClient"
)

// readHandler is called when client starts file download from server
func readHandler(filename string, rf io.ReaderFrom) error {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	st, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	rf.(tftp.OutgoingTransfer).SetSize(st.Size())
	n, err := rf.ReadFrom(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	fmt.Printf("%s: %d bytes sent\n", filename, n)
	return nil
}

// writeHandler is called when client starts file upload to server
func writeHandler(filename string, wt io.WriterTo) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	fmt.Printf("%s: %d bytes received\n", filename, n)
	return nil
}

func pxeServe() {
	// use nil in place of handler to disable read or write operations
	s := tftp.NewServer(readHandler, writeHandler)
	s.SetTimeout(5 * time.Second)  // optional
	err := s.ListenAndServe(":69") // blocks until s.Shutdown() is called
	if err != nil {
		fmt.Fprintf(os.Stdout, "server: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	go pxeServe()

	// listen to incoming udp packets
	pc, err := net.ListenPacket("udp4", ":67")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	for {
		buf := make([]byte, 2000)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}
		go serve(pc, addr, buf[:n])
	}

}

func dumpBytes(m []byte) {
	for i := 0; i < len(m); i++ {
		if i%16 == 0 {
			if i > 0 {
				fmt.Printf("   ")
				for j := i - 16; j < i; j++ {
					c := rune(m[j])
					if !unicode.IsPrint(c) {
						c = ' '
					}
					fmt.Printf("%c ", c)
				}
				fmt.Printf("\n")
			}
			fmt.Printf("%6d: ", i)
		}
		fmt.Printf("%02x ", m[i])
	}
	fmt.Printf("\n")
}

func serve(pc net.PacketConn, addr net.Addr, buf []byte) {
	if *logp {
		fmt.Printf("got %d bytes from %v\n", len(buf), addr)
	}

	if *debug {
		dumpBytes(buf)
	}

	// check source => 67

	// Is bootp request ?

	if len(buf) < 1 {
		if *logp {
			fmt.Printf("buffer too small\n")
		}
		return
	}

	if buf[0] != 1 {
		if *logp {
			fmt.Printf("not a bootp request\n")
		}
		return
	}

	// Parse bootp request

	if len(buf) < 236 {
		if *logp {
			fmt.Printf("buffer too small\n")
		}
		return
	}

	if *logp {
		fmt.Printf("BOOTREQUEST\n")
		fmt.Printf(" - htype   = %d\n", buf[1])
		fmt.Printf(" - hlen    = %d\n", buf[2])
		fmt.Printf(" - hops    = %d\n", buf[3])
		fmt.Printf(" - xid     = %02x\n", buf[4:8])
		fmt.Printf(" - secs    = %02x\n", buf[8:9])
		fmt.Printf(" - unused  = %02x\n", buf[10:12])
		fmt.Printf(" - ciaddr  = %02x\n", buf[12:16])
		fmt.Printf(" - yiaddr  = %02x\n", buf[16:20])
		fmt.Printf(" - siaddr  = %02x\n", buf[20:24])
		fmt.Printf(" - giaddr  = %02x\n", buf[24:28])
		fmt.Printf(" - chaddr  = %02x\n", buf[28:44])
		fmt.Printf(" - sname   = %s\n", cstring(buf[44:108]))
		fmt.Printf(" - file    = %s\n", cstring(buf[108:236]))
	}

	if len(buf) < 240 {
		if *logp {
			fmt.Printf("no more data\n")
		}
		return
	}

	// has rfc1048 extension ?

	if buf[236] != 0x63 || buf[237] != 0x82 || buf[238] != 0x53 || buf[239] != 0x63 {
		if *logp {
			fmt.Printf("unknown extension %02x\n", buf[236:240])
		}
		return
	}

	// Parse rfc1048 extension

	if *logp {
		fmt.Printf("RFC1048 extension\n")
	}

	msgIsDhcpDiscover := false
	msgIsDhcpRequest := true
	msgIsPxe := false

	i := 240
	for i < len(buf) {
		n := buf[i]
		if n == 0xff {
			break
		}
		if n == 0 {
			i++
			continue
		}

		size := buf[i+1]
		switch n {
		case 53:
			if *logp {
				name := "DHCP message type"
				fmt.Printf(" - ext:%03d = %-30s (%d) [%d bytes]\n", i, name, n, size)
			}
			switch buf[i+2] {
			case 1:
				msgIsDhcpDiscover = true
			case 3:
				msgIsDhcpRequest = true
			}

		case 60:
			vcid := string(buf[i+2 : i+2+int(size)])
			if *logp {
				name := "Vendor Class ID"
				fmt.Printf(" - ext:%03d = %-30s (%d) [%d bytes]\n", i, name, n, size)
				fmt.Printf("\t%s\n", vcid)
			}
			if vcid[:8] != pxeClassID {
				msgIsPxe = true
			}
		}

		i += int(size) + 2
	}

	if !msgIsDhcpDiscover {
		if !msgIsDhcpRequest {
			// fmt.Printf("not a dhcp request\n")
			return
		}
		if *logp {
			fmt.Printf("not a dhcp discover\n")
		}
		return
	}

	if !msgIsPxe {
		if *logp {
			fmt.Printf("not a pxe request\n")
		}
		return
	}

	// reply

	nbuf := make([]byte, 1500)

	nbuf[0] = 2 // reply

	nbuf[1] = buf[1] // copy hardware addr type
	nbuf[2] = buf[2] // copy hardware addr length
	nbuf[3] = buf[3] // copy hops

	// copy xid

	copy(nbuf[4:8], buf[4:8])

	// set your ip

	// nbuf[16] = 192
	// nbuf[17] = 168
	// nbuf[18] = 1
	// nbuf[19] = 37

	// copy client hardware addr

	copy(nbuf[28:44], buf[28:44])

	// set boot filename

	copy(nbuf[108:236], "boot.bin\x00")

	// set tftp server ip addr

	nbuf[20] = 192
	nbuf[21] = 168
	nbuf[22] = 1
	nbuf[23] = 49

	i = 236
	copy(nbuf[i:i+4], buf[i:i+4])
	i += 4

	// set Message Type

	nbuf[i+0] = 53
	nbuf[i+1] = 1
	nbuf[i+2] = 2 // DHCP offer
	i += 3

	// set Server Addr

	nbuf[i+0] = 54
	nbuf[i+1] = 4
	nbuf[i+2] = 192
	nbuf[i+3] = 168
	nbuf[i+4] = 1
	nbuf[i+5] = 49
	i += 6

	// set Server Identifier

	l := len(pxeClassID)
	nbuf[i+0] = 60
	nbuf[i+1] = byte(l)
	copy(nbuf[i+2:i+l+2], pxeClassID)
	i += 2 + l

	// Add PXE vendor stuff

	nbuf[i+0] = 43
	nbuf[i+1] = 3
	nbuf[i+2] = 6
	nbuf[i+3] = 1
	nbuf[i+4] = 8
	i += 5

	// done

	nbuf[i] = 255

	// sending request

	if *debug {
		fmt.Printf("sending %d bytes\n", i+1)
		dumpBytes(nbuf[:i+1])
	}

	bcastAddr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:68")

	n, err := pc.WriteTo(nbuf[:i+1], bcastAddr)
	if err != nil {
		if *logp {
			fmt.Printf(" - could not send byte: %v\n", err)
		}
		return
	}

	if *logp {
		fmt.Printf(" - sent %d bytes\n", n)
	}
}

func cstring(buf []byte) string {
	ni := bytes.IndexByte(buf, 0)
	if ni < 0 {
		return "invalid string"
	}
	return string(buf[:ni])
}
