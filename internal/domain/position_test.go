package domain_test

import (
	"testing"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

func amount(v float64) *float64 { return &v }

func TestPositionValidate(t *testing.T) {
	cases := []struct {
		name     string
		position domain.Position
		want     error
	}{
		{
			"a complete position is valid",
			domain.Position{Code: "MGR", Name: "과장", RankLevel: 4, MinSalary: amount(3000000), MaxSalary: amount(5000000)},
			nil,
		},
		{
			"a position with no pay band is valid",
			domain.Position{Code: "MGR", Name: "과장"},
			nil,
		},
		{
			"a position with only a floor is valid",
			domain.Position{Code: "MGR", Name: "과장", MinSalary: amount(3000000)},
			nil,
		},
		{"code is required", domain.Position{Name: "과장"}, domain.ErrPositionCodeEmpty},
		{"name is required", domain.Position{Code: "MGR"}, domain.ErrPositionNameEmpty},
		{
			"rank level cannot be negative",
			domain.Position{Code: "MGR", Name: "과장", RankLevel: -1},
			domain.ErrPositionRankNegative,
		},
		{
			"a negative floor is rejected",
			domain.Position{Code: "MGR", Name: "과장", MinSalary: amount(-1)},
			domain.ErrPositionSalaryNegative,
		},
		{
			"a floor above the ceiling is rejected",
			domain.Position{Code: "MGR", Name: "과장", MinSalary: amount(5000000), MaxSalary: amount(3000000)},
			domain.ErrPositionSalaryRange,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			position := tc.position
			if err := position.Validate(); err != tc.want {
				t.Errorf("Validate() = %v, want %v", err, tc.want)
			}
		})
	}
}
