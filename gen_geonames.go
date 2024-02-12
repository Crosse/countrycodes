//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	countryInfoURL      = "https://download.geonames.org/export/dump/countryInfo.txt"
	businessRegionsJSON = "business_regions.json"
)

type column struct {
	name         string
	dataType     string
	structMember string
}

var columns = []column{
	{name: "ISOAlpha2"},
	{name: "ISOAlpha3"},
	{name: "ISONumeric"},
	{name: "FIPSCode"},
	{name: "Name"},
	{name: "Capital"},
	{name: "Area", dataType: "float64"},
	{name: "Population", dataType: "int"},
	{name: "Continent"},
	{name: "TLD"},
	{name: "CurrencyCode"},
	{name: "CurrencyName"},
	{name: "Phone"},
	{name: "PostalCodeFormat"},
	{name: "PostalCodeRegex"},
	{name: "Languages", dataType: "[]string"},
	{name: "GeoNameID"},
	{name: "Neighbors", dataType: "[]string"},
	{name: "EquivalentFIPSCode"},
	{name: "BusinessRegion"},
}

var (
	fetch        bool
	formatOutput bool
)

func main() {
	flag.BoolVar(&fetch, "fetch", false, "download the latest country info file")
	flag.BoolVar(&formatOutput, "format", true, "format the output using gofmt")
	flag.Parse()

	countryInfoFile := path.Base(countryInfoURL)

	var fetchTime time.Time

	if fetch {
		if err := download(countryInfoURL, countryInfoFile); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fetchTime = time.Now()
	}

	ccFile, err := os.Open(countryInfoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open %s: %v", countryInfoFile, err)
	}
	defer ccFile.Close()

	countries, err := parseCountryCodes(ccFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	regionsFile, err := os.Open(businessRegionsJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open %s: %v", businessRegionsJSON, err)
	}
	defer regionsFile.Close()

	countries, err = parseBusinessRegions(regionsFile, countries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := writeCountryCodes(fetchTime, countries); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, fn := range [](func([]map[string]interface{}) error){
		writeLanguages,
		writeContinents,
		writeISOLookup,
		writeBusinessRegions,
	} {
		if err := fn(countries); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

func download(url, fileName string) error {
	fmt.Printf("downloading %s to %s\n", url, fileName)

	client := http.DefaultClient

	res, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading: %v\n", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got non-200 status when downloading: %d\n", res.StatusCode)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", fileName, err)
	}
	defer f.Close()

	written, err := io.Copy(f, res.Body)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	} else if res.ContentLength > 0 && written != res.ContentLength {
		return fmt.Errorf("failed to write all data to file: %d != %d", written, res.ContentLength)
	}

	return nil
}

func parseCountryCodes(file *os.File) ([]map[string]interface{}, error) {
	fmt.Println("parsing country info file")

	var err error
	var data []map[string]interface{}

	re := regexp.MustCompile(`^(#.*|\s*)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if re.MatchString(line) {
			// blank line or comment
			continue
		}

		country := make(map[string]interface{})

		fields := strings.Split(line, "\t")
		for i, field := range fields {
			col := columns[i]
			if col.name == "" {
				continue
			}

			switch col.dataType {
			case "int":
				country[col.name], err = strconv.Atoi(field)
				if err != nil {
					return nil, fmt.Errorf("error parsing value %q for field %s: %v", field, col.name, err)
				}
			case "float64":
				country[col.name], err = strconv.ParseFloat(field, 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing value %q for field %s: %v", field, col.name, err)
				}
			case "[]string":
				if field == "" {
					country[col.name] = []string{}
				} else {
					country[col.name] = strings.Split(field, ",")
				}
			case "", "string":
				country[col.name] = field
			default:
				return nil, fmt.Errorf("unhandled type %s", col.dataType)
			}
		}
		data = append(data, country)
	}
	return data, nil
}

func parseBusinessRegions(file *os.File, countries []map[string]interface{}) ([]map[string]interface{}, error) {
	fmt.Println("parsing business regions")

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var regions map[string]string
	if err := json.Unmarshal(data, &regions); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	for _, country := range countries {
		iso2 := country["ISOAlpha2"].(string)
		if region, ok := regions[iso2]; !ok {
			return nil, fmt.Errorf("region missing for %q", iso2)
		} else {
			country["BusinessRegion"] = region
		}
	}

	return countries, nil
}

func writeCountryCodes(fetchTime time.Time, countries []map[string]interface{}) error {
	const filename = "country_code_data.go"

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf(`
package countrycodes

// Code generated by gen_geonames.go; DO NOT EDIT.

import "time"

const NumCountries = %d

func FetchTime() time.Time {
	return time.Unix(%d, 0) // %s
}

var countryCodes map[string]CountryCode

func init() {
	countryCodes = map[string]CountryCode{
`,
		len(countries),
		fetchTime.UTC().Unix(),
		fetchTime.UTC().Format(time.RFC3339)))

	for _, country := range countries {
		buf.WriteString(fmt.Sprintf("\t%q: {\n", country["ISOAlpha2"]))

		var structFields []string
		for k, v := range country {
			sb := strings.Builder{}
			sb.WriteString(fmt.Sprintf("\t\t%s: ", k))
			switch val := v.(type) {
			case int:
				sb.WriteString(fmt.Sprintf("%d", val))
			case float32, float64:
				sb.WriteString(fmt.Sprintf("%f", val))
			case []string:
				sb.WriteString("[]string{")
				l := []string{}
				for _, s := range val {
					l = append(l, fmt.Sprintf("%q", strings.TrimSpace(s)))
				}
				sb.WriteString(strings.Join(l, ","))
				sb.WriteString("}")
			case string:
				sb.WriteString(fmt.Sprintf("%q", strings.TrimSpace(val)))
			default:
				return fmt.Errorf("unhandled type %T", val)
			}
			sb.WriteString(",\n")
			structFields = append(structFields, sb.String())
		}
		slices.SortFunc(structFields, func(i, j string) int {
			field1 := strings.TrimSpace(strings.Split(i, ":")[0])
			field2 := strings.TrimSpace(strings.Split(j, ":")[0])

			if field1 == "Name" {
				return -1000
			} else if field2 == "Name" {
				return 1000
			}

			var idx1, idx2 int
			for idx, column := range columns {
				if column.name == field1 {
					idx1 = idx
				}
				if column.name == field2 {
					idx2 = idx
				}
			}
			return idx1 - idx2

		})

		for _, s := range structFields {
			buf.WriteString(s)
		}
		buf.WriteString("\t},\n")
	}
	buf.WriteString("}\n\n")
	buf.WriteString("}\n\n")

	return writeFile(filename, &buf, formatOutput)
}

func writeLanguages(countries []map[string]interface{}) error {
	const filename = "languages.go"

	var buf bytes.Buffer

	buf.WriteString(`
package countrycodes

// Code generated by gen_geonames.go; DO NOT EDIT.

var languages map[string][]CountryCode

func init() {
	languages = map[string][]CountryCode{
`)

	languages := make(map[string][]string)

	for _, country := range countries {
		langs, ok := country["Languages"].([]string)
		if !ok {
			return fmt.Errorf("Languages field for country %s was not a string slice", country["ISOAlpha2"])
		}
		for _, lang := range langs {
			languages[lang] = append(languages[lang], country["ISOAlpha2"].(string))
		}
	}

	for lang, speakers := range languages {
		buf.WriteString(fmt.Sprintf("\t%q: {", lang))
		l := []string{}
		for _, speaker := range speakers {
			l = append(l, fmt.Sprintf("countryCodes[%q]", strings.TrimSpace(speaker)))
		}
		buf.WriteString(strings.Join(l, ","))
		buf.WriteString("},\n")
	}
	buf.WriteString("}\n}\n")

	return writeFile(filename, &buf, formatOutput)
}

func writeContinents(countries []map[string]interface{}) error {
	const filename = "continents.go"

	var buf bytes.Buffer

	buf.WriteString(`
package countrycodes

// Code generated by gen_geonames.go; DO NOT EDIT.

var continents map[string][]CountryCode

func init() {
	continents = map[string][]CountryCode{
`)

	continents := make(map[string][]string)

	for _, country := range countries {
		// build up the continents map
		continent, ok := country["Continent"].(string)
		if !ok {
			return fmt.Errorf("Continent field for country %s was not a string", country["ISOAlpha2"])
		}
		continents[continent] = append(continents[continent], country["ISOAlpha2"].(string))
	}

	for continent, countries := range continents {
		buf.WriteString(fmt.Sprintf("\t%q: {", continent))
		l := []string{}
		for _, country := range countries {
			l = append(l, fmt.Sprintf("countryCodes[%q]", strings.TrimSpace(country)))
		}
		buf.WriteString(strings.Join(l, ","))
		buf.WriteString("},\n")
	}
	buf.WriteString("}\n}\n")

	return writeFile(filename, &buf, formatOutput)
}

func writeISOLookup(countries []map[string]interface{}) error {
	const filename = "country_lookups.go"

	var buf bytes.Buffer

	buf.WriteString(`
package countrycodes

// Code generated by gen_geonames.go; DO NOT EDIT.

var isoLookup map[string]CountryCode

func init() {
	isoLookup = map[string]CountryCode{
`)

	for _, country := range countries {
		buf.WriteString(fmt.Sprintf(
			"\t%q: countryCodes[%q],\n",
			country["ISOAlpha3"], country["ISOAlpha2"]))

	}
	buf.WriteString("}\n}\n")

	return writeFile(filename, &buf, formatOutput)
}

func writeBusinessRegions(countries []map[string]interface{}) error {
	const filename = "business_regions.go"

	var buf bytes.Buffer

	buf.WriteString(`
package countrycodes

// Code generated by gen_geonames.go; DO NOT EDIT.
// Originally taken from https://gist.github.com/richjenks/15b75f1960bc3321e295

var businessRegions map[string][]CountryCode

func init() {
	businessRegions = map[string][]CountryCode{
`)

	businessRegions := make(map[string][]string)

	for _, country := range countries {
		if country["BusinessRegion"] == nil {
			country["BusinessRegion"] = ""
		}

		region, ok := country["BusinessRegion"].(string)
		if !ok {
			return fmt.Errorf("BusinessRegions field for country %s was not a string", country["ISOAlpha2"])
		}
		businessRegions[region] = append(businessRegions[region], country["ISOAlpha2"].(string))

	}

	for region, countries := range businessRegions {
		buf.WriteString(fmt.Sprintf("\t%q: {", region))
		l := []string{}
		for _, country := range countries {
			l = append(l, fmt.Sprintf("countryCodes[%q]", strings.TrimSpace(country)))
		}
		buf.WriteString(strings.Join(l, ","))
		buf.WriteString("},\n")
	}

	buf.WriteString("}\n}\n")

	return writeFile(filename, &buf, formatOutput)
}

func writeFile(filename string, buf *bytes.Buffer, shouldFormat bool) error {
	var err error
	var final []byte

	fmt.Printf("writing out file %s\n", filename)

	if shouldFormat {
		final, err = format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("error formatting generated code for %s: %v", filename, err)
		}
	} else {
		final = buf.Bytes()
	}

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer outFile.Close()

	written, err := outFile.Write(final)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %v", filename, err)
	}
	if written != len(final) {
		return fmt.Errorf(
			"failed to write all data to file %s (%d != %d)", filename,
			written, len(final))
	}

	return nil
}
