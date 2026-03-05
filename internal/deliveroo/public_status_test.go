package deliveroo

import (
	"reflect"
	"testing"
)

func TestParsePublicStatus(t *testing.T) {
	text := `Deliveroo

This order is for Gildas

Estimated arrival

About 6 minutes

The order is out for delivery

Your order’s on track to be delivered on time.

Delivery

Rubel

Bicycle

221 Pentonville Road, FORA, N19UZ

Order Details

Ta’Mini Lebanese Bakery

Order #5040

1x

Baba Ghanouj

1x

Chicken kaak

1x

Falafel Wrap

Discover Deliveroo`

	got := ParsePublicStatus(text)
	if got.Customer != "Gildas" {
		t.Fatalf("customer=%q", got.Customer)
	}
	if got.EstimatedArrival != "About 6 minutes" {
		t.Fatalf("eta=%q", got.EstimatedArrival)
	}
	if got.Status != "The order is out for delivery" {
		t.Fatalf("status=%q", got.Status)
	}
	if got.StatusDetail != "Your order’s on track to be delivered on time." {
		t.Fatalf("detail=%q", got.StatusDetail)
	}
	if got.Courier != "Rubel" || got.Transport != "Bicycle" {
		t.Fatalf("delivery=%+v", got)
	}
	if got.Address != "221 Pentonville Road, FORA, N19UZ" {
		t.Fatalf("address=%q", got.Address)
	}
	if got.Restaurant != "Ta’Mini Lebanese Bakery" {
		t.Fatalf("restaurant=%q", got.Restaurant)
	}
	if got.OrderNumber != "Order #5040" {
		t.Fatalf("order=%q", got.OrderNumber)
	}
	wantItems := []string{"1x Baba Ghanouj", "1x Chicken kaak", "1x Falafel Wrap"}
	if !reflect.DeepEqual(got.Items, wantItems) {
		t.Fatalf("items=%v", got.Items)
	}
}
