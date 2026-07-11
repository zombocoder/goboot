package ambiguous

// Store is satisfied by two components with no primary or qualifier.
type Store interface{ Get() string }

// @Repository
type MemoryStore struct{}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }
func (*MemoryStore) Get() string   { return "memory" }

// @Repository
type DiskStore struct{}

func NewDiskStore() *DiskStore { return &DiskStore{} }
func (*DiskStore) Get() string  { return "disk" }

// Consumer depends on the ambiguous Store interface.
//
// @Service
type Consumer struct{}

func NewConsumer(s Store) *Consumer { return &Consumer{} }
