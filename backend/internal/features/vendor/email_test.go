package vendor

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		name, input, want string
		invalid           bool
	}{
		{name: "omitted"},
		{name: "whitespace", input: " \t\n"},
		{name: "ordinary", input: "vendor@example.com", want: "vendor@example.com"},
		{name: "normalization", input: " \tStore.Name+Sales@EXAMPLE.COM ", want: "Store.Name+Sales@example.com"},
		{name: "hyphenated domain", input: "Store@MY-SHOP.EXAMPLE", want: "Store@my-shop.example"},
		{name: "punycode domain", input: "vendor@xn--bcher-kva.example", want: "vendor@xn--bcher-kva.example"},
		{name: "leading domain hyphen", input: "vendor@-example.com", invalid: true},
		{name: "trailing domain hyphen", input: "vendor@example-.com", invalid: true},
		{name: "domain underscore", input: "vendor@exam_ple.com", invalid: true},
		{name: "oversized domain label", input: "vendor@" + strings.Repeat("a", 64) + ".com", invalid: true},
		{name: "empty domain label", input: "vendor@example..com", invalid: true},
		{name: "domain literal", input: "vendor@[127.0.0.1]", invalid: true},
		{name: "missing domain", input: "vendor@", invalid: true},
		{name: "missing at", input: "vendor.example.com", invalid: true},
		{name: "display name", input: "Vendor <vendor@example.com>", invalid: true},
		{name: "angle brackets", input: "<vendor@example.com>", invalid: true},
		{name: "list", input: "a@example.com,b@example.com", invalid: true},
		{name: "comment", input: "vendor@example.com (Vendor)", invalid: true},
		{name: "newline", input: "vendor@exam\nple.com", invalid: true},
		{name: "consecutive dots", input: "ven..dor@example.com", invalid: true},
		{name: "long local part", input: strings.Repeat("a", 65) + "@example.com", invalid: true},
		{name: "length boundary", input: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61), want: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)},
		{name: "too long", input: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 190), invalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeEmail(tc.input)
			if tc.invalid {
				if err == nil {
					t.Fatal("expected invalid email")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if got != nil {
					t.Fatalf("expected nil, got %q", *got)
				}
			} else if got == nil || *got != tc.want {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}
