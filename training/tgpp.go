package main

import(
    "flag"
    "fmt"
    "code.google.com/p/gopacket"
    "code.google.com/p/gopacket/pcap"
    "code.google.com/p/gopacket/layers"
    "encoding/json"
    "os"
    "gopkg.in/mgo.v2"
    "./base"
)


var configPath string
var start int64 = -1
var config Config

func init() {
    const (
        defaultConfig   = "pp-config.json"
        configUsage     = "the location of the configuration file"
    )
    flag.StringVar(&configPath, "config", defaultConfig, configUsage)
    flag.StringVar(&configPath, "c" , defaultConfig, configUsage+" (shorthand)")
}

type DBConfig struct {
    Database    string  `json:"database"`
    Collection  string  `json:"collection"`
}

type Config struct{
    DB      DBConfig    `json:"db"`
    Raw     string      `json:"pcap"`
    Scans   []string    `json:"scans"`
}


//Read config file and return the configuration.                                                                    
func bootstrap(configPath string, config *Config)error{
        fd, err := os.Open(configPath)
        if err != nil{return err}
        decoder := json.NewDecoder(fd)
        err = decoder.Decode(config)
        if err != nil{return err}
        fmt.Printf("%+v\n",*config)
        return nil
}

// Read from a pcap file and transfer the information into a rawpacket table in psql.
func main(){
    flag.Parse()

    if err := bootstrap(configPath, &config); err != nil{
        panic(err)
    }

    session, err := mgo.Dial("127.0.0.1")
    if err != nil {
        panic(err)
    }
 
    defer session.Close()

    packetOut := make(chan base.Packet, 10)

    if handle, err := pcap.OpenOffline(config.Raw); err != nil{
        panic(err)
    } else{
        packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
        go processPackets(packetSource.Packets(), packetOut)
    }
    db := session.DB("packetgen")
    exportPackets(packetOut, db)
}

func processPackets(packetSource chan gopacket.Packet, PacketOut chan base.Packet){
    for packet := range packetSource {
        handlePacket(packet, PacketOut)  // Do something with a packet here.
    }
    close(PacketOut)
    fmt.Println("Finished read packets source")
}

func handlePacket(packet gopacket.Packet, out chan base.Packet){
    reltimestamp := packet.Metadata().CaptureInfo.Timestamp
    if start == -1{
        start = reltimestamp.UnixNano()
    }
    timestamp := reltimestamp.UnixNano()
    timestamp = timestamp - start
    length := packet.Metadata().CaptureInfo.CaptureLength
    pkt := base.Packet{Timestamp:timestamp, Source:"Test", CaptureLength:length}
    if net := packet.Layer(layers.LayerTypeIPv4); net != nil{
        ipv4 := net.(*layers.IPv4)
        pkt.IPv4Header = base.IPv4Header{
            Version:ipv4.Version,
            IHL:ipv4.IHL,
            TOS:ipv4.TOS,
            Length: ipv4.Length,
            Id: ipv4.Id,
            Flags: uint8(ipv4.Flags),
            FragOffset: ipv4.FragOffset,
            TTL: ipv4.TTL,
            Protocol: uint8(ipv4.Protocol),
            Checksum: ipv4.Checksum,
            SrcIP: ipv4.SrcIP,
            DstIP: ipv4.DstIP,
        }
    } else if net := packet.Layer(layers.LayerTypeIPv6); net != nil{
        ipv6 := net.(*layers.IPv6)
        pkt.IPv6Header = base.IPv6Header{
            Version:ipv6.Version,
            TrafficClass: ipv6.TrafficClass,
            FlowLabel: ipv6.FlowLabel,
            Length:ipv6.Length,
            NextHeader:uint8(ipv6.NextHeader),
            HopLimit:ipv6.HopLimit,
            SrcIP:ipv6.SrcIP,
            DstIP:ipv6.DstIP,
        }
    }

    if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil{
        tcp, _ := tcpLayer.(*layers.TCP)
        pkt.TCPHeader = base.TCPHeader{
            SrcPort:uint16(tcp.SrcPort),
            DstPort:uint16(tcp.DstPort),
            Seq:tcp.Seq,
            Ack:tcp.Ack,
            DataOffset: tcp.DataOffset,
            FIN: tcp.FIN,
            SYN: tcp.SYN,
            RST: tcp.RST,
            PSH: tcp.PSH,
            ACKF: tcp.ACK,
            URG: tcp.URG,
            ECE: tcp.ECE,
            CWR: tcp.CWR,
            NS: tcp.NS,
            Window: tcp.Window,
            Checksum: tcp.Checksum,
            Urgent: tcp.Urgent,
        }
    }else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil{
        udp, _ := udpLayer.(*layers.UDP)
        pkt.UDPHeader = base.UDPHeader{
            SrcPort: uint16(udp.SrcPort),
            DstPort: uint16(udp.DstPort),
            Length: udp.Length,
            Checksum: udp.Checksum,
        }
    }

    if ((pkt.TCPHeader.SrcPort != 0 || pkt.UDPHeader.SrcPort != 0) &&
        (pkt.IPv4Header.Version != 0 || pkt.IPv6Header.Version != 0)){
        out <- pkt
    }
}

// Fan in packets.
func exportPackets(packetSource chan base.Packet, db *mgo.Database){
    // Collection pacekts
    c := db.C("rawpackets")

    index := mgo.Index{
        Key: []string{"number"},
        Unique: true,
        DropDups: true,
        Background: true,
        Sparse: true,
    }

    start, err := c.Count()
    if err != nil{
        panic(err)
    }
    number := start

    err = c.EnsureIndex(index)
    if err != nil{
        panic(err)
    }

    bulk := c.Bulk()


    number += 1

    fmt.Println("Started to read packets")
    for packet := range packetSource{
        packet.Number = number
        bulk.Insert(packet)

        if(number % 10000) == 0{
            _, err := bulk.Run()
            if err != nil{
                panic(err)
            }
            bulk = c.Bulk()
        }
        number += 1
    }
    _, err = bulk.Run()
    if err != nil{
        panic(err)
    }

    s := db.C("sources")
    snumber, err := s.Count()
    if err != nil{
        panic(err)
    }
    source := base.Source{Number: snumber, Start:start, End:number-1,Scans:config.Scans}
    err = s.Insert(source)
    if err != nil{
        panic(err)
    }
    fmt.Println("Finished writing data.")
    fmt.Printf("Inserted packets from %d to %d\n",start, number-1)
}
