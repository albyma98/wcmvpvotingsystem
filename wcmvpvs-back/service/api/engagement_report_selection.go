package api

import "fmt"

// Resolve selection in calculation order so retries with the same set of IDs
// produce the same approved version. A missing selection supports old clients;
// an explicitly empty selection is a validation error.
func validateReportKPISelection(kpis []reportKPI, requested *[]string, fallback []string, format string) ([]string, error) {
	ids := fallback
	if requested != nil {
		ids = *requested
	} else if len(ids) == 0 {
		for _, kpi := range kpis {
			ids = append(ids, kpi.ID)
		}
	}
	limit := 30
	if format == "one_page" {
		limit = 20
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("Seleziona almeno una KPI da inserire nel PDF.")
	}
	if len(ids) > limit {
		return nil, fmt.Errorf("Il formato scelto permette al massimo %d KPI. Modifica la selezione o scegli il report completo.", limit)
	}
	known, chosen := map[string]bool{}, map[string]bool{}
	for _, kpi := range kpis {
		known[kpi.ID] = true
	}
	for _, id := range ids {
		if !known[id] {
			return nil, fmt.Errorf("La selezione contiene una KPI non presente nel calcolo.")
		}
		if chosen[id] {
			return nil, fmt.Errorf("La selezione contiene una KPI duplicata.")
		}
		chosen[id] = true
	}
	selected := []string{}
	for _, kpi := range kpis {
		if chosen[kpi.ID] {
			selected = append(selected, kpi.ID)
		}
	}
	return selected, nil
}
