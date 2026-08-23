package intake

// Status describes the lifecycle of a specimen batch inside the intake
// pipeline. A batch starts as draft, moves to reviewing once the collector
// submits it, then either approved or rejected.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusReviewing Status = "reviewing"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
)

// Item is a single specimen line inside a batch.
type Item struct {
	ID         string
	SampleCode string
	FoodType   string
	Quantity   int
}

// Batch is an intake batch of specimen lines with a small optimistic
// concurrency guard so two intake clerks cannot overwrite each other.
type Batch struct {
	ID         string
	SourceSite string
	Collector  string
	Status     Status
	Revision   int
	Items      []Item
	CreatedAt  string
	UpdatedAt  string
}

// Summary is the aggregated view of one batch.
type Summary struct {
	BatchID       string
	TotalItems    int
	TotalQuantity int
	ByFoodType    map[string]int
	Status        Status
}

// Clone returns a deep copy of the batch. Callers that keep a batch around
// after mutating it must use Clone so the stored copy never changes shape
// underneath concurrent readers.
func (b *Batch) Clone() *Batch {
	if b == nil {
		return nil
	}
	copy := *b
	return &copy
}
