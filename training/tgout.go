package main

import(
	"flag"
	"fmt"
    "./base"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "container/heap"
    "net"
)

var configPath string

type Config struct{
    Clients     int
    Bandwith    int
    Scan        bool
    LocalIp     string
}

type Fetch struct{
    FlowNumber  int // Flow number

}

func init() {
    const (
        defaultConfig   = "config.json"
        configUsage     = "the location of the configuration file"
    )
    flag.StringVar(&configPath, "config", defaultConfig, configUsage)
    flag.StringVar(&configPath, "c" , defaultConfig, configUsage+" (shorthand)")
}


func main(){
    flag.Parse()

    localIp := net.ParseIP("192.168.1.53")

    session, err := mgo.Dial("127.0.0.1")
    if err != nil {
        panic(err)
    }

    defer session.Close()

    db := session.DB("packetgen")

    // Create packet queue.
    pq := make(base.PacketQueue, 0)
    heap.Init(&pq)


    // Just to start were going to use all packets, no filtering.

    convC := db.C("conversations")
    flowC := db.C("flows")
    packetC := db.C("packets")

    convCount:=0
    var conv base.Conversation
    Citer := convC.Find(nil).Iter()
    for Citer.Next(&conv){
        convCount += 1
        var oldLocal net.IP // host to replace with localIP
        for _, ip := range(conv.Hosts){
            if ip.Equal(localIp){
                oldLocal = localIp
            }
        }
        if oldLocal == nil{
            // Default to the first ip to overwrite
            oldLocal = conv.Hosts[0]
        }

        // Grab each flow for that iter.
        var flow base.Flow
        Fiter := flowC.Find(bson.M{"conversation":conv.Number}).Iter()
        for Fiter.Next(&flow){
            // Grab the packets.
            var packet base.Packet
            Piter := packetC.Find(bson.M{"flow":flow.Number}).Iter()
            for Piter.Next(&packet){
                // Keep track of the packet.
                newpacket := packet
                newpacket.SetIp(oldLocal, localIp)
                heap.Push(&pq, &base.Item{Value:newpacket})
            }
        }
    }

    packetsNum := pq.Len()
    // Print out the packets in order? Hopefully..
    for pq.Len() > 0 {
        item := heap.Pop(&pq).(*base.Item)
        fmt.Printf("%+v\n", item.Value)
    }
    fmt.Println(packetsNum)
}