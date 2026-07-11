package badsched

// @Service
type Svc struct{}

func NewSvc() *Svc { return &Svc{} }

// Tick has a @Scheduled annotation with no rate, which is invalid.
//
// @Scheduled
func (s *Svc) Tick() {}
