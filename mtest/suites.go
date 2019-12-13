package mtest

import . "github.com/onsi/ginkgo"

// FunctionsSuite is a test suite that tests small test cases
func FunctionsSuite() {
	Context("ckecli", TestCKECLI)
	Context("kubernetes", TestKubernetes)
}

// OperatorsSuite is a test suite that tests CKE operators
func OperatorsSuite() {
	Context("operators", func() {
		TestOperators(false)
	})
}

// RobustnessSuite is a test suite that tests CKE operators with SSH-not-connected nodes
func RobustnessSuite() {
	Context("operators", func() {
		TestStopCP()
		TestOperators(true)
	})
}

// UpgradeSuite is a test suite that reboots all nodes and upgrade CKE.
func UpgradeSuite() {
	Context("upgrade", func() {
		TestUpgrade()
		Context("kubernetes", TestKubernetes)
	})
}
