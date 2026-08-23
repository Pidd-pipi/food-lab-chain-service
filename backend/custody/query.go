package custody

import (
	"context"
	"strings"
)

// Filter narrows chains by specimen id, food type and time window. The input
// slice is treated as read-only; the returned slice never aliases it. Filter
// honors the context: once canceled it stops scanning and returns the cause
// instead of a half-built result that looks complete to the caller.
func Filter(ctx context.Context, chains []Chain, q Query) ([]Chain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]Chain, 0, len(chains))
	for _, chain := range chains {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if q.SpecimenID != "" && chain.SpecimenID != q.SpecimenID {
			continue
		}
		if q.FoodType != "" && !chainMatchesFoodType(chain, q.FoodType) {
			continue
		}
		if q.From != "" || q.To != "" {
			if !chainInWindow(chain, q.From, q.To) {
				continue
			}
		}
		out = append(out, *chain.Clone())
	}
	return out, nil
}

func chainMatchesFoodType(chain Chain, foodType string) bool {
	for _, node := range chain.Nodes {
		if strings.EqualFold(node.FoodType, foodType) {
			return true
		}
	}
	return false
}

func chainInWindow(chain Chain, from, to string) bool {
	if len(chain.Nodes) == 0 {
		return false
	}
	first := chain.Nodes[0].At
	last := chain.Nodes[len(chain.Nodes)-1].At
	if from != "" && last < from {
		return false
	}
	if to != "" && first > to {
		return false
	}
	return true
}

// Paginate pages a filtered chain list. Page size is capped at 200. A canceled
// context is reported immediately so pagination never returns a page for a
// request the caller has given up on.
func Paginate(ctx context.Context, chains []Chain, q Query) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 25
	}
	if size > 200 {
		size = 200
	}
	total := len(chains)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return Page{Items: chains[start:end], Page: page, PageSize: size, Total: total, HasNext: end < total}, nil
}

// GroupByFoodType counts chains per food type token found in their events. It
// honors the context and stops mid-sweep on cancellation rather than returning
// an undercounted map as if the run had finished.
func GroupByFoodType(ctx context.Context, chains []Chain) (map[string]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, chain := range chains {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, node := range chain.Nodes {
			for _, token := range strings.Fields(node.Action) {
				key := strings.ToLower(strings.Trim(token, " ,;"))
				if key == "" || seen[key] {
					continue
				}
				seen[key] = true
				out[key]++
			}
		}
	}
	return out, nil
}
