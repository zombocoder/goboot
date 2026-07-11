// Package conditions exercises profiles and conditional components (§29).
package conditions

// @Application(name="conditions-app")
type Application struct{}

// Always is unconditional and always present.
//
// @Service(name="always")
type Always struct{}

func NewAlways() *Always { return &Always{} }

// ProdOnly is active only under the "production" profile.
//
// @Service(name="prodOnly")
// @Profile(["production"])
type ProdOnly struct{}

func NewProdOnly() *ProdOnly { return &ProdOnly{} }

// DevOnly is active only under the "dev" profile.
//
// @Service(name="devOnly")
// @Profile(["dev"])
type DevOnly struct{}

func NewDevOnly() *DevOnly { return &DevOnly{} }

// CacheEnabled is present only when the property cache.enabled is "true".
//
// @Service(name="cacheEnabled")
// @ConditionalOnProperty(name="cache.enabled", havingValue="true")
type CacheEnabled struct{}

func NewCacheEnabled() *CacheEnabled { return &CacheEnabled{} }

// NeedsAlways is present only when a component named "always" is present.
//
// @Service(name="needsAlways")
// @ConditionalOnNut(type="always")
type NeedsAlways struct{}

func NewNeedsAlways() *NeedsAlways { return &NeedsAlways{} }

// FallbackClock is present only when no "PrimaryClock" is provided.
//
// @Service(name="fallbackClock")
// @ConditionalOnMissingNut(type="PrimaryClock")
type FallbackClock struct{}

func NewFallbackClock() *FallbackClock { return &FallbackClock{} }

// NeedsProd requires prodOnly, which is itself profile-gated. When production is
// inactive, prodOnly is removed and NeedsProd must cascade out (fixpoint).
//
// @Service(name="needsProd")
// @ConditionalOnNut(type="prodOnly")
type NeedsProd struct{}

func NewNeedsProd() *NeedsProd { return &NeedsProd{} }
