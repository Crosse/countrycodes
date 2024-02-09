package countrycodes

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaughtEmAll(t *testing.T) {
	require.Len(t, countryCodes, NumCountries)
}

func TestFindByISOAlpha(t *testing.T) {
	for iso2, v := range countryCodes {
		require.EqualValues(t, v.ISOAlpha2, iso2)

		cc, ok := FindByISOAlpha2(iso2)
		require.True(t, ok)
		require.Equal(t, v, cc)
		require.Equal(t, iso2, cc.ISOAlpha2)

		cc, ok = FindByISOAlpha3(v.ISOAlpha3)
		require.True(t, ok)
		require.Equal(t, v, cc)

		cc, ok = FindByISOAlpha(iso2)
		require.True(t, ok)
		require.Equal(t, v, cc)

		cc, ok = FindByISOAlpha(v.ISOAlpha3)
		require.True(t, ok)
		require.Equal(t, v, cc)

		cc, ok = FindByISOAlpha2(v.ISOAlpha3)
		require.False(t, ok)
		require.Empty(t, cc)

		cc, ok = FindByISOAlpha3(v.ISOAlpha2)
		require.False(t, ok)
		require.Empty(t, cc)
	}

	for _, fn := range []func(string) (CountryCode, bool){
		FindByISOAlpha2,
		FindByISOAlpha3,
		FindByISOAlpha,
	} {
		name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
		t.Run(name, func(t *testing.T) {
			cc, ok := fn("XX")
			require.False(t, ok)
			require.Empty(t, cc)
		})
	}
}

func BenchmarkFindByISOAlpha(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FindByISOAlpha("US")
	}
}

func BenchmarkFindByISOAlpha2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FindByISOAlpha2("US")
	}
}

func BenchmarkFindByISOAlpha3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FindByISOAlpha3("USA")
	}
}
