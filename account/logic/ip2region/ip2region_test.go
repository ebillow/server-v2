package ip2region

import (
	"testing"
)

func TestGetCountry(t *testing.T) {
	var ip = "103.142.140.132"
	country, err := GetCountry(ip)
	t.Log(country)

	country, err = GetCountry("127.0.0.1")
	t.Log(country, err)
}
