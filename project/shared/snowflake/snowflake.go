package snowflake

import (
	"sync"
	"time"
)

const (
	// 起始时间戳 (2024-01-01 00:00:00 UTC)
	epoch int64 = 1704067200000

	// 位数分配
	nodeBits     = 10
	sequenceBits = 12

	// 最大值
	maxNodeID   = -1 ^ (-1 << nodeBits)
	maxSequence = -1 ^ (-1 << sequenceBits)

	// 位移
	nodeShift      = sequenceBits
	timestampShift = nodeBits + sequenceBits
)

// ID 雪花ID
type ID int64

// String 转换为字符串
func (id ID) String() string {
	return Int64ToString(int64(id))
}

// Int64 转换为 int64
func (id ID) Int64() int64 {
	return int64(id)
}

// Node 雪花ID生成器节点
type Node struct {
	mu        sync.Mutex
	nodeID    int64
	sequence  int64
	lastTime  int64
}

// NewNode 创建雪花ID生成器
func NewNode(nodeID int64) (*Node, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		nodeID = 1
	}
	return &Node{
		nodeID:   nodeID,
		sequence: 0,
	}, nil
}

// Generate 生成雪花ID
func (n *Node) Generate() ID {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == n.lastTime {
		n.sequence = (n.sequence + 1) & maxSequence
		if n.sequence == 0 {
			// 序号用尽，等待下一毫秒
			for now <= n.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.sequence = 0
	}

	n.lastTime = now

	id := ((now - epoch) << timestampShift) |
		(n.nodeID << nodeShift) |
		n.sequence

	return ID(id)
}

// Int64ToString 将 int64 转换为字符串
func Int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte(n%10) + '0'
		n /= 10
	}

	if negative {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}
