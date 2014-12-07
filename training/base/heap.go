// This is adapted from the PriorityQueue Example from
// http://golang.org/pkg/container/heap/#example__intHeap to store packets
package base

type Item struct{
	Value Packet
	index int
}

type PacketQueue []*Item

func (pq PacketQueue) Len()int{return len(pq)}

func (pq PacketQueue) Less(i, j int) bool{
	return pq[i].Value.Timestamp < pq[j].Value.Timestamp
}

func (pq PacketQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PacketQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PacketQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}