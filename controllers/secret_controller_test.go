package controllers

import "testing"

func TestHashString(t *testing.T) {
	url1 := "ssh://git@git.belastingdienst.nl:7999/cpet/opr-paas.git"
	url2 := "ssh://git@git.belastingdienst.nl:7999/cpet/opr-strimzi.git"
	r1 := hashString(url1)
	r2 := hashString(url2)

	if string(r1) == string(r2) {
		t.Errorf("controller.hashString does not produce unique strings")
	}
}
