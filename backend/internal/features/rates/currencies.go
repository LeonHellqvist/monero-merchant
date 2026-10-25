package rates

import (
	"maps"
	"slices"
)

// Currencies the POS can use. All of them are published by Frankfurter.
var supported = map[string]bool{
	"AED": true, "ARS": true, "AUD": true, "BRL": true, "CAD": true,
	"CHF": true, "CLP": true, "CNY": true, "COP": true, "CZK": true,
	"DKK": true, "EUR": true, "GBP": true, "GEL": true, "HKD": true,
	"HUF": true, "IDR": true, "ILS": true, "INR": true, "JPY": true,
	"KES": true, "KRW": true, "MXN": true, "MYR": true, "NGN": true,
	"NOK": true, "NZD": true, "PEN": true, "PHP": true, "PLN": true,
	"RON": true, "RUB": true, "SEK": true, "SGD": true, "THB": true,
	"TRY": true, "UAH": true, "USD": true, "ZAR": true, "ZMW": true,
}

var SupportedCurrencies = slices.Sorted(maps.Keys(supported))
