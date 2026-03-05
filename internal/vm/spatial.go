package vm

import (
	"container/list"
	"sync"

	"github.com/rawbytedev/memory3d/internal/types"
)

// Spatial-aware LRU cache
type SpatialCache struct {
	cache    map[types.Address3D]*list.Element
	ll       *list.List
	capacity int
	mu       sync.RWMutex

	// Spatial prefetching
	prefetchWindow uint32
	prefetchQueue  chan []types.Address3D
}

type CacheEntry struct {
	Data []byte
}

func NewSpatialCache() *SpatialCache {
	return &SpatialCache{
		cache:          make(map[types.Address3D]*list.Element),
		ll:             list.New(),
		capacity:       1000, // Default LRU capacity
		prefetchWindow: 64,   // Prefetch radius in 3D space
		prefetchQueue:  make(chan []types.Address3D, 10),
	}
}

func (c *SpatialCache) Get(addr types.Address3D) ([]byte, bool) {
	c.mu.RLock()
	elem, ok := c.cache[addr]
	c.mu.RUnlock()

	if !ok {
		// Trigger spatial prefetch
		go c.prefetchSpatial(addr)
		return nil, false
	}

	// Move to front (hot)
	c.mu.Lock()
	c.ll.MoveToFront(elem)
	c.mu.Unlock()

	return elem.Value.(*CacheEntry).Data, true
}

// Put adds or updates an entry in the cache
func (c *SpatialCache) Put(addr types.Address3D, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already in cache, update and move to front
	if elem, ok := c.cache[addr]; ok {
		c.ll.MoveToFront(elem)
		elem.Value.(*CacheEntry).Data = data
		return
	}

	// Add new entry to front
	entry := &CacheEntry{Data: data}
	elem := c.ll.PushFront(entry)
	c.cache[addr] = elem

	// Evict oldest if capacity exceeded
	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			// Find and remove from map
			// Note: This is simplified; ideally we'd store addr in entry
			// TODO: Optimize by storing address in CacheEntry
		}
	}
}

// Invalidate removes an entry from the cache
func (c *SpatialCache) Invalidate(addr types.Address3D) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[addr]; ok {
		c.ll.Remove(elem)
		delete(c.cache, addr)
	}
}

// Prefetch based on spatial locality in 3D space
func (c *SpatialCache) prefetchSpatial(center types.Address3D) {
	// Calculate bounding box in 3D
	minX := center.X - uint64(c.prefetchWindow)
	maxX := center.X + uint64(c.prefetchWindow)
	minY := center.Y - c.prefetchWindow
	maxY := center.Y + c.prefetchWindow
	minZ := uint16(max(0, int(center.Z)-int(c.prefetchWindow)))
	maxZ := uint16(min(65535, int(center.Z)+int(c.prefetchWindow)))

	// Generate addresses to prefetch
	var addrs []types.Address3D
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				addr := types.Address3D{X: x, Y: y, Z: z}
				addrs = append(addrs, addr)

				// Batch prefetch
				if len(addrs) >= 64 {
					select {
					case c.prefetchQueue <- addrs:
						addrs = nil
					default:
						// Queue full, skip
					}
				}
			}
		}
	}
}
