package importing

// CompareBeforePromote enforces deterministic compare-before-promote behavior.
func CompareBeforePromote(currentScore, candidateScore int) Decision {
	if candidateScore > currentScore {
		return Decision{Promote: true, Reason: "candidate_better"}
	}
	if candidateScore == currentScore {
		return Decision{Promote: false, Reason: "tie_no_promote"}
	}
	return Decision{Promote: false, Reason: "candidate_not_better"}
}
