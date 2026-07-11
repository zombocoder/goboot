package badlifecycle

// @Service
type Svc struct{}

func NewSvc() *Svc { return &Svc{} }

// Init has an unsupported lifecycle signature: it takes a non-context argument.
//
// @PostConstruct
func (s *Svc) Init(n int) error { return nil }
