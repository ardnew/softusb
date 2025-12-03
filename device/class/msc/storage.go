package msc

import (
	"io"
	"os"
	"sync"
)

// Storage defines the interface for MSC storage backends.
// Implementations provide block-level storage operations.
type Storage interface {
	// BlockSize returns the size of a storage block in bytes.
	BlockSize() uint32

	// BlockCount returns the total number of blocks.
	BlockCount() uint64

	// Read reads blocks starting at lba into buf.
	// Returns number of blocks read or error.
	Read(lba uint64, blocks uint32, buf []byte) (uint32, error)

	// Write writes blocks from buf starting at lba.
	// Returns number of blocks written or error.
	Write(lba uint64, blocks uint32, buf []byte) (uint32, error)

	// Sync flushes any cached writes to storage.
	Sync() error

	// IsReadOnly returns true if storage is read-only.
	IsReadOnly() bool

	// IsRemovable returns true if media is removable.
	IsRemovable() bool

	// IsPresent returns true if media is present (for removable media).
	IsPresent() bool

	// Eject ejects removable media (optional operation).
	// Returns error if not supported or media cannot be ejected.
	Eject() error
}

// MemoryStorage implements Storage interface using an in-memory buffer.
type MemoryStorage struct {
	data      []byte
	blockSize uint32
	readOnly  bool
	removable bool
	present   bool
	mutex     sync.RWMutex
}

// NewMemoryStorage creates an in-memory storage with the given size and block size.
func NewMemoryStorage(size uint64, blockSize uint32) *MemoryStorage {
	return &MemoryStorage{
		data:      make([]byte, size),
		blockSize: blockSize,
		present:   true,
	}
}

// BlockSize returns the block size.
func (m *MemoryStorage) BlockSize() uint32 {
	return m.blockSize
}

// BlockCount returns the number of blocks.
func (m *MemoryStorage) BlockCount() uint64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return uint64(len(m.data)) / uint64(m.blockSize)
}

// Read reads blocks from memory.
func (m *MemoryStorage) Read(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.present {
		return 0, io.EOF
	}

	offset := lba * uint64(m.blockSize)
	length := uint64(blocks) * uint64(m.blockSize)

	if offset+length > uint64(len(m.data)) {
		return 0, io.EOF
	}

	if uint64(len(buf)) < length {
		return 0, io.ErrShortBuffer
	}

	copy(buf, m.data[offset:offset+length])
	return blocks, nil
}

// Write writes blocks to memory.
func (m *MemoryStorage) Write(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.present {
		return 0, io.EOF
	}

	if m.readOnly {
		return 0, os.ErrPermission
	}

	offset := lba * uint64(m.blockSize)
	length := uint64(blocks) * uint64(m.blockSize)

	if offset+length > uint64(len(m.data)) {
		return 0, io.EOF
	}

	if uint64(len(buf)) < length {
		return 0, io.ErrShortBuffer
	}

	copy(m.data[offset:offset+length], buf)
	return blocks, nil
}

// Sync is a no-op for memory storage.
func (m *MemoryStorage) Sync() error {
	return nil
}

// IsReadOnly returns whether the storage is read-only.
func (m *MemoryStorage) IsReadOnly() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.readOnly
}

// SetReadOnly sets the read-only flag.
func (m *MemoryStorage) SetReadOnly(readOnly bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.readOnly = readOnly
}

// IsRemovable returns whether the media is removable.
func (m *MemoryStorage) IsRemovable() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.removable
}

// SetRemovable sets the removable flag.
func (m *MemoryStorage) SetRemovable(removable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.removable = removable
}

// IsPresent returns whether media is present.
func (m *MemoryStorage) IsPresent() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.present
}

// SetPresent sets the media presence flag.
func (m *MemoryStorage) SetPresent(present bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.present = present
}

// Eject ejects the media (sets present to false).
func (m *MemoryStorage) Eject() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.removable {
		return os.ErrPermission
	}

	m.present = false
	return nil
}

// FileStorage implements Storage interface using a file.
type FileStorage struct {
	file      *os.File
	blockSize uint32
	size      uint64
	readOnly  bool
	mutex     sync.RWMutex
}

// NewFileStorage creates file-backed storage.
// If readOnly is true, the file is opened in read-only mode.
func NewFileStorage(path string, blockSize uint32, readOnly bool) (*FileStorage, error) {
	flags := os.O_RDWR
	if readOnly {
		flags = os.O_RDONLY
	}

	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &FileStorage{
		file:      file,
		blockSize: blockSize,
		size:      uint64(stat.Size()),
		readOnly:  readOnly,
	}, nil
}

// BlockSize returns the block size.
func (f *FileStorage) BlockSize() uint32 {
	return f.blockSize
}

// BlockCount returns the number of blocks.
func (f *FileStorage) BlockCount() uint64 {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.size / uint64(f.blockSize)
}

// Read reads blocks from file.
func (f *FileStorage) Read(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	offset := int64(lba * uint64(f.blockSize))
	length := int(blocks * f.blockSize)

	if uint64(offset)+uint64(length) > f.size {
		return 0, io.EOF
	}

	if len(buf) < length {
		return 0, io.ErrShortBuffer
	}

	n, err := f.file.ReadAt(buf[:length], offset)
	if err != nil && err != io.EOF {
		return 0, err
	}

	return uint32(n) / f.blockSize, nil
}

// Write writes blocks to file.
func (f *FileStorage) Write(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.readOnly {
		return 0, os.ErrPermission
	}

	offset := int64(lba * uint64(f.blockSize))
	length := int(blocks * f.blockSize)

	if uint64(offset)+uint64(length) > f.size {
		return 0, io.EOF
	}

	if len(buf) < length {
		return 0, io.ErrShortBuffer
	}

	n, err := f.file.WriteAt(buf[:length], offset)
	if err != nil {
		return 0, err
	}

	return uint32(n) / f.blockSize, nil
}

// Sync flushes file writes to disk.
func (f *FileStorage) Sync() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.readOnly {
		return nil
	}

	return f.file.Sync()
}

// IsReadOnly returns whether the storage is read-only.
func (f *FileStorage) IsReadOnly() bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.readOnly
}

// IsRemovable returns false (file storage is not removable).
func (f *FileStorage) IsRemovable() bool {
	return false
}

// IsPresent returns true (file storage is always present).
func (f *FileStorage) IsPresent() bool {
	return true
}

// Eject is not supported for file storage.
func (f *FileStorage) Eject() error {
	return os.ErrPermission
}

// Close closes the underlying file.
func (f *FileStorage) Close() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		return err
	}
	return nil
}
