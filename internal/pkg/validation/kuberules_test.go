/*
MIT License

Copyright (c) 2019-2024 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{name: "Empty", namespace: "", wantErr: true},
		{name: "Invalid", namespace: "Invalid_Namespace", wantErr: true},
		{name: "Valid", namespace: "valid-namespace", wantErr: false},
		{name: "ValidSingleChar", namespace: "a", wantErr: false},
		{name: "ValidWithNumbers", namespace: "namespace123", wantErr: false},
		{name: "InvalidStartWithHyphen", namespace: "-invalid", wantErr: true},
		{name: "InvalidEndWithHyphen", namespace: "invalid-", wantErr: true},
		{name: "InvalidUpperCase", namespace: "Invalid", wantErr: true},
		{name: "TooLong", namespace: "a1234567890123456789012345678901234567890123456789012345678901234567890", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.namespace)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNamespace_ErrorMessages(t *testing.T) {
	// Test specific error messages match existing behavior
	err := ValidateNamespace("")
	assert.Equal(t, "namespace cannot be empty", err.Error())

	err = ValidateNamespace("invalid_namespace")
	assert.Equal(t, "invalid namespace format", err.Error())

	err = ValidateNamespace("a1234567890123456789012345678901234567890123456789012345678901234567890")
	assert.Equal(t, "invalid namespace format", err.Error())
}

func TestValidateResourceName(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		wantErr      bool
	}{
		{name: "Empty", resourceName: "", wantErr: true},
		{name: "Valid", resourceName: "valid-name", wantErr: false},
		{name: "ValidWithDots", resourceName: "app.v1.service", wantErr: false},
		{name: "ValidSingleChar", resourceName: "a", wantErr: false},
		{name: "ValidWithNumbers", resourceName: "app123", wantErr: false},
		{name: "InvalidStartWithHyphen", resourceName: "-invalid", wantErr: true},
		{name: "InvalidEndWithHyphen", resourceName: "invalid-", wantErr: true},
		{name: "InvalidUpperCase", resourceName: "Invalid", wantErr: true},
		{name: "InvalidSpecialChars", resourceName: "invalid_name", wantErr: true},
		{name: "TooLong", resourceName: "a" + strings.Repeat("b", 253), wantErr: true},
		{name: "ValidMaxLength", resourceName: "a" + strings.Repeat("b", 251) + "c", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceName(tt.resourceName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResourceName_ErrorMessages(t *testing.T) {
	// Test specific error messages
	err := ValidateResourceName("")
	assert.Equal(t, "resource name cannot be empty", err.Error())

	err = ValidateResourceName("invalid_name")
	assert.Equal(t, "invalid resource name format", err.Error())

	longName := "a" + strings.Repeat("b", 254)
	err = ValidateResourceName(longName)
	assert.Contains(t, err.Error(), "resource name too long")
}

func TestValidateRebalanceRate(t *testing.T) {
	tests := []struct {
		name    string
		rate    float32
		wantErr bool
	}{
		{name: "ValidZero", rate: 0.0, wantErr: false},
		{name: "ValidHalf", rate: 0.5, wantErr: false},
		{name: "ValidOne", rate: 1.0, wantErr: false},
		{name: "ValidQuarter", rate: 0.25, wantErr: false},
		{name: "InvalidNegative", rate: -0.1, wantErr: true},
		{name: "InvalidOverOne", rate: 1.1, wantErr: true},
		{name: "InvalidLarge", rate: 2.0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRebalanceRate(tt.rate)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRebalanceRate_ErrorMessages(t *testing.T) {
	// Test specific error messages
	err := ValidateRebalanceRate(-0.1)
	assert.Equal(t, "invalid rebalance rate: -0.100000 (must be between 0 and 1.0)", err.Error())

	err = ValidateRebalanceRate(1.5)
	assert.Equal(t, "invalid rebalance rate: 1.500000 (must be between 0 and 1.0)", err.Error())
}

func TestValidatePodPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{name: "Empty", prefix: "", wantErr: true},
		{name: "Valid", prefix: "valid-prefix", wantErr: false},
		{name: "ValidWithDots", prefix: "app.v1", wantErr: false},
		{name: "ValidSingleChar", prefix: "a", wantErr: false},
		{name: "ValidWithNumbers", prefix: "app123", wantErr: false},
		{name: "ValidEmptyOptional", prefix: "", wantErr: true}, // This should fail
		{name: "InvalidStartWithHyphen", prefix: "-invalid", wantErr: true},
		{name: "InvalidEndWithHyphen", prefix: "invalid-", wantErr: true},
		{name: "InvalidUpperCase", prefix: "Invalid_Prefix", wantErr: true},
		{name: "InvalidSpecialChars", prefix: "invalid_name", wantErr: true},
		{name: "TooLong", prefix: "a" + strings.Repeat("b", 50), wantErr: true},
		{name: "ValidMaxLength", prefix: "a" + strings.Repeat("b", 48) + "c", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePodPrefix(tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePodPrefix_ErrorMessages(t *testing.T) {
	// Test specific error messages
	err := ValidatePodPrefix("")
	assert.Equal(t, "prefix cannot be empty", err.Error())

	err = ValidatePodPrefix("invalid_name")
	assert.Equal(t, "invalid prefix format", err.Error())

	longPrefix := "a" + strings.Repeat("b", 51)
	err = ValidatePodPrefix(longPrefix)
	assert.Contains(t, err.Error(), "prefix too long")
}
