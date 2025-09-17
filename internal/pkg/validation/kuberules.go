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
	"fmt"
	"regexp"
)

// validNamespace is a regex pattern for Kubernetes namespace validation
// Based on Kubernetes DNS-1123 subdomain naming rules
var validNamespace = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// validResourceName is a regex pattern for Kubernetes resource name validation
// Based on Kubernetes DNS-1123 subdomain naming rules with dots allowed
var validResourceName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)

// validPodPrefix is a regex pattern for Kubernetes pod prefix validation
// Based on Kubernetes DNS-1123 subdomain naming rules with dots allowed
var validPodPrefix = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// ValidateNamespace validates the namespace parameter for security
// Returns an error if the namespace is empty or doesn't match Kubernetes naming rules
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Kubernetes namespace naming rules: lowercase alphanumeric and hyphens, max 63 chars
	if !validNamespace.MatchString(namespace) || len(namespace) > 63 {
		return fmt.Errorf("invalid namespace format")
	}

	return nil
}

// ValidateResourceName validates Kubernetes resource names for security
// Returns an error if the name is empty, too long, or doesn't match Kubernetes naming rules
func ValidateResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	if len(name) > 253 {
		return fmt.Errorf("resource name too long: %d characters", len(name))
	}

	// Kubernetes resource naming rules: lowercase alphanumeric, hyphens, and dots
	if !validResourceName.MatchString(name) {
		return fmt.Errorf("invalid resource name format")
	}

	return nil
}

// ValidateRebalanceRate validates the rebalance rate parameter
// Returns an error if the rate is not between 0 and 1.0 (inclusive)
func ValidateRebalanceRate(rate float32) error {
	if rate < 0 || rate > 1.0 {
		return fmt.Errorf("invalid rebalance rate: %f (must be between 0 and 1.0)", rate)
	}
	return nil
}

// ValidatePodPrefix validates the pod prefix parameter for security
// Returns an error if the prefix is empty, too long, or doesn't match Kubernetes naming rules
func ValidatePodPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("prefix cannot be empty")
	}

	if len(prefix) > 50 {
		return fmt.Errorf("prefix too long: %d characters", len(prefix))
	}

	// Kubernetes pod naming rules: lowercase alphanumeric, hyphens, and dots
	if !validPodPrefix.MatchString(prefix) {
		return fmt.Errorf("invalid prefix format")
	}

	return nil
}
