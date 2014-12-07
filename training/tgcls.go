package main

import(
	"flag"
	"fmt"
    "./base"
    "gopkg.in/mgo.v2"
    _ "gopkg.in/mgo.v2/bson"
    "net"
)

var configPath string

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
    _ = configPath
	// Iterate through each one of the packets and assign it to flow.

	session, err := mgo.Dial("127.0.0.1")
    if err != nil {
        panic(err)
    }
 
    defer session.Close()

    db := session.DB("packetgen")
    rawPacketC := db.C("rawpackets")

    flows := make(map[string]*base.Flow, 10)
    conversations := make(map[string]*base.Conversation, 10)
    convFlows := make(map[string][]*base.Flow) // Conversation -> Flows
    flowPackets := make(map[string][]*base.Packet) // Flows -> Packets

    // Iterate through each packet, create flows if needed and
    flowCount := 0
    conversationCount := 0
    var packet base.Packet
    iter := rawPacketC.Find(nil).Iter()
    for iter.Next(&packet) {
        // Process packet
        ep1, ep2 := packet.Endpoints()
        conversationKey := packet.ConversationId()
        flowKey := packet.FlowId()
        if _, ok := conversations[conversationKey]; !ok{
            // Need to create conversation
            newConv := &base.Conversation{
                Number: conversationCount,
                Hosts: []net.IP{ep1.Ip, ep2.Ip},
                Start: packet.Timestamp,
                Scan:false,
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

    convC := db.C("conversations")
    flowC := db.C("flows")
    packetC := db.C("packets")

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

    // Find the duration for the flows and conversations
    fmt.Printf("Flows: %d\n", flowCount)
    fmt.Printf("Conversatiosn: %d\n", conversationCount)
}