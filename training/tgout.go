package main

import(
	"flag"
	"fmt"
    "./base"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "container/heap"
    "net"
    "github.com/lib/pq"
    "database/sql"
    "time"
    "encoding/json"
    "os"
)

var configPath string
var localIp net.IP

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

    localIp = net.ParseIP("192.168.1.99")

    session, err := mgo.Dial("127.0.0.1")
    if err != nil {
        panic(err)
    }

    defer session.Close()
    db := session.DB("packetgen")
    // Create packet queue.
    packetq := make(base.PacketQueue, 0)
    heap.Init(&packetq)

    // Just to start were going to use all packets, no filtering.
    convC := db.C("conversations")
    flowC := db.C("flows")
    packetC := db.C("packets")


    // Sql Connection
    sqldb, err := sql.Open("postgres", "user=terriergen dbname=terriergen password=avahi-daemon sslmode=disable")
    if err != nil {
        panic(err)
    }
    var startId int
    var endId int
    err = sqldb.QueryRow("SELECT count(id) from packets").Scan(&startId)
    if err != nil{
        panic(err)
    }
    endId = startId

    txn, err := sqldb.Begin()
    if err != nil {
        panic(err)
    }

    stmt, err := txn.Prepare(pq.CopyIn("packets", "id", "port","ip","ttl","time"))
    if err != nil {
        panic(err)
    }

    // Json file

    scans := make(map[string]bool, 0)

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

        // Get the ip that is not going to be overwritten
        var remoteIp net.IP
        for _, ip := range(conv.Hosts){
            if !ip.Equal(oldLocal){
                remoteIp = ip
            }
        }
        scans[remoteIp.String()] = conv.Scan

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
                heap.Push(&packetq, &base.Item{Value:newpacket})
            }
        }
    }

    testCases := make([]Testcase, 0)

    // Window size
    time := int64(30000000000)
    pkts := 1000

    // Start && end of sliding windows in nanoseconds
    window := NewWindow(time, pkts)
    // Print out the packets in order? Hopefully..
    for packetq.Len() > 0 {
        item := heap.Pop(&packetq).(*base.Item)
        window.Add(&item.Value)
        if window.Full(){
            testCases = append(testCases, handlePackets(window.Flush(), scans,stmt, startId, &endId)...)
            _, err = stmt.Exec()
            if err != nil {
                panic(err)
            }

            err = stmt.Close()
            if err != nil {
                panic(err)
            }

            stmt, err = txn.Prepare(pq.CopyIn("packets", "id", "port","ip","ttl","time"))
            if err != nil {
                panic(err)
            }
        }
    }
    testCases = append(testCases, handlePackets(window.Flush(), scans, stmt, startId, &endId)...)

    fmt.Println("Flusing db")
    _, err = stmt.Exec()
    if err != nil {
        panic(err)
    }

    err = stmt.Close()
    if err != nil {
        panic(err)
    }

    err = txn.Commit()
    if err != nil {
        panic(err)
    }

    fileName := "out.json"
    // Save testCases to json file
    fmt.Println("Saving to json")
    b, err := json.Marshal(testCases)
    if err != nil{
        panic(err)
    }
    fd, err := os.Create(fileName) 
    if err != nil{
        panic(err)
    }
    _, err = fd.Write(b)
    if err != nil{
        panic(err)
    }

}

func handlePackets(pkts []*base.Packet, scans map[string]bool, stmt *sql.Stmt, startId int, endId *int)[]Testcase{
    ips := make(map[string]net.IP, 0)

    for _, pkt := range(pkts){
        fmt.Printf("%+v\n", pkt)
        var rem base.Endpoint
        var loc base.Endpoint
        // Check if the ip exists and set rem/loc
        rem, loc = (*pkt).Endpoints()
        if loc.Ip.Equal(localIp){
            if _, ok := ips[rem.Ip.String()]; !ok {
                ips[rem.Ip.String()] = rem.Ip
            }
            // Write packet to mysql
            baseTime := time.Time{}
            duration, err := time.ParseDuration(fmt.Sprintf("%dns",pkt.Timestamp))
            if err != nil{
                panic(err)
            }
            (*endId)+=1
            _, err = stmt.Exec(endId,loc.Port, rem.Ip.String(),
                0, baseTime.Add(duration))
            if err != nil{
                panic(err)
            }
        }
    }
    tcs := make([]Testcase, 0)
    for _, ip := range(ips){
        tcs = append(tcs,
            Testcase{
                Start:startId+1,
                End:*endId,
                Scan:scans[ip.String()],
                Ip:ip.String(),
            })
    }
    return tcs
}

func NewWindow(delta int64, count int) Window{
    return Window{
        start:int64(0),
        end:delta,
        now:int64(0),
        delta: delta,
        window: make([]*base.Packet, count),
        pkts:count,
        index:0,
    }
}

type Testcase struct{
    Start       int `json:"Start"`
    End         int `json:"End"`
    Scan        bool `json:"Scan"`
    Ip          string `json:"Ip"`
}

type Window struct{
    start, end, now, delta  int64
    window                  []*base.Packet
    index, pkts             int // Next open spot
}

func (w *Window) Add(p *base.Packet){
    w.window[w.index] = p
    w.index += 1
    w.now = (*p).Timestamp
}

func (w *Window) Full() bool{
    fmt.Printf("Full %d %d %d %d\n",w.now, w.end, w.index, w.pkts)
    if w.now >= w.end{
        return true
    }
    if w.index >= w.pkts{
        return true
    }
    return false
}

func (w *Window) Flush()[]*base.Packet{
    fmt.Printf("0:%d\n", w.index)
    pks := w.window[0:w.index-1]
    w.start = w.now
    w.end = w.start + w.delta
    w.index = 0
    return pks
}