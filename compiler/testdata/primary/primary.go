package primary

// Store is satisfied by two components, but one is marked @Primary.
type Store interface{ Get() string }

// @Repository
// @Primary
type MemoryStore struct{}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }
func (*MemoryStore) Get() string   { return "memory" }

// @Repository
type DiskStore struct{}

func NewDiskStore() *DiskStore { return &DiskStore{} }
func (*DiskStore) Get() string { return "disk" }

// Consumer depends on Store; resolution picks the primary MemoryStore.
//
// @Service(name="consumer")
type Consumer struct{}

func NewConsumer(s Store) *Consumer { return &Consumer{} }
