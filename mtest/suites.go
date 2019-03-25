package mtest

import . "github.com/onsi/ginkgo"

// FunctionsSuite is a test suite that tests small test cases
var FunctionsSuite = func() {
	Context("ckecli", TestCKECLI)
	Context("kubernetes", TestKubernetes)
}

// OperatorsAllSuite is a test suite that tests all CKE operators
var OperatorsAllSuite = func() {
	Context("operators all", TestOperatorsAll)
}

// OperatorsMiscSuite is a test suite that tests miscellaneous CKE operators
var OperatorsMiscSuite = func() {
	Context("operators misc", TestOperatorsMisc)
}
