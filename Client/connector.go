package Client

import (
	. "../Debug"
	"../Pack"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

type Connector struct {
	connectorFunc
}

func (connector *Connector) Init(f connectorFunc) {
	connector.connectorFunc = f
}

func (connector *Connector) Handle(conn net.Conn) (err error) {
	DebugLogger.Println(conn.RemoteAddr(), "trying to connect")
	fmt.Println(conn.RemoteAddr(), "trying to connect")
	connector.init(conn)
	connector.extraInit()
	defer connector.eliminate()
	go connector.connectionHeartBeats()
	err = connector.checkAccess()
	if err != nil {
		return err
	}
	connector.preAction()
	defer connector.postAction()
	for connector.stats() {
		connector.loop()
	}
	return err
}

type connectorFunc interface {
	checkAccess() error
	connectionHeartBeats()
	eliminate()
	extraInit()
	init(net.Conn)
	loop()
	postAction()
	preAction()
	stats() bool
}

type connector struct {
	connectorFunc
	addr       string
	conn       net.Conn
	readWriter *bufio.ReadWriter
	refresh    chan string
	stat       bool
}

func (connector *connector) checkAccess() error { return io.EOF }
func (connector *connector) extraInit()         {}
func (connector *connector) loop()              {}
func (connector *connector) postAction()        {}
func (connector *connector) preAction()         {}
func (connector *connector) stats() bool        { return connector.stat }

func (connector *connector) clearReadBuffer() error {
	var n = connector.readWriter.Reader.Buffered()
	var _, err = connector.readWriter.Discard(n)
	return err
}

func (connector *connector) connectionHeartBeats() {
	for {
		select {
		case <-connector.refresh:
			if !connector.stat {
				return
			}
		case <-time.After(time.Minute):
			connector.stat = false
			break
		}
	}
}

func (connector *connector) eliminate() {
	err := connector.clearReadBuffer()
	if err != nil {
		DebugLogger.Println(err)
	}
	err = connector.readWriter.Flush()
	if err != nil {
		DebugLogger.Println(err)
	}
	err = connector.conn.Close()
	if err != nil {
		DebugLogger.Println(err)
	}
}

func (connector *connector) init(conn net.Conn) {
	connector.conn = conn
	connector.addr = connector.conn.RemoteAddr().String()
	connector.readWriter = bufio.NewReadWriter(bufio.NewReader(connector.conn), bufio.NewWriter(connector.conn))
	connector.stat = true
	connector.refresh = make(chan string, 1)
}

func (connector *connector) refreshLink(stream Pack.Stream) {
	var statMap = make(map[string]string)
	err := json.Unmarshal([]byte(stream), &statMap)
	if err != nil {
		DebugLogger.Println(err)
		return
	}
	if stat, ok := statMap["stat"]; ok {
		switch stat {
		case "open":
			connector.refresh <- ""
		case "close":
			connector.stat = false
			DebugLogger.Println(connector.addr, "close received")
		default:
		}
	}
}

func (connector *connector) testReceiver(stream Pack.Stream) {
	DebugLogger.Println(stream)
	fmt.Println(stream)
	connector.refresh <- ""
}

func (connector *connector) writeRepeat(packet Pack.Packet, t time.Duration) (err error) {
	var ch = make(chan string, 1)
	var stat = true
	go func() {
		var count int
		for count < 3 && stat {
			_ = connector.conn.SetWriteDeadline(time.Now().Add(t))
			_, err = connector.readWriter.WriteString(string(packet))
			if err != nil {
				DebugLogger.Println(err)
			}
			err = connector.readWriter.Flush()
			if err != nil && err != io.EOF {
				DebugLogger.Println(err)
				count++
			} else {
				break
			}
		}
		ch <- ""
		_ = connector.conn.SetWriteDeadline(time.Time{})
	}()
	select {
	case <-ch:
		connector.refresh <- ""
		return nil
	case <-time.After(t):
		_ = connector.conn.SetWriteDeadline(time.Time{})
		return io.EOF
	}
}
