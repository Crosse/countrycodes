package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Crosse/countrycodes"
)

func main() {
	if len(os.Args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <country code>\n", os.Args[0])
		os.Exit(1)
	}

	for _, code := range os.Args[1:] {
		cc, ok := countrycodes.FindByISOAlpha(strings.ToUpper(code))
		if !ok {
			cc, ok = countrycodes.FindByName(code)
			if !ok {
				fmt.Fprintf(os.Stderr, "not an ISO-3166 country code or country name: %q\n", code)
				continue
			}
		}
		fmt.Printf("Name: %s\n", cc.Name)
		fmt.Printf("Business Region: %s\n", cc.BusinessRegion)
		fmt.Printf("ISO-3166-1 alpha-2 code: %s\n", cc.ISOAlpha2)
		fmt.Printf("ISO-3166-1 alpha-3 code: %s\n", cc.ISOAlpha3)
		fmt.Printf("ISO-3166-1 numeric code: %s\n", cc.ISONumeric)
		fmt.Printf("Languages: %s\n", strings.Join(cc.Languages, ", "))
		fmt.Printf("Capital: %s\n", cc.Capital)
		fmt.Printf("Continent: %s\n", cc.Continent)
		fmt.Printf("Currency: %s (%s)\n", cc.CurrencyName, cc.CurrencyCode)
		fmt.Printf("Population: %d\n", cc.Population)
		fmt.Printf("Area: %.0f sq. km\n", cc.Area)

		fmt.Println("Neighboring Countries:")
		for _, n := range cc.Neighbors {
			neigh := countrycodes.GetByISOAlpha2(n)
			fmt.Printf(" - %s (%s)\n", neigh.Name, neigh.ISOAlpha2)
		}
		fmt.Println()
	}
}
