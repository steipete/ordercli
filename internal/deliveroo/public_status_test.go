package deliveroo

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/steipete/ordercli/internal/browserpage"
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

func TestParsePublicStatusCompleted(t *testing.T) {
	text := `Deliveroo

Log in

Sign up

This order is for Gildas

Right on time 🙌

Your order took 20 minutes. Enjoy!

Delivery

221 Pentonville Road, FORA, N19UZ

Order Details

Ta’Mini Lebanese Bakery

Order #5040

1x

Baba Ghanouj

1x

Chicken kaak

Discover Deliveroo`

	got := ParsePublicStatus(text)
	if got.Customer != "Gildas" {
		t.Fatalf("customer=%q", got.Customer)
	}
	if got.Status != "Right on time 🙌" {
		t.Fatalf("status=%q", got.Status)
	}
	if got.StatusDetail != "Your order took 20 minutes. Enjoy!" {
		t.Fatalf("detail=%q", got.StatusDetail)
	}
	if got.Courier != "" || got.Transport != "" {
		t.Fatalf("unexpected delivery agent fields: %+v", got)
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
	wantItems := []string{"1x Baba Ghanouj", "1x Chicken kaak"}
	if !reflect.DeepEqual(got.Items, wantItems) {
		t.Fatalf("items=%v", got.Items)
	}
}

func TestFetchPublicStatus(t *testing.T) {
	orig := readBrowserPageText
	t.Cleanup(func() { readBrowserPageText = orig })

	readBrowserPageText = func(_ context.Context, targetURL string, opts browserpage.Options) (browserpage.Result, error) {
		if targetURL != "https://deliveroo.example/status" {
			t.Fatalf("targetURL=%q", targetURL)
		}
		if opts.Timeout != 5*time.Second {
			t.Fatalf("timeout=%s", opts.Timeout)
		}
		if !opts.Headless {
			t.Fatal("expected headless browser fetch")
		}
		return browserpage.Result{
			FinalURL: "https://deliveroo.example/final",
			Title:    "Deliveroo",
			Text: "This order is for Gildas\nDelivery\n221 Pentonville Road, FORA, N19UZ\nOrder Details\nTa'Mini\nOrder #5040",
		}, nil
	}

	got, err := FetchPublicStatus(context.Background(), "https://deliveroo.example/status", 5*time.Second)
	if err != nil {
		t.Fatalf("FetchPublicStatus: %v", err)
	}
	if got.URL != "https://deliveroo.example/final" || got.Title != "Deliveroo" {
		t.Fatalf("got=%+v", got)
	}
	if got.Customer != "Gildas" || got.Address != "221 Pentonville Road, FORA, N19UZ" {
		t.Fatalf("got=%+v", got)
	}
	if got.RawText == "" {
		t.Fatal("expected raw text to be preserved")
	}
}

func TestFetchPublicStatus_Error(t *testing.T) {
	orig := readBrowserPageText
	t.Cleanup(func() { readBrowserPageText = orig })

	readBrowserPageText = func(_ context.Context, _ string, _ browserpage.Options) (browserpage.Result, error) {
		return browserpage.Result{}, errors.New("boom")
	}

	if _, err := FetchPublicStatus(context.Background(), "https://deliveroo.example/status", time.Second); err == nil || err.Error() != "boom" {
		t.Fatalf("err=%v", err)
	}
}

func TestPublicStatusDetailsString(t *testing.T) {
	got := PublicStatus{
		Restaurant:       "Ta'Mini",
		OrderNumber:      "Order #5040",
		Customer:         "Gildas",
		Status:           "Right on time",
		StatusDetail:     "Delivered",
		EstimatedArrival: "Now",
		Courier:          "Rubel",
		Transport:        "Bicycle",
		Address:          "221 Pentonville Road, FORA, N19UZ",
		URL:              "https://deliveroo.example/status",
		Items:            []string{"1x Falafel Wrap", "1x Baba Ghanouj"},
	}.DetailsString()

	want := "restaurant=Ta'Mini\norder=Order #5040\nfor=Gildas\nstatus=Right on time\ndetail=Delivered\neta=Now\ncourier=Rubel\ntransport=Bicycle\naddress=221 Pentonville Road, FORA, N19UZ\nurl=https://deliveroo.example/status\nitems:\n  1x Falafel Wrap\n  1x Baba Ghanouj"
	if got != want {
		t.Fatalf("got=\n%s\nwant=\n%s", got, want)
	}
}

func TestLooksLikeAddress(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{line: "221 Pentonville Road", want: true},
		{line: "Pentonville Road, FORA", want: true},
		{line: "Bicycle", want: false},
	}
	for _, tc := range tests {
		if got := looksLikeAddress(tc.line); got != tc.want {
			t.Fatalf("looksLikeAddress(%q)=%v want %v", tc.line, got, tc.want)
		}
	}
}
