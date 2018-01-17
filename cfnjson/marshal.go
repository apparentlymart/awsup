package cfnjson

import (
	"encoding/json"
	"fmt"

	"github.com/apparentlymart/awsup/eval"
	"github.com/hashicorp/hcl2/hcl"
)

func Marshal(template *eval.FlatTemplate) ([]byte, hcl.Diagnostics) {
	raw, diags := PrepareStructure(template)
	if diags.HasErrors() {
		return nil, diags
	}

	ret, err := json.Marshal(raw)
	if err != nil {
		// Should never happen, since PrepareStructure should always produce
		// something valid.
		panic(fmt.Errorf("PrepareStructure produced non-JSON-able data: %s", err))
	}
	return ret, diags
}
