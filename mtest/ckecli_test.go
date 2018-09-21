package mtest

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should issue client certificate for etcd and connect to the CKE managed etcd", func() {
		By("execute ckecli etcd user-add")
		stdout := ckecli("etcd", "user-add", "mtest")
		type response struct {
			Crt string `json:"certificate"`
			Key string `json:"private_key"`
		}
		var res response
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("execute etcdctl")
		c := localTempFile(res.Crt)
		k := localTempFile(res.Key)
		_, _, err = etcdctl(host1, c.Name(), k.Name(), "endpoint", "health")
		Expect(err).NotTo(HaveOccurred())
	})
})
