// SPDX-License-Identifier: MIT
package mempool

import (
	"errors"
	"log"
	"sync"
)

var (
	ErrTooLargeAlloc = errors.New("too large allocation request")
	ErrInvalidAddr   = errors.New("invalid address")
	ErrOverrunAccess = errors.New("overrun access")
)

const safeIntBits = 53
const blockLowBits = 28 // configurable
const blockHighBits = safeIntBits - blockLowBits

const MemBlockSlot = (1 << blockHighBits) - 1 // # of blocks (id=0 is reserved for NULL)
const MemBlockLimit = (1 << blockLowBits) - 1 // limit of blocks[bytes]

type MemBlock []byte

type MemPool struct {
	mu     sync.Mutex
	blocks map[uint32]*MemBlock
	nextID uint32
}

// id -> address
func id2addr(id uint32) uint64 {
	return uint64(id) << blockLowBits
}

// address -> (id, offset)
func addr2id(addr uint64) (uint32, int) {
	id := uint32((addr >> blockLowBits) & ((1 << blockHighBits) - 1))
	offset := int(addr & ((1 << blockLowBits) - 1))
	return id, offset
}

func NewMemPool() *MemPool {
	log.Printf("MemPool: slot=%d limit=%d", MemBlockSlot, MemBlockLimit)
	return &MemPool{
		blocks: make(map[uint32]*MemBlock),
		nextID: 1,
	}
}

func (p *MemPool) Alloc(size uint64) (uint64, error) {
	if size > MemBlockLimit {
		return 0, ErrTooLargeAlloc
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.nextID >= MemBlockSlot {
		panic("Out of memory (MemBlock full)")
	}
	block := make(MemBlock, size)
	id := p.nextID
	p.blocks[id] = &block
	p.nextID++
	return id2addr(id), nil
}

func (p *MemPool) Free(addr uint64) error {
	if addr == 0 {
		// ISO C shall accept `free(0)`
		return nil
	}
	id, _ := addr2id(addr)

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.blocks[id]; !exists {
		return ErrInvalidAddr
	}
	delete(p.blocks, id)
	return nil
}

func (p *MemPool) Access(addr uint64, pval *byte) (byte, error) {
	id, offset := addr2id(addr)
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.blocks[id]; !exists {
		return 0, ErrInvalidAddr
	}
	block := p.blocks[id]
	if offset >= len(*block) {
		return 0, ErrOverrunAccess
	}
	// write byte (optional)
	if pval != nil {
		log.Printf("MemPool: W[%d:%d] 0x%x", id, offset, *pval)
		(*block)[offset] = *pval
	}
	// read byte
	value := (*block)[offset]
	log.Printf("MemPool: R[%d:%d] 0x%x", id, offset, value)
	return value, nil
}
