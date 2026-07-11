package unknownann

// Widget carries an annotation that is not in the registry, which must surface
// as a warning rather than blocking the build.
//
// @Frobnicate(level=11)
// @Service
type Widget struct{}

func NewWidget() *Widget { return &Widget{} }
