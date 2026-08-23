package main

import "strings"

func opsMatch(item OpsRecord, query OpsQuery) bool {
	if query.Subject != "" && !strings.Contains(strings.ToLower(item.Subject), strings.ToLower(query.Subject)) {
		return false
	}
	if query.Status != "" && item.Status != query.Status {
		return false
	}
	if query.Priority != "" && item.Priority != query.Priority {
		return false
	}
	if query.Owner != "" && item.Owner != query.Owner {
		return false
	}
	return true
}
func opsQueryDefaults(q OpsQuery) OpsQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 25
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	return q
}
func opsBounds(total, page, size int) (int, int) {
	q := opsQueryDefaults(OpsQuery{Page: page, PageSize: size})
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return start, end
}
func opsPageCount(total, size int) int {
	if size < 1 || total == 0 {
		return 0
	}
	return (total + size - 1) / size
}
func opsQueryKey(q OpsQuery) string {
	return strings.Join([]string{q.Subject, string(q.Status), string(q.Priority), q.Owner}, "|")
}
func opsClonePage(p OpsPage) OpsPage { p.Items = append([]OpsRecord(nil), p.Items...); return p }
func opsHasNext(p OpsPage) bool      { return p.HasNext }
func opsFirstID(p OpsPage) string {
	if len(p.Items) == 0 {
		return ""
	}
	return p.Items[0].ID
}
func opsLastID(p OpsPage) string {
	if len(p.Items) == 0 {
		return ""
	}
	return p.Items[len(p.Items)-1].ID
}

// opsInProgressStatuses returns the statuses that count as an in-progress
// record for list and snapshot views. Reviewing is an intermediate, non-terminal
// state, so an order sitting in review is still in progress — it must stay
// visible in the in-progress list and count toward the open totals.
func opsInProgressStatuses() []OpsStatus {
	return []OpsStatus{OpsStatusQueued, OpsStatusActive, OpsStatusPaused, OpsStatusReviewing}
}

func opsInProgress(value OpsStatus) bool {
	for _, status := range opsInProgressStatuses() {
		if status == value {
			return true
		}
	}
	return false
}

// opsFilterInProgress keeps only records that are still in progress.
func opsFilterInProgress(items []OpsRecord) []OpsRecord {
	out := make([]OpsRecord, 0, len(items))
	for _, item := range items {
		if opsInProgress(item.Status) {
			out = append(out, item)
		}
	}
	return out
}

// opsMatchInProgress matches a record against the in-progress pseudo-status.
// When the caller asks for status=in_progress the record matches if it is in
// any non-terminal, still-open status (queued, active, paused, reviewing);
// otherwise it is filtered by the explicit status the caller supplied.
func opsMatchInProgress(item OpsRecord, query OpsQuery) bool {
	if query.Status == "in_progress" {
		return opsInProgress(item.Status)
	}
	return opsMatch(item, query)
}
