// SPDX-License-Identifier: MIT
package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yohhoy/malloc-server/mempool"
)

var pool *mempool.MemPool

type MallocRequest struct {
	Size uint64 `json:"size"`
}

type MallocResponse struct {
	Address uint64 `json:"addr"`
	Size    uint64 `json:"size"`
}

type FreeRequest struct {
	Address uint64 `json:"addr"`
}

type MemWriteRequest struct {
	Value byte `json:"val"`
}

type MemReadResponse struct {
	Value byte `json:"val"`
}

// Status Code: Undefined Behavior
const HttpStatusUB int = http.StatusTeapot

// void* malloc(size_t size)
func handleMalloc(c *gin.Context) {
	var req MallocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"addr":  0,
			"error": "Invalid malloc parameter"})
		return
	}
	if req.Size == 0 {
		// SEI CERT C Coding Standard, Recommendations, MEM04-C
		// https://wiki.sei.cmu.edu/confluence/display/c/MEM04-C.+Beware+of+zero-length+allocations
		c.JSON(http.StatusBadRequest, gin.H{
			"addr":  0,
			"error": "Violation of MEM04-C"})
		return
	}
	addr, err := pool.Alloc(req.Size)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"addr": addr, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, MallocResponse{
		Address: addr,
		Size:    req.Size,
	})
}

// void free(void* addr)
func handleFree(c *gin.Context) {
	var req FreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid free parameter"})
		return
	}
	err := pool.Free(req.Address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// void write(void* addr, byte val)
func writeMemory(c *gin.Context) {
	addr, err := strconv.ParseUint(c.Param("addr"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid write memory parameter"})
		return
	}
	var req MemWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid write memory parameter"})
		return
	}
	_, err = pool.Access(addr, &req.Value)
	if err != nil {
		c.JSON(HttpStatusUB, gin.H{"error": "UB"})
		return
	}
	c.JSON(http.StatusOK, gin.H{})
}

// byte read(void* addr)
func readMemory(c *gin.Context) {
	addr, err := strconv.ParseUint(c.Param("addr"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid read memory parameter"})
		return
	}
	val, err := pool.Access(addr, nil)
	if err != nil {
		c.JSON(HttpStatusUB, gin.H{"error": "UB"})
		return
	}
	c.JSON(http.StatusOK, MemReadResponse{Value: val})
}

func handleNotImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func router() *gin.Engine {
	r := gin.Default()
	// memory allocation/deallocation
	r.POST("/memory/malloc", handleMalloc)
	r.POST("/memory/calloc", handleNotImplemented)
	r.POST("/memory/realloc", handleNotImplemented)
	r.POST("/memory/free", handleFree)
	// byte-wise write/read access
	r.PUT("/memory/:addr", writeMemory)
	r.GET("/memory/:addr", readMemory)
	return r
}

func main() {
	log.Println("malloc REST Server")
	pool = mempool.NewMemPool()
	//gin.SetMode(gin.ReleaseMode)
	router().Run("localhost:8080")
}
