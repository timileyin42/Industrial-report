package registry

// ChainVerifyResult is the outcome of walking an audit table's hash
// chain (migrations/0013_audit_log_tamper_evidence.sql) end to end.
// FirstBadID is the earliest row whose stored hash no longer matches
// what the chain expects — everything after it is provably suspect too,
// since each row's hash depends on the one before it.
type ChainVerifyResult struct {
	Valid         bool
	MismatchCount int64
	FirstBadID    *int64
}

// toChainVerifyResult normalizes sqlc's min(id) FILTER(...) result,
// which comes back as `interface{}` (either int64 or nil) since sqlc
// can't infer nullability from a FILTER clause.
func toChainVerifyResult(mismatchCount int64, firstBadID interface{}) ChainVerifyResult {
	res := ChainVerifyResult{Valid: mismatchCount == 0, MismatchCount: mismatchCount}
	if id, ok := firstBadID.(int64); ok {
		res.FirstBadID = &id
	}
	return res
}
