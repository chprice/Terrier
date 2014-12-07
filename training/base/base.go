package base

import(
    "net"
    "fmt"
)

const (
    TCPFlowType = 1
    UDPFlowType = 2
)

type Packet struct{
    Number          int // UID
    Timestamp       int64 // Relative to flow start
    CaptureLength   int // Number of bytes
    Source          string
    IPv4Header      IPv4Header
    IPv6Header      IPv6Header
    TCPHeader       TCPHeader
    UDPHeader       UDPHeader
    Flow            int
}

func (p Packet) Endpoints() (Endpoint, Endpoint){
    var ip1, ip2 net.IP
    var p1, p2 uint16

    if p.IPv4Header.SrcIP != nil{
        ip1 = p.IPv4Header.SrcIP
        ip2 = p.IPv4Header.DstIP
    }else{
        ip1 = p.IPv6Header.SrcIP
        ip2 = p.IPv6Header.DstIP
    }
    flowType := 0
    if p.UDPHeader.SrcPort != 0{
        flowType = UDPFlowType
        p1 = p.UDPHeader.SrcPort
        p2 = p.UDPHeader.DstPort
    }else{
        flowType = TCPFlowType
        p1 = p.TCPHeader.SrcPort
        p2 = p.TCPHeader.DstPort
    }
    return Endpoint{Type:flowType,Ip:ip1, Port:p1}, Endpoint{Type:flowType,Ip:ip2, Port:p2}
}

// Sets the endpoint given to the new endpoint
func (p *Packet) SetIp(ip, newip net.IP){
    if p.IPv4Header.SrcIP != nil{
        if p.IPv4Header.SrcIP.Equal(ip){
            p.IPv4Header.SrcIP = newip
        }else{
            p.IPv4Header.DstIP = newip
        }
    }else{
        if p.IPv6Header.SrcIP.Equal(ip){
            p.IPv6Header.SrcIP = newip
        }else{
            p.IPv6Header.DstIP = newip
        }
    }
}

func (p Packet) FlowId()string{
    return FlowId(p.Endpoints())
}

func (p Packet) ConversationId()string{
    return ConversationId(p.Endpoints())
}

func ConversationId(ep1, ep2 Endpoint) string{
    var first, second string
    if ep1.Ip.String() < ep2.Ip.String(){
        first = ep1.Ip.String()
        second = ep2.Ip.String()
    }else{
        first = ep2.Ip.String()
        second = ep1.Ip.String()
    }
    return fmt.Sprintf("%s:%s",first, second)
}

// Return a string which represents a flow id key
func FlowId(ep1, ep2 Endpoint) string{
    var first, second string
    if ep1.Id() < ep2.Id(){
        first = ep1.Id()
        second = ep2.Id()
    }else{
        first = ep2.Id()
        second = ep1.Id()
    }
    return fmt.Sprintf("%s:%s",first, second)
}

type Endpoint struct{
    Type int // TCP or UDP
    Ip net.IP
    Port uint16
}

func (e Endpoint) Id() string{
    return fmt.Sprintf("%d:%v:%d",e.Type,e.Ip,e.Port)
}

type Conversation struct {
    Number      int // UID
    Hosts       []net.IP
    Start       int64
    Endpoint    int64
    Duration    int64
    TotalBytes  int
    Throughput  int64
    Scan        bool
}

type Flow struct {
    Number              int // UID
    Type                int // 1 == tcp, 2 = udp
    Ep1, Ep2            Endpoint
    Packets             int // The number of packets in this flow
    Throughput          int64 // The throughput of the flow. bits/time
    Start               int64 // Start time relative to the Converstaion
    Endpoint            int64
    Duration            int64
    Conversation        int
    TotalBytes          int 
}

func (f Flow) FlowId()string{
    return FlowId(f.Ep1, f.Ep2)
}

type IPv4Header struct{
    Version    uint8
    IHL        uint8
    TOS        uint8
    Length     uint16
    Id         uint16
    Flags      uint8
    FragOffset uint16
    TTL        uint8
    Protocol   uint8
    Checksum   uint16
    SrcIP      net.IP
    DstIP      net.IP
}

type IPv6Header struct{
    Version      uint8
    TrafficClass uint8
    FlowLabel    uint32
    Length       uint16
    NextHeader   uint8
    HopLimit     uint8
    SrcIP        net.IP
    DstIP        net.IP
}

type TCPHeader struct {
    SrcPort, DstPort                            uint16
    Seq                                         uint32
    Ack                                         uint32 `bson:"Ack"`
    DataOffset                                  uint8
    FIN, SYN, RST, PSH, ACKF, URG, ECE, CWR, NS bool
    Window                                      uint16
    Checksum                                    uint16
    Urgent                                      uint16
}

type UDPHeader struct {
    SrcPort, DstPort    uint16
    Length              uint16
    Checksum            uint16
}

