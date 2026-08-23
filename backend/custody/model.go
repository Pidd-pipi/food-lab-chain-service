package custody

// Node is one custody event inside a specimen chain.
type Node struct {
	Step       int
	EventID    string
	SpecimenID string
	FoodType   string
	Actor      string
	Action     string
	At         string
}

// Chain is the assembled custody trail for one specimen.
type Chain struct {
	SpecimenID string
	Nodes      []Node
	Complete   bool
}

// Clone returns a deep copy so callers can annotate a chain without changing
// the shared assembly.
func (c *Chain) Clone() *Chain {
	if c == nil {
		return nil
	}
	copy := *c
	copy.Nodes = append([]Node(nil), c.Nodes...)
	return &copy
}

// Query filters and pages chains.
type Query struct {
	SpecimenID string
	FoodType   string
	From       string
	To         string
	Page       int
	PageSize   int
}

// Page is a paginated chain listing.
type Page struct {
	Items    []Chain
	Page     int
	PageSize int
	Total    int
	HasNext  bool
}
