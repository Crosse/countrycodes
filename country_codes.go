package countrycodes

import "strings"

// CountryCode represents a country's ISO codes and other information.
type CountryCode struct {
	// ISOAlpha2 is the 2-letter ISO-3166-1 alpha-2 code.
	ISOAlpha2 string
	// ISOAlpha3 is the 3-letter ISO-3166-1 alpha-3 code.
	ISOAlpha3 string
	// ISONumeric is the ISO-3166-1 numeric code.
	ISONumeric string
	// FIPSCode is the FIPS 10-4 country code.
	FIPSCode string
	// Name is the name of the country.
	Name string
	// Capital is the capital of the country.
	Capital string
	// Area is the area of the country in square kilometers.
	Area float64
	// Population is the population of the country.
	Population int
	// Continent is the continent the country is on.
	Continent string
	// TLD is the top-level domain of the country.
	TLD string
	// CurrencyCode is the ISO-4217 currency code.
	CurrencyCode string
	// CurrencyName is the name of the currency.
	CurrencyName       string
	Phone              string
	PostalCodeFormat   string
	PostalCodeRegex    string
	Languages          []string
	GeoNameID          string
	Neighbors          []string
	EquivalentFIPSCode string
	BusinessRegion     string
}

func FindByName(name string) (CountryCode, bool) {
	for _, c := range countryCodes {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return CountryCode{}, false
}

func FindByISOAlpha2(code string) (CountryCode, bool) {
	c, ok := countryCodes[code]
	return c, ok
}

func GetByISOAlpha2(code string) CountryCode {
	return countryCodes[code]
}

func FindByISOAlpha3(code string) (CountryCode, bool) {
	c, ok := isoLookup[code]
	return c, ok
}

func GetByISOAlpha3(code string) CountryCode {
	return isoLookup[code]
}

func FindByISOAlpha(code string) (CountryCode, bool) {
	var c CountryCode
	var ok bool
	if len(code) == 2 {
		c, ok = FindByISOAlpha2(code)
	} else if len(code) == 3 {
		c, ok = FindByISOAlpha3(code)
	}

	return c, ok
}

func GetByISOAlpha(code string) CountryCode {
	var c CountryCode
	if len(code) == 2 {
		c, _ = FindByISOAlpha2(code)
	} else if len(code) == 3 {
		c, _ = FindByISOAlpha3(code)
	}

	return c
}

func FindByLanguage(code string) ([]CountryCode, bool) {
	c, ok := languages[code]
	if !ok {
		return []CountryCode{}, false
	}

	return c, true
}

func FindByContinent(continent string) ([]CountryCode, bool) {
	c, ok := continents[continent]
	if !ok {
		return []CountryCode{}, false
	}

	return c, true
}

func FindByBusinessRegion(region string) ([]CountryCode, bool) {
	c, ok := businessRegions[region]
	if !ok {
		return []CountryCode{}, false
	}

	return c, true
}

//go:generate go run gen_geonames.go -fetch
