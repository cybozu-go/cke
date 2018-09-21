package mtest

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should connect vault and etcd", func() {
		By("execute ckecli etcd user-add")
		Eventually(func() error {
			stdout, _, err := execAt(host1, "/data/ckecli", "etcd", "user-add", "mtest")
			if err != nil {
				return err
			}
			type response struct {
				Crt string `json:"certificate"`
				Key string `json:"private_key"`
			}
			var res response
			err = json.Unmarshal(stdout, &res)
			if err != nil {
				return err
			}
			return nil
		}).Should(Succeed())
	})
})
