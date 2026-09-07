package domain

import "math"

// Monetary precision policy.
//
// Amount columns in PostgreSQL are DECIMAL(18,2) (see db/migrations/000005).
// The Go models carry them as float64, so every value that is about to be
// persisted or compared MUST first be quantized to the same 2-decimal scale
// the database uses. Without that step PostgreSQL rounds each row on its own
// (33.333 -> 33.33) while Go keeps the unrounded value, and the header total
// of a voucher stops matching the sum of its lines.
//
// The long-term fix is an exact type (shopspring/decimal or int64 minor
// units). That is deliberately NOT done here: DebitAmount/CreditAmount are
// float64 in internal/dto and internal/handler as well, so switching the type
// would break every caller outside this package. Quantize-then-compare gives
// the same observable behaviour for 2-decimal money without touching the
// public field types.
const (
	// AmountScale is the number of decimal places stored by the database.
	AmountScale = 2

	// amountFactor is 10^AmountScale.
	amountFactor = 100.0

	// BalanceEpsilon is the tolerance used when comparing two monetary
	// values for equality. It is half of the smallest representable unit
	// (0.01 / 2), so two values that quantize to the same stored amount
	// always compare equal, and values that differ by one unit never do.
	BalanceEpsilon = 0.005
)

// RoundAmount quantizes a monetary value to AmountScale decimal places using
// half-away-from-zero rounding, which is what PostgreSQL's numeric type does.
//
// Note: float64 cannot represent every 2-decimal value exactly, so a value
// that is mathematically exactly on a .005 boundary may round in either
// direction depending on its binary representation. Callers must therefore
// quantize once, as early as possible, and use the quantized value everywhere
// afterwards rather than re-deriving it.
func RoundAmount(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*amountFactor) / amountFactor
}

// AmountsEqual reports whether two monetary values are equal within
// BalanceEpsilon. Use this instead of == for any float64 money comparison.
func AmountsEqual(a, b float64) bool {
	return math.Abs(a-b) < BalanceEpsilon
}

// IsZeroAmount reports whether a monetary value rounds to zero at the stored
// scale. An amount of 0.004 is "non-zero" in Go but is stored as 0.00, which
// violates the chk_entry_amount constraint, so it must be treated as zero.
func IsZeroAmount(v float64) bool {
	return math.Abs(v) < BalanceEpsilon
}

// SumAmounts adds monetary values, quantizing each addend and the result so
// the total matches what the database computes from the stored rows.
func SumAmounts(values ...float64) float64 {
	var total float64
	for _, v := range values {
		total += RoundAmount(v)
	}
	return RoundAmount(total)
}
