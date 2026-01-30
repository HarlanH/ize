package ize

import (
	"fmt"

	"ize/internal/logger"
)

// Clause represents a single facet with one or more values (OR of values)
// e.g., brand:Samsung OR brand:LG
type Clause struct {
	FacetName string   // The facet name (e.g., "brand")
	Values    []string // The values to match (OR semantics)
}

// DecisionList represents a cluster's filter rule as a conjunction of clauses
// Structure: (facet1:val1 OR facet1:val2) AND (facet2:val3) AND ...
// This maps directly to Algolia's facetFilters format
type DecisionList struct {
	Clauses []Clause // AND of these clauses (max 3 recommended)
}

// ToAlgoliaFilter converts the decision list to Algolia's facetFilters format
// Returns [][]string where outer array is AND, inner arrays are OR
// e.g., [["brand:Samsung", "brand:LG"], ["color:Black"]]
func (d DecisionList) ToAlgoliaFilter() [][]string {
	if len(d.Clauses) == 0 {
		return nil
	}

	filters := make([][]string, 0, len(d.Clauses))
	for _, clause := range d.Clauses {
		if len(clause.Values) == 0 {
			continue
		}
		orGroup := make([]string, 0, len(clause.Values))
		for _, value := range clause.Values {
			orGroup = append(orGroup, fmt.Sprintf("%s:%s", clause.FacetName, value))
		}
		filters = append(filters, orGroup)
	}
	return filters
}

// Matches tests whether an item's facet set matches this decision list
// All clauses must match (AND semantics), and within a clause, any value matches (OR semantics)
func (d DecisionList) Matches(fs FacetSet) bool {
	if len(d.Clauses) == 0 {
		return true // Empty rule matches everything
	}

	for _, clause := range d.Clauses {
		clauseMatches := false
		for _, value := range clause.Values {
			key := fmt.Sprintf("%s:%s", clause.FacetName, value)
			if fs[key] {
				clauseMatches = true
				break
			}
		}
		if !clauseMatches {
			return false // AND semantics: all clauses must match
		}
	}
	return true
}

// String returns a human-readable representation of the decision list
func (d DecisionList) String() string {
	if len(d.Clauses) == 0 {
		return "(empty rule)"
	}

	var parts []string
	for _, clause := range d.Clauses {
		if len(clause.Values) == 1 {
			parts = append(parts, fmt.Sprintf("%s:%s", clause.FacetName, clause.Values[0]))
		} else {
			var orParts []string
			for _, v := range clause.Values {
				orParts = append(orParts, fmt.Sprintf("%s:%s", clause.FacetName, v))
			}
			parts = append(parts, fmt.Sprintf("(%s)", joinStrings(orParts, " OR ")))
		}
	}
	return joinStrings(parts, " AND ")
}

// RuleQuality holds metrics about how well a decision list captures a cluster
type RuleQuality struct {
	Precision float64 // What fraction of rule matches are true cluster members
	Recall    float64 // What fraction of cluster members match the rule
	F1        float64 // Harmonic mean of precision and recall
}

// Rule fitting constants
const (
	// MaxClausesInRule is the maximum number of facet clauses in a decision list
	MaxClausesInRule = 3

	// MinLiftThreshold is the minimum lift for a value to be included in a clause
	// Lift = P(value|positive) / P(value|all) - values with lift > 1 are over-represented in positives
	MinLiftThreshold = 1.2
)

// valueStats tracks counts for a facet value in positive and total sets
type valueStats struct {
	positiveCount int
	totalCount    int
}

// facetValueStats maps facetName -> value -> stats
type facetValueStats map[string]map[string]*valueStats

// collectFacetStats builds statistics about facet values across positive and all items
func collectFacetStats(positiveSet map[int]bool, allFacetSets []FacetSet) facetValueStats {
	stats := make(facetValueStats)

	for idx, fs := range allFacetSets {
		isPositive := positiveSet[idx]
		for facetKey := range fs {
			facetName, facetValue := parseFacetKey(facetKey)
			if facetName == "" {
				continue
			}
			if stats[facetName] == nil {
				stats[facetName] = make(map[string]*valueStats)
			}
			if stats[facetName][facetValue] == nil {
				stats[facetName][facetValue] = &valueStats{}
			}
			stats[facetName][facetValue].totalCount++
			if isPositive {
				stats[facetName][facetValue].positiveCount++
			}
		}
	}

	return stats
}

// selectValuesWithLift returns values that have lift >= threshold
func selectValuesWithLift(values map[string]*valueStats, totalPositives, totalItems int) []string {
	var selected []string
	for value, stats := range values {
		if stats.totalCount == 0 || stats.positiveCount == 0 {
			continue
		}
		// Lift = P(value|positive) / P(value|all)
		pValueGivenPositive := float64(stats.positiveCount) / float64(totalPositives)
		pValue := float64(stats.totalCount) / float64(totalItems)
		if pValue > 0 {
			lift := pValueGivenPositive / pValue
			if lift >= MinLiftThreshold {
				selected = append(selected, value)
			}
		}
	}
	return selected
}

// fitDecisionList fits a decision list rule for a cluster using greedy facet selection
// positiveIndices: indices of items in the cluster (positive examples)
// allFacetSets: facet sets for all items
// Returns the fitted rule and quality metrics
func fitDecisionList(positiveIndices []int, allFacetSets []FacetSet, log *logger.Logger) (*DecisionList, *RuleQuality) {
	if len(positiveIndices) == 0 || len(allFacetSets) == 0 {
		return &DecisionList{}, &RuleQuality{}
	}

	// Build positive set for quick lookup
	positiveSet := make(map[int]bool)
	for _, idx := range positiveIndices {
		positiveSet[idx] = true
	}

	totalItems := len(allFacetSets)
	totalPositives := len(positiveIndices)

	// Collect facet value statistics
	facetStats := collectFacetStats(positiveSet, allFacetSets)

	// Greedy clause selection
	clauses := selectClausesGreedy(facetStats, positiveIndices, allFacetSets, totalPositives, totalItems, log)

	rule := &DecisionList{Clauses: clauses}
	quality := computeRuleQuality(*rule, positiveIndices, allFacetSets)

	log.Debug("fitDecisionList: completed",
		"clauses", len(clauses),
		"precision", fmt.Sprintf("%.3f", quality.Precision),
		"recall", fmt.Sprintf("%.3f", quality.Recall),
		"f1", fmt.Sprintf("%.3f", quality.F1),
	)

	return rule, quality
}

// selectClausesGreedy performs greedy selection of facet clauses to maximize recall
func selectClausesGreedy(facetStats facetValueStats, positiveIndices []int, allFacetSets []FacetSet, totalPositives, totalItems int, log *logger.Logger) []Clause {
	var clauses []Clause
	usedFacets := make(map[string]bool)

	for len(clauses) < MaxClausesInRule {
		bestClause, bestFacet, bestRecallGain, bestNewRecall := findBestClause(
			clauses, facetStats, usedFacets, positiveIndices, allFacetSets, totalPositives, totalItems,
		)

		if bestFacet == "" {
			break // No improvement found
		}

		clauses = append(clauses, bestClause)
		usedFacets[bestFacet] = true

		log.Debug("fitDecisionList: added clause",
			"facet", bestFacet,
			"values_count", len(bestClause.Values),
			"recall_gain", fmt.Sprintf("%.3f", bestRecallGain),
			"new_recall", fmt.Sprintf("%.3f", bestNewRecall),
		)
	}

	return clauses
}

// findBestClause finds the best facet clause to add given current clauses
func findBestClause(currentClauses []Clause, facetStats facetValueStats, usedFacets map[string]bool, positiveIndices []int, allFacetSets []FacetSet, totalPositives, totalItems int) (Clause, string, float64, float64) {
	bestFacet := ""
	bestClause := Clause{}
	bestRecallGain := 0.0
	bestNewRecall := 0.0

	currentRule := DecisionList{Clauses: currentClauses}
	currentRecall := computeRecall(currentRule, positiveIndices, allFacetSets)

	for facetName, values := range facetStats {
		if usedFacets[facetName] {
			continue
		}

		selectedValues := selectValuesWithLift(values, totalPositives, totalItems)
		if len(selectedValues) == 0 {
			continue
		}

		candidateClause := Clause{FacetName: facetName, Values: selectedValues}
		candidateClauses := append(currentClauses, candidateClause)
		candidateRule := DecisionList{Clauses: candidateClauses}

		newRecall := computeRecall(candidateRule, positiveIndices, allFacetSets)
		recallGain := newRecall - currentRecall

		if shouldSelectClause(len(currentClauses), newRecall, recallGain, candidateRule, currentRule, positiveIndices, allFacetSets, bestNewRecall, bestRecallGain) {
			bestFacet = facetName
			bestClause = candidateClause
			bestRecallGain = recallGain
			bestNewRecall = newRecall
		}
	}

	return bestClause, bestFacet, bestRecallGain, bestNewRecall
}

// shouldSelectClause determines if a candidate clause should replace the current best
func shouldSelectClause(numClauses int, newRecall, recallGain float64, candidateRule, currentRule DecisionList, positiveIndices []int, allFacetSets []FacetSet, bestNewRecall, bestRecallGain float64) bool {
	if numClauses == 0 {
		// First clause: maximize recall
		return newRecall > bestNewRecall
	}

	// Subsequent clauses: only add if recall doesn't drop too much and precision improves
	if recallGain >= -0.1 && newRecall >= 0.5 {
		newPrecision := computePrecision(candidateRule, positiveIndices, allFacetSets)
		currentPrecision := computePrecision(currentRule, positiveIndices, allFacetSets)
		if newPrecision > currentPrecision {
			return newRecall > bestNewRecall || (newRecall == bestNewRecall && recallGain > bestRecallGain)
		}
	}
	return false
}

// computeRecall calculates what fraction of positives match the rule
func computeRecall(rule DecisionList, positiveIndices []int, allFacetSets []FacetSet) float64 {
	if len(positiveIndices) == 0 {
		return 0
	}
	matches := 0
	for _, idx := range positiveIndices {
		if rule.Matches(allFacetSets[idx]) {
			matches++
		}
	}
	return float64(matches) / float64(len(positiveIndices))
}

// computePrecision calculates what fraction of rule matches are positives
func computePrecision(rule DecisionList, positiveIndices []int, allFacetSets []FacetSet) float64 {
	positiveSet := make(map[int]bool)
	for _, idx := range positiveIndices {
		positiveSet[idx] = true
	}

	totalMatches := 0
	truePositives := 0
	for idx, fs := range allFacetSets {
		if rule.Matches(fs) {
			totalMatches++
			if positiveSet[idx] {
				truePositives++
			}
		}
	}

	if totalMatches == 0 {
		return 0
	}
	return float64(truePositives) / float64(totalMatches)
}

// computeRuleQuality calculates precision, recall, and F1 for a rule
func computeRuleQuality(rule DecisionList, positiveIndices []int, allFacetSets []FacetSet) *RuleQuality {
	precision := computePrecision(rule, positiveIndices, allFacetSets)
	recall := computeRecall(rule, positiveIndices, allFacetSets)

	var f1 float64
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	return &RuleQuality{
		Precision: precision,
		Recall:    recall,
		F1:        f1,
	}
}

// fitAndReassign fits decision list rules to each cluster and reassigns items based on rules
// Items can belong to multiple clusters if they match multiple rules (overlapping clusters)
func fitAndReassign(groups []ClusterGroup, allItems []Result, facetSets []FacetSet, log *logger.Logger) []ClusterGroup {
	if len(groups) == 0 {
		return groups
	}

	// Build item index lookup (Result.ID -> index in allItems)
	itemIndex := make(map[string]int)
	for i, item := range allItems {
		itemIndex[item.ID] = i
	}

	// Phase 1: Fit rules for each cluster based on original membership
	clusterRules := fitRulesForClusters(groups, itemIndex, facetSets, log)

	// Phase 2: Reassign items based on rules (allows overlapping membership)
	newGroups := reassignItemsByRules(clusterRules, allItems, facetSets)

	// Phase 3: Recalculate TopFacets and Stats for each cluster
	for i := range newGroups {
		newGroups[i].TopFacets = calculateTopFacets(newGroups[i].Items, facetSets, itemIndex)
		newGroups[i].Stats = ClusterStats{
			Size:      len(newGroups[i].Items),
			TopFacets: newGroups[i].TopFacets,
		}

		log.Debug("fitAndReassign: reassigned cluster",
			"cluster", i,
			"new_size", len(newGroups[i].Items),
		)
	}

	return newGroups
}

// clusterRuleInfo holds the fitted rule and metadata for a cluster
type clusterRuleInfo struct {
	rule    *DecisionList
	quality *RuleQuality
	name    string
}

// fitRulesForClusters fits decision list rules for each cluster
func fitRulesForClusters(groups []ClusterGroup, itemIndex map[string]int, facetSets []FacetSet, log *logger.Logger) []clusterRuleInfo {
	rules := make([]clusterRuleInfo, len(groups))

	for i, group := range groups {
		positiveIndices := make([]int, 0, len(group.Items))
		for _, item := range group.Items {
			if idx, ok := itemIndex[item.ID]; ok {
				positiveIndices = append(positiveIndices, idx)
			}
		}

		rule, quality := fitDecisionList(positiveIndices, facetSets, log)

		// Generate name from the rule - this ensures unique names for different rules
		name := rule.String()
		if name == "(empty rule)" {
			name = group.Name // Fallback to original name if rule is empty
		}

		rules[i] = clusterRuleInfo{
			rule:    rule,
			quality: quality,
			name:    name,
		}

		log.Debug("fitAndReassign: fitted rule for cluster",
			"cluster", i,
			"original_size", len(group.Items),
			"rule", rule.String(),
			"recall", fmt.Sprintf("%.3f", quality.Recall),
			"precision", fmt.Sprintf("%.3f", quality.Precision),
		)
	}

	return rules
}

// reassignItemsByRules creates new cluster groups by applying rules to all items
func reassignItemsByRules(clusterRules []clusterRuleInfo, allItems []Result, facetSets []FacetSet) []ClusterGroup {
	newGroups := make([]ClusterGroup, len(clusterRules))
	for i := range newGroups {
		newGroups[i] = ClusterGroup{
			Name:        clusterRules[i].name,
			Items:       []Result{},
			Rule:        clusterRules[i].rule,
			RuleQuality: clusterRules[i].quality,
		}
	}

	// Assign each item to all clusters whose rules it matches
	for idx, fs := range facetSets {
		for i, cr := range clusterRules {
			if cr.rule.Matches(fs) {
				newGroups[i].Items = append(newGroups[i].Items, allItems[idx])
			}
		}
	}

	return newGroups
}
