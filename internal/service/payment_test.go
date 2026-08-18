package service

import (
	"testing"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

func TestComputeDiscount(t *testing.T) {
	tests := []struct {
		name   string
		coupon domain.Coupon
		amount int
		want   int
	}{
		{
			name:   "percent half",
			coupon: domain.Coupon{DiscountType: domain.CouponDiscountPercent, DiscountValue: 50},
			amount: 1000,
			want:   500,
		},
		{
			name:   "fixed capped by amount",
			coupon: domain.Coupon{DiscountType: domain.CouponDiscountFixed, DiscountValue: 2000},
			amount: 1500,
			want:   1500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDiscount(&tt.coupon, tt.amount)
			if got != tt.want {
				t.Fatalf("computeDiscount() = %d, want %d", got, tt.want)
			}
		})
	}
}
