package stellarnet

import (
	"testing"

	"github.com/stellar/go/build"
	"github.com/stellar/go/xdr"
)

func TestVerifyEnvelope(t *testing.T) {
	SetNetwork(build.PublicNetwork)
	defer SetNetwork(build.TestNetwork)
	b64 := "AAAAAANjzBWOC6YJo49wLshbTPMAmHnZ1I5AESV73e605u3DAAAnEAAAAAAAAAAAAAAAAQAAAABdRIV+AAAAAF1EhqoAAAAAAAAAAQAAAAEAAAAAc35v3HkfCY0CYiA898rk/9hkUeNCTCneeOKQHyo1HJcAAAAKAAAAEFN0ZWxsYXJwb3J0IGF1dGgAAAABAAAAQMCsw7hA+QQnW9t2MfAU92Sqa7eD1udjvaS5BSO9AJFXuELyBmzw+l+GhIry01cM6nz5HKleHf+wDn2jXYYlFKQAAAAAAAAAAbTm7cMAAABAnoRu4cp4cl9UEYqyRIfAIiLhoSU7h77vU9yV2S1RSNZfhc/YaXlMnlLkb9CAeLho1nVMOQnGNzQ55gWJzXXQDQ=="
	var env xdr.TransactionEnvelope
	if err := xdr.SafeUnmarshalBase64(b64, &env); err != nil {
		t.Fatal(err)
	}
	if err := VerifyEnvelope(env); err != nil {
		t.Fatal(err)
	}
}
