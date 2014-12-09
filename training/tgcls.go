package main

import(
	"flag"
	"fmt"
    "./base"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "net"
)

var configPath  string
var source       int

func init() {
    const (
        defaultConfig   = "config.json"
        configUsage     = "the location of the configuration file"
        defaultSource   = -1
        sourceUsage     = "the source to classify (default: 0, all)"
    )
    flag.StringVar(&configPath, "config", defaultConfig, configUsage)
    flag.StringVar(&configPath, "c" , defaultConfig, configUsage+" (shorthand)")
    flag.IntVar(&source, "source", defaultSource, sourceUsage)
    flag.IntVar(&source, "s", defaultSource, sourceUsage +" (shorthand)")
}

func main(){
	flag.Parse()
    _ = configPath

	// Iterate through each one of the packets and assign it to flow.
    // Starting at start and ending at end.
	session, err := mgo.Dial("127.0.0.1")
    if err != nil {
        panic(err)
    }
    defer session.Close()
    // List of scans
    db := session.DB("packetgen")
    if source == -1{
        // Iterate through each source in the db. classify it's range.
        sources := db.C("sources")
        var src base.Source
        iter := sources.Find(nil).Iter()
        for iter.Next(&src){
            // Get scans in this source
            var scans = make(map[string]bool)
            for _, ip := range(src.Scans){
                scans[ip] = true
            }
            classifyRange(db, src.Start, src.End,scans)
        }
        err := iter.Err()
        if err != nil{
            panic(err)
        }
    }
}


func classifyRange(db *mgo.Database, start, end int, scans map[string]bool){
    rawPacketC := db.C("rawpackets")

    flows := make(map[string]*base.Flow, 10)
    conversations := make(map[string]*base.Conversation, 10)
    convFlows := make(map[string][]*base.Flow) // Conversation -> Flows
    flowPackets := make(map[string][]*base.Packet) // Flows -> Packets

    convC := db.C("conversations")
    flowC := db.C("flows")
    packetC := db.C("packets")

    index := mgo.Index{
        Key: []string{"flow"},
        Unique: false,
        DropDups: false,
        Background: true,
    }
    err := packetC.EnsureIndex(index)
    if err != nil{
        panic(err)
    }
    index = mgo.Index{
        Key: []string{"conversation"},
        Unique: false,
        DropDups: false,
        Background: true,
    }
    err = flowC.EnsureIndex(index)
    if err != nil{
        panic(err)
    }
    // Iterate through each packet, create flows if needed and
    flowCount, err := flowC.Count()
    if err != nil{
        panic(err)
    }
    flowStart := flowCount

    conversationCount, err := convC.Count()
    if err != nil{
        panic(err)
    }
    convStart := conversationCount

    var packet base.Packet
    iter := rawPacketC.Find(bson.M{"number": bson.M{"$gte":start, "$lt":end}}).Iter()
    for iter.Next(&packet) {
        // Process packet
        ep1, ep2 := packet.Endpoints()
        conversationKey := packet.ConversationId()
        isScan := false
        var scanIp net.IP
        if _, ok := scans[ep1.Ip.String()]; ok{
            scanIp = ep1.Ip
            isScan = true
        }
        if _, ok := scans[ep2.Ip.String()]; ok{
            scanIp = ep2.Ip
            isScan = true
        }
        flowKey := packet.FlowId()
        if _, ok := conversations[conversationKey]; !ok{
            // Need to create conversation
            newConv := &base.Conversation{
                Number: conversationCount,
                Hosts: []net.IP{ep1.Ip, ep2.Ip},
                Start: packet.Timestamp,
                Scan: isScan,
                Scanner:scanIp,
            }
            conversationCount += 1
            conversations[conversationKey] = newConv
            convFlows[conversationKey] = make([]*base.Flow, 0)
        }
        conversation := conversations[conversationKey]

        if _, ok := flows[flowKey]; !ok{
            // Flow needs to be created
            newFlow := &base.Flow{
                Number:flowCount,
                Type:ep1.Type,
                Ep1:ep1,
                Ep2:ep2,
                Packets:1,
                Start:packet.Timestamp,
                Conversation:conversation.Number,
            }
            flows[flowKey] = newFlow
            flowCount += 1
            flowPackets[flowKey] = make([]*base.Packet, 0)
            convFlows[conversationKey] = append(convFlows[conversationKey], newFlow)
        }
        flow := flows[flowKey]
        (*flow).Endpoint = packet.Timestamp
        (*conversation).Endpoint = packet.Timestamp
        (*flow).Packets += 1
        newPacket := packet

        (*conversation).TotalBytes += newPacket.CaptureLength
        (*flow).TotalBytes += newPacket.CaptureLength

        newPacket.Flow = flow.Number
        flowPackets[flowKey] = append(flowPackets[flowKey],&newPacket)
    }
    for key, conversation := range(conversations){
        conversation.Duration = conversation.Endpoint - conversation.Start
        seconds := conversation.Duration/int64(1000000000)
        if seconds == 0{
            seconds = int64(1)
        }
        throughput :=  int64((int64(conversation.TotalBytes) * int64(8))/ seconds)
        conversation.Throughput = throughput
        for _, flow := range(convFlows[key]){
            flow.Duration = flow.Endpoint - flow.Start
            seconds = flow.Duration/int64(1000000000)
            if seconds == 0{
                seconds = int64(1)
            }
            throughput = int64(int64(flow.TotalBytes * 8) / seconds)
            flow.Throughput = throughput
            for _, packet := range(flowPackets[flow.FlowId()]){
                err = packetC.Insert(packet)
                if err != nil{
                    panic(err)
                }
            }
            err = flowC.Insert(flow)
            if err != nil{
                panic(err)
            }
        }
        err := convC.Insert(conversation)
        if err != nil{
            panic(err)
        }
    }
    fmt.Printf("Indexed %d packets\n", end-start)
    fmt.Printf("Flows: %d(%dnew)\n", flowCount, flowCount- flowStart)
    fmt.Printf("Conversatiosn: %d(%d new)\n", conversationCount, conversationCount-convStart)
}