package mtest

import . "github.com/onsi/ginkgo"

// FunctionsSuite is a test suite that tests small test cases
var FunctionsSuite = func() {
	Context("ckecli", TestCKECLI)
	Context("kubernetes", TestKubernetes)
}

// OperatorsSuite is a test suite that tests CKE operators
var OperatorsSuite = func() {
	Context("operators", TestOperators)
}

// RobustnessSuite is a test suite that tests CKE operators with SSH-not-connected nodes
var RobustnessSuite = func() {
	Context("operators", func() {
		TestStopCP()
		TestOperators()
	})
}
