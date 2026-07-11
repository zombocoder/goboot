package model

// Controller groups a controller component with its base path and the routes
// its methods expose (§17).
type Controller struct {
	// Component is the controller component itself, wired like any other.
	Component *Component
	// BasePath is the path prefix from @RequestMapping, e.g. /api/v1/users.
	BasePath string
	// Routes are the endpoints declared by the controller's methods, in
	// deterministic order.
	Routes []*Route
}
