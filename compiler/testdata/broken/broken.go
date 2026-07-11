package broken

// Broken carries an annotation but the package does not type-check, exercising
// the loader's error surfacing.
//
// @Service
type Broken struct{}

func use() { _ = undefinedSymbol }
