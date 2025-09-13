package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"sync"
)

const (
	PUT_OP byte = 0
	DEL_OP      = 1
)

type WalWriter struct {
	tree *BPlusTree

	mu               sync.Mutex
	file             *os.File
	buffer           bytes.Buffer
	flushC           chan struct{}
	stopC            chan struct{}
	RecoverC         chan struct{}
	RecoverCallbackC chan struct{}
}

type LogEntry struct {
	op    byte
	key   string
	value string
}

func (this *WalWriter) Append(e *LogEntry) (err error) {
	this.mu.Lock()
	// serialization and write to buffer
	payload := &bytes.Buffer{}
	payload.WriteByte(e.op)
	err = binary.Write(payload, binary.LittleEndian, uint32(len(e.key)))
	if err != nil {
		panic("wal append serialization err, check memory.")
	}
	payload.WriteString(e.key)
	err = binary.Write(payload, binary.LittleEndian, uint32(len(e.value)))
	if err != nil {
		panic("wal append serialization err, check memory.")
	}
	payload.WriteString(e.value)

	err = binary.Write(&this.buffer, binary.LittleEndian, uint32(payload.Len()))
	if err != nil {
		panic("wal append serialization err, check memory.")
	}
	this.buffer.Write(payload.Bytes())

	this.mu.Unlock()
	// block it so avoid async problem
	this.flushC <- struct{}{}
	return nil
}

func NewWalWriter(path string, tree *BPlusTree) (*WalWriter, error) {
	// 需要RDWR权限才能支持Truncate操作，但不立即截断
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	res := &WalWriter{
		tree:             tree,
		file:             file,
		flushC:           make(chan struct{}, 1), //no buffer
		stopC:            make(chan struct{}),    // once notify
		RecoverC:         make(chan struct{}, 1),
		RecoverCallbackC: make(chan struct{}, 1),
	}
	go res.run() //it has state

	return res, nil
}

func (this *WalWriter) run() {
	// strong consistency
	for {
		select {
		case <-this.flushC:
			this.memory2disk()
		case <-this.stopC:
			// memory2disk for strong consistency
			this.memory2disk()
			return
		case <-this.RecoverC:
			err := this.disk2memory()
			if err != nil {
				panic(err)
			}
			this.RecoverCallbackC <- struct{}{}
		}

	}
}

// memory2disk to disk
func (this *WalWriter) memory2disk() {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.buffer.Len() == 0 {
		return
	}
	// 确保写入到文件末尾
	_, err := this.file.Seek(0, 2) // SEEK_END
	if err != nil {
		panic("seek wal file error")
	}
	_, err = this.file.Write(this.buffer.Bytes())
	if err != nil {
		// no tolerate
		panic("write wal buffer error, please check if memory resources ok.")
	}
	err = this.file.Sync()
	if err != nil {
		panic("write wal disk error, please check if disk resources ok.")
	}
	// reset buffer for Next log
	this.buffer.Reset()
}
func (this *WalWriter) disk2memory() error {
	var entries []LogEntry
	_, err := this.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(this.file)
	if err != nil {
		return err
	}
	offset := 0
	for offset < len(data) {
		if offset+4 > len(data) {
			return errors.New("header err")
		}
		payloadLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+payloadLen > len(data) {
			return errors.New("payload len err")
		}
		payload := data[offset : offset+payloadLen]
		offset += payloadLen

		// trans to entry
		var entry LogEntry
		p := 0
		entry.op = payload[p]
		p++

		if p+4 > len(payload) {
			return errors.New("payload key len err")
		}
		keyLen := int(binary.LittleEndian.Uint32(payload[p : p+4]))
		p += 4

		if p+keyLen > len(payload) {
			return errors.New("payload key err")
		}
		entry.key = string(payload[p : p+keyLen])
		p += keyLen

		if p+4 > len(payload) {
			return errors.New("payload value len err")
		}
		valueLen := int(binary.LittleEndian.Uint32(payload[p : p+4]))
		p += 4

		if p+valueLen > len(payload) {
			return errors.New("payload value err")
		}
		entry.value = string(payload[p : p+valueLen])
		p += valueLen

		entries = append(entries, entry)
	}
	for _, e := range entries {
		if e.op == PUT_OP {
			err := this.tree.Put(e.key, e.value)
			if err != nil {
				panic(err)
			}
		} else if e.op == DEL_OP {
			ok, err := this.tree.Del(e.key)
			if err != nil {
				// 在 WAL 恢复时，删除不存在的 key 是正常的，只记录日志
				log.Printf("WAL recovery: key '%s' not found during delete operation", e.key)
			} else if !ok {
				log.Printf("WAL recovery: key '%s' not found during delete operation", e.key)
			}
		}
	}
	return nil
}

func (this *WalWriter) Clear() {
	this.mu.Lock()
	defer this.mu.Unlock()
	err := this.file.Truncate(0)
	if err != nil {
		panic(err)
	}
	// 重置文件指针到开头
	_, err = this.file.Seek(0, 0)
	if err != nil {
		panic(err)
	}
}

func (this *WalWriter) Close() error {
	close(this.stopC)
	return this.file.Close()
}
